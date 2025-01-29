package jvmattach

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

type jvmDecoder struct {
	rd  io.Reader
	err error
}

func (d *jvmDecoder) error() error {
	return d.err
}

func (d *jvmDecoder) readByte() (byte, bool /* eof */, bool /* ok */) {
	if d.err != nil {
		return 0, false, false
	}

	buf := []byte{0}
	_, err := d.rd.Read(buf)
	if err == io.EOF {
		return 0, true, true
	}
	if err != nil {
		d.err = fmt.Errorf("failed to receive byte: %w", err)
		return 0, false, false
	}
	return buf[0], false, true
}

func (d *jvmDecoder) readInt() (int, bool) {
	buf := make([]byte, 0, 4)
	for {
		digit, eof, ok := d.readByte()
		if !ok {
			return 0, false
		}
		if eof || digit == byte('\n') {
			break
		}
		buf = append(buf, digit)
	}
	if len(buf) == 0 {
		d.err = errors.New("unexpected EOF when reading integer number")
		return 0, false
	}
	num, err := strconv.Atoi(string(buf))
	if err != nil {
		d.err = fmt.Errorf("failed to parse integer number: %w", err)
		return 0, false
	}
	return num, true
}

func (d *jvmDecoder) readString() (string, bool) {
	if d.err != nil {
		return "", false
	}
	data, err := io.ReadAll(d.rd)
	if err != nil {
		d.err = fmt.Errorf("failed to read string: %w", err)
		return "", false
	}
	return string(data), true
}
