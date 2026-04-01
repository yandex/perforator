package agent

import (
	"fmt"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func PythonInternalsOffsetsByVersion(version *python.PythonVersion) (*unwinder.PythonInternalsOffsets, error) {
	if version == nil {
		return nil, fmt.Errorf("nil version provided")
	}
	return pythonOffsets.Get(offsetloader.NewLanguageVersion(version.Major, version.Minor, version.Micro))
}

func IsVersionSupported(version *python.PythonVersion) bool {
	_, err := PythonInternalsOffsetsByVersion(version)
	return err == nil
}

// Only supported python version config must be passed here.
func ParsePythonUnwinderConfig(conf *python.PythonConfig) *unwinder.PythonConfig {
	offsets, _ := PythonInternalsOffsetsByVersion(conf.Version)
	return &unwinder.PythonConfig{
		Version:                     offsetloader.NewLanguageVersion(conf.Version.Major, conf.Version.Minor, conf.Version.Micro).EncodeToUint32(),
		PyThreadStateTlsOffset:      uint64(-conf.PyThreadStateTLSOffset),
		PyRuntimeRelativeAddress:    conf.RelativePyRuntimeAddress,
		PyInterpHeadRelativeAddress: conf.RelativePyInterpHeadAddress,
		AutoTssKeyRelativeAddress:   conf.RelativeAutoTSSkeyAddress,
		UnicodeTypeSizeLog2:         conf.UnicodeTypeSizeLog2,
		Offsets:                     *offsets,
	}
}
