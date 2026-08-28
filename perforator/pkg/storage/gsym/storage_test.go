package gsym

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/storage/blob/fs"
	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type fakeMetaStorage struct {
	metas map[string]*gsymmeta.GSYMMeta
}

func (f *fakeMetaStorage) GetGSYMs(_ context.Context, buildIDs []string) ([]*gsymmeta.GSYMMeta, error) {
	res := make([]*gsymmeta.GSYMMeta, 0, len(buildIDs))
	for _, id := range buildIDs {
		if meta, ok := f.metas[id]; ok {
			res = append(res, meta)
		}
	}
	return res, nil
}

func (f *fakeMetaStorage) StoreGSYM(context.Context, gsymmeta.GSYMMeta) error { panic("unused") }

func (f *fakeMetaStorage) CollectExpiredGSYMs(context.Context, time.Duration, *util.Pagination) ([]*gsymmeta.GSYMMeta, error) {
	panic("unused")
}

func (f *fakeMetaStorage) RemoveGSYMs(context.Context, []string) error { panic("unused") }

type memWriterAt struct {
	b []byte
}

func (m *memWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if need := int(off) + len(p); need > len(m.b) {
		m.b = append(m.b, make([]byte, need-len(m.b))...)
	}
	copy(m.b[off:], p)
	return len(p), nil
}

func newTestStorage(t *testing.T, metas map[string]*gsymmeta.GSYMMeta) *GSYMStorage {
	l := xlog.ForTest(t)
	blobStorage, err := fs.NewFSStorage(fs.FSStorageConfig{Root: t.TempDir()}, l)
	require.NoError(t, err)
	return NewStorage(&fakeMetaStorage{metas: metas}, blobStorage, l)
}

func putBlob(t *testing.T, s *GSYMStorage, key string, content []byte) {
	require.NoError(t, s.blobStorage.Put(context.Background(), key, bytes.NewReader(content)))
}

func compress(t *testing.T, payload []byte) []byte {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = enc.Write(payload)
	require.NoError(t, err)
	require.NoError(t, enc.Close())
	return buf.Bytes()
}

func testMeta(buildID string, size uint64) *gsymmeta.GSYMMeta {
	return &gsymmeta.GSYMMeta{BuildID: buildID, UncompressedSize: size}
}

func TestLoadGSYM_Roundtrip(t *testing.T) {
	payload := bytes.Repeat([]byte("gsym data "), 1000)
	s := newTestStorage(t, map[string]*gsymmeta.GSYMMeta{
		"a": testMeta("a", uint64(len(payload))),
	})
	putBlob(t, s, "a", compress(t, payload))

	w := &memWriterAt{}
	meta, err := s.LoadGSYM(context.Background(), "a", w)
	require.NoError(t, err)
	require.Equal(t, "a", meta.BuildID)
	require.Equal(t, payload, w.b)
}

func TestLoadGSYM_ExceedsDeclaredSize(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 4096)
	declared := uint64(len(payload)) - 1
	s := newTestStorage(t, map[string]*gsymmeta.GSYMMeta{"a": testMeta("a", declared)})
	putBlob(t, s, "a", compress(t, payload))

	w := &memWriterAt{}
	_, err := s.LoadGSYM(context.Background(), "a", w)
	require.ErrorContains(t, err, "continues past its declared size")
	require.LessOrEqual(t, len(w.b), int(declared))
}

func TestLoadGSYM_EndsShortOfDeclaredSize(t *testing.T) {
	payload := []byte("short payload")
	s := newTestStorage(t, map[string]*gsymmeta.GSYMMeta{
		"a": testMeta("a", uint64(len(payload))+7),
	})
	putBlob(t, s, "a", compress(t, payload))

	_, err := s.LoadGSYM(context.Background(), "a", &memWriterAt{})
	require.ErrorContains(t, err, "short of its declared size")
}

func TestLoadGSYM_CorruptTail(t *testing.T) {
	payload := bytes.Repeat([]byte("y"), 4096)
	blob := compress(t, payload)
	blob[len(blob)-1] ^= 0xff
	s := newTestStorage(t, map[string]*gsymmeta.GSYMMeta{
		"a": testMeta("a", uint64(len(payload))),
	})
	putBlob(t, s, "a", blob)

	_, err := s.LoadGSYM(context.Background(), "a", &memWriterAt{})
	// The decoder may report the damaged frame either on the final data read
	// or at the end probe; either way it must not read as a length mismatch,
	// and it must not be accepted.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "short of its declared size")
}

func TestLoadGSYM_ImplausibleSize(t *testing.T) {
	for _, size := range []uint64{0, math.MaxUint64} {
		s := newTestStorage(t, map[string]*gsymmeta.GSYMMeta{"a": testMeta("a", size)})
		putBlob(t, s, "a", compress(t, []byte("data")))

		_, err := s.LoadGSYM(context.Background(), "a", &memWriterAt{})
		require.ErrorContains(t, err, "implausible uncompressed_size")
	}
}

func TestLoadGSYM_NotFound(t *testing.T) {
	s := newTestStorage(t, nil)
	_, err := s.LoadGSYM(context.Background(), "missing", &memWriterAt{})
	require.Error(t, err)
}
