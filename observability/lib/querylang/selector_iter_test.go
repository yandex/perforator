package querylang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parserv2 "github.com/yandex/perforator/observability/lib/querylang/parser/v2"
)

func TestCandidates(t *testing.T) {
	for _, tc := range []struct {
		Query    string
		Expected map[string][]string
	}{
		{
			Query: `{ a = "b" }`,
			Expected: map[string][]string{
				"a": {`"b"`},
			},
		},
		{
			Query: `{ a = "b", a = "c", b = "d", b != "d" }`,
			Expected: map[string][]string{
				// Can find a starting set, no candidates remained.
				"a": {},
				"b": {},
			},
		},
		{
			Query: `{ a = "b|d", a = "c|d" }`,
			Expected: map[string][]string{
				"a": {`"d"`},
			},
		},
		{
			Query: `{ a = "b|d", a != "c|d" }`,
			Expected: map[string][]string{
				"a": {`"b"`},
			},
		},
		{
			Query: `{ a = "b|*", a != "c|*" }`,
			Expected: map[string][]string{
				// Cannot find a starting set.
				"a": nil,
			},
		},
		{
			Query: `{ a = "a", b = "b", c = "c" }`,
			Expected: map[string][]string{
				"a": {`"a"`},
				"b": {`"b"`},
				"c": {`"c"`},
			},
		},
		{
			Query: `{ a = "a|-", b = "x|y*" }`,
			Expected: map[string][]string{
				// Cannot find a starting set.
				"a": nil,
				"b": nil,
			},
		},
		{
			Query: `{ a = "x|y", a != "y*" }`,
			Expected: map[string][]string{
				"a": {`"x"`, `"y"`},
			},
		},
		{
			Query: `{ b = "x|y|z", b != "y|z*" }`,
			Expected: map[string][]string{
				"b": {`"x"`, `"z"`},
			},
		},
		{
			Query: `{ a = "x|y|z", a =~ "x", a !~ "y" }`,
			Expected: map[string][]string{
				"a": {`"x"`, `"y"`, `"z"`},
			},
		},
		{
			Query: `{ a != "x|y" }`,
			Expected: map[string][]string{
				"a": nil,
			},
		},
	} {
		t.Run(tc.Query, func(t *testing.T) {
			parser := parserv2.NewParser()
			selector, err := parser.ParseSelector(tc.Query)
			require.NoError(t, err)

			candidates := selector.CandidateValues()

			assert.Len(t, candidates, len(tc.Expected))

			for field, expected := range tc.Expected {
				c, ok := candidates[field]
				assert.True(t, ok)

				if expected == nil {
					assert.Nil(t, c)
				} else {
					values := make([]string, 0, len(c))
					for _, v := range c {
						values = append(values, v.Repr())
					}
					assert.Equal(t, expected, values)
				}
			}
		})
	}
}
