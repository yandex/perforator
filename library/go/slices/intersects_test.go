package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestIntersection(t *testing.T) {
	type testCase[E comparable] struct {
		a           []E
		b           []E
		expected    []E
		expectedErr error
	}

	testCases := []testCase[string]{
		{nil, nil, nil, nil},
		{nil, []string{"c", "d"}, nil, nil},
		{[]string{"a", "b"}, nil, nil, nil},
		{[]string{"a", "b"}, []string{"c", "d"}, []string(nil), nil},
		{[]string{"a", "b", "c"}, []string{"c", "d"}, []string{"c"}, nil},
		{[]string{"a", "b", "b", "c"}, []string{"c", "d", "b", "b"}, []string{"b", "b", "c"}, nil},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			isec := slices.Intersection(tc.a, tc.b)
			assert.Equal(t, tc.expected, isec)
		})
	}
}

func BenchmarkIntersection(b *testing.B) {
	benchCases := []struct {
		a, b []string
	}{
		{nil, nil},
		{[]string{"a", "b"}, []string{"c", "d"}},
		{[]string{"a", "b"}, []string{"b", "d"}},
		{[]string{"a", "b"}, []string{"b", "d"}},
		{
			[]string{
				"Lorem", "Ipsum", "is", "simply", "dummy", "text", "of",
				"the", "printing", "and", "typesetting", "industry.",
				"Lorem", "Ipsum", "has", "been", "the", "industry's", "standard",
				"dummy", "text", "ever", "since", "the", "1500s,", "when", "an",
				"unknown", "printer", "took", "a", "galley", "of", "type", "and",
				"scrambled", "it", "to", "make", "a", "type", "specimen", "book.",
			},
			[]string{
				"It", "has", "survived", "not", "only", "five", "centuries,", "but",
				"also", "the", "leap", "into", "electronic", "typesetting,",
				"remaining", "essentially", "unchanged.", "It", "was", "popularized",
				"in", "the", "1960s", "with", "the", "release", "of", "Letraset",
				"sheets", "containing", "Lorem", "Ipsum", "passages,", "and", "more",
				"recently", "with", "desktop", "publishing", "software", "like",
				"Aldus", "PageMaker", "including", "versions", "of", "Lorem", "Ipsum",
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slices.Intersection(benchCases[i%len(benchCases)].a, benchCases[i%len(benchCases)].b)
	}
}
