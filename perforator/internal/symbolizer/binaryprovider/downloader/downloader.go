package downloader

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/semaphore"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/filecache"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

const (
	binaryFilePrefix = "binary_"
	gsymFilePrefix   = "gsym_"

	downloadTimeout = 10 * time.Minute
)

type downloadableStorage interface {
	size(ctx context.Context, buildID string) (uint64, error)
	download(ctx context.Context, buildID string, writer io.WriterAt) error
}

type downloaderMetrics struct {
	downloadsInFly metrics.IntGauge
	downloadTimer  metrics.Timer
}

// fileHandle is a ready, pinned cache file. It satisfies binaryprovider.FileHandle.
type fileHandle struct {
	ref *filecache.Ref
}

func (h *fileHandle) Path() string { return h.ref.Path() }
func (h *fileHandle) Close()       { h.ref.Release() }

type Downloader struct {
	l xlog.Logger
	r metrics.Registry

	fileCache *filecache.Cache
	semaphore *semaphore.Weighted

	metrics *downloaderMetrics
}

type Config struct {
	MaxSimultaneousDownloads uint64 // 0 means unlimited
}

func NewDownloader(
	l xlog.Logger,
	r metrics.Registry,
	cache *filecache.Cache,
	conf Config,
) *Downloader {
	maxDownloads := int64(conf.MaxSimultaneousDownloads)
	if maxDownloads == 0 {
		maxDownloads = math.MaxInt64
	}

	downloader := &Downloader{
		l:         l,
		r:         r,
		fileCache: cache,
		semaphore: semaphore.NewWeighted(maxDownloads),
	}
	downloader.registerMetrics()

	return downloader
}

func (d *Downloader) registerMetrics() {
	d.metrics = &downloaderMetrics{
		downloadsInFly: d.r.IntGauge("binaries.downloads_in_fly.gauge"),
		downloadTimer:  d.r.Timer("binaries.downloads.timer"),
	}
}

// runDownload performs the actual storage fetch for a single cache fill. The
// semaphore bounds concurrent downloads (and thus peak memory, since gsym blobs
// are decompressed in full); coalesced waiters never reach here.
func (d *Downloader) runDownload(
	ctx context.Context,
	storage downloadableStorage,
	buildID string,
	writer io.WriterAt,
) error {
	if err := d.semaphore.Acquire(ctx, 1); err != nil {
		return err
	}
	defer d.semaphore.Release(1)

	d.metrics.downloadsInFly.Add(1)
	defer d.metrics.downloadsInFly.Add(-1)

	d.l.Info(ctx, "Start binary download", log.String("build_id", buildID))
	ts := time.Now()
	if err := storage.download(ctx, buildID, writer); err != nil {
		d.l.Error(ctx, "Failed to download binary", log.String("build_id", buildID), log.Error(err))
		return err
	}
	duration := time.Since(ts)
	d.l.Info(ctx, "Downloaded binary", log.String("build_id", buildID), log.Duration("duration", duration))
	d.metrics.downloadTimer.RecordDuration(duration)
	return nil
}

func (d *Downloader) acquire(
	ctx context.Context,
	storage downloadableStorage,
	buildID string,
	filePrefix string,
) (binaryprovider.FileHandle, error) {
	key := getBinaryFileEntryName(buildID, filePrefix)

	ref, err := d.fileCache.GetOrFill(ctx, key, func(ctx context.Context, w filecache.Writer) error {
		// ctx here is cache-owned (survives the requester); the timeout bounds
		// the download itself.
		ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
		defer cancel()

		size, err := storage.size(ctx, buildID)
		if err != nil {
			return err
		}
		if err := w.Charge(ctx, int64(size)); err != nil {
			return err
		}
		return d.runDownload(ctx, storage, buildID, w)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to acquire filecache item for %s: %w", key, err)
	}

	return &fileHandle{ref: ref}, nil
}

func getBinaryFileEntryName(buildID string, prefix string) string {
	buildIDAsPath := strings.ReplaceAll(buildID, "/", "%")

	return prefix + buildIDAsPath
}

// binaryDownloadAdapter adapts binary.Storage to downloadableStorage.
type binaryDownloadAdapter struct {
	storage binarystorage.Storage
}

func (a *binaryDownloadAdapter) size(ctx context.Context, buildID string) (uint64, error) {
	binaries, err := a.storage.GetBinaries(ctx, []string{buildID})
	if err != nil {
		return 0, err
	}
	if len(binaries) == 0 {
		return 0, fmt.Errorf("no binary %s: %w", buildID, binarystorage.ErrNotFound)
	}
	binary := binaries[0]
	if binary.BlobInfo == nil {
		return 0, fmt.Errorf("there is no blob for binary %s", buildID)
	}

	return effectiveBinarySize(binary), nil
}

func effectiveBinarySize(meta *binarymeta.BinaryMeta) uint64 {
	if meta.Compression == compressionpb.CompressionMethod_None {
		return meta.BlobInfo.Size
	}

	return meta.UncompressedSize
}

func (a *binaryDownloadAdapter) download(ctx context.Context, buildID string, writer io.WriterAt) error {
	_, err := a.storage.LoadBinary(ctx, buildID, writer)
	return err
}

// gsymDownloadAdapter adapts gsym.Storage to downloadableStorage.
// Size returns the uncompressed size (the final on-disk size after decompression).
// Download fetches the compressed blob from storage and decompresses it with zstd before writing to writer.
type gsymDownloadAdapter struct {
	storage gsymstorage.Storage
}

func (a *gsymDownloadAdapter) size(ctx context.Context, buildID string) (uint64, error) {
	gsyms, err := a.storage.GetGSYMs(ctx, []string{buildID})
	if err != nil {
		return 0, err
	}
	if len(gsyms) == 0 {
		return 0, fmt.Errorf("no GSYM for binary %s", buildID)
	}
	return gsyms[0].UncompressedSize, nil
}

func (a *gsymDownloadAdapter) download(ctx context.Context, buildID string, writer io.WriterAt) error {
	var buf aws.WriteAtBuffer
	buf.GrowthCoeff = 1.5

	_, err := a.storage.LoadGSYM(ctx, buildID, &buf)
	if err != nil {
		return err
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return err
	}

	result, err := decoder.DecodeAll(buf.Bytes(), nil)
	if err != nil {
		return err
	}

	_, err = writer.WriteAt(result, 0)
	return err
}

type artifactDownloader struct {
	downloader *Downloader
	storage    downloadableStorage
	prefix     string
}

func (d *artifactDownloader) Acquire(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	return d.downloader.acquire(ctx, d.storage, buildID, d.prefix)
}

func NewBinaryDownloader(downloader *Downloader, binaryStorage binarystorage.Storage) binaryprovider.BinaryProvider {
	return &artifactDownloader{
		downloader: downloader,
		storage:    &binaryDownloadAdapter{storage: binaryStorage},
		prefix:     binaryFilePrefix,
	}
}

func NewGSYMDownloader(downloader *Downloader, gsymStorage gsymstorage.Storage) binaryprovider.BinaryProvider {
	return &artifactDownloader{
		downloader: downloader,
		storage:    &gsymDownloadAdapter{storage: gsymStorage},
		prefix:     gsymFilePrefix,
	}
}

func CreateDownloaders(
	fileCacheConfig *filecache.Config,
	maxSimultaneousDownloads uint32,
	l xlog.Logger,
	reg metrics.Registry,
	binaryStorage binarystorage.Storage,
	gsymStorage gsymstorage.Storage,
) (binaryprovider.BinaryProvider, binaryprovider.BinaryProvider, error) {
	fileCache, err := filecache.NewFileCache(fileCacheConfig, reg)
	if err != nil {
		return nil, nil, err
	}

	downloaderInstance := NewDownloader(
		l.WithName("Downloader"),
		reg,
		fileCache,
		Config{
			MaxSimultaneousDownloads: uint64(maxSimultaneousDownloads),
		},
	)

	return NewBinaryDownloader(downloaderInstance, binaryStorage), NewGSYMDownloader(downloaderInstance, gsymStorage), nil
}
