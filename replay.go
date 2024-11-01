package main

import (
	"bytes"
	"fmt"
	"golang.org/x/net/websocket"
	"sync"
	"webshell/ttyrec"
)

func replayHandler(ws *websocket.Conn) {

	logger.Info("Replaying session")
	var err error

	replayer, err := ttyrec.NewReplayer("ttyrec.bin")
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load audit file: %v", err))
		return
	}

	defer func() {
		logger.Info("Stopping terminal")
		replayer.Close()
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	wsWriter := WsWriter{ws: ws}

	// temp
	go func() {
		replayer.Play(wsWriter)
	}()

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

				specialPayload := bytes.Trim(b[1:], " \n\r\t\x00\x01")
				logger.Info(string(specialPayload))
				if len(specialPayload) == 0 {
					continue
				}

				if string(specialPayload) == "PING" {
					logger.Debug("PING")
					continue
				}

				if string(specialPayload) == "PLAY" {
					go func() {
						replayer.Play(wsWriter)
					}()
					continue
				}
			}
		}

	}()

	wg.Wait()
}

type WsWriter struct {
	ws *websocket.Conn
}

func (w WsWriter) Write(b []byte) (int, error) {
	err := websocket.Message.Send(w.ws, b)
	if err != nil {
		return 0, err
	}
	return len(b), err
}
