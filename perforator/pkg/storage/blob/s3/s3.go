package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	s3client "github.com/yandex/perforator/perforator/pkg/s3"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	defaultDownloadConcurrency = 20
	MaximumShards              = 256
	UploadConcurrency          = 20

	AwsNotFoundCode = "NotFound"
)

var _ models.Storage = (*S3Storage)(nil)

type mdsStorageMetrics struct {
	bytesDownloaded metrics.Counter
	bytesUploaded   metrics.Counter
	// degradedDownloads counts Get calls that could not use the parallel
	// ranged path, by reason.
	degradedDownloads metrics.CounterVec
}

type S3Storage struct {
	bucket string
	l      xlog.Logger

	client   s3iface.S3API
	uploader *s3manager.Uploader
	deleter  *s3manager.BatchDelete
	// downloader is used with an explicit Range only: that path downloads
	// the single range synchronously (no internal part fan-out) while
	// keeping the SDK's interrupted-body retry loop.
	downloader *s3manager.Downloader

	downloadConcurrency int
	downloadPartSize    int64

	metrics *mdsStorageMetrics
}

type WriteAtBuffer = aws.WriteAtBuffer

func NewS3Storage(l xlog.Logger, reg metrics.Registry, client *s3client.Client, bucket string) (*S3Storage, error) {
	return &S3Storage{
		bucket: bucket,
		l:      l.WithName("s3storage"),
		client: client,
		uploader: s3manager.NewUploaderWithClient(client, func(d *s3manager.Uploader) {
			d.Concurrency = UploadConcurrency
		}),
		downloader: s3manager.NewDownloaderWithClient(client, func(d *s3manager.Downloader) {
			d.Concurrency = 1
		}),
		downloadConcurrency: defaultDownloadConcurrency,
		downloadPartSize:    s3manager.DefaultDownloadPartSize,
		deleter: s3manager.NewBatchDeleteWithClient(client, func(d *s3manager.BatchDelete) {
			d.BatchSize = s3manager.DefaultBatchSize
		}),
		metrics: &mdsStorageMetrics{
			bytesDownloaded:   reg.Counter("downloaded.bytes"),
			bytesUploaded:     reg.Counter("uploaded.bytes"),
			degradedDownloads: reg.CounterVec("degraded_downloads.count", []string{"reason"}),
		},
	}, nil
}

// Delete implements Storage
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	l := s.keylog(key)
	l.Debug(ctx, "Deleting S3 object")

	_, err := s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})

	if err != nil {
		l.Error(ctx, "Failed to remove S3 object", log.Error(err))
		return fmt.Errorf("failed to remove value: %w", err)
	}
	return nil
}

// DeleteObjects implements storage
func (s *S3Storage) DeleteObjects(ctx context.Context, keys []string) error {
	l := s.l.With(log.Array("keys", keys))

	objectIDs := make([]*s3.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		key := key
		objectIDs = append(objectIDs, &s3.ObjectIdentifier{
			Key: &key,
		})
	}

	l.Debug(ctx, "Deleting S3 objects")

	output, err := s.client.DeleteObjectsWithContext(
		ctx,
		&s3.DeleteObjectsInput{
			Bucket: &s.bucket,
			Delete: &s3.Delete{
				Objects: objectIDs,
			},
		},
	)

	if len(output.Errors) != 0 {
		for _, err := range output.Errors {
			l.Error(ctx, "Failed to delete object from S3",
				log.Any("message", err.Message),
				log.Any("code", err.Code),
				log.Any("key", err.Key),
			)
		}
	}

	return err
}

func isNoExistError(err error) bool {
	var aerr awserr.Error
	if errors.As(err, &aerr) {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchKey:
			return true
		// shitty
		case AwsNotFoundCode:
			return true
		default:
		}
	}
	return false
}

// Get implements Storage. The first ranged request discovers the blob size;
// larger blobs are streamed with parallel ranged fetches ahead of the reader.
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Range:  aws.String(fmt.Sprintf("bytes=0-%d", s.downloadPartSize-1)),
	})
	if err != nil {
		if isInvalidRangeError(err) {
			// A ranged GET on a zero-length blob is unsatisfiable. Rather
			// than assign the error code any broader meaning, fall back to a
			// plain GET: an empty object yields an empty stream, anything
			// else downloads sequentially.
			s.degraded("invalid_range")
			return s.getSequential(ctx, key)
		}
		if isNoExistError(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		}
		s.keylog(key).Error(ctx, "Failed to fetch blob", log.Error(err))
		return nil, err
	}

	if out.ContentRange == nil {
		// The server ignored Range and is returning the whole object: stream
		// it sequentially instead of buffering an unbounded body.
		s.degraded("range_ignored")
		return &meteredReader{r: out.Body, counter: s.metrics.bytesDownloaded}, nil
	}
	start, end, total, ok := parseContentRange(*out.ContentRange)
	if !ok {
		// Unknown total (e.g. "bytes 0-N/*", the same case the aws
		// downloader degrades on): one sequential unranged GET.
		s.degraded("unknown_total")
		_ = out.Body.Close()
		return s.getSequential(ctx, key)
	}
	firstLen := min(total, s.downloadPartSize)
	if start != 0 || end != firstLen-1 {
		// The response answers a different interval than requested; do not
		// risk splicing it in as part zero.
		s.degraded("interval_mismatch")
		_ = out.Body.Close()
		return s.getSequential(ctx, key)
	}
	etag := out.ETag
	if etag == nil || *etag == "" {
		// Without an ETag the later parts cannot be pinned to this object
		// version, and a concurrent overwrite could splice two versions into
		// one stream.
		s.degraded("no_etag")
		_ = out.Body.Close()
		return s.getSequential(ctx, key)
	}

	first := make([]byte, firstLen)
	_, err = io.ReadFull(out.Body, first)
	_ = out.Body.Close()
	if err != nil {
		// Interrupted initial body: refetch the range through the retrying
		// downloader.
		s.degraded("first_part_refetch")
		first, err = s.fetchPart(ctx, key, etag, 0, firstLen-1)
		if err != nil {
			return nil, err
		}
	} else {
		s.metrics.bytesDownloaded.Add(firstLen)
	}

	if total <= s.downloadPartSize {
		return io.NopCloser(bytes.NewReader(first)), nil
	}

	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		return s.fetchPart(ctx, key, etag, start, end)
	}

	return newRangedReader(ctx, fetch, first, total, s.downloadConcurrency, s.downloadPartSize), nil
}

// fetchPart downloads bytes [start, end] of the object through the SDK
// downloader, which retries interrupted bodies; the explicit Range keeps it to
// a single synchronous ranged request. IfMatch pins the object version seen by
// the initial request so a concurrent overwrite cannot splice two versions
// into one stream.
func (s *S3Storage) fetchPart(ctx context.Context, key string, etag *string, start, end int64) ([]byte, error) {
	buf := aws.NewWriteAtBuffer(make([]byte, 0, end-start+1))
	n, err := s.downloader.DownloadWithContext(ctx, buf, &s3.GetObjectInput{
		Bucket:  &s.bucket,
		Key:     &key,
		Range:   aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
		IfMatch: etag,
	})
	if err != nil {
		return nil, err
	}
	if n != end-start+1 {
		return nil, fmt.Errorf("range %d-%d of %s: expected %d bytes, got %d", start, end, key, end-start+1, n)
	}
	s.metrics.bytesDownloaded.Add(n)
	return buf.Bytes(), nil
}

func (s *S3Storage) degraded(reason string) {
	s.metrics.degradedDownloads.With(map[string]string{"reason": reason}).Inc()
}

func isInvalidRangeError(err error) bool {
	var aerr awserr.Error
	return errors.As(err, &aerr) && aerr.Code() == "InvalidRange"
}

// parseContentRange parses a Content-Range header of the form
// "bytes 0-123/456"; "*" as the total means the server does not know it.
// Mirrors the aws downloader's private setTotalBytes:
// https://github.com/aws/aws-sdk-go/blob/main/service/s3/s3manager/download.go
func parseContentRange(contentRange string) (start, end, total int64, ok bool) {
	interval, totalStr, found := strings.Cut(contentRange, "/")
	if !found || totalStr == "*" {
		return 0, 0, 0, false
	}
	total, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil || total < 0 {
		return 0, 0, 0, false
	}
	unit, rangeStr, found := strings.Cut(interval, " ")
	if !found || unit != "bytes" {
		return 0, 0, 0, false
	}
	startStr, endStr, found := strings.Cut(rangeStr, "-")
	if !found {
		return 0, 0, 0, false
	}
	start, err = strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, 0, false
	}
	end, err = strconv.ParseInt(endStr, 10, 64)
	if err != nil || end < start || end >= total {
		return 0, 0, 0, false
	}
	return start, end, total, true
}

func (s *S3Storage) getSequential(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if isNoExistError(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		}
		return nil, err
	}
	return &meteredReader{r: out.Body, counter: s.metrics.bytesDownloaded}, nil
}

// meteredReader charges the transfer counter as bytes arrive.
type meteredReader struct {
	r       io.ReadCloser
	counter metrics.Counter
}

func (m *meteredReader) Read(p []byte) (int, error) {
	n, err := m.r.Read(p)
	m.counter.Add(int64(n))
	return n, err
}

func (m *meteredReader) Close() error { return m.r.Close() }

func (s *S3Storage) Size(ctx context.Context, key string) (uint64, error) {
	l := s.keylog(key)
	l.Debug(ctx, "Fetching blob size")

	res, err := s.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})

	if err != nil {
		if isNoExistError(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		} else {
			l.Warn(ctx, "Failed to get value size", log.Error(err))
		}

		return 0, err
	}

	return uint64(*res.ContentLength), nil
}

type s3writer struct {
	key string
	l   xlog.Logger

	w    io.WriteCloser
	werr error

	uploadedBytes        atomic.Uint64
	uploadedBytesCounter metrics.Counter

	g        errgroup.Group
	uploader *s3manager.Uploader
}

func (s *S3Storage) newS3Writer(key string, uploader *s3manager.Uploader, l xlog.Logger) *s3writer {
	return &s3writer{
		key:                  key,
		uploader:             uploader,
		l:                    l.WithName("s3writer"),
		uploadedBytes:        atomic.Uint64{},
		uploadedBytesCounter: s.metrics.bytesUploaded,
	}
}

func (w *s3writer) start(ctx context.Context, bucket string) {
	rd, wr := io.Pipe()

	w.g.SetLimit(1)
	w.g.Go(func() error {
		w.l.Debug(ctx, "Starting s3 uploader")
		_, err := w.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
			Key:    &w.key,
			Body:   rd,
			Bucket: &bucket,
		})
		w.l.Debug(ctx, "Finished s3 uploader")
		if err != nil {
			_ = rd.CloseWithError(err)
		} else {
			_ = rd.Close()
		}
		return err
	})

	w.w = wr
}

func (w *s3writer) Write(p []byte) (int, error) {
	if w.werr != nil {
		return 0, w.werr
	}
	n, err := w.w.Write(p)
	w.uploadedBytes.Add(uint64(n))
	if err != nil {
		w.werr = err
		return 0, err
	}
	return n, nil
}

func (w *s3writer) Commit() (string, error) {
	if w.werr != nil {
		return "", w.werr
	}

	err := w.w.Close()
	if err != nil {
		return "", err
	}

	err = w.g.Wait()
	if err != nil {
		return "", err
	}

	w.uploadedBytesCounter.Add(int64(w.uploadedBytes.Load()))

	return w.key, nil
}

// Put implements Storage
func (s *S3Storage) Put(ctx context.Context, key string) (wr models.Writer, err error) {
	l := s.keylog(key)

	defer func() {
		if err != nil {
			l.Error(ctx, "Failed to upload value", log.Error(err))
		}
	}()

	uploader := s.newS3Writer(key, s.uploader, l)
	uploader.start(ctx, s.bucket)

	return uploader, nil
}

func getShardPrefix(shardParams *storage.ShardParams) (string, error) {
	if shardParams.NumShards > MaximumShards {
		return "", fmt.Errorf("num shards must be <= %d", MaximumShards)
	}

	return fmt.Sprintf("%02x", shardParams.ShardIndex), nil
}

func (s *S3Storage) List(ctx context.Context, pagination *models.Pagination, shards *storage.ShardParams) (result []string, err error) {
	var output *s3.ListObjectsV2Output
	var keyFrom *string
	if pagination.KeyFrom != "" {
		keyFrom = &pagination.KeyFrom
	}

	shardPrefix, err := getShardPrefix(shards)
	if err != nil {
		return nil, err
	}
	output, err = s.client.ListObjectsV2WithContext(
		ctx,
		&s3.ListObjectsV2Input{
			Bucket:     &s.bucket,
			MaxKeys:    ptr.Int64(int64(pagination.Limit)),
			StartAfter: keyFrom,
			Prefix:     &shardPrefix,
		},
	)
	if err != nil {
		return nil, err
	}

	result = []string{}
	for _, object := range output.Contents {
		result = append(result, *object.Key)
	}

	return
}

func (s *S3Storage) keylog(key string) xlog.Logger {
	return s.l.With(log.String("key", key))
}
