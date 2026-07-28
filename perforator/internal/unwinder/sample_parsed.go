package unwinder

// RecordSampleParsed is the decoded form of a sample, decoupled from the
// fixed-layout BPF struct. Variable-length fields are slices.
type RecordSampleParsed struct {
	// Fixed header fields (embedded from generated BTF struct)
	RecordSampleHeader

	// Variable-length fields (these shadow the SectionDesc fields in RecordSampleHeader)
	Cgroups   []uint64
	KernStack []uint64
	UserStack []uint64
	LBR       []BranchRecord
	TLS       []ThreadLocalVariableCollectResult

	// Interpreter stacks (decoded from language sections)
	PythonStack InterpreterStack
	PhpStack    InterpreterStack
	JvmStack    JvmStack
	LuaStack    InterpreterStack
}

// NewRecordSampleParsed creates a RecordSampleParsed with pre-allocated
// backing slices to avoid allocations on the hot path.
func NewRecordSampleParsed() *RecordSampleParsed {
	return &RecordSampleParsed{
		Cgroups:   make([]uint64, 0, 16),
		KernStack: make([]uint64, 0, 127),
		UserStack: make([]uint64, 0, 128),
		LBR:       make([]BranchRecord, 0, 32),
		TLS:       make([]ThreadLocalVariableCollectResult, 0, 4),
	}
}
