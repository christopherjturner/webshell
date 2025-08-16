package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
	// "webshell/ttyrec"
)

const (
	shell              = "/bin/bash"
	maxBufferSizeBytes = 1024 * 256
)

type Shell struct {
	config   Config
	timeout  Timeout
	sessions *SessionManager
}

func (s Shell) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Accept the WS connection
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// TODO: look at better ways of getting the id
	id := r.URL.Path

	newProc := func() (*ShellProcess, error) {
		shellProcess := &ShellProcess{}
		err = shellProcess.Start(shell)
		return shellProcess, err
	}

	// Request the session
	logger.Info(fmt.Sprintf("Getting session %s", id))
	sess, err := s.sessions.GetSession(id, newProc)
	if err != nil {
		logger.Error(err.Error())
		_ = ws.Close(websocket.StatusInternalError, "Failed to start session")
		return
	}

	// Attach the client
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	_ = sess.Attach(ctx, ws)
	logger.Info(fmt.Sprintf("Attached websocket to session %s", id))
	defer sess.Detach()

	for {
		_, b, err := ws.Read(ctx)
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

				if err := sess.Resize(uint16(rows), uint16(cols)); err != nil {
					logger.Warn(fmt.Sprintf("Failed to resize tty, error: %s", err))
				}
				continue
			}

			logger.Info("Unknown special payload " + specialPayload)
		}

		// Send user input to shell process
		_, err = sess.WriteToTTY(b)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to write to TTY: %s", err))
		}
	}
}

// WebShell's websocket handler
func (s Shell) shellHandler(ctxReq context.Context, ws *websocket.Conn, shellProc *ShellProcess) {

	logger.Info("New webshell session")

	ctxLocal, cancelLocal := context.WithCancel(ctxReq)
	defer cancelLocal()

	var wg sync.WaitGroup
	wg.Add(2)
	activeConnections.Add(1)
	defer activeConnections.Done()

	// Gracefully stop the session
	defer func() {
		logger.Info("Stopping webshell")
		cancelLocal()
		wg.Wait() // Wait for both goroutines to finish.

		if err := shellProc.Kill(); err != nil {
			logger.Error("Failed to kill shell process")
		}
		if err := ws.Close(websocket.StatusNormalClosure, "Connection closed"); err != nil {
			logger.Error(fmt.Sprintf("Failed to close websocket: %s", err))
		}
	}()

	// Shell -> User
	go func() {
		defer wg.Done()
		buffer := make([]byte, maxBufferSizeBytes)
		for {
			l, err := shellProc.Read(buffer)
			if err != nil {
				logger.Error(fmt.Sprintf("err from shellProc: %s", err))
				ws.Write(ctxLocal, websocket.MessageBinary, []byte("Session Ended"))
				break
			}

			if err := ws.Write(ctxLocal, websocket.MessageBinary, buffer[:l]); err != nil {
				logger.Error("Failed to forward tty to ws %s", err)
			}
		}
	}()

	// WS -> TTY
	go func() {
		defer wg.Done()
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
