package sampletype

import (
	"github.com/google/pprof/profile"
)

const (
	SampleTypeCPU    = "cpu"
	SampleTypeWall   = "wall"
	SampleTypeLBR    = "lbr"
	SampleTypeSignal = "signal"

	SampleTypeCPUCycles   = SampleTypeCPU + ".cycles"
	SampleTypeWallSeconds = SampleTypeWall + ".seconds"
	SampleTypeLbrStacks   = SampleTypeLBR + ".stacks"
	SampleTypeSignalCount = SampleTypeSignal + ".count"
)

func SampleTypeToString(sampleType *profile.ValueType) string {
	return sampleType.Type + "." + sampleType.Unit
}
