package slices_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	stdslices "golang.org/x/exp/slices"

	"github.com/yandex/perforator/library/go/slices"
)

func TestFilter(t *testing.T) {
	t.Run("even", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Filter(s, func(v int) bool {
			return v&1 != 1
		})
		expected := []int{2, 4, 6, 8, 10}
		assert.Equal(t, expected, res)
	})
	t.Run("custom_type", func(t *testing.T) {
		type mySlice []int
		s := mySlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Filter(s, func(v int) bool {
			return v&1 != 1
		})
		expected := mySlice{2, 4, 6, 8, 10}
		assert.Equal(t, expected, res)
	})
}

func TestReduce(t *testing.T) {
	t.Run("even_only", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Reduce(s, func(v int) bool {
			return v&1 != 1
		})
		expected := []int{2, 4, 6, 8, 10}
		assert.Equal(t, fmt.Sprintf("%p", s), fmt.Sprintf("%p", res), "slices have different pointers")
		assert.Equal(t, expected, res)
	})
	t.Run("vowel_only", func(t *testing.T) {
		valid := []rune{'a', 'e', 'i', 'o', 'u', 'y'}
		s := []rune{'a', 'b', 'b', 'y', 'k', 'f', 'h', 'o', 'e', 'g', 'i', 'r', 't', 'd', 'q', 'u', 's', 'w'}
		res := slices.Reduce(s, func(v rune) bool {
			return stdslices.Contains(valid, v)
		})
		expected := []rune{'a', 'y', 'o', 'e', 'i', 'u'}
		assert.Equal(t, fmt.Sprintf("%p", s), fmt.Sprintf("%p", res), "slices have different pointers")
		assert.Equal(t, expected, res)
	})
}
