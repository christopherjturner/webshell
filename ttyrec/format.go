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
	// TODO: keep ref to underlying file
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

	if rec.Header.Version != VERSION {
		return nil, fmt.Errorf("unsupport recording version %d", rec.Header.Version)
	}

	if rec.Header.AuditOffset > 0 && rec.Header.AuditLength > 0 {
		rec.Audit = io.NewSectionReader(r, rec.Header.AuditOffset, rec.Header.AuditLength)
	}

	if rec.Header.TimingOffset > 0 && rec.Header.TimingLength > 0 {
		numberOfTimings := int(rec.Header.TimingLength) / binary.Size(Timing{})
		rec.Timings = make([]Timing, numberOfTimings)

		tr := io.NewSectionReader(r, rec.Header.TimingOffset, rec.Header.TimingLength)
		if err := binary.Read(tr, binary.LittleEndian, &rec.Timings); err != nil {
			return nil, err
		}

		// Add extra end-of-file timing
		rec.Timings = append(rec.Timings, Timing{
			Offset: rec.Header.AuditLength,
			Time:   rec.Timings[len(rec.Timings)-1].Time,
		})
	}

	return rec, nil
}

func Save(dest io.WriteSeeker, audit io.Reader, timings io.Reader) error {

	header := Header{
		Magic:   MAGIC,
		Version: VERSION,
	}

	// Write the header, we will come back and fill in the missing values at the end
	err := binary.Write(dest, binary.LittleEndian, header)
	if err != nil {
		return err
	}

	// Write the audit file
	w, err := io.Copy(dest, audit)
	if err != nil {
		return err
	}

	header.AuditOffset = int64(binary.Size(header))
	header.AuditLength = w

	// Write the timings
	w, err = io.Copy(dest, timings)
	if err != nil {
		return err
	}

	header.TimingOffset = int64(header.AuditOffset + header.AuditLength)
	header.TimingLength = w

	// Update the header
	dest.Seek(0, 0)
	err = binary.Write(dest, binary.LittleEndian, header)
	if err != nil {
		return err
	}

	return nil
}
