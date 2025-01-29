package slices

import (
	"golang.org/x/exp/slices"
)

// ContainsString checks if string slice contains given string.
// Deprecated: use Contains instead.
var ContainsString = slices.Contains[[]string, string]

// Contains checks if slice contains given element.
func Contains[E comparable](slice []E, element E) bool {
	return slices.Contains(slice, element)
}

// ContainsAny checks if slice contains any of given elements.
func ContainsAny[E comparable](slice, elements []E) bool {
	return len(Intersection(slice, elements)) > 0
}

// ContainsAll checks if slice contains all given elements, order independent.
func ContainsAll[E comparable](slice, elements []E) bool {
	set := make(map[E]struct{}, len(slice))
	for _, v := range slice {
		set[v] = struct{}{}
	}
	for _, v := range elements {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}
