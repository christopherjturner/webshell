package ttyrec

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
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
	writer    AuditWriter
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
		auditFile: auditFile,
		writer:    NewGzipAuditWriter(auditFile),
		precision: 100,
		enabled:   true,
	}
	return rec, nil
}

func (r *Recorder) Save() error {

	// Stop any writes while save is in progress
	r.enabled = false

	if err := r.writer.Close(); err != nil {
		return err
	}

	// Reopen file
	f, err := os.OpenFile(r.auditFile.Name(), os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer r.writer.Close()

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
		AuditLength:  int64(r.writer.Len()),
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

	n, err := r.writer.Write(b)
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
	fmt.Printf("read %d bytes, wrote %d bytes\n", len(b), n)
	return n, nil
}

// A fake recorder, used when auditing is disabled
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

type AuditWriter interface {
	io.WriteCloser
	Len() int
}

type StandardAuditWriter struct {
	n    int
	file *os.File
}

func (w *StandardAuditWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	w.n += n
	return n, err
}

func (w StandardAuditWriter) Len() int {
	return w.n
}

func (w StandardAuditWriter) Close() error {
	return w.file.Close()
}

type GzipAuditWriter struct {
	file *os.File
	wc   *writeCounter
	gzw  *gzip.Writer
}

func NewGzipAuditWriter(f *os.File) *GzipAuditWriter {
	wc := &writeCounter{}
	mw := io.MultiWriter(wc, f)
	gzw := gzip.NewWriter(mw)

	return &GzipAuditWriter{
		file: f,
		wc:   wc,
		gzw:  gzw,
	}
}

func (w *GzipAuditWriter) Write(p []byte) (int, error) {
	n, err := w.gzw.Write(p)
	return n, err
}

func (w GzipAuditWriter) Len() int {
	return w.wc.n
}

func (w GzipAuditWriter) Close() error {
	if err := w.gzw.Close(); err != nil {
		return err
	}
	return w.file.Close()
}

type writeCounter struct {
	n int
}

func (w *writeCounter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}
