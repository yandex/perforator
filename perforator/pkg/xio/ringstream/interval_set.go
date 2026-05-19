package ringstream

// interval is a half-open written range [start, end).
type interval struct{ start, end int64 }

// intervalSet tracks a set of non-overlapping written [start, end) intervals
// and maintains a contiguous frontier: the largest value f such that all bytes
// in [0, f) have been covered by added intervals.
//
// Intervals may be added in any order; the frontier advances automatically as
// gaps between the frontier and pending intervals are filled.
type intervalSet struct {
	pending  []interval // sorted by start, non-overlapping, all start > frontier
	frontier int64
}

// add records that [start, end) has been written and advances the frontier as
// far as contiguous coverage allows.
func (s *intervalSet) add(start, end int64) {
	if start <= s.frontier {
		if end > s.frontier {
			s.frontier = end
		}
		s.drain()
		return
	}
	// Insert into pending sorted by start.
	idx := 0
	for idx < len(s.pending) && s.pending[idx].start < start {
		idx++
	}
	s.pending = append(s.pending, interval{})
	copy(s.pending[idx+1:], s.pending[idx:])
	s.pending[idx] = interval{start: start, end: end}
	s.merge()
	// No drain needed: all pending[i].start > frontier after insertion.
}

// merge collapses adjacent/overlapping intervals in s.pending after insertion.
func (s *intervalSet) merge() {
	if len(s.pending) <= 1 {
		return
	}
	out := s.pending[:1]
	for _, iv := range s.pending[1:] {
		last := &out[len(out)-1]
		if iv.start <= last.end {
			if iv.end > last.end {
				last.end = iv.end
			}
			continue
		}
		out = append(out, iv)
	}
	s.pending = out
}

// drain advances the frontier through any pending intervals that are now
// contiguous with it.
func (s *intervalSet) drain() {
	for len(s.pending) > 0 && s.pending[0].start <= s.frontier {
		h := s.pending[0]
		if h.end > s.frontier {
			s.frontier = h.end
		}
		s.pending = s.pending[1:]
	}
}
