// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

package util

// CloneSliceFunc creates a copy of A by first applying convert to each element.
func CloneSliceFunc[A, B any](original []A, convert func(item A) B) []B {
	clone := make([]B, len(original))
	for i := 0; i < len(original); i++ {
		clone[i] = convert(original[i])
	}
	return clone
}
