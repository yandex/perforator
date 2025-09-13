package unwinder

import (
	"fmt"

	"github.com/yandex/perforator/library/go/core/resource"
)

type ProgramRequirements struct {
	Debug bool
	JVM   bool
	PHP   bool
}

func LoadProg(reqs ProgramRequirements) ([]byte, error) {
	var name string

	if reqs.Debug {
		name = "debug"
	} else {
		name = "release"
	}
	if reqs.JVM {
		name += ".jvm"
	}

	if reqs.PHP {
		name += ".php"
	}

	name = fmt.Sprintf("ebpf/unwinder.%s.elf", name)

	data := resource.Get(name)
	if data == nil {
		return nil, fmt.Errorf("missing program resource %q for requirements %+v", name, reqs)
	}

	return data, nil
}
