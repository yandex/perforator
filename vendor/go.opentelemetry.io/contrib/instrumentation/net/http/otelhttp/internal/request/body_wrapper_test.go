// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFirstCall = errors.New("first call")

func TestBodyWrapper(t *testing.T) {
	bw := NewBodyWrapper(io.NopCloser(strings.NewReader("hello world")), func(int64) {})

	data, err := io.ReadAll(bw)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))

	assert.Equal(t, int64(11), bw.BytesRead())
	assert.Equal(t, io.EOF, bw.Error())
}

type multipleErrorsReader struct {
	calls int
}

type errorWrapper struct{}

func (errorWrapper) Error() string {
	return "subsequent calls"
}

func (mer *multipleErrorsReader) Read([]byte) (int, error) {
	mer.calls = mer.calls + 1
	if mer.calls == 1 {
		return 0, errFirstCall
	}

	return 0, errorWrapper{}
}

func TestBodyWrapperWithErrors(t *testing.T) {
	bw := NewBodyWrapper(io.NopCloser(&multipleErrorsReader{}), func(int64) {})

	data, err := io.ReadAll(bw)
	require.Equal(t, errFirstCall, err)
	assert.Equal(t, "", string(data))
	require.Equal(t, errFirstCall, bw.Error())

	data, err = io.ReadAll(bw)
	require.Equal(t, errorWrapper{}, err)
	assert.Equal(t, "", string(data))
	require.Equal(t, errorWrapper{}, bw.Error())
}

func TestConcurrentBodyWrapper(t *testing.T) {
	bw := NewBodyWrapper(io.NopCloser(strings.NewReader("hello world")), func(int64) {})

	go func() {
		_, _ = io.ReadAll(bw)
	}()

	assert.NotNil(t, bw.BytesRead())
	assert.Eventually(t, func() bool {
		return errors.Is(bw.Error(), io.EOF)
	}, time.Second, 10*time.Millisecond)
}
