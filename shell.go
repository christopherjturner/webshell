package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"

	"webshell/ttyrec"
)

const (
	shell              = "/bin/bash"
	maxBufferSizeBytes = 1024 * 256
)

type Shell struct {
	config  Config
	timeout Timeout
}

func (s Shell) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Accept the WS connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// Start shell process
	shellProcess := &ShellProcess{}
	err = shellProcess.Start(shell)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// Attach auditing if required
	if s.config.AuditTTY {
		timestamp := time.Now().Format(time.RFC3339)
		auditFile := fmt.Sprintf("%s_%s.tty.audit", timestamp, s.config.Token)
		recorder, err := ttyrec.NewRecorder(s.config.AuditPath, auditFile)
		if err != nil {
			logger.Error(err.Error())
			http.Error(w, "Audit setup failed", http.StatusInternalServerError)
			return
		}
		shellProcess.WithTTYRecorder(recorder)
		logger.Info(fmt.Sprintf("Recording TTY data to %s/%s", s.config.AuditPath, auditFile))
		defer recorder.Save()
	}

	if s.config.AuditExec {
		if err := shellProcess.WithAuditing(); err != nil {
			logger.Error(err.Error())
			http.Error(w, "Audit setup failed", http.StatusInternalServerError)
			return
		}
	}

	// Pass to websocket handler
	s.timeout.Start()
	s.shellHandler(r.Context(), conn, shellProcess)

}

// WebShell's websocket handler
func (s Shell) shellHandler(ctxReq context.Context, ws *websocket.Conn, shellProc *ShellProcess) {

	logger.Info("New webshell session")

	ctxLocal, cancelLocal := context.WithCancel(ctxReq)
	defer cancelLocal()

	var wg sync.WaitGroup
	wg.Add(1)
	activeConnections.Add(1)
	defer activeConnections.Done()

	// Gracefully stop the session
	defer func() {
		logger.Info("Stopping webshell")
		if err := shellProc.Kill(); err != nil {
			logger.Error("Failed to kill shell process")
		}
		if err := ws.Close(websocket.StatusNormalClosure, "Connection closed"); err != nil {
			logger.Error(fmt.Sprintf("Failed to close websocket: %s", err))
		}
	}()

	// Shell -> User
	go func() {
		buffer := make([]byte, maxBufferSizeBytes)
		for {
			l, err := shellProc.Read(buffer)
			if err != nil {
				ws.Write(ctxLocal, websocket.MessageText, []byte("Session Ended"))
				break
			}

			if err := ws.Write(ctxLocal, websocket.MessageText, buffer[:l]); err != nil {
				logger.Error("Failed to forward tty to ws")
			}
		}
		wg.Done()
	}()

	// User -> Shell
	go func() {

		// Kill the shell when the reader exits. TODO: can do move this outside the go routines?
		defer shellProc.Kill()

		for {
			_, b, err := ws.Read(ctxLocal)
			if err != nil {
				logger.Warn(fmt.Sprintf("Websocket closed: %s", err))
				break
			}

			s.timeout.Ping()

			b = bytes.Trim(b, "\x00")

			if len(b) == 0 {
				continue
			}

			// Special purpose payloads.
			if b[0] == 1 {
				specialPayload := string(bytes.Trim(b[1:], " \n\r\t\x00\x01"))

				if specialPayload == "PING" {
					continue
				}

				// Resize payload (SIZE COL ROW)
				if strings.HasPrefix(specialPayload, "SIZE") {
					fields := strings.Fields(specialPayload)
					if len(fields) != 3 {
						logger.Error("Invalid resize payload: " + specialPayload)
						continue
					}

					cols, errCol := strconv.ParseInt(fields[1], 10, 16)
					rows, errRow := strconv.ParseInt(fields[2], 10, 16)

					if errCol != nil || errRow != nil {
						logger.Error("Invalid resize payload: " + specialPayload)
						continue
					}

					logger.Debug(fmt.Sprintf("Resizing tty to use %d rows and %d columns...", rows, cols))

					if err := pty.Setsize(shellProc.tty, &pty.Winsize{
						Rows: uint16(rows),
						Cols: uint16(cols),
					}); err != nil {
						logger.Warn(fmt.Sprintf("Failed to resize tty, error: %s", err))
					}
					continue
				}

				logger.Info("Unknown special payload " + specialPayload)
			}

			// Send user input to shell process
			_, err = shellProc.Write(b)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to write to TTY: %s", err))
			}
		}

	}()

	// Stop the handler if the global context is cancelled
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-globalCtx.Done():
				logger.Debug("Cancelling Websocket Handler")
				if err := shellProc.Kill(); err != nil {
					logger.Error(err.Error())
				}
				if err := ws.CloseNow(); err != nil {
					logger.Error(err.Error())
				}
				return
			case <-ctxLocal.Done():
				logger.Info("Request Cancelled")
				return
			case <-ticker.C:
				s.timeout.Ping()
			}
		}
	}()

	wg.Wait()
}
