package parserv2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parserv2 "github.com/yandex/perforator/observability/lib/querylang/parser/v2"
)

type testCase struct {
	Query         string
	ExpectedRepr  string
	ExpectedStr   string
	ExpectedError string
}

func TestParseSelector(t *testing.T) {
	for _, tc := range []testCase{
		{
			Query:        `{"project"="a", service="b", "cluster"=c, other==d}`,
			ExpectedRepr: `"project" = "a" AND "service" = "b" AND "cluster" = "c" AND "other" = "d"`,
		},
		{
			Query:        `{project="a*", service="b?2", cluster != "c?3|c*4"}`,
			ExpectedRepr: `"project" glob "a*" AND "service" glob "b?2" AND "cluster" !glob "c?3" AND "cluster" !glob "c*4"`,
		},
		{
			Query:        `{project="a*", service=="b*", cluster!="c*", other!=="d|c"}`,
			ExpectedRepr: `"project" glob "a*" AND "service" = "b*" AND "cluster" !glob "c*" AND "other" != "d|c"`,
		},
		{
			Query:        `{a="*", b="-", c=-}`,
			ExpectedRepr: `"a" exists AND "b" !exists AND "c" !exists`,
		},
		{
			Query:        `{a!="*", b!="-"}`,
			ExpectedRepr: `"a" !exists AND "b" exists`,
		},
		{
			Query:        `{a=~"1", b!~"2"}`,
			ExpectedRepr: `"a" regex "1" AND "b" !regex "2"`,
		},
		{
			Query:        `{x > 10, y="a|b|c"}`,
			ExpectedRepr: `"x" > 10 AND ("y" = "a" OR "y" = "b" OR "y" = "c")`,
		},
		{
			Query:        `{x > -10}`,
			ExpectedRepr: `"x" > -10`,
		},
		{
			Query:        `{x <= -1.01e2}`,
			ExpectedRepr: `"x" <= -101`,
		},
		{
			Query:        `{x > "abc"}`,
			ExpectedRepr: `"x" > "abc"`,
		},
		{
			Query:        `{x = "abc*"}`,
			ExpectedRepr: `"x" glob "abc*"`,
		},
		{
			Query:        `{x != "*abc"}`,
			ExpectedRepr: `"x" !glob "*abc"`,
		},
		{
			Query:        `{x = "a|abc*"}`,
			ExpectedRepr: `("x" = "a" OR "x" glob "abc*")`,
		},
		{
			Query:        `{x != "a|b"}`,
			ExpectedRepr: `"x" != "a" AND "x" != "b"`,
		},
		{
			Query:        `{x != "a|abc*"}`,
			ExpectedRepr: `"x" != "a" AND "x" !glob "abc*"`,
		},
		{
			Query:        `{x = "a|*|-"}`,
			ExpectedRepr: `("x" = "a" OR "x" exists OR "x" !exists)`,
		},
		{
			Query:        `{x != "a|*|-"}`,
			ExpectedRepr: `"x" != "a" AND "x" !exists AND "x" exists`,
		},
		{
			Query:        `{x =* "a|b"}`,
			ExpectedRepr: `("x" isubstring "a" OR "x" isubstring "b")`,
		},
		{
			Query:        `{x !=* "a|b"}`,
			ExpectedRepr: `"x" !isubstring "a" AND "x" !isubstring "b"`,
		},
		{
			Query:        `{}`,
			ExpectedRepr: ``,
		},
		{
			Query:        `{x = k}`,
			ExpectedRepr: `"x" = "k"`,
		},
		{
			Query:        `{x = 1k}`,
			ExpectedRepr: `"x" = 1000`,
		},
		{
			Query:        `{x = 1.5M}`,
			ExpectedRepr: `"x" = 1500000`,
		},
		{
			Query:        `{x = 123E}`,
			ExpectedRepr: `"x" = 123000000000000000000`,
		},
		// weird but allowed by grammar. 1_000 * 1_000
		{
			Query:        `{x = 1e3k}`,
			ExpectedRepr: `"x" = 1000000`,
		},
		{
			Query:        `{x = -1e3k}`,
			ExpectedRepr: `"x" = -1000000`,
		},
		{
			Query:        `{x > 1s}`,
			ExpectedRepr: `"x" > 1s`,
		},
		{
			Query:        `{x >= 1ms}`,
			ExpectedRepr: `"x" >= 1ms`,
		},
		{
			Query:        `{x <= '1s'}`,
			ExpectedRepr: `"x" <= "1s"`,
		},
		{
			Query:        `{a =~ "1|2"}`,
			ExpectedRepr: `"a" regex "1|2"`,
		},
		{
			Query:        `{x !~ "a|b"}`,
			ExpectedRepr: `"x" !regex "a|b"`,
		},

		{
			Query:         `{x > 1p}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x > 1ks}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x > 1sk}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x > 1e3s}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x > 1se3}`,
			ExpectedError: `syntax error`,
		},

		{
			Query:         `{x > -1s}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x = 1s}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x = -+10}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x = 10|20}`,
			ExpectedError: `syntax error`,
		},
		{
			Query:         `{x > }`,
			ExpectedError: "syntax error",
		},
		{
			Query:         `{x =~ 123}`,
			ExpectedError: "syntax error",
		},
		{
			Query:         `{x = 1EE}`,
			ExpectedError: `syntax error`,
		},

		{
			Query:         `({x = 123})`,
			ExpectedError: "syntax error",
		},
		{
			Query:         `function({x = 123})`,
			ExpectedError: "syntax error",
		},
	} {
		t.Run(tc.Query, func(t *testing.T) {
			p := parserv2.NewParser()
			s, err := p.ParseSelector(tc.Query)
			if tc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.ExpectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedRepr, s.Repr())
			}
		})
	}
}

func TestParseExpression(t *testing.T) {
	for _, tc := range []testCase{
		{
			Query:        `{}`,
			ExpectedStr:  `{}`,
			ExpectedRepr: `{}`,
		},
		{
			Query:        `{a = b, c > 10, d = 'a|b'}`,
			ExpectedStr:  `{"a" = "b", "c" > 10, "d" = "a|b"}`,
			ExpectedRepr: `{"a" = "b" AND "c" > 10 AND ("d" = "a" OR "d" = "b")}`,
		},
		{
			Query:        `some_func()`,
			ExpectedStr:  `some_func()`,
			ExpectedRepr: `some_func()`,
		},
		{
			Query:        `some_func('a', 1, 2.5)`,
			ExpectedStr:  `some_func("a", 1, 2.5)`,
			ExpectedRepr: `some_func("a", 1, 2.5)`,
		},
		{
			Query:        `some_func(a, 1, 2.5)`,
			ExpectedStr:  `some_func(a, 1, 2.5)`,
			ExpectedRepr: `some_func(a, 1, 2.5)`,
		},
		{
			Query:        `filter({a = b}, all(eq('a', 'b')))`,
			ExpectedStr:  `filter({"a" = "b"}, all(eq("a", "b")))`,
			ExpectedRepr: `filter({"a" = "b"}, all(eq("a", "b")))`,
		},
		{
			Query:        `filter({a = 1}, any( all(eq('a', 'b'), eq('c', 'd')), eq('x', 'y') ))`,
			ExpectedStr:  `filter({"a" = 1}, any(all(eq("a", "b"), eq("c", "d")), eq("x", "y")))`,
			ExpectedRepr: `filter({"a" = 1}, any(all(eq("a", "b"), eq("c", "d")), eq("x", "y")))`,
		},
		{
			Query:        `filter({a = b}, x -> x)`,
			ExpectedStr:  `filter({"a" = "b"}, x -> x)`,
			ExpectedRepr: `filter({"a" = "b"}, (x) -> x)`,
		},
		{
			Query:        `filter((x, y) -> x)`,
			ExpectedStr:  `filter((x, y) -> x)`,
			ExpectedRepr: `filter((x, y) -> x)`,
		},
		{
			Query:        `filter(x -> eq(get(x, 'a'), 'x'))`,
			ExpectedStr:  `filter(x -> eq(get(x, "a"), "x"))`,
			ExpectedRepr: `filter((x) -> eq(get(x, "a"), "x"))`,
		},
		{
			Query:        `filter({project="smth", service='wow'}, x -> all(eq(get(x, 'a'), 'x')))`,
			ExpectedStr:  `filter({"project" = "smth", "service" = "wow"}, x -> all(eq(get(x, "a"), "x")))`,
			ExpectedRepr: `filter({"project" = "smth" AND "service" = "wow"}, (x) -> all(eq(get(x, "a"), "x")))`,
		},

		{
			Query:         `filter({a = b}) by 'a'`,
			ExpectedError: `semantic error`,
		},
		{
			Query:         `{a = b} + {c = d}`,
			ExpectedError: `semantic error`,
		},
		{
			Query:         `!filter({a = b})`,
			ExpectedError: `semantic error`,
		},
		{
			Query:         `filter({a = b}) && true`,
			ExpectedError: `semantic error`,
		},
		{
			Query:         `filter({a = b}) || true`,
			ExpectedError: `semantic error`,
		},
	} {
		t.Run(tc.Query, func(t *testing.T) {
			p := parserv2.NewParser()
			e, err := p.ParseExpression(tc.Query)
			if tc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.ExpectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedRepr, e.Repr())

				str, err := e.ToString()
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedStr, str)
			}
		})
	}
}
