// Package profileresult defines the serialized output shared by profiling backends.
package profileresult

import (
	"errors"
	"time"

	gprofile "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
)

// Meta describes a serialized profile, independently of the pipeline that built it.
// Counts are reported by the builder, before any further wire-format aggregation.
type Meta struct {
	BuildIDs        []string
	Envs            []string
	EventTypes      []string
	SignalTypes     []string
	PIDSampleCounts map[int64]int
	SampleCount     int
	StartTimestamp  time.Time
	EndTimestamp    time.Time
}

// Result is the common flush/storage contract for Go and native builders.
// Bundle and Meta must describe the same profile and remain immutable after flush.
type Result struct {
	Bundle *bundle.ProfileBundle
	Meta   Meta
}

// ParsePprof returns an independently mutable profile for consumers needing pprof.
func (r *Result) ParsePprof() (*gprofile.Profile, error) {
	if r == nil || r.Bundle == nil {
		return nil, errors.New("profile has no serialized body")
	}
	data, err := r.Bundle.GetOrConvertPprof()
	if err != nil {
		return nil, err
	}
	return gprofile.ParseData(data)
}
