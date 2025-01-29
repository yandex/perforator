package slices

import (
	"cmp"
	"slices"
)

// Sort is like slices.Sort but returns sorted copy of given slice
func Sort[T cmp.Ordered](s []T) []T {
	s2 := make([]T, len(s))
	copy(s2, s)
	slices.Sort(s2)
	return s2
}

// SortBy sorts a slice in place using given sortKey
func SortBy[S ~[]E, E any, U cmp.Ordered](x S, sortKey func(E) U) {
	slices.SortFunc(x, func(a, b E) int { return cmp.Compare(sortKey(a), sortKey(b)) })
}

// SortDescBy sorts a slice in place using given sortKey in descending order
func SortDescBy[S ~[]E, E any, U cmp.Ordered](x S, sortKey func(E) U) {
	slices.SortFunc(x, func(a, b E) int { return cmp.Compare(sortKey(b), sortKey(a)) })
}

// SortStableBy sorts a slice in place using given sortKey, uses stable sorting
func SortStableBy[S ~[]E, E any, U cmp.Ordered](x S, sortKey func(E) U) {
	slices.SortStableFunc(x, func(a, b E) int { return cmp.Compare(sortKey(a), sortKey(b)) })
}

// SortDescStableBy sorts a slice in place using given sortKey in descending order, uses stable sorting
func SortDescStableBy[S ~[]E, E any, U cmp.Ordered](x S, sortKey func(E) U) {
	slices.SortStableFunc(x, func(a, b E) int { return cmp.Compare(sortKey(b), sortKey(a)) })
}
