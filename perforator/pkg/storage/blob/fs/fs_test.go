package fs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/storage/blob"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func newTestStorage(t *testing.T) (*FSStorage, string) {
	root := t.TempDir()
	storage, err := NewFSStorage(FSStorageConfig{Root: root}, xlog.ForTest(t))
	require.NoError(t, err)
	return storage, root
}

func TestPutRoundtripAndMode(t *testing.T) {
	storage, root := newTestStorage(t)
	require.NoError(t, storage.Put(context.Background(), "k", bytes.NewReader([]byte("payload"))))

	r, err := storage.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), data)

	info, err := os.Stat(filepath.Join(root, "store", "k"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestPutReplacesExistingKey(t *testing.T) {
	storage, _ := newTestStorage(t)
	require.NoError(t, storage.Put(context.Background(), "k", bytes.NewReader([]byte("old"))))
	require.NoError(t, storage.Put(context.Background(), "k", bytes.NewReader([]byte("new"))))

	r, err := storage.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("new"), data)
}

func TestPutFailureLeavesNoTrace(t *testing.T) {
	storage, root := newTestStorage(t)
	failing := io.MultiReader(bytes.NewReader([]byte("partial")), failingReader{})
	require.Error(t, storage.Put(context.Background(), "k", failing))

	entries, err := os.ReadDir(filepath.Join(root, "store"))
	if err == nil {
		require.Empty(t, entries)
	} else {
		require.True(t, os.IsNotExist(err))
	}
}

func TestGetAndSizeNotFound(t *testing.T) {
	storage, _ := newTestStorage(t)

	_, getErr := storage.Get(context.Background(), "missing")
	_, sizeErr := storage.Size(context.Background(), "missing")

	var getNotExist *blob.ErrNotExist
	var sizeNotExist *blob.ErrNotExist
	require.ErrorAs(t, getErr, &getNotExist)
	require.ErrorAs(t, sizeErr, &sizeNotExist)
	require.Equal(t, "missing", getNotExist.Key)
	require.Equal(t, "missing", sizeNotExist.Key)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
