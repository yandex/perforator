package slices

import (
	"slices"

	"golang.org/x/exp/constraints"
)

// MergeSorted merges two sorted slices. Returns nil in case of a or b is not sorted
func MergeSorted[E constraints.Ordered](a, b []E) []E {
	if a == nil || b == nil || !slices.IsSorted(a) || !slices.IsSorted(b) {
		return nil
	}

	result := make([]E, len(a)+len(b))
	i, j, k := 0, 0, 0

	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			result[k] = a[i]
			k++
			i++
		} else {
			result[k] = b[j]
			k++
			j++
		}
	}

	for i < len(a) {
		result[k] = a[i]
		k++
		i++
	}

	for j < len(b) {
		result[k] = b[j]
		k++
		j++
	}

	return result
}

// UniqueMergeSorted merges two sorted slices and returns merged slice without duplicates. Returns nil in case of a or b is not sorted
func UniqueMergeSorted[E constraints.Ordered](a, b []E) []E {
	if a == nil || b == nil || !slices.IsSorted(a) || !slices.IsSorted(b) {
		return nil
	}

	result := []E{}
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			if len(result) == 0 || result[len(result)-1] != a[i] {
				result = append(result, a[i])
			}
			i++
		} else {
			if len(result) == 0 || result[len(result)-1] != b[j] {
				result = append(result, b[j])
			}
			j++
		}
	}

	for i < len(a) {
		if len(result) == 0 || result[len(result)-1] != a[i] {
			result = append(result, a[i])
		}
		i++
	}

	for j < len(b) {
		if len(result) == 0 || result[len(result)-1] != b[j] {
			result = append(result, b[j])
		}
		j++
	}

	return result
}
