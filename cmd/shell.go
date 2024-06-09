package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
	log.Println("New session started")
	var err error

	cmd := exec.Command(cmd)
	cmd.Env = os.Environ()
	tty, err := pty.Start(cmd)

	if err != nil {
		log.Printf("Failed to start shell: %s", err)
		websocket.Message.Send(ws, "Failed to start shell")
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	defer func() {
		log.Println("Stopping terminal")
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("failed to stop process: %s", err)
		}

		if _, err := cmd.Process.Wait(); err != nil {
			log.Printf("Failed to wait process: %s", err)
		}

		if err := tty.Close(); err != nil {
			log.Printf("Failed to close tty: %s", err)
		}

		if err := ws.Close(); err != nil {
			log.Printf("Failed to close websocket: %s", err)
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
				log.Println("failed to forward tty to ws")
				continue
			}
		}
	}()

	// WS to TTY
	go func() {
		buffer := make([]byte, maxBufferSizeBytes)
		for {
			if err = websocket.Message.Receive(ws, &buffer); err != nil {
				fmt.Println("Websocket closed: ", err)
				break
			}

			b := bytes.Trim(buffer, "\x00")

			if b[0] == 1 {
				resizeMessage := bytes.Trim(b[1:], " \n\r\t\x00\x01")
				ttySize := &TTYSize{}

				if err := json.Unmarshal(resizeMessage, ttySize); err != nil {
					log.Printf("failed to unmarshal received resize message '%s': %s", string(resizeMessage), err)
					continue
				}
				log.Printf("resizing tty to use %v rows and %v columns...", ttySize.Rows, ttySize.Cols)
				if err := pty.Setsize(tty, &pty.Winsize{
					Rows: ttySize.Rows,
					Cols: ttySize.Cols,
				}); err != nil {
					log.Printf("failed to resize tty, error: %s", err)
				}
				continue
			}

			// forward to the shell
			written, err := tty.Write(b)
			if err != nil {
				log.Println(err)
			}

			// log commands entered
			_, err = logBuffer.Write(b[:written])
			if err != nil {
				log.Printf("log buffer error %v", err)
			}
			for _, i := range b[:written] {
				if i == 13 {
					log.Printf(string(logBuffer.Bytes()))
					logBuffer.Reset()
					break
				}
			}

		}
	}()

	wg.Wait()
}
