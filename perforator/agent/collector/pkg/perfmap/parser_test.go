package perfmap

import (
	"bytes"
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseString(t *testing.T, data string) []symbol {
	t.Helper()

	reader := bytes.NewBuffer([]byte(data))
	syms, err := parse(reader)
	require.NoError(t, err, "unexected error when parsing data %q", data)
	slices.SortFunc(syms, func(a, b symbol) int {
		return cmp.Or(
			cmp.Compare(a.offset, b.offset),
			cmp.Compare(a.size, b.size),
			cmp.Compare(a.name, b.name),
		)
	})
	return syms
}

func TestSimple(t *testing.T) {
	data := `100 ff foo
200 fe bar`
	syms := mustParseString(t, data)
	assert.Equal(t, []symbol{
		{index: 0, offset: 0x100, size: 0xff, name: "foo"},
		{index: 1, offset: 0x200, size: 0xfe, name: "bar"},
	}, syms)
}

func TestHexPrefix(t *testing.T) {
	// NodeJS does not add 0x prefix, JVM does.
	// We support both cases.
	data := `0x100 0xff foo
0x200 0xfe bar`
	syms := mustParseString(t, data)
	assert.Equal(t, []symbol{
		{index: 0, offset: 0x100, size: 0xff, name: "foo"},
		{index: 1, offset: 0x200, size: 0xfe, name: "bar"},
	}, syms)
}

func TestWhiteSpaceInNames(t *testing.T) {
	data := `100 ff void TypicalClass.method(java.lang.String s, int x)
200 fe bar`
	syms := mustParseString(t, data)
	assert.Equal(t, []symbol{
		{index: 0, offset: 0x100, size: 0xff, name: "void TypicalClass.method(java.lang.String s, int x)"},
		{index: 1, offset: 0x200, size: 0xfe, name: "bar"},
	}, syms)
}
