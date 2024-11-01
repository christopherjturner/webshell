package ttyrec

import (
	"encoding/binary"
	"os"
	"time"
)

type Recorder struct {
	ttyFile   *os.File
	timeFile  *os.File
	lastWrite int64
	offset    int
	precision int64
	active    bool
}

func NewRecorder() (*Recorder, error) {
	ttyFile, err := os.Create("ttyrec.data")
	if err != nil {
		return nil, err
	}

	timeFile, err := os.Create("ttyrec.time")
	if err != nil {
		return nil, err
	}

	rec := &Recorder{
		ttyFile:   ttyFile,
		timeFile:  timeFile,
		precision: 100,
		active:    true,
	}

	return rec, nil
}

func (r *Recorder) Save(filepath string) error {

	outfile, err := os.Create(filepath)
	if err != nil {
		return err
	}

	r.active = false

	r.ttyFile.Sync()
	r.timeFile.Sync()

	r.ttyFile.Seek(0, 0)
	r.timeFile.Seek(0, 0)

	return Save(outfile, r.ttyFile, r.timeFile)
}

func (r Recorder) Close() error {
	errTty := r.ttyFile.Close()
	errTime := r.timeFile.Close()

	if errTty != nil {
		return errTty
	}
	if errTime != nil {
		return errTime
	}

	return nil
}

func (r *Recorder) Write(b []byte) (int, error) {

	if !r.active {
		return 0, nil
	}

	// Write to audit temp file.
	n, err := r.ttyFile.Write(b)
	if err != nil {
		return n, err
	}

	// Write to timing temp file.
	thisWrite := time.Now().UnixMilli()
	if r.lastWrite == 0 {
		r.lastWrite = thisWrite
	}

	// Only write entries every x milliseconds.
	if (thisWrite - r.lastWrite) > r.precision {
		r.lastWrite = thisWrite
		timing := Timing{
			Offset: int64(r.offset),
			Time:   thisWrite,
		}
		err = binary.Write(r.timeFile, binary.LittleEndian, timing)
		if err != nil {
			return n, err
		}

	}

	r.offset += n
	return n, nil
}
