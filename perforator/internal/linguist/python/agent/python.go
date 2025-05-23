package agent

import (
	"fmt"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func PythonInternalsOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonInternalsOffsets, error) {
	if version == nil {
		return nil, fmt.Errorf("nil version provided")
	}

	versionKey := encodeVersion(version)

	if offsets, ok := pythonVersionOffsets[versionKey]; ok {
		return offsets, nil
	}

	return nil, fmt.Errorf("no offsets available for Python %d.%d.%d", version.Major, version.Minor, version.Micro)
}

var (
	minSupportedVersion = encodeVersion(&python.PythonVersion{
		Major: 3,
		Minor: 2,
		Micro: 1,
	})
)

func IsVersionSupported(version *python.PythonVersion) bool {
	if version == nil {
		return false
	}

	versionKey := encodeVersion(version)
	if versionKey < minSupportedVersion {
		return false
	}
	_, ok := pythonVersionOffsets[versionKey]
	return ok
}

func encodeVersion(version *python.PythonVersion) uint32 {
	return version.Micro + (version.Minor << 8) + (version.Major << 16)
}

// Only supported python version config must be passed here.
func ParsePythonUnwinderConfig(conf *python.PythonConfig) *unwinder.PythonConfig {
	offsets, _ := PythonInternalsOffsetsByVersion(conf.Version)
	return &unwinder.PythonConfig{
		Version:                   encodeVersion(conf.Version),
		PyThreadStateTlsOffset:    uint64(-conf.PyThreadStateTLSOffset),
		PyRuntimeRelativeAddress:  conf.RelativePyRuntimeAddress,
		AutoTssKeyRelativeAddress: conf.RelativeAutoTSSkeyAddress,
		UnicodeTypeSizeLog2:       conf.UnicodeTypeSizeLog2,
		Offsets:                   *offsets,
	}
}
