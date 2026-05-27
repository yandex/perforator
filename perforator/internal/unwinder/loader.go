package unwinder

import (
	"fmt"

	"github.com/yandex/perforator/library/go/core/resource"
)

type KernelCompatibilityLevel int

const (
	KernelCompatibilityLevelNone KernelCompatibilityLevel = iota
	KernelCompatibilityLevel5_4
)

type ProgramRequirements struct {
	Debug               bool
	KernelCompatibility KernelCompatibilityLevel
}

func LoadProg(reqs ProgramRequirements) ([]byte, error) {
	var name string

	if reqs.Debug {
		name = "debug"
	} else {
		name = "release"
	}

	if reqs.KernelCompatibility == KernelCompatibilityLevel5_4 {
		name += ".k54"
	}

	name = fmt.Sprintf("ebpf/unwinder.%s.elf", name)

	data := resource.Get(name)
	if data == nil {
		return nil, fmt.Errorf("missing program resource %q for requirements %+v", name, reqs)
	}

	return data, nil
}
