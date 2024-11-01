package ttyrec

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {

	// Write audit to a new temp file.
	out, err := os.CreateTemp("", "ttyrec.out")
	if err != nil {
		t.Fatal("failed to create tmp file")
	}
	defer out.Close()

	filename := out.Name()

	// Setup audit recording.
	auditData := []byte("foo\r\nbaz\r\nbar\r\n")
	audit := bytes.NewReader(auditData)

	// Setup timing data.
	timingsData := []Timing{
		{1, 500},
		{2, 600},
	}
	b := []byte{}
	bb := bytes.NewBuffer(b)

	err = binary.Write(bb, binary.LittleEndian, &timingsData)
	if err != nil {
		t.Fatal("failed to setup timing data ")
	}

	timings := bytes.NewReader(bb.Bytes())

	// Save the tty recording.
	err = Save(out, audit, timings)
	if err != nil {
		t.Fatal(err)
	}
	out.Close()

	// Reopen the same recording.
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Load the recording data.
	rec, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}

	recordingFromFile, err := io.ReadAll(rec.Audit)
	if err != nil {
		t.Fatal(err)
	}

	// Check the audit data matches.
	if !bytes.Equal(recordingFromFile, auditData) {
		t.Errorf("loaded recording did not match input\nwant: %X\ngot:  %X",
			auditData,
			recordingFromFile,
		)
	}

	// Check the timing data matches.
	if len(rec.Timings) != len(timingsData) {
		t.Errorf("timing data didnt load correctly. want %d items got %d",
			len(timingsData),
			len(rec.Timings),
		)
	}

	for i := range rec.Timings {
		if rec.Timings[i].Offset != timingsData[i].Offset {
			t.Errorf("timing[%d].Offset want %d got %d",
				i,
				timingsData[i].Offset,
				rec.Timings[i].Offset,
			)
		}
	}
}
