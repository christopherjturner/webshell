package ttyrec

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"time"
)

type TTYRecorder interface {
	io.WriteCloser
	Save() error
}

type Recorder struct {
	ttyFile   *os.File
	timeFile  *os.File
	lastWrite int64
	offset    int
	precision int64
	enabled   bool
	auditDir  string
	auditFile string
}

func NewRecorder(auditDir, auditFile string) (*Recorder, error) {

	err := os.MkdirAll(auditDir, 0600)
	if err != nil {
		return nil, err
	}

	ttyFile, err := os.CreateTemp(auditDir, "ttyrec.data")
	if err != nil {
		return nil, err
	}

	timeFile, err := os.CreateTemp(auditDir, "ttyrec.time")
	if err != nil {
		return nil, err
	}

	rec := &Recorder{
		ttyFile:   ttyFile,
		timeFile:  timeFile,
		precision: 100,
		enabled:   true,
		auditDir:  auditDir,
		auditFile: auditFile,
	}

	return rec, nil
}

func (r *Recorder) Save() error {

	outfile, err := os.Create(filepath.Join(r.auditDir, r.auditFile))
	if err != nil {
		return err
	}

	r.enabled = false

	ttyFile, err := os.Open(r.ttyFile.Name())
	if err != nil {
		return err
	}
	defer ttyFile.Close()

	timeFile, err := os.Open(r.timeFile.Name())
	if err != nil {
		return err
	}
	defer timeFile.Close()

	return Save(outfile, ttyFile, timeFile)
}

func (r Recorder) Close() error {
	errTty := r.ttyFile.Close()
	errTime := r.timeFile.Close()

	_ = os.Remove(r.ttyFile.Name())
	_ = os.Remove(r.timeFile.Name())

	if errTty != nil {
		return errTty
	}

	if errTime != nil {
		return errTime
	}

	return nil
}

func (r *Recorder) Write(b []byte) (int, error) {

	if !r.enabled {
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

type NoOpRecorder struct{}

func (r *NoOpRecorder) Write(b []byte) (int, error) {
	return len(b), nil
}

func (r *NoOpRecorder) Save() error {
	return nil
}

func (r NoOpRecorder) Close() error {
	return nil
}
