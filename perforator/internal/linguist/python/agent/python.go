package agent

import (
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func PythonInternalsOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonInternalsOffsets, error) {
	threadStateOffsets, err := PyThreadStateOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	interpreterFrameOffsets, err := PyInterpreterFrameOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	codeObjectOffsets, err := PyCodeObjectOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	asciiObjectOffsets, err := PyASCIIObjectOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	cframeOffsets, err := PyCframeOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	runtimeOffsets, err := PyRuntimeOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	interpreterStateOffsets, err := PyInterpreterStateOffsetsByVersion(version)
	if err != nil {
		return nil, err
	}

	return &unwinder.PythonInternalsOffsets{
		PyRuntimeStateOffsets:     *runtimeOffsets,
		PyInterpreterStateOffsets: *interpreterStateOffsets,
		PyThreadStateOffsets:      *threadStateOffsets,
		PyCframeOffsets:           *cframeOffsets,
		PyInterpreterFrameOffsets: *interpreterFrameOffsets,
		PyCodeObjectOffsets:       *codeObjectOffsets,
		PyAsciiObjectOffsets:      *asciiObjectOffsets,
	}, nil
}

func IsVersionSupported(version *python.PythonVersion) bool {
	return version.Major == 3 && (version.Minor == 12 || version.Minor == 13)
}

func encodeVersion(version *python.PythonVersion) uint32 {
	return version.Micro + (version.Minor << 8) + (version.Major)<<16
}

func ParsePythonUnwinderConfig(conf *python.PythonConfig) *unwinder.PythonConfig {
	if conf != nil && conf.PyThreadStateTLSOffset < 0 && conf.Version != nil && IsVersionSupported(conf.Version) {
		offsets, _ := PythonInternalsOffsetsByVersion(conf.Version)
		return &unwinder.PythonConfig{
			Version:                  encodeVersion(conf.Version),
			PyThreadStateTlsOffset:   uint64(-conf.PyThreadStateTLSOffset),
			PyRuntimeRelativeAddress: conf.RelativePyRuntimeAddress,
			Offsets:                  *offsets,
		}
	}

	return nil
}
