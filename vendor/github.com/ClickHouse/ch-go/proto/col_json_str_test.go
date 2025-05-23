package proto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/ch-go/internal/gold"
)

var testJSONValues = []string{
	"{\"x\": 5, \"y\": 10}",
	"{\"a\": \"test\", \"b\": \"test2\"}",
	"{\"a\": \"obj test\", \"b\": {\"c\": 20}}",
}

func TestColJSONBytes(t *testing.T) {
	testColumn(t, "json_bytes", func() ColumnOf[[]byte] {
		return new(ColJSONBytes)
	}, []byte(testJSONValues[0]), []byte(testJSONValues[1]), []byte(testJSONValues[2]))
}

func TestColJSONStr_AppendBytes(t *testing.T) {
	var data ColJSONStr

	data.AppendBytes([]byte(testJSONValues[0]))
	data.AppendBytes([]byte(testJSONValues[1]))
	data.AppendBytes([]byte(testJSONValues[2]))

	var buf Buffer
	data.EncodeColumn(&buf)

	t.Run("Golden", func(t *testing.T) {
		gold.Bytes(t, buf.Buf, "col_json_str_bytes")
	})
	t.Run("Ok", func(t *testing.T) {
		br := bytes.NewReader(buf.Buf)
		r := NewReader(br)

		var dec ColJSONStr
		require.NoError(t, dec.DecodeColumn(r, 3))
		require.Equal(t, data, dec)

		t.Run("ForEach", func(t *testing.T) {
			var output []string
			f := func(i int, s string) error {
				output = append(output, s)
				return nil
			}
			require.NoError(t, dec.ForEach(f))
			require.Equal(t, testJSONValues, output)
		})
	})
}

func TestColJSONStr_EncodeColumn(t *testing.T) {
	var data ColJSONStr

	input := testJSONValues
	rows := len(input)
	for _, s := range input {
		data.Append(s)
	}

	var buf Buffer
	data.EncodeColumn(&buf)

	t.Run("Golden", func(t *testing.T) {
		gold.Bytes(t, buf.Buf, "col_json_str")
	})
	t.Run("Ok", func(t *testing.T) {
		br := bytes.NewReader(buf.Buf)
		r := NewReader(br)

		var dec ColJSONStr
		require.NoError(t, dec.DecodeColumn(r, rows))
		require.Equal(t, data, dec)

		t.Run("ForEach", func(t *testing.T) {
			var output []string
			f := func(i int, s string) error {
				output = append(output, s)
				return nil
			}
			require.NoError(t, dec.ForEach(f))
			require.Equal(t, input, output)
		})
	})
	t.Run("EOF", func(t *testing.T) {
		r := NewReader(bytes.NewReader(nil))

		var dec ColJSONStr
		require.ErrorIs(t, dec.DecodeColumn(r, rows), io.EOF)
	})
}

func BenchmarkColJSONStr_DecodeColumn(b *testing.B) {
	const rows = 1_000
	var data ColJSONStr
	for i := 0; i < rows; i++ {
		data.Append("{\"x\": 5}")
	}

	var buf Buffer
	data.EncodeColumn(&buf)

	br := bytes.NewReader(buf.Buf)
	r := NewReader(br)

	var dec ColJSONStr
	if err := dec.DecodeColumn(r, rows); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(buf.Buf)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		br.Reset(buf.Buf)
		r.raw.Reset(br)
		dec.Reset()

		if err := dec.DecodeColumn(r, rows); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkColJSONStr_EncodeColumn(b *testing.B) {
	const rows = 1_000
	var data ColJSONStr
	for i := 0; i < rows; i++ {
		data.Append("{\"x\": 5}")
	}

	var buf Buffer
	data.EncodeColumn(&buf)

	b.SetBytes(int64(len(buf.Buf)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		data.EncodeColumn(&buf)
	}
}
