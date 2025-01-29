package perfevent

type Type string

// See man 2 perf_event_open for the description of the event types.

const (
	// Hardware events
	CPUCycles             Type = "CPUCycles"
	CPUInstructions       Type = "CPUInstructions"
	CacheReferences       Type = "CacheReferences"
	CacheMisses           Type = "CacheMisses"
	BranchInstructions    Type = "BranchInstructions"
	BranchMisses          Type = "BranchMisses"
	BusCycles             Type = "BusCycles"
	StalledCyclesFrontend Type = "StalledCyclesFrontend"
	StalledCyclesBackend  Type = "StalledCyclesBackend"
	RefCPUCycles          Type = "RefCPUCycles"

	// Software events
	CPUClock        Type = "CPUClock" // cpu clock is broken: https://stackoverflow.com/a/56967896
	TaskClock       Type = "TaskClock"
	PageFaults      Type = "PageFaults"
	ContextSwitches Type = "ContextSwitches"
	CPUMigrations   Type = "CPUMigrations"
	PageFaultsMin   Type = "PageFaultsMin"
	PageFaultsMaj   Type = "PageFaultsMaj"
	AlignmentFaults Type = "AlignmentFaults"
	EmulationFaults Type = "EmulationFaults"
	Dummy           Type = "Dummy"

	// Some of the hardware cache events
	L1DataCacheLoadReferences         Type = "L1DataCacheLoadReferences"
	L1DataCacheLoadMisses             Type = "L1DataCacheLoadMisses"
	L1DataCacheStoreReferences        Type = "L1DataCacheStoreReferences"
	L1DataCacheStoreMisses            Type = "L1DataCacheStoreMisses"
	L1InstructionCacheLoadReferences  Type = "L1InstructionCacheLoadReferences"
	L1InstructionCacheLoadMisses      Type = "L1InstructionCacheLoadMisses"
	L1InstructionCacheStoreReferences Type = "L1InstructionCacheStoreReferences"
	L1InstructionCacheStoreMisses     Type = "L1InstructionCacheStoreMisses"
	LLCacheLoadReferences             Type = "LLCacheLoadReferences"
	LLCacheLoadMisses                 Type = "LLCacheLoadMisses"
	LLCacheStoreReferences            Type = "LLCacheStoreReferences"
	LLCacheStoreMisses                Type = "LLCacheStoreMisses"
	DataTLBLoadReferences             Type = "DataTLBLoadReferences"
	DataTLBLoadMisses                 Type = "DataTLBLoadMisses"
	DataTLBStoreReferences            Type = "DataTLBStoreReferences"
	DataTLBStoreMisses                Type = "DataTLBStoreMisses"
	InstructionTLBLoadReferences      Type = "InstructionTLBLoadReferences"
	InstructionTLBLoadMisses          Type = "InstructionTLBLoadMisses"
	InstructionTLBStoreReferences     Type = "InstructionTLBStoreReferences"
	InstructionTLBStoreMisses         Type = "InstructionTLBStoreMisses"
	LocalNodeMemoryLoadReferences     Type = "LocalNodeMemoryLoadReferences"
	LocalNodeMemoryLoadMisses         Type = "LocalNodeMemoryLoadMisses"
	LocalNodeMemoryStoreReferences    Type = "LocalNodeMemoryStoreReferences"
	LocalNodeMemoryStoreMisses        Type = "LocalNodeMemoryStoreMisses"
)
