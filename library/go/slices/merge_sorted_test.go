package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestMerge(t *testing.T) {
	type testCase[E comparable] struct {
		a        []E
		b        []E
		expected []E
	}

	testCases := []testCase[string]{
		{nil, nil, nil},
		{[]string{}, []string{}, []string{}},
		{[]string{""}, []string{""}, []string{"", ""}},
		{nil, []string{"c", "d"}, nil},
		{[]string{"a", "b"}, nil, nil},
		{[]string{}, []string{"c", "d"}, []string{"c", "d"}},
		{[]string{"a", "b"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "b", "c"}, []string{"c", "d"}, []string{"a", "b", "c", "c", "d"}},
		{[]string{"a", "b", "b", "c"}, []string{"c", "d", "b", "b"}, nil},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			isec := slices.MergeSorted(tc.a, tc.b)
			assert.Equal(t, tc.expected, isec)
		})
	}
}

func TestUniqueMerge(t *testing.T) {
	type testCase[E comparable] struct {
		a        []E
		b        []E
		expected []E
	}

	testCases := []testCase[string]{
		{nil, nil, nil},
		{[]string{}, []string{}, []string{}},
		{[]string{""}, []string{""}, []string{""}},
		{nil, []string{"c", "d"}, nil},
		{[]string{"a", "b"}, nil, nil},
		{[]string{}, []string{"c", "d"}, []string{"c", "d"}},
		{[]string{"a", "b"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "b", "c"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "b", "b", "c"}, []string{"c", "d", "b", "b"}, nil},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			isec := slices.UniqueMergeSorted(tc.a, tc.b)
			assert.Equal(t, tc.expected, isec)
		})
	}
}
