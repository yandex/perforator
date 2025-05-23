//go:build go1.23

package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColMapRange(t *testing.T) {
	var (
		enc = &ColMap[string, string]{
			Keys:   &ColStr{},
			Values: &ColStr{},
		}
		expected = []map[string]string{
			{
				"a": "b",
				"c": "d",
			},
			{
				"e": "f",
			},
		}
	)
	enc.AppendArr(expected)

	var buf Buffer
	enc.EncodeColumn(&buf)

	var (
		dec = &ColMap[string, string]{
			Keys:   &ColStr{},
			Values: &ColStr{},
		}
		got []map[string]string
	)
	require.NoError(t, dec.DecodeColumn(buf.Reader(), enc.Rows()))
	for rowIdx := range dec.Rows() {
		row := map[string]string{}
		for k, v := range dec.RowRange(rowIdx) {
			row[k] = v
		}
		got = append(got, row)
	}
	require.Equal(t, expected, got)
}
