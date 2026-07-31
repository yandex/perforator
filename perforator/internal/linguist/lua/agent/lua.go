package agent

import (
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/lua"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

var (
	minSupportedVersion = offsetloader.NewLanguageVersion(2, 1, 0).EncodeToUint32()
	maxSupportedVersion = offsetloader.NewLanguageVersion(2, 2, 0).EncodeToUint32()
)

func IsVersionSupported(version *lua.LuaVersion) bool {
	if version == nil {
		return false
	}

	versionKey := offsetloader.NewLanguageVersion(version.Major, version.Minor, 0).EncodeToUint32()
	if versionKey < minSupportedVersion || versionKey > maxSupportedVersion {
		return false
	}

	return true
}

func ParseLuaUnwinderConfig(conf *lua.LuaConfig) *unwinder.LuaConfig {
	return &unwinder.LuaConfig{
		OffsetGToL:        conf.OffsetGToL,
		OffsetGToDispatch: conf.OffsetGToDispatch,
		OffsetGToVmState:  conf.OffsetGToVmState,
		BinarySize:        conf.BinarySize,
		VmStartPc:         conf.VmStartPc,
		VmEndPc:           conf.VmEndPc,
	}
}
