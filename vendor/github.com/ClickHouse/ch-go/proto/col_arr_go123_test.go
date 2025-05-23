//go:build go1.23

package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColArrRange(t *testing.T) {
	var (
		enc      = new(ColStr).Array()
		expected = [][]string{
			{"foo", "bar", "foo", "foo", "baz"},
			{"foo", "baz"},
		}
	)
	enc.AppendArr(expected)

	var buf Buffer
	enc.EncodeColumn(&buf)

	var (
		dec = new(ColStr).Array()
		got [][]string
	)
	require.NoError(t, dec.DecodeColumn(buf.Reader(), enc.Rows()))
	for rowIdx := range dec.Rows() {
		var row []string
		for e := range dec.RowRange(rowIdx) {
			row = append(row, e)
		}
		got = append(got, row)
	}
	require.Equal(t, expected, got)
}
