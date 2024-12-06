package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"webshell/ttyrec"

	"github.com/coder/websocket"
)

type Replayer struct{}

func (rp Replayer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Accept the WS connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		logger.Error(err.Error())
		return
	}

	replayHandler(conn)
}

func replayHandler(ws *websocket.Conn) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("Replaying session")
	var err error

	replayer, err := ttyrec.NewReplayer(config.ReplayFile)
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

	wsWriter, err := ws.Writer(ctx, websocket.MessageBinary)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// temp
	go func() {
		replayer.Play(wsWriter)
	}()

	go func() {

		for {
			_, b, err := ws.Read(ctx)
			if err != nil {
				logger.Warn(fmt.Sprintf("Websocket closed: %s", err))
				break
			}

			b = bytes.Trim(b, "\x00")

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
