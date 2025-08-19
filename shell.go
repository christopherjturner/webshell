package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/coder/websocket"
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

	// TODO: we assume we're using the token for the session id
	// maybe we'd be better using the cookie, presuming its set.
	id := base64.StdEncoding.EncodeToString([]byte(r.RequestURI))

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
