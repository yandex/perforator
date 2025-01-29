package agent

import (
	"errors"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

const (
	UnspecifiedOffset = uint32((1 << 32) - 1)
)

// Offsets from https://github.com/python/cpython/blob/main/Include/cpython/pystate.h#L59
func PyThreadStateOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonThreadStateOffsets, error) {
	switch {
	case version.Major == 3 && version.Minor == 13:
		return &unwinder.PythonThreadStateOffsets{
			CframeOffset:         UnspecifiedOffset,
			CurrentFrameOffset:   72,
			NativeThreadIdOffset: 160,
			PrevThreadOffset:     0,
			NextThreadOffset:     8,
		}, nil
	case version.Major == 3 && version.Minor == 12:
		return &unwinder.PythonThreadStateOffsets{
			CframeOffset:         56,
			CurrentFrameOffset:   UnspecifiedOffset,
			NativeThreadIdOffset: 144,
			PrevThreadOffset:     0,
			NextThreadOffset:     8,
		}, nil
	default:
		return nil, errors.New("no pythreadstate offsets for this version")
	}
}

// Offsets from https://github.com/python/cpython/blob/a4562fedadb73fe1e978dece65c3bcefb4606678/Include/internal/pycore_frame.h#L62
func PyInterpreterFrameOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonInterpreterFrameOffsets, error) {
	switch {
	case version.Major == 3 && version.Minor >= 12 && version.Minor <= 13:
		return &unwinder.PythonInterpreterFrameOffsets{
			FCodeOffset:    0,
			PreviousOffset: 8,
			OwnerOffset:    70,
		}, nil
	default:
		return nil, errors.New("no py interpreter frame offsets for this version")
	}
}

// Offsets from https://github.com/python/cpython/blob/3.13/Include/internal/pycore_interp.h#L94C1-L146C36
func PyInterpreterStateOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonInterpreterStateOffsets, error) {
	switch {
	case version.Major == 3 && version.Minor == 12:
		return &unwinder.PythonInterpreterStateOffsets{
			NextOffset:        0,
			ThreadsHeadOffset: 72,
		}, nil
	case version.Major == 3 && version.Minor == 13:
		return &unwinder.PythonInterpreterStateOffsets{
			NextOffset:        7264,
			ThreadsHeadOffset: 7344,
		}, nil
	default:
		return nil, errors.New("no py interpreter state offsets for this version")
	}
}

// Offsets from https://github.com/python/cpython/blob/a4562fedadb73fe1e978dece65c3bcefb4606678/Include/cpython/code.h#L73
func PyCodeObjectOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonCodeObjectOffsets, error) {
	if version.Major == 3 && version.Minor >= 12 && version.Minor <= 13 {
		return &unwinder.PythonCodeObjectOffsets{
			CoFirstlinenoOffset: 68,
			FilenameOffset:      112,
			QualnameOffset:      128,
		}, nil
	}

	return nil, errors.New("no py code object offsets for this version")
}

// Offsets from https://github.com/python/cpython/blob/a4562fedadb73fe1e978dece65c3bcefb4606678/Include/cpython/unicodeobject.h#L54
func PyASCIIObjectOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonAsciiObjectOffsets, error) {
	if version.Major == 3 && version.Minor >= 12 && version.Minor <= 13 {
		return &unwinder.PythonAsciiObjectOffsets{
			LengthOffset:           16,
			DataOffset:             40,
			StateOffset:            32,
			AsciiBit:               6,
			CompactBit:             5,
			StaticallyAllocatedBit: 7,
		}, nil
	}

	return nil, errors.New("no py ascii object offsets for this version")
}

// Offsets from https://github.com/python/cpython/blob/3.12/Include/cpython/pystate.h#L67
func PyCframeOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonCframeOffsets, error) {
	if version.Major == 3 && version.Minor == 12 {
		return &unwinder.PythonCframeOffsets{
			CurrentFrameOffset: 0,
		}, nil
	}

	if version.Major == 3 && version.Minor == 13 {
		return &unwinder.PythonCframeOffsets{
			CurrentFrameOffset: UnspecifiedOffset,
		}, nil
	}

	return nil, errors.New("no py cframe offsets for this version")
}

// Offsets from https://github.com/python/cpython/blob/3.13/Include/internal/pycore_runtime.h#L208
func PyRuntimeOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonRuntimeStateOffsets, error) {
	if version.Major == 3 && version.Minor == 12 {
		return &unwinder.PythonRuntimeStateOffsets{
			PyInterpretersMainOffset: 48,
		}, nil
	} else if version.Major == 3 && version.Minor == 13 {
		return &unwinder.PythonRuntimeStateOffsets{
			PyInterpretersMainOffset: 640,
		}, nil
	}

	return nil, errors.New("no py runtime offsets for this version")
}
