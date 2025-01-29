package disjointsegmentsets

// Item is a helper interface for the Prune alrogithm
type Item interface {
	SegmentBegin() uint64
	SegmentEnd() uint64
	GenerationNumber() int
}

// Prune removes all items which overlap with newer (as determined by Generation()) once.
// Precondition: input slice must be sorted by Start().
// Note that input slice is modified by the call.
// Return values: (items that were retained, items that were pruned)
func Prune[T Item](items []T) ([]T, []T) {
	// Indices of maps to prune
	invalidated := make(map[int]struct{})

	// The index of the leftmost item such that for the current item i holds
	// items[i].Begin() < items[firstincover].End().
	firstincover := -1
	bestincover := -1

	for i, item := range items {
		if firstincover == -1 || items[firstincover].SegmentEnd() <= item.SegmentBegin() {
			// No overlap
			firstincover = i
			bestincover = i
			continue
		}

		if items[bestincover].GenerationNumber() > item.GenerationNumber() {
			invalidated[i] = struct{}{}
		} else {
			invalidated[bestincover] = struct{}{}

			bestincover = i
		}
	}

	pruned := make([]T, 0, len(invalidated))

	it := 0
	for i := range items {
		_, has := invalidated[i]
		if has {
			pruned = append(pruned, items[i])
		} else {
			items[it] = items[i]
			it++
		}
	}
	return items[:it], pruned
}
