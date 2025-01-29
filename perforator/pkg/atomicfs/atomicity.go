package atomicfs

type AtomicityLevel int

const (
	AtomicityFull = iota
	AtomicityNoSync
)
