package logging

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
)

type SlogWriter struct {
	log *slog.Logger
}

func NewSlogWriter(logger *slog.Logger) *SlogWriter {
	return &SlogWriter{
		log: logger,
	}
}

func (w *SlogWriter) Write(p []byte) (int, error) {
	// TODO: do we want to truncate input over a certain length?
	w.log.Info(string(p))
	return len(p), nil
}

const bufferSize = 1024 * 16

type AuditWriter struct {
	buf *bufio.Writer
	out io.Writer
}

func NewAuditWriter(w io.Writer) *AuditWriter {
	return &AuditWriter{
		buf: bufio.NewWriterSize(w, bufferSize),
		out: w,
	}
}

func (l *AuditWriter) Write(p []byte) (int, error) {
	n, err := l.buf.Write(p)
	if bytes.Contains(p, []byte{0x0D}) {
		l.buf.Flush()
	}
	return n, err
}
