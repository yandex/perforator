package binary

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xio"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

var (
	ErrNotFound = errors.New("binary not found")
)

type TransactionalWriter interface {
	io.Writer
	Commit(ctx context.Context) error
	Abort(ctx context.Context) error
}

type BinaryStorage struct {
	logger xlog.Logger
	reg    metrics.Registry

	metaStorage binarymeta.Storage
	blobStorage blob.Storage

	failedS3DeleteCounter metrics.Counter
}

func NewStorage(
	metaStorage binarymeta.Storage,
	blobStorage blob.Storage,
	logger xlog.Logger,
	reg metrics.Registry,
) *BinaryStorage {
	reg = reg.WithPrefix("binaries")

	return &BinaryStorage{
		metaStorage:           metaStorage,
		blobStorage:           blobStorage,
		logger:                logger,
		reg:                   reg,
		failedS3DeleteCounter: reg.Counter("failed_to_delete_blobs_error.count"),
	}
}

type BinaryStorageWriter struct {
	written    atomic.Uint64
	buildID    string
	storage    *BinaryStorage
	commiter   binarymeta.Commiter
	blobWriter blob.Writer
	lastPing   time.Time
	logger     xlog.Logger
	ctx        context.Context
}

func NewBinaryStorageWriter(
	ctx context.Context,
	buildID string,
	commiter binarymeta.Commiter,
	storage *BinaryStorage,
	writer blob.Writer,
) (*BinaryStorageWriter, error) {
	return &BinaryStorageWriter{
		buildID:    buildID,
		storage:    storage,
		commiter:   commiter,
		blobWriter: writer,
		lastPing:   time.Now(),
		logger:     storage.logger.With(log.String("build_id", buildID)),
		ctx:        ctx,
	}, nil
}

func (w *BinaryStorageWriter) maybePing() {
	if time.Since(w.lastPing) > 30*time.Second {
		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		defer cancel()
		err := w.commiter.Ping(ctx)
		if err != nil {
			w.logger.Warn(ctx,
				"Failed ping for binary in upload progress",
				log.Error(err),
			)
		} else {
			w.lastPing = time.Now()
		}
	}
}

func (w *BinaryStorageWriter) Write(p []byte) (int, error) {
	w.maybePing()
	n, err := w.blobWriter.Write(p)
	w.written.Add(uint64(n))
	return n, err
}

func (w *BinaryStorageWriter) Abort(ctx context.Context) error {
	// TODO: abort blob.Writer
	return w.commiter.Abort(ctx)
}

func (w *BinaryStorageWriter) Commit(ctx context.Context) error {
	blobID, err := w.blobWriter.Commit()
	if err != nil {
		return err
	}

	w.logger.Debug(ctx, "Uploaded binary blob")

	err = w.commiter.Commit(ctx, &storage.BlobInfo{ID: blobID, Size: w.written.Load()})
	if err != nil {
		deleteErr := w.storage.blobStorage.Delete(ctx, blobID)
		if deleteErr != nil {
			w.logger.Error(ctx,
				"Failed to delete blob after unsuccessful commit attempt",
				log.Error(deleteErr),
			)
		}
		return err
	}

	w.logger.Info(ctx, "Successfully stored binary")

	return nil
}

func (s *BinaryStorage) StoreBinary(
	ctx context.Context,
	buildID string,
	timestamp time.Time,
	opts ...binarymeta.Option,
) (TransactionalWriter, error) {
	commiter, err := s.metaStorage.StoreBinary(ctx, buildID, timestamp, opts...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = commiter.Abort(ctx)
		}
	}()

	writer, err := s.blobStorage.Put(ctx, buildID)
	if err != nil {
		return nil, err
	}

	return NewBinaryStorageWriter(ctx, buildID, commiter, s, writer)
}

func (s *BinaryStorage) loadBinaryMeta(
	ctx context.Context,
	buildID string,
) (*binarymeta.BinaryMeta, error) {
	metas, err := s.metaStorage.GetBinaries(ctx, []string{buildID})
	if err != nil {
		return nil, err
	}

	if len(metas) == 0 {
		return nil, ErrNotFound
	}

	if len(metas) > 1 { // possible?
		return nil, fmt.Errorf(
			"fetched unexpected number of binary metas (expected 1, got %d)",
			len(metas),
		)
	}

	return metas[0], nil
}

// loadBlob writes the binary described by meta into w, decompressing it on
// the fly when the blob is compressed at rest.
func (s *BinaryStorage) loadBlob(
	ctx context.Context,
	meta *binarymeta.BinaryMeta,
	w io.WriterAt,
) error {
	r, err := s.openBlob(ctx, meta)
	if err != nil {
		return err
	}
	defer r.Close()

	if _, err := io.Copy(io.NewOffsetWriter(w, 0), r); err != nil {
		return fmt.Errorf("binary %s: %w", meta.BuildID, err)
	}
	return nil
}

// openBlob returns the binary's byte stream: the blob as stored, wrapped in a
// decompressor when meta says it is compressed at rest. Compressed streams
// are checked against the declared uncompressed size — consumers size caches
// by it, so a deviating stream errors instead of ending.
func (s *BinaryStorage) openBlob(ctx context.Context, meta *binarymeta.BinaryMeta) (io.ReadCloser, error) {
	if meta.BlobInfo == nil {
		return nil, fmt.Errorf("no blob for binary %s", meta.BuildID)
	}

	switch meta.Compression {
	case compressionpb.CompressionMethod_None:
		r, err := s.blobStorage.Get(ctx, meta.BlobInfo.ID)
		if err != nil {
			return nil, err
		}
		// The stream is the blob, so its length is the size recorded for the
		// blob itself: uncompressed_size only mirrors it, and on rows written
		// before that column existed it is zero.
		return xio.NewReadMultiCloser(xio.NewSizedReader(r, int64(meta.BlobInfo.Size)), r), nil
	case compressionpb.CompressionMethod_Zstd:
		if meta.UncompressedSize == 0 || meta.UncompressedSize > math.MaxInt64 {
			return nil, fmt.Errorf("binary %s: implausible uncompressed_size %d", meta.BuildID, meta.UncompressedSize)
		}
		r, err := s.blobStorage.Get(ctx, meta.BlobInfo.ID)
		if err != nil {
			return nil, err
		}
		decoder, err := zstd.NewReader(r)
		if err != nil {
			_ = r.Close()
			return nil, fmt.Errorf("binary %s: failed to create zstd reader: %w", meta.BuildID, err)
		}
		// zstd's own Close does not close its source, so both are owned here.
		decoded := decoder.IOReadCloser()
		return xio.NewReadMultiCloser(xio.NewSizedReader(decoded, int64(meta.UncompressedSize)), decoded, r), nil
	case compressionpb.CompressionMethod_Unknown:
		return nil, fmt.Errorf("binary %s: compression is unknown", meta.BuildID)
	default:
		return nil, fmt.Errorf("binary %s: unsupported compression method: %s", meta.BuildID, meta.Compression)
	}
}

func (s *BinaryStorage) LoadBinary(
	ctx context.Context,
	buildID string,
	writer io.WriterAt,
) (*binarymeta.BinaryMeta, error) {
	meta, err := s.loadBinaryMeta(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if err := s.loadBlob(ctx, meta, writer); err != nil {
		return nil, err
	}

	return meta, nil
}

func (s *BinaryStorage) fillBlobSize(ctx context.Context, meta *binarymeta.BinaryMeta) error {
	var err error
	meta.BlobInfo.Size, err = s.blobStorage.Size(ctx, meta.BlobInfo.ID)
	if err != nil {
		noExistErr := &blob.ErrNoExist{}
		if !errors.As(err, &noExistErr) {
			return err
		}

		// set blob info to nil if no exist error for blob
		meta.BlobInfo = nil
	}

	return nil
}

func (s *BinaryStorage) GetBinaries(
	ctx context.Context,
	buildIDs []string,
) ([]*binarymeta.BinaryMeta, error) {
	metas, err := s.metaStorage.GetBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}

	res := make([]*binarymeta.BinaryMeta, 0, len(metas))
	for _, meta := range metas {
		meta := meta

		if meta.BlobInfo == nil || meta.BlobInfo.ID == "" || meta.Status == binarymeta.InProgress {
			res = append(res, meta)
			continue
		}

		err = s.fillBlobSize(ctx, meta)
		if err != nil {
			return nil, err
		}

		res = append(res, meta)
	}

	return res, nil
}

func (s *BinaryStorage) CollectExpired(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
	shardParams *storage.ShardParams,
) ([]*storage.ObjectMeta, error) {
	metas, err := s.metaStorage.CollectExpiredBinaries(ctx, ttl, pagination)
	if err != nil {
		return nil, err
	}

	result := make([]*storage.ObjectMeta, 0, len(metas))
	for _, meta := range metas {
		result = append(result, &storage.ObjectMeta{
			ID:                meta.BuildID,
			LastUsedTimestamp: meta.LastUsedTimestamp,
		})
	}

	return result, nil
}

func (s *BinaryStorage) Delete(
	ctx context.Context,
	IDs []string,
) error {
	metas, err := s.metaStorage.GetBinaries(ctx, IDs)
	if err != nil {
		return err
	}

	failedToDeleteBlobs := make(map[string]struct{}, 0)
	for _, meta := range metas {
		l := s.logger.With(log.String("build_id", string(meta.BuildID)), log.Any("blob_info", meta.BlobInfo))

		err = s.blobStorage.Delete(ctx, meta.BuildID)
		if err != nil {
			var noExistErr *blob.ErrNoExist
			if errors.As(err, &noExistErr) {
				l.Info(ctx, "Blob to delete was not found")
			} else {
				l.Error(
					ctx,
					"Failed to delete binary blob",
					log.Error(err),
				)
				s.failedS3DeleteCounter.Inc()
				failedToDeleteBlobs[meta.BuildID] = struct{}{}
			}
		} else {
			l.Info(ctx, "Deleted binary blob")
		}
	}

	metas = slices.DeleteFunc(metas, func(meta *binarymeta.BinaryMeta) bool {
		_, ok := failedToDeleteBlobs[meta.BuildID]
		return ok
	})

	idsToDelete := make([]string, 0, len(metas))
	for _, meta := range metas {
		idsToDelete = append(idsToDelete, meta.BuildID)
	}

	err = s.metaStorage.RemoveBinaries(ctx, idsToDelete)
	if err != nil {
		return err
	}

	return err
}
