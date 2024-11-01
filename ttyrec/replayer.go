package ttyrec

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	STOP = iota
	PLAY
)

type Replayer struct {
	file   *os.File
	Record *TTYRecording
	Speed  int64
}

func NewReplayer(pathToFile string) (*Replayer, error) {

	f, err := os.Open(pathToFile)
	if err != nil {
		return nil, err
	}

	record, err := Load(f)
	if err != nil {
		return nil, err
	}

	return &Replayer{
		file:   f,
		Record: record,
		Speed:  2,
	}, nil
}

func (a Replayer) Close() error {
	return a.file.Close()
}

func (a *Replayer) Reset() {
	fmt.Printf("Resetting")
}

func (a *Replayer) PlaybackSpeed(speed int64) {
	a.Speed = speed
}

func (a *Replayer) Play(w io.Writer) {

	var offset int64
	var lastTime int64
	timings := a.Record.Timings
	a.Record.Audit.Seek(0, 0)

	fmt.Println("Starting replay")

	for i := 0; i < len(timings); i++ {
		fmt.Printf("frame %d\n", i)
		// Copy chunk.
		toCopy := int64(timings[i].Offset - offset)
		offset = timings[i].Offset

		n, err := io.CopyN(w, a.Record.Audit, toCopy)
		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}

		fmt.Printf("wrote %d bytes to the websocket\n", n)

		// Handle timings.
		if lastTime == 0 {
			lastTime = timings[i].Time
		}

		sleepFor := (timings[i].Time - lastTime)
		if a.Speed > 0 {
			sleepFor /= a.Speed
		} else if a.Speed < 0 {
			sleepFor *= a.Speed
		} else {
			sleepFor = 0
		}

		time.Sleep(time.Duration(sleepFor) * time.Millisecond)
		lastTime = timings[i].Time
	}

	fmt.Println("End of replay")
}

func (a *Replayer) PlayFrame(w io.Writer, i int, delay bool) {

	if i < 0 || i >= len(a.Record.Timings) {
		w.Write([]byte("\r\nInvalid frame\r\n"))
		return
	}

	// Find the frame prior to the requested one.
	frameEnd := a.Record.Timings[i]
	frameStart := Timing{Offset: 0, Time: frameEnd.Time}
	if i > 0 {
		frameStart = a.Record.Timings[i-1]
	}

	fmt.Printf("Starting replay of frame %d\n", i)

	// Copy chunk.
	a.Record.Audit.Seek(frameStart.Offset, 0)
	size := int64(frameEnd.Offset - frameStart.Offset)
	_, err := io.CopyN(w, a.Record.Audit, size)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// Recreate input pause, if required.
	if delay {
		sleepFor := (frameEnd.Time - frameStart.Time)
		if a.Speed > 0 {
			sleepFor /= a.Speed
		} else if a.Speed < 0 {
			sleepFor *= a.Speed
		} else {
			sleepFor = 0
		}

		time.Sleep(time.Duration(sleepFor) * time.Millisecond)
	}
}
