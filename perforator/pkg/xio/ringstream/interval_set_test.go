package ringstream

import "testing"

func TestIntervalSet_InOrder(t *testing.T) {
	var s intervalSet
	s.add(0, 5)
	if s.frontier != 5 {
		t.Fatalf("frontier=%d, want 5", s.frontier)
	}
	s.add(5, 10)
	if s.frontier != 10 {
		t.Fatalf("frontier=%d, want 10", s.frontier)
	}
	if len(s.pending) != 0 {
		t.Fatalf("pending=%v, want empty", s.pending)
	}
}

func TestIntervalSet_OutOfOrder(t *testing.T) {
	var s intervalSet
	// Write [5,10) before [0,5)
	s.add(5, 10)
	if s.frontier != 0 {
		t.Fatalf("frontier=%d, want 0 before gap is filled", s.frontier)
	}
	if len(s.pending) != 1 {
		t.Fatalf("pending=%v, want 1 entry", s.pending)
	}
	s.add(0, 5)
	if s.frontier != 10 {
		t.Fatalf("frontier=%d, want 10", s.frontier)
	}
	if len(s.pending) != 0 {
		t.Fatalf("pending=%v, want empty", s.pending)
	}
}

func TestIntervalSet_GapNotYetFilled(t *testing.T) {
	var s intervalSet
	s.add(0, 3)
	s.add(7, 10)
	if s.frontier != 3 {
		t.Fatalf("frontier=%d, want 3 (gap [3,7) not filled)", s.frontier)
	}
	s.add(3, 7)
	if s.frontier != 10 {
		t.Fatalf("frontier=%d, want 10", s.frontier)
	}
}

func TestIntervalSet_OverlappingPending(t *testing.T) {
	var s intervalSet
	// Add two overlapping out-of-order intervals, then fill the prefix.
	s.add(3, 8)
	s.add(6, 12)
	if s.frontier != 0 {
		t.Fatalf("frontier=%d, want 0", s.frontier)
	}
	if len(s.pending) != 1 || s.pending[0] != (interval{3, 12}) {
		t.Fatalf("pending=%v, want [{3,12}]", s.pending)
	}
	s.add(0, 3)
	if s.frontier != 12 {
		t.Fatalf("frontier=%d, want 12", s.frontier)
	}
}

func TestIntervalSet_OverlapWithFrontier(t *testing.T) {
	var s intervalSet
	s.add(0, 5)
	// Retry: re-add overlapping range [3,8) — should just extend frontier.
	s.add(3, 8)
	if s.frontier != 8 {
		t.Fatalf("frontier=%d, want 8", s.frontier)
	}
	if len(s.pending) != 0 {
		t.Fatalf("pending=%v, want empty", s.pending)
	}
}

func TestIntervalSet_PendingMergeChain(t *testing.T) {
	var s intervalSet
	// Build a chain of pending intervals, then fill prefix.
	s.add(10, 20)
	s.add(20, 30)
	s.add(30, 40)
	if s.frontier != 0 {
		t.Fatalf("frontier=%d, want 0", s.frontier)
	}
	s.add(0, 10)
	if s.frontier != 40 {
		t.Fatalf("frontier=%d, want 40", s.frontier)
	}
	if len(s.pending) != 0 {
		t.Fatalf("pending=%v, want empty", s.pending)
	}
}

func TestIntervalSet_FrontierNotAdvancedByDisjointPending(t *testing.T) {
	var s intervalSet
	s.add(0, 5)
	s.add(10, 15)
	s.add(20, 25)
	// Frontier is 5; pending has two disjoint intervals.
	if s.frontier != 5 {
		t.Fatalf("frontier=%d, want 5", s.frontier)
	}
	if len(s.pending) != 2 {
		t.Fatalf("pending=%v, want 2 entries", s.pending)
	}
	// Fill the first gap.
	s.add(5, 10)
	if s.frontier != 15 {
		t.Fatalf("frontier=%d, want 15", s.frontier)
	}
	if len(s.pending) != 1 {
		t.Fatalf("pending=%v, want 1 entry", s.pending)
	}
	// Fill the second gap.
	s.add(15, 20)
	if s.frontier != 25 {
		t.Fatalf("frontier=%d, want 25", s.frontier)
	}
	if len(s.pending) != 0 {
		t.Fatalf("pending=%v, want empty", s.pending)
	}
}
