// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespWriterWriteHeader(t *testing.T) {
	rw := NewRespWriterWrapper(&httptest.ResponseRecorder{}, func(int64) {})

	rw.WriteHeader(http.StatusTeapot)
	assert.Equal(t, http.StatusTeapot, rw.statusCode)
	assert.True(t, rw.wroteHeader)

	rw.WriteHeader(http.StatusGone)
	assert.Equal(t, http.StatusTeapot, rw.statusCode)
}

func TestRespWriterFlush(t *testing.T) {
	rw := NewRespWriterWrapper(&httptest.ResponseRecorder{}, func(int64) {})

	rw.Flush()
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.True(t, rw.wroteHeader)
}

type nonFlushableResponseWriter struct{}

func (_ nonFlushableResponseWriter) Header() http.Header {
	return http.Header{}
}

func (_ nonFlushableResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (_ nonFlushableResponseWriter) WriteHeader(int) {}

func TestRespWriterFlushNoFlusher(t *testing.T) {
	rw := NewRespWriterWrapper(nonFlushableResponseWriter{}, func(int64) {})

	rw.Flush()
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.True(t, rw.wroteHeader)
}

func TestConcurrentRespWriterWrapper(t *testing.T) {
	rw := NewRespWriterWrapper(&httptest.ResponseRecorder{}, func(int64) {})

	go func() {
		_, _ = rw.Write([]byte("hello world"))
	}()

	assert.NotNil(t, rw.BytesWritten())
	assert.NotNil(t, rw.StatusCode())
	assert.NoError(t, rw.Error())
}
