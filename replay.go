package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func replayHandler(ws *websocket.Conn) {

	logger.Info("Replaying session")
	var err error

	auditFile, err := os.Open("audit.bin")
	if err != nil {
		panic(err) // TODO: handle better
	}

	timings, err := loadTimings("timings.bin")
	if err != nil {
		panic(err)
	}

	defer func() {
		logger.Info("Stopping terminal")
		auditFile.Close()
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	wsWriter := WsWriter{ws: ws}

	// TTY to WS
	go func() {

		var offset int64
		var lastTime int64

		for i := 0; i < len(timings); i++ {
			toCopy := int64(timings[i].Offset - offset)

			io.CopyN(wsWriter, auditFile, toCopy)
			fmt.Printf("writing %d->%d\n", offset, timings[i].Offset)
			offset = timings[i].Offset

			if lastTime == 0 {
				lastTime = timings[i].Time
			}
			sleepFor := time.Duration(timings[i].Time - lastTime)
			fmt.Printf("sleeping %d\n", sleepFor)
			time.Sleep(sleepFor * time.Millisecond)
			lastTime = timings[i].Time
		}
		io.Copy(wsWriter, auditFile)
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

func loadTimings(filePath string) ([]Timing, error) {
	timings := []Timing{}

	f, err := os.Open(filePath)
	if err != nil {
		return []Timing{}, err
	}
	defer f.Close()

	for {
		var offset int64
		var timestamp int64

		err := binary.Read(f, binary.LittleEndian, &offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading from file: %v", err)
		}

		err = binary.Read(f, binary.LittleEndian, &timestamp)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading from file: %v", err)
		}

		// Append the value to the slice
		timings = append(timings, Timing{offset, timestamp})
	}

	return timings, nil

}

type Timing struct {
	Offset int64
	Time   int64
}

type AuditReplayer struct {
	File    *os.File
	Timings []Timing
}

func NewAuditReplayer() *AuditReplayer {

	return &AuditReplayer{}
}

func (a *AuditReplayer) Play(w io.Writer) {
	var offset int64
	var lastTime int64

	for i := 0; i < len(a.Timings); i++ {
		toCopy := int64(a.Timings[i].Offset - offset)

		io.CopyN(w, a.File, toCopy)
		offset = a.Timings[i].Offset

		if lastTime == 0 {
			lastTime = a.Timings[i].Time
		}
		sleepFor := time.Duration(a.Timings[i].Time - lastTime)
		fmt.Printf("sleeping %d\n", sleepFor)
		time.Sleep(sleepFor * time.Millisecond)
		lastTime = a.Timings[i].Time
	}
	io.Copy(w, a.File)
}
