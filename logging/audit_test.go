package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestBufferedLogOnlyWritesOnNewLine(t *testing.T) {

	w := strings.Builder{}
	audit := NewAuditLogger(&w)

	audit.Write([]byte("1"))
	if w.Len() != 0 {
		t.Fatalf("w was written to unexpectedly")
	}

	audit.Write([]byte("2"))
	if w.Len() != 0 {
		t.Fatalf("w was written to unexpectedly")
	}

	audit.Write([]byte("34"))
	if w.Len() != 0 {
		t.Fatalf("w was written to unexpectedly")
	}

	audit.Write([]byte{'5', '\n'})
	if w.String() != "12345\n" {
		t.Fatalf("w didnt have the expected content [%s]", w.String())
	}
}

func TestBufferedLogWritesWhenBufferIsFull(t *testing.T) {

	w := strings.Builder{}
	audit := NewAuditLogger(&w)

	input := bytes.Repeat([]byte{'x'}, bufferSize-1)

	audit.Write(input)
	if w.Len() != 0 {
		t.Fatalf("w was written to unexpectedly")
	}

	audit.Write([]byte("zz"))
	if w.Len() != bufferSize {
		t.Fatalf("w wasn't the expected size %d != %d", w.Len(), bufferSize)
	}
}
