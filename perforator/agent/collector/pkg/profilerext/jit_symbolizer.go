package profilerext

import "github.com/yandex/perforator/perforator/pkg/linux"

type JITSymbol struct {
	Name string
}

type JITSymbolizerOutput struct {
	Symbols []JITSymbol
	// This should not be a real mapping (i.e. elf file) name, but a pseudo-mapping
	// representing the runtime.
	MappingName string
}

// LocalSymbolizer resolves jit-ed IPs in samples into function names.
type JITSymbolizer interface {
	// Note that this function has neither context nor error result, because
	// it is not supposed to contain complex logic for now.
	// Symbolizer is expected to return at least one output item on success.
	Resolve(pid linux.CurrentNamespacePID, ip uint64) (JITSymbolizerOutput, bool)
}
