package ttyrec

import (
	"encoding/binary"
	"io"
	"os"
	"time"
)

type TTYRecorder interface {
	io.WriteCloser
	Save() error
}

type Recorder struct {
	auditFile *os.File
	timings   []Timing

	lastWrite int64
	auditSize int
	precision int64
	enabled   bool
}

func NewRecorder(filename string) (*Recorder, error) {

	auditFile, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	// Reserve header space at the start of file.
	if err := binary.Write(auditFile, binary.LittleEndian, Header{}); err != nil {
		return nil, err
	}

	rec := &Recorder{
		precision: 100,
		enabled:   true,
		auditFile: auditFile,
	}
	return rec, nil
}

func (r *Recorder) Save() error {

	// Stop any writes while save is in progress
	r.enabled = false
	err := r.auditFile.Close()
	if err != nil {
		return err
	}

	// Reopen file
	f, err := os.OpenFile(r.auditFile.Name(), os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer r.auditFile.Close()

	// Append timing data to the end.
	f.Seek(0, io.SeekEnd)
	if err = binary.Write(f, binary.LittleEndian, r.timings); err != nil && err != io.EOF {
		return err
	}

	// Update header with new offsets.
	auditOffset := int64(binary.Size(Header{}))
	header := Header{
		Magic:        MAGIC,
		Version:      VERSION,
		AuditOffset:  auditOffset,
		AuditLength:  int64(r.auditSize),
		TimingOffset: auditOffset + int64(r.auditSize),
		TimingLength: int64(binary.Size(r.timings)),
	}

	f.Seek(0, io.SeekStart)
	if err = binary.Write(f, binary.LittleEndian, &header); err != nil && err != io.EOF {
		return err
	}

	return f.Close()
}

func (r Recorder) Close() error {
	return r.auditFile.Close()
}

func (r *Recorder) Write(b []byte) (int, error) {

	if !r.enabled {
		return 0, nil
	}

	n, err := r.auditFile.Write(b)
	if err != nil {
		return n, err
	}

	thisWrite := time.Now().UnixMilli()
	if r.lastWrite == 0 {
		r.lastWrite = thisWrite
	}

	// Only write entries every x milliseconds.
	if (thisWrite - r.lastWrite) > r.precision {
		r.lastWrite = thisWrite
		timing := Timing{
			Offset: int64(r.auditSize),
			Time:   thisWrite,
		}
		r.timings = append(r.timings, timing)
	}

	r.auditSize += n

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
