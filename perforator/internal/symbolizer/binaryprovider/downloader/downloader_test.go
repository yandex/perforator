package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/filecache"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	blobfs "github.com/yandex/perforator/perforator/pkg/storage/blob/fs"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

// fakeStorage serves zero bytes for every known buildID; onDownload, when
// set, replaces the default download body.
type fakeStorage struct {
	sizes      map[string]uint64
	onDownload func(ctx context.Context, buildID string, w io.WriterAt) error
	downloads  atomic.Int32
}

func (f *fakeStorage) size(ctx context.Context, buildID string) (uint64, error) {
	size, ok := f.sizes[buildID]
	if !ok {
		return 0, fmt.Errorf("no binary %s", buildID)
	}
	return size, nil
}

func (f *fakeStorage) download(ctx context.Context, buildID string, w io.WriterAt) error {
	f.downloads.Add(1)
	if f.onDownload != nil {
		return f.onDownload(ctx, buildID, w)
	}
	_, err := w.WriteAt(make([]byte, f.sizes[buildID]), 0)
	return err
}

func newTestProvider(t *testing.T, maxSize string, config *Config, fake *fakeStorage) (context.Context, binaryprovider.BinaryProvider) {
	l := xlog.ForTest(t)
	reg := metricsmock.NewRegistry(nil)

	fileCache, err := filecache.NewFileCache(
		&filecache.Config{MaxSize: maxSize, RootPath: t.TempDir()},
		reg,
	)
	require.NoError(t, err)

	d := NewDownloader(l, reg, fileCache, *config)
	provider := &artifactDownloader{downloader: d, storage: fake, prefix: binaryFilePrefix}

	return context.Background(), provider
}

func TestDownloader_Simple(t *testing.T) {
	fake := &fakeStorage{sizes: map[string]uint64{"a": 1}}
	ctx, provider := newTestProvider(t, "100G", &Config{MaxSimultaneousDownloads: 1}, fake)

	handle, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(handle.Path(), binaryFilePrefix+"a"))

	fi, err := os.Stat(handle.Path())
	require.NoError(t, err)
	require.Equal(t, int64(1), fi.Size())

	handle.Close()
}

func TestDownloader_SameBinarySingleDownload(t *testing.T) {
	fake := &fakeStorage{sizes: map[string]uint64{"a": 1}}
	ctx, provider := newTestProvider(t, "100G", &Config{MaxSimultaneousDownloads: 1}, fake)

	h1, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)
	h2, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)

	require.Equal(t, h1.Path(), h2.Path())
	require.Equal(t, int32(1), fake.downloads.Load()) // exactly one download despite two acquires

	h1.Close()
	h2.Close()
}

func TestDownloader_DownloadErrorNotCached(t *testing.T) {
	downloadErr := errors.New("download failed")
	fake := &fakeStorage{
		sizes: map[string]uint64{"a": 2},
		onDownload: func(context.Context, string, io.WriterAt) error {
			return downloadErr
		},
	}
	ctx, provider := newTestProvider(t, "100G", &Config{MaxSimultaneousDownloads: 1}, fake)

	_, err := provider.Acquire(ctx, "a")
	require.ErrorIs(t, err, downloadErr)

	// Not cached: a second acquire retries the download.
	_, err = provider.Acquire(ctx, "a")
	require.ErrorIs(t, err, downloadErr)
	require.Equal(t, int32(2), fake.downloads.Load())
}

func TestDownloader_FirstCallerCancelableWaiterGetsBinary(t *testing.T) {
	started := make(chan struct{})
	proceed := make(chan struct{})
	fake := &fakeStorage{
		sizes: map[string]uint64{"a": 1},
		onDownload: func(_ context.Context, _ string, w io.WriterAt) error {
			close(started)
			<-proceed
			_, err := w.WriteAt([]byte{0}, 0)
			return err
		},
	}
	ctx, provider := newTestProvider(t, "100G", &Config{MaxSimultaneousDownloads: 1}, fake)

	firstCtx, cancel := context.WithCancel(ctx)
	firstDone := make(chan error, 1)
	go func() { _, err := provider.Acquire(firstCtx, "a"); firstDone <- err }()
	<-started

	waiterDone := make(chan error, 1)
	var handle binaryprovider.FileHandle
	go func() {
		var err error
		handle, err = provider.Acquire(ctx, "a") // keeps the download alive
		waiterDone <- err
	}()
	time.Sleep(20 * time.Millisecond)

	// The first caller gives up; the download continues for the waiter.
	cancel()
	require.ErrorIs(t, <-firstDone, context.Canceled)

	close(proceed)
	require.NoError(t, <-waiterDone)
	require.Equal(t, int32(1), fake.downloads.Load())
	handle.Close()
}

func TestDownloader_WaiterCancelableWhileFillContinues(t *testing.T) {
	started := make(chan struct{})
	proceed := make(chan struct{})
	fake := &fakeStorage{
		sizes: map[string]uint64{"a": 1},
		onDownload: func(_ context.Context, _ string, w io.WriterAt) error {
			close(started)
			<-proceed
			_, err := w.WriteAt([]byte{0}, 0)
			return err
		},
	}
	ctx, provider := newTestProvider(t, "100G", &Config{MaxSimultaneousDownloads: 1}, fake)

	fillerDone := make(chan error, 1)
	go func() {
		h, err := provider.Acquire(ctx, "a")
		if err == nil {
			h.Close()
		}
		fillerDone <- err
	}()
	<-started

	waiterCtx, cancel := context.WithCancel(ctx)
	waiterDone := make(chan error, 1)
	go func() {
		_, err := provider.Acquire(waiterCtx, "a") // coalesces onto the in-flight fill
		waiterDone <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-waiterDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("canceled waiter did not return while the fill was in flight")
	}

	close(proceed)
	require.NoError(t, <-fillerDone)

	// The shared fill survived the waiter's cancellation: the next acquire hits.
	h, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, int32(1), fake.downloads.Load())
	h.Close()
}

func TestDownloader_Concurrent(t *testing.T) {
	const distinct = 20
	sizes := make(map[string]uint64, distinct)
	for i := 0; i < distinct; i++ {
		sizes[fmt.Sprintf("b%d", i)] = 10 // 10 bytes each; ~10 fit in 100B
	}
	fake := &fakeStorage{sizes: sizes}
	ctx, provider := newTestProvider(t, "100B", &Config{MaxSimultaneousDownloads: 2}, fake)

	var succeeded atomic.Uint32
	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < 100; i++ {
		buildID := fmt.Sprintf("b%d", i%distinct)
		g.Go(func() error {
			handle, err := provider.Acquire(ctx, buildID)
			if err != nil {
				return err
			}
			require.True(t, strings.HasSuffix(handle.Path(), binaryFilePrefix+buildID))
			time.Sleep(time.Millisecond)
			handle.Close()
			succeeded.Add(1)
			return nil
		})
	}

	require.NoError(t, g.Wait())
	require.Equal(t, uint32(100), succeeded.Load()) // capacity contention waits, never fails
}

func TestEffectiveBinarySize(t *testing.T) {
	t.Run("zstd uses uncompressed size", func(t *testing.T) {
		size := effectiveBinarySize(&binarymeta.BinaryMeta{
			BuildID:          "a",
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: 1024,
			BlobInfo:         &storage.BlobInfo{Size: 100},
		})
		require.Equal(t, uint64(1024), size)
	})

	t.Run("legacy none falls back to blob size", func(t *testing.T) {
		size := effectiveBinarySize(&binarymeta.BinaryMeta{
			BuildID:     "a",
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 500},
		})
		require.Equal(t, uint64(500), size)
	})
}

// fakeBinaryMeta and fakeGSYMMeta serve fixed metas; every other method panics
// through the nil embedded interface.
type fakeBinaryMeta struct {
	binarymeta.Storage
	metas map[string]*binarymeta.BinaryMeta
}

func (f *fakeBinaryMeta) GetBinaries(_ context.Context, buildIDs []string) ([]*binarymeta.BinaryMeta, error) {
	res := make([]*binarymeta.BinaryMeta, 0, len(buildIDs))
	for _, id := range buildIDs {
		if meta, ok := f.metas[id]; ok {
			res = append(res, meta)
		}
	}
	return res, nil
}

type fakeGSYMMeta struct {
	gsymmeta.Storage
	metas map[string]*gsymmeta.GSYMMeta
}

func (f *fakeGSYMMeta) GetGSYMs(_ context.Context, buildIDs []string) ([]*gsymmeta.GSYMMeta, error) {
	res := make([]*gsymmeta.GSYMMeta, 0, len(buildIDs))
	for _, id := range buildIDs {
		if meta, ok := f.metas[id]; ok {
			res = append(res, meta)
		}
	}
	return res, nil
}

func zstdCompress(t *testing.T, payload []byte) []byte {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf, zstd.WithEncoderConcurrency(1))
	require.NoError(t, err)
	_, err = enc.Write(payload)
	require.NoError(t, err)
	require.NoError(t, enc.Close())
	return buf.Bytes()
}

func newBlobStorageWith(t *testing.T, key string, content []byte) blob.Storage {
	blobStorage, err := blobfs.NewFSStorage(blobfs.FSStorageConfig{Root: t.TempDir()}, xlog.ForTest(t))
	require.NoError(t, err)

	require.NoError(t, blobStorage.Put(context.Background(), key, bytes.NewReader(content)))

	return blobStorage
}

func newTestDownloader(t *testing.T) *Downloader {
	reg := metricsmock.NewRegistry(nil)
	fileCache, err := filecache.NewFileCache(
		&filecache.Config{MaxSize: "16MiB", RootPath: t.TempDir()},
		reg,
	)
	require.NoError(t, err)
	return NewDownloader(xlog.ForTest(t), reg, fileCache, Config{})
}

// TestBinaryDownloader_ZstdChargeMatchesFile drives the production adapter to
// pin the cache accounting invariant: the size charged before the download
// must be the number of bytes the download then writes. Charging the
// compressed size would let the cache overshoot its budget on every
// compressed binary.
func TestBinaryDownloader_ZstdChargeMatchesFile(t *testing.T) {
	ctx := context.Background()
	payload := bytes.Repeat([]byte("binary payload "), 500)
	blobStorage := newBlobStorageWith(t, "build-1", zstdCompress(t, payload))

	binStorage := binarystorage.NewStorage(
		&fakeBinaryMeta{metas: map[string]*binarymeta.BinaryMeta{
			"build-1": {
				BuildID:          "build-1",
				BlobInfo:         &storage.BlobInfo{ID: "build-1"},
				Compression:      compressionpb.CompressionMethod_Zstd,
				UncompressedSize: uint64(len(payload)),
			},
		}},
		blobStorage,
		xlog.ForTest(t),
		metricsmock.NewRegistry(nil),
	)

	charged, err := (&binaryDownloadAdapter{storage: binStorage}).size(ctx, "build-1")
	require.NoError(t, err)
	require.Equal(t, uint64(len(payload)), charged)

	handle, err := NewBinaryDownloader(newTestDownloader(t), binStorage).Acquire(ctx, "build-1")
	require.NoError(t, err)
	defer handle.Close()

	onDisk, err := os.ReadFile(handle.Path())
	require.NoError(t, err)
	require.Equal(t, int(charged), len(onDisk), "cache file size must match the charged size")
	require.Equal(t, payload, onDisk)
}

// TestGSYMDownloader_DecompressesOnce pins that the GSYM adapter lands exactly
// the decompressed bytes on disk — neither the compressed blob nor a
// double-decompressed one.
func TestGSYMDownloader_DecompressesOnce(t *testing.T) {
	ctx := context.Background()
	payload := bytes.Repeat([]byte("gsym payload "), 500)
	compressed := zstdCompress(t, payload)
	blobStorage := newBlobStorageWith(t, "build-2", compressed)

	gsymStorage := gsymstorage.NewStorage(
		&fakeGSYMMeta{metas: map[string]*gsymmeta.GSYMMeta{
			"build-2": {
				BuildID:          "build-2",
				CompressedSize:   uint64(len(compressed)),
				UncompressedSize: uint64(len(payload)),
			},
		}},
		blobStorage,
		xlog.ForTest(t),
	)

	charged, err := (&gsymDownloadAdapter{storage: gsymStorage}).size(ctx, "build-2")
	require.NoError(t, err)
	require.Equal(t, uint64(len(payload)), charged)

	handle, err := NewGSYMDownloader(newTestDownloader(t), gsymStorage).Acquire(ctx, "build-2")
	require.NoError(t, err)
	defer handle.Close()

	onDisk, err := os.ReadFile(handle.Path())
	require.NoError(t, err)
	require.Equal(t, int(charged), len(onDisk), "cache file size must match the charged size")
	require.Equal(t, payload, onDisk)
}
