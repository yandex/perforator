package loader

import "github.com/yandex/perforator/library/go/core/resource"

func LoadProg(debug bool) []byte {
	var name string

	if debug {
		name = "ebpf/prog.debug.elf"
	} else {
		name = "ebpf/prog.release.elf"
	}

	return resource.MustGet(name)
}
