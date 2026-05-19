package ringstream

// ringBuf is a fixed-capacity byte buffer that maps ever-increasing logical
// offsets to physical positions via modular arithmetic. It does not track
// read/write progress — callers are responsible for range validation.
type ringBuf struct {
	data     []byte
	capacity int64
}

func newRingBuf(capacity int) ringBuf {
	return ringBuf{
		data:     make([]byte, capacity),
		capacity: int64(capacity),
	}
}

// write copies p into the ring at logical offset off, handling wrap-around.
func (r *ringBuf) write(off int64, p []byte) {
	start := off % r.capacity
	tail := r.capacity - start
	if int64(len(p)) <= tail {
		copy(r.data[start:], p)
		return
	}
	copy(r.data[start:], p[:tail])
	copy(r.data[0:], p[tail:])
}

// read copies len(p) bytes from the ring starting at logical offset off into p.
func (r *ringBuf) read(off int64, p []byte) {
	start := off % r.capacity
	tail := r.capacity - start
	if int64(len(p)) <= tail {
		copy(p, r.data[start:start+int64(len(p))])
		return
	}
	copy(p, r.data[start:r.capacity])
	copy(p[tail:], r.data[0:int64(len(p))-tail])
}
