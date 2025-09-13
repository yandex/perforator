package slices_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/slices"
)

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		slices   [][]int
		expected []int
	}{
		{name: "no slices", slices: [][]int{}, expected: []int{}},
		{name: "empty slices", slices: [][]int{{}, {}, {}}, expected: []int{}},
		{name: "only one slice", slices: [][]int{{1, 2, 3}}, expected: []int{2, 1, 3}},
		{name: "one non-empty slice", slices: [][]int{{1, 2, 3}, {}, {}}, expected: []int{2, 1, 3}},
		{name: "overlapping slices with duplicates", slices: [][]int{{1, 2, 3, 2}, {2, 3, 4, 4}, {1, 2}}, expected: []int{4, 2, 3, 1}},
		{name: "nil slices", slices: [][]int{[]int(nil), []int(nil)}, expected: []int{}},
		{name: "one non-nil slice", slices: [][]int{{1, 2, 3}, []int(nil)}, expected: []int{2, 1, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := slices.Union(tt.slices...)
			require.Equal(t, slices.Sorted(tt.expected), slices.Sorted(res))
		})
	}
}
