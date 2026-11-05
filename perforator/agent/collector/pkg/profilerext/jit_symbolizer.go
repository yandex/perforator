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

type Address struct {
	// IP is address of instruction.
	// For non-leaf frames, it is currently address of last byte of the CALL instruction.
	IP uint64
	// Leaf is true if this is a leaf frame (i.e. it is not caller of any other frame).
	Leaf bool
}

// JITSymbolizer resolves jit-ed IPs in samples into function names.
type JITSymbolizer interface {
	// Note that this function has neither context nor error result, because
	// it is not supposed to contain complex logic for now.
	// Symbolizer is expected to return at least one output item on success.
	Resolve(pid linux.CurrentNamespacePID, addr Address) (JITSymbolizerOutput, bool)
}

// ProcessJITSymbolizer resolves jit-ed IPs in samples into function names.
type ProcessJITSymbolizer interface {
	// ResolveForProcess returns symbol (or several symbols representing inline chain)
	// corresponding to the address in the process identified by key.
	// It should not contain complex logic and should be infallible.
	ResolveForProcess(key linux.ProcessKey, addr Address) (JITSymbolizerOutput, bool)
}
