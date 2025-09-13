package slices

import "golang.org/x/exp/maps"

// Union returns a slice containing unique elements from all input slices
func Union[E comparable](slices ...[]E) []E {
	seen := make(map[E]struct{})
	for _, slice := range slices {
		for _, v := range slice {
			seen[v] = struct{}{}
		}
	}
	return maps.Keys(seen)
}
