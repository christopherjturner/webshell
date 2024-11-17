package ttyrec

import (
	"fmt"
)

import (
	"encoding/binary"
	"io"
)

const (
	MAGIC   uint32 = 0xDC3443CD
	VERSION byte   = 0x01
)

type Header struct {
	Magic             uint32
	Version           byte
	AuditCompression  byte
	TimingCompression byte
	Flags             byte
	AuditOffset       int64
	AuditLength       int64
	TimingOffset      int64
	TimingLength      int64
}

type Timing struct {
	Time   int64
	Offset int64
}

type TTYRecording struct {
	Header  Header
	Audit   *io.SectionReader
	Timings []Timing
}

type ReaderAtCloser interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

func Load(r ReaderAtCloser) (*TTYRecording, error) {

	rec := &TTYRecording{}
	header := Header{}
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	rec.Header = header

	if rec.Header.Magic != MAGIC {
		return nil, fmt.Errorf("invalid file, invalid header ID %d", rec.Header.Magic)
	}

	switch rec.Header.Version {
	case 0x01:
		return loadV1(r, rec)
	default:
		return nil, fmt.Errorf("unsupport recording version %d", rec.Header.Version)
	}
}

func loadV1(r ReaderAtCloser, rec *TTYRecording) (*TTYRecording, error) {
	if rec.Header.AuditOffset > 0 && rec.Header.AuditLength > 0 {
		rec.Audit = io.NewSectionReader(r, rec.Header.AuditOffset, rec.Header.AuditLength)
	}

	if rec.Header.TimingOffset > 0 && rec.Header.TimingLength > 0 {
		numberOfTimings := int(rec.Header.TimingLength) / binary.Size(Timing{})
		rec.Timings = make([]Timing, numberOfTimings)

		tr := io.NewSectionReader(r, rec.Header.TimingOffset, rec.Header.TimingLength)
		if err := binary.Read(tr, binary.LittleEndian, &rec.Timings); err != nil && err != io.EOF {
			fmt.Printf("%v\n", rec)
			return nil, fmt.Errorf("failed to load timings: %v", err)
		}

		// Add extra end-of-file timing
		rec.Timings = append(rec.Timings, Timing{
			Offset: rec.Header.AuditLength,
			Time:   rec.Timings[len(rec.Timings)-1].Time,
		})
	}

	return rec, nil
}
