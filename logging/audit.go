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

type AuditLogger struct {
	buf *bufio.Writer
	out io.Writer
}

func NewAuditLogger(w io.Writer) *AuditLogger {
	return &AuditLogger{
		buf: bufio.NewWriterSize(w, bufferSize),
		out: w,
	}
}

func (l *AuditLogger) Write(p []byte) (int, error) {
	n, err := l.buf.Write(p)
	if l.buf.Buffered() > 0 && bytes.Contains(p, []byte{'\n'}) {
		l.buf.Flush()
	}
	return n, err
}
