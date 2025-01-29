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
	storage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	BinaryFilePrefix = "binary_"
	GSYMFilePrefix   = "gsym_"

	DownloadTimeout      = 10 * time.Minute
	BinarySizesCacheSize = 10000

	DefaultDownloadsQueueSize = 10000
)

type binaryType int

const (
	binaryTypeDefault binaryType = iota
	binaryTypeGSYM
)

type downloaderMetrics struct {
	scheduledBinaries metrics.Counter
	downloadsInFly    metrics.IntGauge

	downloadTimer metrics.Timer
}

type binary struct {
	acquiredFile  *asyncfilecache.AcquiredFileReference
	info          *binaryprovider.BinaryInfo
	binaryType    binaryType
	binaryStorage storage.Storage
}

func (b *binary) load(ctx context.Context, writer io.WriterAt, done func() error) error {
	var err error

	switch b.binaryType {
	case binaryTypeDefault:
		_, err = b.binaryStorage.LoadBinary(ctx, b.info.BuildID, writer)
	case binaryTypeGSYM:
		gsymWriter := newGSYMWriter(writer, done)
		gsymDone := func() error {
			return gsymWriter.Done()
		}

		writer = gsymWriter
		done = gsymDone
		_, err = b.binaryStorage.LoadBinary(ctx, b.info.BuildID, writer)
	}
	if err != nil {
		return err
	}

	return done()
}

type Downloader struct {
	l xlog.Logger
	r metrics.Registry

	fileCache *asyncfilecache.FileCache

	binariesQueue chan *binary

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
		maxQueueSize = DefaultDownloadsQueueSize
	}

	downloader := &Downloader{
		l:             l,
		r:             r,
		fileCache:     cache,
		semaphore:     semaphore.NewWeighted(int64(conf.MaxSimultaneousDownloads)),
		binariesQueue: make(chan *binary, maxQueueSize),
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

type gsymWriter struct {
	buffer aws.WriteAtBuffer

	writer io.WriterAt
	done   func() error
}

func newGSYMWriter(writer io.WriterAt, done func() error) *gsymWriter {
	return &gsymWriter{
		buffer: aws.WriteAtBuffer{
			GrowthCoeff: 1.5,
		},
		writer: writer,
		done:   done,
	}
}

func (w *gsymWriter) WriteAt(p []byte, off int64) (n int, err error) {
	return w.buffer.WriteAt(p, off)
}

func (w *gsymWriter) Done() error {
	var err error
	defer func() {
		if err != nil {
			_ = w.done()
		}
	}()

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return err
	}

	result, err := decoder.DecodeAll(w.buffer.Bytes(), []byte{})
	if err != nil {
		return err
	}

	_, err = w.writer.WriteAt(result, 0)
	if err != nil {
		return err
	}

	return w.done()
}

func (d *Downloader) performDownload(ctx context.Context, binary *binary) error {
	writer, done, err := binary.acquiredFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = done()
		}
	}()

	return binary.load(ctx, writer, done)
}

func (d *Downloader) runDownload(ctx context.Context, download *binary) {
	defer d.semaphore.Release(1)
	d.metrics.downloadsInFly.Add(1)
	defer d.metrics.downloadsInFly.Add(-1)

	ctx, cancel := context.WithTimeout(ctx, DownloadTimeout)
	defer cancel()

	l := d.l.With(
		log.String("build_id", download.info.BuildID),
		log.UInt64("size", download.info.Size),
		log.String("function", "runDownload"),
	)
	l.Info(ctx, "Start binary download")
	ts := time.Now()

	err := d.performDownload(ctx, download)
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
		var req *binary
		select {
		case req = <-d.binariesQueue:
		case <-ctx.Done():
			return ctx.Err()
		}

		_ = d.semaphore.Acquire(ctx, 1)
		go d.runDownload(ctx, req)
	}
}

func getBinaryMeta(ctx context.Context, binaryStorage storage.Storage, buildID string) (*binarymeta.BinaryMeta, error) {
	var err error
	ctx, span := otel.Tracer("Symbolizer").Start(
		ctx, "downloader.(*Downloader).getBinaryMeta",
		trace.WithAttributes(attribute.String("buildID", buildID)),
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	binaries, err := binaryStorage.GetBinaries(ctx, []string{buildID})
	if err != nil {
		return nil, err
	}
	if len(binaries) == 0 {
		return nil, fmt.Errorf("no binary %s found", buildID)
	}

	return binaries[0], nil
}

func (d *Downloader) scheduleBinaryForDownload(ctx context.Context, binary *binary) error {
	d.metrics.scheduledBinaries.Inc()

	select {
	case d.binariesQueue <- binary:
	case <-ctx.Done():
		return ctx.Err()
	}

	d.l.Info(ctx, "Scheduled binary for download", log.String("build_id", binary.info.BuildID))
	return nil
}

func (d *Downloader) acquire(
	ctx context.Context,
	sizeCache *lru.Cache,
	binaryStorage storage.Storage,
	buildID string,
	binaryType binaryType,
	filePrefix string,
) (binaryprovider.FileHandle, error) {
	sz, err := getBinarySize(ctx, sizeCache, binaryStorage, buildID, binaryType)
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
		err = d.scheduleBinaryForDownload(ctx, &binary{
			acquiredFile:  acquiredRef,
			info:          binaryInfo,
			binaryType:    binaryType,
			binaryStorage: binaryStorage,
		})
		if err != nil {
			return nil, err
		}
	}

	return acquiredRef, nil
}

func getBinarySize(
	ctx context.Context,
	sizeCache *lru.Cache,
	binaryStorage storage.Storage,
	buildID string,
	binaryType binaryType,
) (uint64, error) {
	sizeFromCache, ok := sizeCache.Get(buildID)
	if ok {
		return sizeFromCache.(uint64), nil
	}

	meta, err := getBinaryMeta(ctx, binaryStorage, buildID)
	if err != nil {
		return 0, err
	}
	if meta.BlobInfo == nil {
		return 0, fmt.Errorf("there is no blob for binary %s", buildID)
	}

	cachedSize := meta.BlobInfo.Size
	if binaryType == binaryTypeGSYM {
		if meta.GSYMBlobInfo == nil {
			return 0, fmt.Errorf("these is no GSYM for binary %s", buildID)
		}
		cachedSize = meta.GSYMBlobInfo.Size
	}

	sizeCache.Add(buildID, cachedSize)

	return cachedSize, nil
}

func getBinaryFileEntryName(buildID string, prefix string) string {
	buildIDAsPath := strings.ReplaceAll(buildID, "/", "%")

	return prefix + buildIDAsPath
}

type BinaryDownloader struct {
	downloader    *Downloader
	binaryStorage storage.Storage

	sizeCache *lru.Cache
}

func NewBinaryDownloader(downloader *Downloader, binaryStorage storage.Storage) (*BinaryDownloader, error) {
	sizeCache, err := lru.New(BinarySizesCacheSize)
	if err != nil {
		return nil, err
	}

	return &BinaryDownloader{
		downloader:    downloader,
		binaryStorage: binaryStorage,
		sizeCache:     sizeCache,
	}, nil
}

func (d *BinaryDownloader) Acquire(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	return d.downloader.acquire(ctx, d.sizeCache, d.binaryStorage, buildID, binaryTypeDefault, BinaryFilePrefix)
}

type GSYMDownloader struct {
	downloader    *Downloader
	binaryStorage storage.Storage

	sizeCache *lru.Cache
}

func NewGSYMDownloader(downloader *Downloader, binaryStorage storage.Storage) (*GSYMDownloader, error) {
	sizeCache, err := lru.New(BinarySizesCacheSize)
	if err != nil {
		return nil, err
	}

	return &GSYMDownloader{
		downloader:    downloader,
		binaryStorage: binaryStorage,
		sizeCache:     sizeCache,
	}, nil
}

func (d *GSYMDownloader) Acquire(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	return d.downloader.acquire(ctx, d.sizeCache, d.binaryStorage, buildID, binaryTypeGSYM, GSYMFilePrefix)
}

func CreateDownloaders(
	fileCacheConfig *asyncfilecache.Config,
	maxSimultaneousDownloads uint32,
	l xlog.Logger,
	reg metrics.Registry,
	binaryStorage storage.Storage,
	gsymStorage storage.Storage,
) (*Downloader, *BinaryDownloader, *GSYMDownloader, error) {
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
