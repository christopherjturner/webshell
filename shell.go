package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/net/websocket"

	"webshell/strace"
	"webshell/ttyrec"
)

const shell = "/bin/bash"
const maxBufferSizeBytes = 1024 * 256

type TTYSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	X    uint16 `json:"x"`
	Y    uint16 `json:"y"`
}

func runAs(cmd *exec.Cmd, user *user.User) {
	uid, _ := strconv.ParseInt(user.Uid, 10, 32)
	gid, _ := strconv.ParseInt(user.Gid, 10, 32)

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", user.HomeDir))
}

func filterEnv(o []string) []string {
	environ := []string{}
	for _, e := range o {
		key, _, _ := strings.Cut(e, "=")
		if _, found := restrictedEnvVars[key]; !found {
			environ = append(environ, e)
		}
	}
	return environ
}

func shellHandler(ws *websocket.Conn) {

	ctxLocal, cancelLocal := context.WithCancel(ws.Request().Context())
	logger.Info("New webshell session started")
	var err error

	// Start the shell
	cmd := exec.Command(shell)
	cmd.Env = filterEnv(os.Environ())
	cmd.Dir = config.HomeDir

	if config.User != nil {
		logger.Info(fmt.Sprintf("Running %s as %s", shell, config.User.Username))
		runAs(cmd, config.User)
	}

	tty, err := pty.Start(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start %s: %s", shell, err))
		websocket.Message.Send(ws, "Failed to start shell")
		return
	}

	activeConnections.Add(1)
	var wg sync.WaitGroup
	wg.Add(1)

	// TTY Recorder
	// When TTY Recorder is disabled we use a non-functioning version of it in its place.
	// Avoids having to scatter if statements all of the place etc.
	var recorder ttyrec.TTYRecorder = &ttyrec.NoOpRecorder{}

	if config.AuditTTY {
		// TTY recordings are written down to "<token>-<pid>.audit".
		auditFile := fmt.Sprintf("%s_%d.tty.audit", config.Token, cmd.Process.Pid)
		recorder, err = ttyrec.NewRecorder(config.AuditPath, auditFile)
		if err != nil {
			logger.Error(fmt.Sprintf("TTYRec failed to start: %v", err))
			return
		}

		logger.Info("TTY auditing is enabled")
	}

	// Syscall Exec auditing
	// Requires strace to be installed.
	// AuditExec will error if its enabled and strace is not found.
	if config.AuditExec {
		straceAudit := strace.NewStraceLogger(auditLogger)
		if err := straceAudit.Attach(cmd.Process.Pid); err != nil {
			logger.Error(fmt.Sprintf("Syscall auditing failed to start: %v", err))
			return
		}

		logger.Info("Syscall auditing is enabled")
	}

	// Gracefully stop the session
	defer func() {
		logger.Info(fmt.Sprintf("Stopping shell PID %d", cmd.Process.Pid))
		if err := cmd.Process.Kill(); err != nil {
			logger.Error(fmt.Sprintf("failed to stop process: %s", err))
		}

		if _, err := cmd.Process.Wait(); err != nil {
			logger.Error(fmt.Sprintf("Failed to wait process: %s", err))
		}

		if err := tty.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close tty: %s", err))
		}

		if err := ws.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close websocket: %s", err))
		}

		// Save the TTY recording.
		if err := recorder.Save(); err != nil {
			logger.Error(fmt.Sprintf("Failed to save ttyrec: %s", err))
		}
		logger.Info("Audit file written")

		if err := recorder.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close TTYRecorder: %s", err))
		}
		activeConnections.Done()
	}()

	// Shell -> User
	go func() {
		buffer := make([]byte, maxBufferSizeBytes)

		for {
			l, err := tty.Read(buffer)
			if err != nil {
				websocket.Message.Send(ws, "session ended")
				wg.Done()
				return
			}

			if err := websocket.Message.Send(ws, buffer[:l]); err != nil {
				logger.Error("failed to forward tty to ws")
				continue
			}

			// Copy data to TTY Recorder.
			// TODO: maybe use a TeeReader instead
			if _, err := recorder.Write(buffer[:l]); err != nil {
				logger.Error("failed record tty to ws")
				continue
			}
		}

		// Close the socket, this will stop User -> Shell
		ws.Close()
	}()

	// User -> Shell
	go func() {
		buffer := make([]byte, maxBufferSizeBytes)
		for {
			if err = websocket.Message.Receive(ws, &buffer); err != nil {
				logger.Warn(fmt.Sprintf("Websocket closed: %s", err))
				break
			}

			b := bytes.Trim(buffer, "\x00")

			// Special purpose payloads.
			if b[0] == 1 {
				specialPayload := bytes.Trim(b[1:], " \n\r\t\x00\x01")

				if len(specialPayload) == 0 {
					continue
				}

				// Response to ping messages
				if string(specialPayload) == "PING" {
					logger.Debug("PING")
					continue
				}

				// TODO: maybe do a non-json version of this payload?
				ttySize := &TTYSize{}

				if err := json.Unmarshal(specialPayload, ttySize); err != nil {
					logger.Warn(fmt.Sprintf("failed to unmarshal received resize message '%s': %s", string(specialPayload), err))
					continue
				}

				logger.Debug(fmt.Sprintf("resizing tty to use %v rows and %v columns...", ttySize.Rows, ttySize.Cols))

				if err := pty.Setsize(tty, &pty.Winsize{
					Rows: ttySize.Rows,
					Cols: ttySize.Cols,
				}); err != nil {
					logger.Warn(fmt.Sprintf("failed to resize tty, error: %s", err))
				}
				continue
			}

			// Forward to the process.
			_, err := tty.Write(b)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to write to TTY: %s", err))
			}
		}

		// Kill the shell, this will stop the Shell -> User routine
		cmd.Process.Kill()
	}()

	// Stop the handler if the global context is cancelled
	go func() {
		select {
		case <-globalCtx.Done():
			logger.Debug("Cancelling Websocket Handler")
			_ = ws.Close()
			return
		case <-ctxLocal.Done():
		}
	}()

	wg.Wait()
	cancelLocal() // Stop the global context listener
}
