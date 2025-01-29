package unwinder

import "github.com/yandex/perforator/library/go/core/resource"

func LoadProg(debug bool) []byte {
	var name string

	if debug {
		name = "ebpf/unwinder.debug.elf"
	} else {
		name = "ebpf/unwinder.release.elf"
	}

	return resource.MustGet(name)
}
