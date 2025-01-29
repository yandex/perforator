package downloader

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/metrics"
	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	mock_binary "github.com/yandex/perforator/perforator/pkg/storage/binary/mock"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func createLogger() (xlog.Logger, error) {
	lconf := zap.KVConfig(log.DebugLevel)
	lconf.OutputPaths = []string{"stderr"}
	return xlog.TryNew(zap.New(lconf))
}

func newTestObjects(
	t *testing.T,
	cacheConfig *asyncfilecache.Config,
	config *Config,
) (
	context.Context,
	context.CancelFunc,
	xlog.Logger,
	metrics.Registry,
	*mock_binary.MockStorage,
	*BinaryDownloader,
) {
	l, err := createLogger()
	require.NoError(t, err)
	reg := metricsmock.NewRegistry(nil)

	fileCache, err := asyncfilecache.NewFileCache(
		cacheConfig,
		l,
		reg,
	)
	require.NoError(t, err)
	fileCache.EvictReleased()

	ctrl := gomock.NewController(t)
	storage := mock_binary.NewMockStorage(ctrl)

	downloader, err := NewDownloader(l, reg, fileCache, *config)
	require.NoError(t, err)

	binaryDownloader, err := NewBinaryDownloader(downloader, storage)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func(t *testing.T) {
		err := downloader.RunBackgroundDownloader(ctx)
		if !errors.Is(err, context.Canceled) {
			require.NoError(t, err)
		}
	}(t)
	return ctx, cancel, l, reg, storage, binaryDownloader
}

func TestDownloader_Simple(t *testing.T) {
	ctx, cancel, _, _, mockStorage, downloader := newTestObjects(
		t,
		&asyncfilecache.Config{
			MaxSize:  "100G",
			MaxItems: 100000,
			RootPath: "./binaries",
		},
		&Config{
			MaxSimultaneousDownloads: 1,
			MaxQueueSize:             100000,
		},
	)
	defer cancel()

	mockStorage.EXPECT().GetBinaries(gomock.Any(), []string{"a"}).Return(
		[]*binarymeta.BinaryMeta{
			{
				BuildID: "a",
				BlobInfo: &storage.BlobInfo{
					Size: 1,
				},
			},
		},
		nil,
	).AnyTimes()
	var writer *asyncfilecache.WriterAt
	mockStorage.EXPECT().
		LoadBinary(gomock.Any(), "a", gomock.AssignableToTypeOf(writer)).
		DoAndReturn(func(_ any, _ string, writer *asyncfilecache.WriterAt) (*binarymeta.BinaryMeta, error) {
			time.Sleep(100 * time.Millisecond)
			_, err := writer.WriteAt([]byte{'a'}, 0)
			if err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{
				BuildID: "a",
				BlobInfo: &storage.BlobInfo{
					Size: 1,
				},
			}, nil
		})

	acquiredBinary, err := downloader.Acquire(ctx, "a")
	require.NoError(t, err)

	fileRef := acquiredBinary.(*asyncfilecache.AcquiredFileReference)
	state := fileRef.State()
	require.True(t, state == asyncfilecache.Absent || state == asyncfilecache.Opened)

	err = acquiredBinary.WaitStored(ctx)
	require.NoError(t, err)

	suffix := fmt.Sprintf("%sa", BinaryFilePrefix)
	require.True(
		t,
		strings.HasSuffix(acquiredBinary.Path(), suffix),
		"path %s must end with %s",
		acquiredBinary.Path(),
		suffix,
	)

	err = acquiredBinary.WaitStored(ctx)
	require.NoError(t, err)

	acquiredBinary.Close()
}

func TestDownloader_SameBinarySimple(t *testing.T) {
	ctx, cancel, _, _, mockStorage, downloader := newTestObjects(
		t,
		&asyncfilecache.Config{
			MaxSize:  "100G",
			MaxItems: 100000,
			RootPath: "./binaries",
		},
		&Config{
			MaxSimultaneousDownloads: 1,
			MaxQueueSize:             100000,
		},
	)
	defer cancel()

	buildID := "a"

	mockStorage.EXPECT().GetBinaries(gomock.Any(), []string{buildID}).Return(
		[]*binarymeta.BinaryMeta{
			{
				BlobInfo: &storage.BlobInfo{
					Size: 1,
				},
			},
		},
		nil,
	).AnyTimes()
	var writer *asyncfilecache.WriterAt
	mockStorage.EXPECT().
		LoadBinary(gomock.Any(), "a", gomock.AssignableToTypeOf(writer)).
		DoAndReturn(func(_ any, _ string, writer *asyncfilecache.WriterAt) (*binarymeta.BinaryMeta, error) {
			time.Sleep(100 * time.Millisecond)
			_, err := writer.WriteAt([]byte{'a'}, 0)
			if err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{
				BuildID: buildID,
				BlobInfo: &storage.BlobInfo{
					Size: 1,
				},
			}, nil
		})

	acquiredBinary1, err := downloader.Acquire(ctx, buildID)
	require.NoError(t, err)

	err = acquiredBinary1.WaitStored(ctx)
	require.NoError(t, err)
	suffix := fmt.Sprintf("%s%s", BinaryFilePrefix, buildID)
	require.True(
		t,
		strings.HasSuffix(acquiredBinary1.Path(), suffix),
		"path %s must end with %s",
		acquiredBinary1.Path(),
		suffix,
	)

	acquiredBinary2, err := downloader.Acquire(ctx, buildID)
	require.NoError(t, err)

	err = acquiredBinary2.WaitStored(ctx)
	require.NoError(t, err)
	require.True(
		t,
		strings.HasSuffix(acquiredBinary2.Path(), suffix),
		"path %s must end with %s",
		acquiredBinary2.Path(),
		suffix,
	)

	acquiredBinary1.Close()
	acquiredBinary2.Close()
}

func TestCachedDownloader_ErrorHandling(t *testing.T) {
	ctx, cancel, _, _, mockStorage, downloader := newTestObjects(
		t,
		&asyncfilecache.Config{
			MaxSize:  "100G",
			MaxItems: 100000,
			RootPath: "./binaries",
		},
		&Config{
			MaxSimultaneousDownloads: 1,
			MaxQueueSize:             100000,
		},
	)
	defer cancel()

	buildID := "a"
	mockStorage.EXPECT().GetBinaries(gomock.Any(), []string{buildID}).Return(
		[]*binarymeta.BinaryMeta{
			{
				BlobInfo: &storage.BlobInfo{
					Size: 2,
				},
			},
		},
		nil,
	).AnyTimes()
	var writer *asyncfilecache.WriterAt
	mockStorage.EXPECT().
		LoadBinary(gomock.Any(), "a", gomock.AssignableToTypeOf(writer)).
		DoAndReturn(func(_ any, _ string, writer *asyncfilecache.WriterAt) (*binarymeta.BinaryMeta, error) {
			_, err := writer.WriteAt([]byte{'a'}, 0)
			if err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{
				BuildID: buildID,
				BlobInfo: &storage.BlobInfo{
					Size: 2,
				},
			}, nil
		})

	acquiredBinary, err := downloader.Acquire(ctx, "a")
	require.NoError(t, err)

	err = acquiredBinary.WaitStored(ctx)
	require.Error(t, err)

	acquiredBinary.Close()
}

type Binary struct {
	Meta *binarymeta.BinaryMeta
	Body []byte
}

type Request struct {
	Binaries []*Binary
}

func buildRequests(binaries uint32, requestsCount uint32) ([]*Request, map[string]*Binary) {
	allBinaries := make([]*Binary, 0, binaries)
	resultAllBinaries := map[string]*Binary{}

	for i := uint32(0); i < binaries; i++ {
		bin := &Binary{
			Meta: &binarymeta.BinaryMeta{
				BuildID: fmt.Sprintf("%d", i),
				BlobInfo: &storage.BlobInfo{
					Size: uint64(i),
				},
			},
			Body: make([]byte, i),
		}
		allBinaries = append(allBinaries, bin)
		resultAllBinaries[fmt.Sprintf("%d", i)] = bin
	}

	requests := make([]*Request, 0, requestsCount)
	for i := uint32(0); i < requestsCount; i++ {
		requestBinaries := allBinaries[0:i]
		if i%2 == 1 {
			requestBinaries = allBinaries[i:]
		}

		requests = append(requests, &Request{
			Binaries: requestBinaries,
		})
	}

	return requests, resultAllBinaries
}

func TestCachedDownloader_Concurrent(t *testing.T) {
	ctx, cancel, l, _, mockStorage, downloader := newTestObjects(
		t,
		&asyncfilecache.Config{
			MaxSize:  "100B",
			MaxItems: 1000000,
			RootPath: "./binaries",
		},
		&Config{
			MaxQueueSize:             100000,
			MaxSimultaneousDownloads: 2,
		},
	)
	defer cancel()

	requestsCount := uint32(10)
	requests, allBinaries := buildRequests(20, requestsCount)

	var writer *asyncfilecache.WriterAt
	for _, bin := range allBinaries {
		bin := bin
		mockStorage.EXPECT().GetBinaries(gomock.Any(), []string{bin.Meta.BuildID}).Return(
			[]*binarymeta.BinaryMeta{
				{
					BlobInfo: &storage.BlobInfo{
						Size: bin.Meta.BlobInfo.Size,
					},
					BuildID: bin.Meta.BuildID,
				},
			},
			nil,
		).AnyTimes()
		mockStorage.EXPECT().
			LoadBinary(gomock.Any(), bin.Meta.BuildID, gomock.AssignableToTypeOf(writer)).
			DoAndReturn(func(_ any, _ string, writer *asyncfilecache.WriterAt) (*binarymeta.BinaryMeta, error) {
				time.Sleep(3 * time.Millisecond)
				_, err := writer.WriteAt(make([]byte, bin.Meta.BlobInfo.Size), 0)
				if err != nil {
					return nil, err
				}
				return &binarymeta.BinaryMeta{
					BuildID: bin.Meta.BuildID,
					BlobInfo: &storage.BlobInfo{
						Size: bin.Meta.BlobInfo.Size,
					},
				}, nil
			}).AnyTimes()
	}

	g, _ := errgroup.WithContext(ctx)

	successfulRequests := atomic.Uint32{}

	for i, req := range requests {
		reqCopy := req
		jitter := time.Duration(i) * time.Millisecond
		g.Go(func() error {
			acquiredBinaries := map[string]binaryprovider.FileHandle{}

			failedToAcquire := false

			for _, bin := range reqCopy.Binaries {
				binary, err := downloader.Acquire(ctx, bin.Meta.BuildID)
				if err != nil {
					failedToAcquire = true
					break
				}

				acquiredBinaries[bin.Meta.BuildID] = binary
			}

			for buildID, acquiredBinary := range acquiredBinaries {
				err := acquiredBinary.WaitStored(ctx)
				require.NoError(t, err)
				require.True(t, strings.HasSuffix(acquiredBinary.Path(), getBinaryFileEntryName(buildID, BinaryFilePrefix)))
			}

			if !failedToAcquire {
				successfulRequests.Add(1)
				time.Sleep(20*time.Millisecond + jitter)
			}

			for _, acquiredBinary := range acquiredBinaries {
				acquiredBinary.Close()
			}

			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	require.Greater(t, successfulRequests.Load(), uint32(0))

	l.Logger().Infof("%d successfull requests out of %d", successfulRequests.Load(), requestsCount)
}
