package jvmattach

import "io"

type jvmEncoder struct {
	wr  io.Writer
	err error
}

func (e *jvmEncoder) error() error {
	return e.err
}

func (e *jvmEncoder) writeString(s string) {
	if e.err != nil {
		return
	}
	if len(s) > 0 {
		_, err := e.wr.Write([]byte(s))
		if err != nil {
			e.err = err
			return
		}
	}
	_, err := e.wr.Write([]byte{0})
	if err != nil {
		e.err = err
	}
}

const protocolVersion = "1"

func (e *jvmEncoder) writeCommand(args [4]string) {
	e.writeString(protocolVersion)
	for _, s := range args {
		e.writeString(s)
	}
}
