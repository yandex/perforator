package xio

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSizedReader_ExactSize(t *testing.T) {
	r := NewSizedReader(strings.NewReader("payload"), 7)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "payload", string(got))
}

func TestSizedReader_ShortStream(t *testing.T) {
	r := NewSizedReader(strings.NewReader("payload"), 10)

	_, err := io.ReadAll(r)
	require.ErrorContains(t, err, "ends 3 bytes short of its declared size")
}

func TestSizedReader_LongStream(t *testing.T) {
	r := NewSizedReader(strings.NewReader("payload"), 4)

	got, err := io.ReadAll(r)
	require.ErrorContains(t, err, "continues past its declared size")
	require.Equal(t, "payl", string(got), "nothing past the declared size reaches the caller")
}

// stutterReader returns (0, nil) once, which io.Reader permits and callers
// must treat as "nothing happened".
type stutterReader struct {
	src       io.Reader
	stuttered bool
}

func (r *stutterReader) Read(p []byte) (int, error) {
	if !r.stuttered {
		r.stuttered = true
		return 0, nil
	}
	return r.src.Read(p)
}

func TestSizedReader_ToleratesEmptyRead(t *testing.T) {
	r := NewSizedReader(&stutterReader{src: strings.NewReader("payload")}, 7)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "payload", string(got))
}

type recordingCloser struct {
	name   string
	closed *[]string
	err    error
}

func (c recordingCloser) Close() error {
	*c.closed = append(*c.closed, c.name)
	return c.err
}

func TestReadMultiCloser_ClosesAllInOrder(t *testing.T) {
	var closed []string
	r := NewReadMultiCloser(
		strings.NewReader("payload"),
		recordingCloser{name: "outer", closed: &closed},
		recordingCloser{name: "inner", closed: &closed},
	)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "payload", string(got))
	require.NoError(t, r.Close())
	require.Equal(t, []string{"outer", "inner"}, closed)
}

func TestReadMultiCloser_JoinsCloseErrors(t *testing.T) {
	var closed []string
	boom := errors.New("boom")
	r := NewReadMultiCloser(
		strings.NewReader(""),
		recordingCloser{name: "outer", closed: &closed, err: boom},
		recordingCloser{name: "inner", closed: &closed},
	)

	require.ErrorIs(t, r.Close(), boom)
	require.Equal(t, []string{"outer", "inner"}, closed, "a failing closer must not skip the rest")
}
