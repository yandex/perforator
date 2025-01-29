package pprofmetrics

import (
	"github.com/google/pprof/profile"
)

type ProfileMetricsAccumulator struct {
	samplesNumber int64
	stackMaxDepth int64

	stackFramesSum           int64
	unsymbolizedLocationsSum int64
}

func NewProfileMetricsAccumulator(p *profile.Profile) *ProfileMetricsAccumulator {
	a := &ProfileMetricsAccumulator{}
	a.addProfile(p)
	return a
}

////////////////////////////////////////////////////////////////////////////////

func (a *ProfileMetricsAccumulator) isUnsymbolized(l *profile.Location) bool {
	return (l.Mapping == nil || l.Mapping.File != "[JIT]") && len(l.Line) == 0
}

func (a *ProfileMetricsAccumulator) addProfile(p *profile.Profile) {
	a.samplesNumber = int64(len(p.Sample))

	for _, s := range p.Sample {
		// Max Depth.
		if int64(len(s.Location)) > a.stackMaxDepth {
			a.stackMaxDepth = int64(len(s.Location))
		}

		// Frames.
		a.stackFramesSum += int64(len(s.Location))

		// Unsymbolized.
		for _, l := range s.Location {
			if a.isUnsymbolized(l) {
				a.unsymbolizedLocationsSum++
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func (a *ProfileMetricsAccumulator) StackMaxDepth() int64 {
	return a.stackMaxDepth
}

func (a *ProfileMetricsAccumulator) StackFramesSum() int64 {
	return a.stackFramesSum
}

func (a *ProfileMetricsAccumulator) SamplesNumber() int64 {
	return a.samplesNumber
}

func (a *ProfileMetricsAccumulator) UnsymbolizedNumberSum() int64 {
	return a.unsymbolizedLocationsSum
}
