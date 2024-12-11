package ttyrec

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"testing"
)

type WriteCounter struct {
	n int
}

func (w *WriteCounter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}

func (w WriteCounter) Len() int {
	return w.n
}

func xTestReadWriteCompress(t *testing.T) {
	wc := &WriteCounter{}
	var buf bytes.Buffer
	mw := io.MultiWriter(wc, &buf)

	zw := gzip.NewWriter(mw)

	input := []byte("11111 11111111111111111111111111111111111111111111111111111111111 hello")
	inputBuffer := bytes.NewReader(input)

	b, err := io.Copy(zw, inputBuffer)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("input %d, output %d (wc %d) vs %d\n %02X\n", len(input), b, wc.Len(), buf.Len(), buf)

	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}

	bb := []byte{}
	br := bytes.NewBuffer(bb)

	io.Copy(br, zr)
	println(br.String())
}
