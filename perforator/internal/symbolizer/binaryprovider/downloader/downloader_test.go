package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/filecache"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	mock_binary "github.com/yandex/perforator/perforator/pkg/storage/binary/mock"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

func newTestObjects(t *testing.T, maxSize string, config *Config) (
	context.Context,
	*mock_binary.MockStorage,
	binaryprovider.BinaryProvider,
) {
	l := xlog.ForTest(t)
	reg := metricsmock.NewRegistry(nil)

	fileCache, err := filecache.NewFileCache(
		&filecache.Config{MaxSize: maxSize, RootPath: t.TempDir()},
		reg,
	)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	st := mock_binary.NewMockStorage(ctrl)

	downloaderInstance := NewDownloader(l, reg, fileCache, *config)
	provider := NewBinaryDownloader(downloaderInstance, st)

	return context.Background(), st, provider
}

// expectBinary stubs an uncompressed binary of blobSize bytes. loadTimes < 0
// means the download may happen any number of times (e.g. across evictions).
func expectBinary(st *mock_binary.MockStorage, buildID string, blobSize uint64, loadTimes int) {
	st.EXPECT().GetBinaries(gomock.Any(), []string{buildID}).Return(
		[]*binarymeta.BinaryMeta{{
			BuildID:     buildID,
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: blobSize},
		}},
		nil,
	).AnyTimes()

	call := st.EXPECT().
		LoadBinary(gomock.Any(), buildID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, w io.WriterAt) (*binarymeta.BinaryMeta, error) {
			if _, err := w.WriteAt(make([]byte, blobSize), 0); err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{BuildID: buildID, BlobInfo: &storage.BlobInfo{Size: blobSize}}, nil
		})
	if loadTimes < 0 {
		call.AnyTimes()
	} else {
		call.Times(loadTimes)
	}
}

func TestDownloader_Simple(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})
	expectBinary(st, "a", 1, 1)

	handle, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(handle.Path(), binaryFilePrefix+"a"))

	fi, err := os.Stat(handle.Path())
	require.NoError(t, err)
	require.Equal(t, int64(1), fi.Size())

	handle.Close()
}

func TestDownloader_SameBinarySingleDownload(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})
	expectBinary(st, "a", 1, 1) // exactly one download despite two acquires

	h1, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)
	h2, err := provider.Acquire(ctx, "a")
	require.NoError(t, err)

	require.Equal(t, h1.Path(), h2.Path())

	h1.Close()
	h2.Close()
}

func TestDownloader_DownloadErrorNotCached(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})
	st.EXPECT().GetBinaries(gomock.Any(), []string{"a"}).Return(
		[]*binarymeta.BinaryMeta{{
			BuildID:     "a",
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 2},
		}},
		nil,
	).AnyTimes()
	downloadErr := errors.New("download failed")
	st.EXPECT().LoadBinary(gomock.Any(), "a", gomock.Any()).
		Return(nil, downloadErr).Times(2)

	_, err := provider.Acquire(ctx, "a")
	require.ErrorIs(t, err, downloadErr)

	// Not cached: a second acquire retries the download.
	_, err = provider.Acquire(ctx, "a")
	require.ErrorIs(t, err, downloadErr)
}

func TestDownloader_FirstCallerCancelableWaiterGetsBinary(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})

	st.EXPECT().GetBinaries(gomock.Any(), []string{"a"}).Return(
		[]*binarymeta.BinaryMeta{{
			BuildID:     "a",
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 1},
		}},
		nil,
	).AnyTimes()
	started := make(chan struct{})
	proceed := make(chan struct{})
	st.EXPECT().LoadBinary(gomock.Any(), "a", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, w io.WriterAt) (*binarymeta.BinaryMeta, error) {
			close(started)
			<-proceed
			if _, err := w.WriteAt([]byte{0}, 0); err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{BuildID: "a"}, nil
		}).Times(1)

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
	handle.Close()
}

func TestDownloader_WaiterCancelableWhileFillContinues(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})

	st.EXPECT().GetBinaries(gomock.Any(), []string{"a"}).Return(
		[]*binarymeta.BinaryMeta{{
			BuildID:     "a",
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 1},
		}},
		nil,
	).AnyTimes()
	started := make(chan struct{})
	proceed := make(chan struct{})
	st.EXPECT().LoadBinary(gomock.Any(), "a", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, w io.WriterAt) (*binarymeta.BinaryMeta, error) {
			close(started)
			<-proceed
			if _, err := w.WriteAt([]byte{0}, 0); err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{BuildID: "a"}, nil
		}).Times(1)

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
	h.Close()
}

func TestDownloader_ZstdAcquireSize(t *testing.T) {
	const uncompressedSize = 1024 * 1024

	ctx, st, provider := newTestObjects(t, "100G", &Config{MaxSimultaneousDownloads: 1})
	st.EXPECT().GetBinaries(gomock.Any(), []string{"z"}).Return(
		[]*binarymeta.BinaryMeta{{
			BuildID:          "z",
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: uncompressedSize,
			BlobInfo:         &storage.BlobInfo{Size: 4096},
		}},
		nil,
	).AnyTimes()
	st.EXPECT().LoadBinary(gomock.Any(), "z", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, w io.WriterAt) (*binarymeta.BinaryMeta, error) {
			if _, err := w.WriteAt(make([]byte, uncompressedSize), 0); err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{BuildID: "z"}, nil
		}).Times(1)

	handle, err := provider.Acquire(ctx, "z")
	require.NoError(t, err)

	fi, err := os.Stat(handle.Path())
	require.NoError(t, err)
	require.Equal(t, int64(uncompressedSize), fi.Size()) // charged the uncompressed size

	handle.Close()
}

func TestDownloader_Concurrent(t *testing.T) {
	ctx, st, provider := newTestObjects(t, "100B", &Config{MaxSimultaneousDownloads: 2})

	const distinct = 20
	for i := 0; i < distinct; i++ {
		expectBinary(st, fmt.Sprintf("b%d", i), 10, -1) // 10 bytes each; ~10 fit in 100B
	}

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
