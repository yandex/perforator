package slices

import (
	"iter"
	"maps"
)

// Zip converts slices ([]T, []M) to iter.Seq2[T, M]
func Zip[S1 ~[]T, S2 ~[]M, T, M any](a S1, b S2) iter.Seq2[T, M] {
	return func(yield func(T, M) bool) {
		n := min(len(a), len(b))
		for i := range n {
			if !yield(a[i], b[i]) {
				return
			}
		}
	}
}

// ZipToMap converts slices ([]K, []V) to map[K]V
func ZipToMap[S1 ~[]K, S2 ~[]V, K comparable, V any](keys S1, values S2) map[K]V {
	return maps.Collect(Zip(keys, values))
}
