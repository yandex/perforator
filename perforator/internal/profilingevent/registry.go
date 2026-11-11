package profilingevent

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
)

var (
	registeredEventTypes []Type
	nameToEventType      = make(map[string]Type)
	aliasToEventType     = make(map[string]Type)
)

func GetTypeByNameOrAlias(name string) Type {
	if typ, ok := nameToEventType[name]; ok {
		return typ
	}
	if typ, ok := aliasToEventType[name]; ok {
		return typ
	}
	return nil
}

func registerType(typ Type, aliases []string) (Type, error) {
	if _, ok := nameToEventType[typ.Name()]; ok {
		return nil, fmt.Errorf("event with name %q is already registered", typ.Name())
	}
	for _, alias := range aliases {
		if _, ok := aliasToEventType[alias]; ok {
			return nil, fmt.Errorf("event with alias %q is already registered", alias)
		}
	}
	registeredEventTypes = append(registeredEventTypes, typ)

	nameToEventType[typ.Name()] = typ
	for _, alias := range aliases {
		aliasToEventType[alias] = typ
	}
	return typ, nil
}

func mustRegisterType[T Type](typ T, aliases []string) T {
	_, err := registerType(typ, aliases)
	if err != nil {
		panic(err)
	}
	return typ
}

func getCacheConfig(id, op, result uint32) uint64 {
	return uint64(id | (op << 8) | (result << 16))
}

// See man 2 perf_event_open for the description of the event types.

var (
	CPUCycles = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "cpu", Unit: "cycles"},
			Type:   unix.PERF_TYPE_HARDWARE,
			Config: unix.PERF_COUNT_HW_CPU_CYCLES,
		},
		[]string{"CPUCycles", "cycles"},
	)
	CPUInstructions = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "instructions", Unit: "count"},
			Type:   unix.PERF_TYPE_HARDWARE,
			Config: unix.PERF_COUNT_HW_INSTRUCTIONS,
		},
		[]string{"CPUInstructions"},
	)
	CacheReferences = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "cache-references", Unit: "count"},
			Type:   unix.PERF_TYPE_HARDWARE,
			Config: unix.PERF_COUNT_HW_CACHE_REFERENCES,
		},
		[]string{"CacheReferences"},
	)
	CacheMisses = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "cache-misses", Unit: "count"},
			Type:   unix.PERF_TYPE_HARDWARE,
			Config: unix.PERF_COUNT_HW_CACHE_MISSES,
		},
		[]string{"CacheMisses"},
	)
	LLCacheLoadMisses = mustRegisterType(
		&perfevent.Type{
			Meta: perfevent.TypeMeta{Name: "LLC-load-misses", Unit: "count"},
			Type: unix.PERF_TYPE_HW_CACHE,
			Config: getCacheConfig(
				unix.PERF_COUNT_HW_CACHE_LL,
				unix.PERF_COUNT_HW_CACHE_OP_READ,
				unix.PERF_COUNT_HW_CACHE_RESULT_MISS,
			),
		},
		[]string{"LLCacheLoadMisses"},
	)
	LLCacheStoreMisses = mustRegisterType(
		&perfevent.Type{
			Meta: perfevent.TypeMeta{Name: "LLC-store-misses", Unit: "count"},
			Type: unix.PERF_TYPE_HW_CACHE,
			Config: getCacheConfig(
				unix.PERF_COUNT_HW_CACHE_LL,
				unix.PERF_COUNT_HW_CACHE_OP_WRITE,
				unix.PERF_COUNT_HW_CACHE_RESULT_MISS,
			),
		},
		[]string{"LLCacheStoreMisses"},
	)
	// AMD-specific hardware events
	AMDFam19hBRS = mustRegisterType(
		&perfevent.Type{
			// TODO: replace with name: lbr, and unit: stacks ?
			Meta: perfevent.TypeMeta{Name: "AMDFam19hBRS"},
			// Has to be a raw event
			Type: unix.PERF_TYPE_RAW,
			// AMD-specific constant, not present in unix package
			// https://github.com/torvalds/linux/blob/07e27ad16399afcd693be20211b0dfae63e0615f/arch/x86/events/perf_event.h#L1453
			Config: 0xc4,
		},
		nil,
	)
	// Software events
	// cpu clock is broken: https://stackoverflow.com/a/56967896
	CPUClock = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "cpu-clock", Unit: "seconds"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_CPU_CLOCK,
		},
		[]string{"CPUClock"},
	)
	TaskClock = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "task-clock", Unit: "seconds"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_TASK_CLOCK,
		},
		[]string{"TaskClock"},
	)
	PageFaults = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "page-faults", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_PAGE_FAULTS,
		},
		[]string{"PageFaults", "faults"},
	)
	ContextSwitches = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "context-switches", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_CONTEXT_SWITCHES,
		},
		[]string{"ContextSwitches", "cs"},
	)
	CPUMigrations = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "cpu-migrations", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_CPU_MIGRATIONS,
		},
		[]string{"CPUMigrations", "migrations"},
	)
	PageFaultsMin = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "minor-faults", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_PAGE_FAULTS_MIN,
		},
		[]string{"PageFaultsMin"},
	)
	PageFaultsMaj = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "major-faults", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_PAGE_FAULTS_MAJ,
		},
		[]string{"PageFaultsMaj"},
	)
	AlignmentFaults = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "alignment-faults", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_ALIGNMENT_FAULTS,
		},
		[]string{"AlignmentFaults"},
	)
	EmulationFaults = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "emulation-faults", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_EMULATION_FAULTS,
		},
		[]string{"EmulationFaults"},
	)
	Dummy = mustRegisterType(
		&perfevent.Type{
			Meta:   perfevent.TypeMeta{Name: "dummy", Unit: "count"},
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_DUMMY,
		},
		nil,
	)
)
