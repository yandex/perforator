package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestDedup(t *testing.T) {
	testCases := []struct {
		given, expected []string
	}{
		{
			[]string{"42"},
			[]string{"42"},
		},
		{
			[]string{"1", "2", "3", "4", "4", "3", "2", "1"},
			[]string{"1", "2", "3", "4"},
		},
		{
			[]string{"ololo", "trololo"},
			[]string{"ololo", "trololo"},
		},
		{
			[]string{"yandex", "google", "bing", "bing", "bing"},
			[]string{"bing", "google", "yandex"},
		},
		{
			[]string{"4", "3", "2", "1"},
			[]string{"1", "2", "3", "4"},
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, slices.DedupStrings(tc.given))
	}
}

func TestDedupBools(t *testing.T) {
	testCases := []struct {
		given, expected []bool
	}{
		{
			[]bool{true},
			[]bool{true},
		},
		{
			[]bool{true, false, true, true, false, false, true},
			[]bool{false, true},
		},
		{
			[]bool{true, true, true, true},
			[]bool{true},
		},
		{
			[]bool{true, false},
			[]bool{false, true},
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, slices.DedupBools(tc.given))
	}
}

func BenchmarkDedup(b *testing.B) {
	benchCases := []struct {
		name string
		sets [][]string
	}{
		{
			"small",
			[][]string{
				{"1", "2", "3", "4", "4", "3", "2", "1"},
				{"ololo", "trololo"},
				{"yandex", "google", "bing", "bing", "bing"},
			},
		},
		{
			"large_duplicated",
			[][]string{
				{
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
					"same", "same", "same", "same", "same", "same",
				},
				{
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
					"case", "case", "case", "case", "case", "case",
				},
			},
		},
		{
			"large_uniq",
			[][]string{
				{
					"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
					"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
					"u", "v", "w", "x", "y", "z", "A", "B", "C", "D",
					"E", "F", "G", "H", "I", "J", "K", "L", "M", "N",
					"O", "P", "Q", "R", "S", "T", "U", "V", "W", "X",
					"Y", "Z", "0", "1", "2", "3", "4", "5", "6", "7",
					"8", "9",
				},
				{
					"Lorem", "ipsum", "dolor", "sit", "amet", "consectetur",
					"adipiscing", "elit", "sed", "do", "eiusmod", "tempor",
					"incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua",
				},
			},
		},
	}

	b.ReportAllocs()
	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				slices.Dedup(bc.sets[i%len(bc.sets)])
			}
		})
	}
}
