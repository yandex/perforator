package downloader

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	lru "github.com/hashicorp/golang-lru"
	"github.com/klauspost/compress/zstd"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/semaphore"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	binaryFilePrefix = "binary_"
	gsymFilePrefix   = "gsym_"

	downloadTimeout      = 10 * time.Minute
	binarySizesCacheSize = 10000

	defaultDownloadsQueueSize = 10000
)

type downloadableStorage interface {
	size(ctx context.Context, buildID string) (uint64, error)
	download(ctx context.Context, buildID string, writer io.WriterAt) error
}

type downloaderMetrics struct {
	scheduledBinaries metrics.Counter
	downloadsInFly    metrics.IntGauge

	downloadTimer metrics.Timer
}

type downloadItem struct {
	acquiredFile *asyncfilecache.AcquiredFileReference
	info         *binaryprovider.BinaryInfo
	storage      downloadableStorage
}

func (b *downloadItem) load(ctx context.Context, writer io.WriterAt, done func() error) error {
	err := b.storage.download(ctx, b.info.BuildID, writer)
	if err != nil {
		return err
	}

	return done()
}

type Downloader struct {
	l xlog.Logger
	r metrics.Registry

	fileCache *asyncfilecache.FileCache

	binariesQueue chan *downloadItem

	semaphore *semaphore.Weighted

	metrics *downloaderMetrics
}

type Config struct {
	MaxQueueSize             uint64
	MaxSimultaneousDownloads uint64
}

func NewDownloader(
	l xlog.Logger,
	r metrics.Registry,
	cache *asyncfilecache.FileCache,
	conf Config,
) (*Downloader, error) {
	maxQueueSize := conf.MaxQueueSize
	if maxQueueSize == 0 {
		maxQueueSize = defaultDownloadsQueueSize
	}

	downloader := &Downloader{
		l:             l,
		r:             r,
		fileCache:     cache,
		semaphore:     semaphore.NewWeighted(int64(conf.MaxSimultaneousDownloads)),
		binariesQueue: make(chan *downloadItem, maxQueueSize),
	}
	downloader.registerMetrics()

	return downloader, nil
}

func (d *Downloader) registerMetrics() {
	d.r.FuncIntGauge(
		"binaries.downloads_scheduled.gauge",
		func() int64 {
			return int64(len(d.binariesQueue))
		},
	)

	d.metrics = &downloaderMetrics{
		scheduledBinaries: d.r.Counter("binaries.downloads_scheduled.count"),
		downloadsInFly:    d.r.IntGauge("binaries.downloads_in_fly.gauge"),
		downloadTimer:     d.r.Timer("binaries.downloads.timer"),
	}
}

func (d *Downloader) performDownload(ctx context.Context, item *downloadItem) error {
	writer, done, err := item.acquiredFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = done()
		}
	}()

	err = item.load(ctx, writer, done)
	return err
}

func (d *Downloader) runDownload(ctx context.Context, item *downloadItem) {
	defer d.semaphore.Release(1)
	d.metrics.downloadsInFly.Add(1)
	defer d.metrics.downloadsInFly.Add(-1)

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	l := d.l.With(
		log.String("build_id", item.info.BuildID),
		log.UInt64("size", item.info.Size),
		log.String("function", "runDownload"),
	)
	l.Info(ctx, "Start binary download")
	ts := time.Now()

	err := d.performDownload(ctx, item)
	if err != nil {
		l.Error(ctx, "Failed to download binary")
		return
	}

	duration := time.Since(ts)
	l.Info(ctx, "Downloaded binary", log.Duration("duration", duration))
	d.metrics.downloadTimer.RecordDuration(duration)
}

func (d *Downloader) RunBackgroundDownloader(ctx context.Context) error {
	for {
		var req *downloadItem
		select {
		case req = <-d.binariesQueue:
		case <-ctx.Done():
			return ctx.Err()
		}

		_ = d.semaphore.Acquire(ctx, 1)
		go d.runDownload(ctx, req)
	}
}

func (d *Downloader) scheduleForDownload(ctx context.Context, item *downloadItem) error {
	d.metrics.scheduledBinaries.Inc()

	select {
	case d.binariesQueue <- item:
	case <-ctx.Done():
		return ctx.Err()
	}

	d.l.Info(ctx, "Scheduled binary for download", log.String("build_id", item.info.BuildID))
	return nil
}

func getSize(
	ctx context.Context,
	sizeCache *lru.Cache,
	storage downloadableStorage,
	buildID string,
) (uint64, error) {
	sizeFromCache, ok := sizeCache.Get(buildID)
	if ok {
		return sizeFromCache.(uint64), nil
	}

	var err error
	ctx, span := otel.Tracer("Symbolizer").Start(
		ctx, "downloader.getSize",
		trace.WithAttributes(attribute.String("buildID", buildID)),
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	sz, err := storage.size(ctx, buildID)
	if err != nil {
		return 0, err
	}

	sizeCache.Add(buildID, sz)
	return sz, nil
}

func (d *Downloader) acquire(
	ctx context.Context,
	sizeCache *lru.Cache,
	storage downloadableStorage,
	buildID string,
	filePrefix string,
) (binaryprovider.FileHandle, error) {
	sz, err := getSize(ctx, sizeCache, storage, buildID)
	if err != nil {
		return nil, err
	}

	entry := getBinaryFileEntryName(buildID, filePrefix)

	acquiredRef, inserted, err := d.fileCache.Acquire(entry, sz)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire filecache item for %s: %w", entry, err)
	}

	binaryInfo := &binaryprovider.BinaryInfo{
		BuildID: buildID,
		Size:    sz,
	}

	if inserted {
		err = d.scheduleForDownload(ctx, &downloadItem{
			acquiredFile: acquiredRef,
			info:         binaryInfo,
			storage:      storage,
		})
		if err != nil {
			return nil, err
		}
	}

	return acquiredRef, nil
}

func getBinaryFileEntryName(buildID string, prefix string) string {
	buildIDAsPath := strings.ReplaceAll(buildID, "/", "%")

	return prefix + buildIDAsPath
}

// binaryDownloadAdapter adapts binary.Storage to DownloadableStorage.
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
	if binaries[0].BlobInfo == nil {
		return 0, fmt.Errorf("there is no blob for binary %s", buildID)
	}
	return binaries[0].BlobInfo.Size, nil
}

func (a *binaryDownloadAdapter) download(ctx context.Context, buildID string, writer io.WriterAt) error {
	_, err := a.storage.LoadBinary(ctx, buildID, writer)
	return err
}

// gsymDownloadAdapter adapts gsym.Storage to DownloadableStorage.
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

	sizeCache *lru.Cache
}

func (d *artifactDownloader) Acquire(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	return d.downloader.acquire(ctx, d.sizeCache, d.storage, buildID, d.prefix)
}

func NewBinaryDownloader(downloader *Downloader, binaryStorage binarystorage.Storage) (binaryprovider.BinaryProvider, error) {
	sizeCache, err := lru.New(binarySizesCacheSize)
	if err != nil {
		return nil, err
	}

	return &artifactDownloader{
		downloader: downloader,
		storage:    &binaryDownloadAdapter{storage: binaryStorage},
		prefix:     binaryFilePrefix,
		sizeCache:  sizeCache,
	}, nil
}

func NewGSYMDownloader(downloader *Downloader, gsymStorage gsymstorage.Storage) (binaryprovider.BinaryProvider, error) {
	sizeCache, err := lru.New(binarySizesCacheSize)
	if err != nil {
		return nil, err
	}

	return &artifactDownloader{
		downloader: downloader,
		storage:    &gsymDownloadAdapter{storage: gsymStorage},
		prefix:     gsymFilePrefix,
		sizeCache:  sizeCache,
	}, nil
}

func CreateDownloaders(
	fileCacheConfig *asyncfilecache.Config,
	maxSimultaneousDownloads uint32,
	l xlog.Logger,
	reg metrics.Registry,
	binaryStorage binarystorage.Storage,
	gsymStorage gsymstorage.Storage,
) (*Downloader, binaryprovider.BinaryProvider, binaryprovider.BinaryProvider, error) {
	fileCache, err := asyncfilecache.NewFileCache(
		fileCacheConfig,
		l,
		reg,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	downloaderInstance, err := NewDownloader(
		l.WithName("Downloader"),
		reg,
		fileCache,
		Config{
			MaxSimultaneousDownloads: uint64(maxSimultaneousDownloads),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	binaryDownloader, err := NewBinaryDownloader(
		downloaderInstance,
		binaryStorage,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	gsymDownloader, err := NewGSYMDownloader(
		downloaderInstance,
		gsymStorage,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return downloaderInstance, binaryDownloader, gsymDownloader, nil
}
