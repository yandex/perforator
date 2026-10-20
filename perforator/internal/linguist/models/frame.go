package models

type LineInfoOutcome uint8

const (
	LineInfoSkipped LineInfoOutcome = iota
	LineInfoResolved
	LineInfoUnavailable
	LineInfoNoLine
	LineInfoFailed
)

// ResolvedInterpreterFrame owns its strings. Address is the address to emit,
// independent of the raw frame identity used by symbolization caches.
type ResolvedInterpreterFrame struct {
	Mapping      string
	Name         string
	Filename     string
	Address      uint64
	StartLine    int32
	Line         int32
	Unsymbolized bool
	HasStartLine bool
	HasLine      bool
	DoNotCache   bool
	LineInfo     LineInfoOutcome
}
