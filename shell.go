package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"golang.org/x/net/websocket"
)

const cmd = "/bin/bash"
const maxBufferSizeBytes = 1024 * 256

type TTYSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	X    uint16 `json:"x"`
	Y    uint16 `json:"y"`
}

func shellHandler(ws *websocket.Conn) {

	lb := []byte{}
	logBuffer := bytes.NewBuffer(lb)
	logger.Info("New webshell session started")
	var err error

	cmd := exec.Command(cmd)
	cmd.Env = os.Environ()
	tty, err := pty.Start(cmd)

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start shell: %s", err))
		websocket.Message.Send(ws, "Failed to start shell")
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	defer func() {
		logger.Info("Stopping terminal")
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

	}()

	// TTY to WS
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
		}
	}()

	// WS to TTY
	go func() {
		buffer := make([]byte, maxBufferSizeBytes)
		for {
			if err = websocket.Message.Receive(ws, &buffer); err != nil {
				logger.Warn(fmt.Sprintf("Websocket closed: %s", err))
				break
			}

			b := bytes.Trim(buffer, "\x00")

			// Handle resize message from the terminal.
			if b[0] == 1 {
				resizeMessage := bytes.Trim(b[1:], " \n\r\t\x00\x01")
				ttySize := &TTYSize{}

				if err := json.Unmarshal(resizeMessage, ttySize); err != nil {
					logger.Warn(fmt.Sprintf("failed to unmarshal received resize message '%s': %s", string(resizeMessage), err))
					continue
				}
				logger.Info("resizing tty to use %v rows and %v columns...", ttySize.Rows, ttySize.Cols)
				if err := pty.Setsize(tty, &pty.Winsize{
					Rows: ttySize.Rows,
					Cols: ttySize.Cols,
				}); err != nil {
					logger.Warn(fmt.Sprintf("failed to resize tty, error: %s", err))
				}
				continue
			}

			// forward to the shell
			written, err := tty.Write(b)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to write to TTY: %s", err))
			}

			// Log commands entered
			// TODO: Test how this handles very large inputs.
			//       It should probably flush the buffer after a certain size is reached and/or truncate big inputs
			_, err = logBuffer.Write(b[:written])
			if err != nil {
				logger.Error(fmt.Sprintf("log buffer error %s", err))
			}
			for _, i := range b[:written] {
				if i == 13 {
					logger.Info(logBuffer.String())
					logBuffer.Reset()
					break
				}
			}

		}
	}()

	wg.Wait()
}
