package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestEqualUnordered(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			"equal sets",
			[]string{"a", "b", "c"},
			[]string{"c", "b", "a"},
			true,
		},
		{
			"empty slices",
			[]string{},
			[]string{},
			true,
		},
		{
			"nil slices",
			nil,
			nil,
			true,
		},
		{
			"slices with duplicates",
			[]string{"a", "b", "a"},
			[]string{"b", "b", "b", "a"},
			false,
		},
		{
			"slices with duplicates",
			[]string{"a", "b", "a"},
			[]string{"a", "b", "b", "a"},
			false,
		},
		{
			"not equal slices",
			[]string{"a", "a", "a"},
			[]string{"b"},
			false,
		},
		{
			"a contains more elements",
			[]string{"a", "b", "c"},
			[]string{"a", "b"},
			false,
		},
		{
			"b contains more elements",
			[]string{"a", "b"},
			[]string{"a", "b", "c"},
			false,
		},
		{
			"slice with equal length and duplicates",
			[]string{"a", "b", "b"},
			[]string{"a", "a", "b"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slices.EqualUnordered(tt.a, tt.b))
		})
	}
}
