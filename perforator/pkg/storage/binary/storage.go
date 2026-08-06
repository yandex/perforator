package binary

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync/atomic"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

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

func (s *BinaryStorage) LoadBinary(
	ctx context.Context,
	buildID string,
	writer io.WriterAt,
) (*binarymeta.BinaryMeta, error) {
	meta, err := s.loadBinaryMeta(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if meta.BlobInfo == nil {
		return nil, fmt.Errorf("no blob for binary %s", meta.BuildID)
	}

	r, err := s.blobStorage.Get(ctx, meta.BlobInfo.ID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	_, err = io.Copy(io.NewOffsetWriter(writer, 0), r)
	if err != nil {
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
