package ttyrec

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestSaveAndLoadRecording(t *testing.T) {

	// Setup recorder.
	f, _ := os.CreateTemp("", "TestSaveAndLoadAuditing")
	f.Close()
	defer os.Remove(f.Name())

	rec, err := NewRecorder(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	rec.precision = -1

	// Write the test data down in two writes
	testData := []byte("test data 111 2222 4444 11111111111111111111111111111111111111111111111111111111111111111111111111111111111")
	rec.Write(testData[:25])
	rec.Write(testData[25:])

	// Save recording.
	err = rec.Save()
	if err != nil {
		t.Fatal(err)
	}

	// Reload the recorder)
	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatal("Could not reload auditfile: " + f.Name())
	}

	audit, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}

	// Validate content of recording.
	fmt.Printf("audit, %v", audit)
	if audit.Header.AuditOffset != 40 {
		t.Errorf("auditOffset is wrong. want %d got %d", 40, audit.Header.AuditOffset)
	}

	if audit.Header.AuditLength > int64(len(testData)) {
		t.Errorf("auditLength is wrong.  %d >  %d", audit.Header.AuditLength, len(testData))
	}

	gzr, err := gzip.NewReader(audit.Audit)
	if err != nil {
		t.Fatal(err)
	}

	auditData, err := io.ReadAll(gzr)
	if err != nil {
		t.Errorf("failed to read audit data: %v", err)
	}

	if !bytes.Equal(auditData, testData) {
		t.Errorf("audit data was wrong:  \nwant %s  \ngot  %s", string(testData), string(auditData))
	}

	// Expect 1 timing per write + 1 to mark the end.
	if len(audit.Timings) != 3 {
		t.Errorf("wrong number of timings, wanted 3, got %d", len(audit.Timings))
	}
}
