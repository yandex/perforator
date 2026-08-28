package binary

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/fs"
	blobmodels "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

func bg() context.Context { return context.Background() }

type fakeMetaStorage struct {
	metas map[string]*binarymeta.BinaryMeta
	begin func(context.Context, string, time.Time, ...binarymeta.Option) (binarymeta.UploadClaim, error)
}

func (f *fakeMetaStorage) BeginUpload(ctx context.Context, buildID string, timestamp time.Time, opts ...binarymeta.Option) (binarymeta.UploadClaim, error) {
	return f.begin(ctx, buildID, timestamp, opts...)
}

func (f *fakeMetaStorage) GetBinaries(_ context.Context, buildIDs []string) ([]*binarymeta.BinaryMeta, error) {
	res := make([]*binarymeta.BinaryMeta, 0, len(buildIDs))
	for _, id := range buildIDs {
		if meta, ok := f.metas[id]; ok {
			res = append(res, meta)
		}
	}
	return res, nil
}

func (f *fakeMetaStorage) CollectExpiredBinaries(context.Context, time.Duration, *util.Pagination) ([]*binarymeta.BinaryMeta, error) {
	return nil, nil
}

func (f *fakeMetaStorage) RemoveBinaries(context.Context, []string) error { return nil }

type fakeUploadClaim struct {
	commit func(context.Context, *storage.BlobInfo) error
	abort  func(context.Context) error
}

func (f *fakeUploadClaim) Commit(ctx context.Context, blobInfo *storage.BlobInfo) error {
	return f.commit(ctx, blobInfo)
}

func (f *fakeUploadClaim) Ping(context.Context) error { return nil }

func (f *fakeUploadClaim) Abort(ctx context.Context) error {
	return f.abort(ctx)
}

// memWriterAt collects concurrent WriteAts into a growing buffer.
type memWriterAt struct {
	mu sync.Mutex
	b  []byte
}

func (m *memWriterAt) WriteAt(p []byte, off int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if end := int(off) + len(p); end > len(m.b) {
		m.b = append(m.b, make([]byte, end-len(m.b))...)
	}
	copy(m.b[off:], p)
	return len(p), nil
}

func newTestStorage(t *testing.T, metas map[string]*binarymeta.BinaryMeta) *BinaryStorage {
	l := xlog.ForTest(t)
	fsStorage, err := fs.NewFSStorage(fs.FSStorageConfig{Root: t.TempDir()}, l)
	require.NoError(t, err)

	return NewStorage(&fakeMetaStorage{metas: metas}, fsStorage, l, metricsmock.NewRegistry(nil))
}

func putBlob(t *testing.T, s *BinaryStorage, key string, content []byte) {
	require.NoError(t, s.blobStorage.Put(bg(), key, bytes.NewReader(content)))
}

func compress(t *testing.T, payload []byte) []byte {
	enc, err := zstd.NewWriter(nil)
	require.NoError(t, err)
	compressed := enc.EncodeAll(payload, nil)
	require.NoError(t, enc.Close())
	return compressed
}

func testMeta(buildID, blobID string, method compressionpb.CompressionMethod, uncompressedSize uint64) *binarymeta.BinaryMeta {
	return &binarymeta.BinaryMeta{
		BuildID:          buildID,
		BlobInfo:         &storage.BlobInfo{ID: blobID},
		Compression:      method,
		UncompressedSize: uncompressedSize,
	}
}

func newStoreTestStorage(t *testing.T, claim binarymeta.UploadClaim, blobStorage blobmodels.Storage) *BinaryStorage {
	metaStorage := &fakeMetaStorage{begin: func(context.Context, string, time.Time, ...binarymeta.Option) (binarymeta.UploadClaim, error) {
		return claim, nil
	}}
	return NewStorage(metaStorage, blobStorage, xlog.ForTest(t), metricsmock.NewRegistry(nil))
}

func TestStoreBinary_PutAndAbortFailureReturnsPutError(t *testing.T) {
	putErr := errors.New("put failed")
	abortErr := errors.New("abort failed")
	events := []string{}
	claim := &fakeUploadClaim{abort: func(ctx context.Context) error {
		require.NoError(t, ctx.Err())
		events = append(events, "abort")
		return abortErr
	}}
	blobStorage := &fakeBlobStorage{put: func(context.Context, string, io.Reader) error {
		events = append(events, "put")
		return putErr
	}}

	_, err := newStoreTestStorage(t, claim, blobStorage).StoreBinary(bg(), "a", time.Now(), bytes.NewReader([]byte("data")))
	require.ErrorIs(t, err, putErr)
	require.NotErrorIs(t, err, abortErr)
	require.Equal(t, []string{"put", "abort"}, events)
}

func TestStoreBinary_CancelledCommitCleanupOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payload := []byte("data")
	events := []string{}
	claim := &fakeUploadClaim{
		commit: func(ctx context.Context, blobInfo *storage.BlobInfo) error {
			events = append(events, "commit")
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			require.Equal(t, &storage.BlobInfo{ID: "a", Size: uint64(len(payload))}, blobInfo)
			return ctx.Err()
		},
		abort: func(ctx context.Context) error {
			events = append(events, "abort")
			require.NoError(t, ctx.Err())
			return nil
		},
	}
	blobStorage := &fakeBlobStorage{
		put: func(_ context.Context, _ string, src io.Reader) error {
			events = append(events, "put")
			data, err := io.ReadAll(src)
			require.NoError(t, err)
			require.Equal(t, payload, data)
			cancel()
			return nil
		},
		delete: func(ctx context.Context, _ string) error {
			events = append(events, "delete")
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			return ctx.Err()
		},
	}

	_, err := newStoreTestStorage(t, claim, blobStorage).StoreBinary(ctx, "a", time.Now(), bytes.NewReader(payload))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"put", "commit", "delete", "abort"}, events)
}

func TestLoadBinary_Uncompressed(t *testing.T) {
	payload := []byte("uncompressed payload")
	meta := testMeta("a", "blob-a", compressionpb.CompressionMethod_None, 0)
	meta.BlobInfo.Size = uint64(len(payload))
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{"a": meta})
	putBlob(t, s, "blob-a", payload)

	var w memWriterAt
	m, err := s.LoadBinary(bg(), "a", &w)
	require.NoError(t, err)
	require.Equal(t, "a", m.BuildID)
	require.Equal(t, payload, w.b)
}

func TestLoadBinary_UncompressedSizeMismatch(t *testing.T) {
	payload := []byte("uncompressed payload")
	meta := testMeta("a", "blob-a", compressionpb.CompressionMethod_None, 0)
	meta.BlobInfo.Size = uint64(len(payload)) + 5
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{"a": meta})
	putBlob(t, s, "blob-a", payload)

	var w memWriterAt
	_, err := s.LoadBinary(bg(), "a", &w)
	require.ErrorContains(t, err, "short of its declared size")
}

func TestLoadBinary_ZstdRoundtrip(t *testing.T) {
	payload := bytes.Repeat([]byte("zstd payload "), 1000)
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{
		"z": testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, uint64(len(payload))),
	})
	putBlob(t, s, "blob-z", compress(t, payload))

	var w memWriterAt
	_, err := s.LoadBinary(bg(), "z", &w)
	require.NoError(t, err)
	require.Equal(t, payload, w.b)
}

func TestLoadBinary_NotFound(t *testing.T) {
	s := newTestStorage(t, nil)
	var w memWriterAt
	_, err := s.LoadBinary(bg(), "missing", &w)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestLoadBlob_UnknownCompression(t *testing.T) {
	s := newTestStorage(t, nil)
	var w memWriterAt
	err := s.loadBlob(bg(), testMeta("u", "blob-u", compressionpb.CompressionMethod_Unknown, 1), &w)
	require.ErrorContains(t, err, "compression is unknown")
}

func TestLoadBlob_ImplausibleUncompressedSize(t *testing.T) {
	for _, size := range []uint64{0, math.MaxUint64} {
		// The size is rejected before the blob is fetched, so no blob is needed.
		s := newTestStorage(t, nil)

		err := s.loadBlob(bg(), testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, size), &memWriterAt{})
		require.ErrorContains(t, err, "implausible uncompressed_size")
	}
}

func TestLoadBlob_DecompressedEndsShortOfDeclaredSize(t *testing.T) {
	payload := []byte("short payload")
	declared := uint64(len(payload)) + 7
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{
		"sh": testMeta("sh", "blob-short", compressionpb.CompressionMethod_Zstd, declared),
	})
	putBlob(t, s, "blob-short", compress(t, payload))

	err := s.loadBlob(bg(), testMeta("sh", "blob-short", compressionpb.CompressionMethod_Zstd, declared), &memWriterAt{})
	require.ErrorContains(t, err, "short of its declared size")
}

func TestLoadBlob_DecompressedExceedsDeclaredSize(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 4096)
	declared := uint64(len(payload)) - 1
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{
		"z": testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, declared),
	})
	putBlob(t, s, "blob-z", compress(t, payload))

	var w memWriterAt
	_, err := s.LoadBinary(bg(), "z", &w)
	require.ErrorContains(t, err, "continues past its declared size")
	require.LessOrEqual(t, uint64(len(w.b)), declared) // nothing past the declared size reached the writer
}

func TestLoadBlob_UnsupportedCompression(t *testing.T) {
	s := newTestStorage(t, nil)
	var w memWriterAt
	err := s.loadBlob(bg(), testMeta("g", "blob-g", compressionpb.CompressionMethod(999), 1), &w)
	require.ErrorContains(t, err, "unsupported compression method")
}

func TestLoadBlob_CorruptZstd(t *testing.T) {
	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{
		"z": testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, 100),
	})
	putBlob(t, s, "blob-z", []byte("this is not a zstd stream"))

	var w memWriterAt
	_, err := s.LoadBinary(bg(), "z", &w)
	require.Error(t, err)
}

// TestLoadBlob_CorruptZstdTail pins the probe read: a frame whose trailer is
// damaged decodes cleanly for every declared byte, so only advancing past the
// last one surfaces the corruption.
func TestLoadBlob_CorruptZstdTail(t *testing.T) {
	payload := bytes.Repeat([]byte("y"), 4096)
	blob := compress(t, payload)
	blob[len(blob)-1] ^= 0xff

	s := newTestStorage(t, map[string]*binarymeta.BinaryMeta{
		"z": testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, uint64(len(payload))),
	})
	putBlob(t, s, "blob-z", blob)

	var w memWriterAt
	_, err := s.LoadBinary(bg(), "z", &w)
	// The decoder may report the damaged frame either on the final data read
	// or at the end probe; either way it must not read as a length mismatch,
	// and it must not be accepted.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "short of its declared size")
}

func TestLoadBlob_BlobLoadErrorPropagates(t *testing.T) {
	boom := errors.New("blob load failed")
	s := NewStorage(
		&fakeMetaStorage{},
		&fakeBlobStorage{get: func(context.Context, string) (io.ReadCloser, error) { return nil, boom }},
		xlog.ForTest(t),
		metricsmock.NewRegistry(nil),
	)

	err := s.loadBlob(bg(), testMeta("z", "blob-z", compressionpb.CompressionMethod_Zstd, 100), &memWriterAt{})
	require.ErrorIs(t, err, boom)
}

// fakeBlobStorage overrides Get; every other Storage method panics via the nil
// embedded interface.
type fakeBlobStorage struct {
	blobmodels.Storage
	put    func(ctx context.Context, key string, src io.Reader) error
	get    func(ctx context.Context, key string) (io.ReadCloser, error)
	delete func(ctx context.Context, key string) error
}

func (f *fakeBlobStorage) Put(ctx context.Context, key string, src io.Reader) error {
	return f.put(ctx, key, src)
}

func (f *fakeBlobStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return f.get(ctx, key)
}

func (f *fakeBlobStorage) Delete(ctx context.Context, key string) error {
	return f.delete(ctx, key)
}
