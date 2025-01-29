package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/resource"
)

func TestResource(t *testing.T) {
	assert.Equal(t, []byte("hello world"), resource.Get("/a.txt"))

	bindata, err := os.ReadFile("testdata/b.bin")
	assert.NoError(t, err)
	assert.Equal(t, bindata, resource.Get("/b.bin"))

	assert.Equal(t, []byte("bar"), resource.Get("foo"))

	assert.Equal(t, []byte("handle this"), resource.Get("testdata/collision.txt"))

	assert.Equal(t, []string{"/a.txt", "/b.bin", "foo", "testdata/collision.txt"}, resource.Keys())
}

func TestDoubleRegister(t *testing.T) {
	resource.InternalRegister("panic", []byte("foo"))

	require.Panics(t, func() {
		resource.InternalRegister("panic", []byte("foo"))
	})
}
