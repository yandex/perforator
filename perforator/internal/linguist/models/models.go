package models

const (
	UnsymbolizedInterpreterLocation = "<unsymbolized interpreter>"
)

// Language identifies an interpreter using the shared BPF language IDs.
type Language uint8

type RuntimeFrameCounts struct {
	Frames       int64
	Unsymbolized int64
}
