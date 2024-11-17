package ttyrec

import (
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

	// Write test data to recorder.
	rec.Write([]byte("test data\n"))
	rec.Write([]byte("exit\n"))

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

	if audit.Header.AuditLength != 15 {
		t.Errorf("auditLength is wrong. want %d got %d", 15, audit.Header.AuditLength)
	}

	auditData, err := io.ReadAll(audit.Audit)
	if err != nil {
		t.Errorf("failed to read audit data: %v", err)
	}

	if string(auditData) != "test data\nexit\n" {
		t.Errorf("audit data was wrong, got %s", auditData)
	}

	if len(audit.Timings) != 3 {
		t.Errorf("wrong number of timings, wanted 3, got %d", len(audit.Timings))
	}
}
