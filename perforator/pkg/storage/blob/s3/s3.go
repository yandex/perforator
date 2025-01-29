package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	MaximumShards       = 256
	UploadConcurrency   = 20
	DownloadConcurrency = 20

	AwsNotFoundCode = "NotFound"
)

var _ models.Storage = (*S3Storage)(nil)

type mdsStorageMetrics struct {
	bytesDownloaded metrics.Counter
	bytesUploaded   metrics.Counter
}

type S3Storage struct {
	bucket string
	l      xlog.Logger

	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	deleter    *s3manager.BatchDelete

	metrics *mdsStorageMetrics
}

type WriteAtBuffer = aws.WriteAtBuffer

func NewS3Storage(l xlog.Logger, reg metrics.Registry, client *s3.S3, bucket string) (*S3Storage, error) {
	return &S3Storage{
		bucket: bucket,
		l:      l.WithName("s3storage"),
		client: client,
		uploader: s3manager.NewUploaderWithClient(client, func(d *s3manager.Uploader) {
			d.Concurrency = UploadConcurrency
		}),
		downloader: s3manager.NewDownloaderWithClient(client, func(d *s3manager.Downloader) {
			d.Concurrency = DownloadConcurrency
		}),
		deleter: s3manager.NewBatchDeleteWithClient(client, func(d *s3manager.BatchDelete) {
			d.BatchSize = s3manager.DefaultBatchSize
		}),
		metrics: &mdsStorageMetrics{
			bytesDownloaded: reg.Counter("downloaded.bytes"),
			bytesUploaded:   reg.Counter("uploaded.bytes"),
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

// Get implements Storage
func (s *S3Storage) Get(ctx context.Context, key string, w io.WriterAt) error {
	l := s.keylog(key)
	l.Debug(ctx, "Fetching blob")
	start := time.Now()

	count, err := s.downloader.DownloadWithContext(ctx, w, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})

	s.metrics.bytesDownloaded.Add(count)

	if err != nil {
		if isNoExistError(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		}
		l.Error(ctx, "Failed to fetch blob", log.Error(err))
		return err
	} else {
		l.Debug(ctx, "Fetched blob",
			log.Int64("size", count),
			log.Duration("duration", time.Since(start)),
		)
	}

	return nil
}

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
		}

		l.Info(ctx, "Failed to get value size", log.Error(err))
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
