package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/net/websocket"

	"webshell/strace"
	"webshell/ttyrec"
)

const (
	shell              = "/bin/bash"
	maxBufferSizeBytes = 1024 * 256
)

var restrictedEnvVars = map[string]bool{
	"AUDIT_UPLOAD_URL": true,
}

// Sets a command to run as a different user.
func runAs(cmd *exec.Cmd, user *user.User) {
	uid, _ := strconv.ParseInt(user.Uid, 10, 32)
	gid, _ := strconv.ParseInt(user.Gid, 10, 32)

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", user.HomeDir))
}

// Removes any restricted keys from the parent environment
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

// WebShell's websocket handler
func shellHandler(ws *websocket.Conn) {

	logger.Info("New webshell session started")

	ctxLocal, cancelLocal := context.WithCancel(ws.Request().Context())
	defer cancelLocal()

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

	// TTY Recorder
	// When TTY Recorder is disabled we use a non-functioning version of it in its place.
	// Avoids having to scatter if statements all of the place etc.
	var recorder ttyrec.TTYRecorder = &ttyrec.NoOpRecorder{}

	if config.AuditTTY {
		// TTY recordings are written down to "<token>-<pid>.audit".
		auditFile := fmt.Sprintf("%s_%d.tty.audit", config.Token, cmd.Process.Pid)
		recorder, err = ttyrec.NewRecorder(filepath.Join(config.AuditPath, auditFile))
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

	activeConnections.Add(1)
	var wg sync.WaitGroup
	wg.Add(1)

	// Gracefully stop the session
	defer func() {
		logger.Info(fmt.Sprintf("Stopping shell PID %d", cmd.Process.Pid))
		if err := cmd.Process.Kill(); err != nil {
			logger.Error(fmt.Sprintf("Failed to stop process: %s", err))
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
				break
			}

			if err := websocket.Message.Send(ws, buffer[:l]); err != nil {
				logger.Error("Failed to forward tty to ws")
				continue
			}

			// Copy data to TTY Recorder.
			// TODO: maybe use a TeeReader instead
			if _, err := recorder.Write(buffer[:l]); err != nil {
				logger.Error("Failed record tty to ws")
				continue
			}
		}

		// Close the socket, this will stop User -> Shell
		_ = ws.Close()
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

					if err := pty.Setsize(tty, &pty.Winsize{
						Rows: uint16(rows),
						Cols: uint16(cols),
					}); err != nil {
						logger.Warn(fmt.Sprintf("Failed to resize tty, error: %s", err))
					}

					continue
				}

				logger.Info("Unknown special payload " + specialPayload)
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
}
