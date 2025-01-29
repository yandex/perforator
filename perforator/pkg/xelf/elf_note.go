package xelf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
)

type ElfNoteType uint32

//nolint:st1003
const (
	NT_GNU_BUILD_ID ElfNoteType = 3
)

const (
	elfNoteNameSizeLimit = 16
	elfNoteDescSizeLimit = 256
)

type ElfNote struct {
	Type        ElfNoteType
	Name        string
	Description []byte
}

type elfNoteReader struct {
	file *elf.File
	r    io.ReadSeeker
	note *ElfNote
	err  error
}

func newElfNoteReader(file *elf.File, r io.ReadSeeker) *elfNoteReader {
	return &elfNoteReader{file, r, &ElfNote{}, nil}
}

func (r *elfNoteReader) Error() error {
	return r.err
}

func (r *elfNoteReader) Next() bool {
	if r.err != nil {
		return false
	}

	err := r.trynext()
	if err == io.EOF {
		return false
	}
	if err != nil {
		r.err = fmt.Errorf("failed to decode ELF PT_NOTE header: %w", err)
		return false
	}

	return true
}

func (r *elfNoteReader) Note() *ElfNote {
	return r.note
}

// PT_NOTE layout is described there: https://docs.oracle.com/cd/E19683-01/816-1386/6m7qcoblj/index.html#chapter6-18048
func (r *elfNoteReader) trynext() error {
	//nolint:st1003
	type Elf64_Nhdr struct {
		NameSize uint32
		DescSize uint32
		Type     ElfNoteType
	}
	var nhdr Elf64_Nhdr

	err := binary.Read(r.r, r.file.ByteOrder, &nhdr)
	if err != nil {
		return err
	}
	if nhdr.NameSize > elfNoteNameSizeLimit {
		return fmt.Errorf("too large SHT_NOTE name size: %d while limit is %d", nhdr.DescSize, elfNoteDescSizeLimit)
	}
	if nhdr.DescSize > elfNoteDescSizeLimit {
		return fmt.Errorf("too large SHT_NOTE desc size: %d while limit is %d", nhdr.DescSize, elfNoteDescSizeLimit)
	}

	r.note.Type = nhdr.Type

	buf, err := r.read(nhdr.NameSize)
	if err != nil {
		return err
	}
	r.note.Name = string(buf)

	r.note.Description, err = r.read(nhdr.DescSize)
	if err != nil {
		return err
	}

	return nil
}

func (r *elfNoteReader) read(size uint32) ([]byte, error) {
	buf := make([]byte, size)
	_, err := r.r.Read(buf)
	if err != nil {
		return nil, err
	}

	// Skip any padding.
	if mod := int64(size) % 4; mod != 0 {
		_, err = r.r.Seek(4-mod, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
	}

	// Discard last null terminator.
	return bytes.TrimRight(buf, "\x00"), nil
}
