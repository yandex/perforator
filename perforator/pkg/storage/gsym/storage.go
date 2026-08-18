package gsym

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/yandex/perforator/library/go/core/log"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xio"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var errNotFound = errors.New("gsym not found")

type GSYMStorage struct {
	logger      xlog.Logger
	metaStorage gsymmeta.Storage
	blobStorage blob.Storage
}

func NewStorage(
	metaStorage gsymmeta.Storage,
	blobStorage blob.Storage,
	logger xlog.Logger,
) *GSYMStorage {
	return &GSYMStorage{
		metaStorage: metaStorage,
		blobStorage: blobStorage,
		logger:      logger,
	}
}

func (s *GSYMStorage) loadGSYMMeta(
	ctx context.Context,
	buildID string,
) (*gsymmeta.GSYMMeta, error) {
	metas, err := s.metaStorage.GetGSYMs(ctx, []string{buildID})
	if err != nil {
		return nil, err
	}

	if len(metas) == 0 {
		return nil, errNotFound
	}

	if len(metas) > 1 {
		return nil, fmt.Errorf(
			"fetched unexpected number of gsym metas (expected 1, got %d)",
			len(metas),
		)
	}

	return metas[0], nil
}

// LoadGSYM writes the decompressed GSYM for buildID into writer (blobs are
// always zstd-compressed at rest).
func (s *GSYMStorage) LoadGSYM(
	ctx context.Context,
	buildID string,
	writer io.WriterAt,
) (*gsymmeta.GSYMMeta, error) {
	meta, err := s.loadGSYMMeta(ctx, buildID)
	if err != nil {
		return nil, err
	}
	if meta.UncompressedSize == 0 || meta.UncompressedSize > math.MaxInt64 {
		return nil, fmt.Errorf("gsym %s: implausible uncompressed_size %d", buildID, meta.UncompressedSize)
	}

	r, err := s.blobStorage.Get(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	decoder, err := zstd.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer decoder.Close()

	sized := xio.NewSizedReader(decoder, int64(meta.UncompressedSize))
	if _, err := io.Copy(io.NewOffsetWriter(writer, 0), sized); err != nil {
		return nil, fmt.Errorf("gsym %s: %w", buildID, err)
	}

	return meta, nil
}

func (s *GSYMStorage) GetGSYMs(
	ctx context.Context,
	buildIDs []string,
) ([]*gsymmeta.GSYMMeta, error) {
	return s.metaStorage.GetGSYMs(ctx, buildIDs)
}

func (s *GSYMStorage) CollectExpired(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
	shardParams *storage.ShardParams,
) ([]*storage.ObjectMeta, error) {
	metas, err := s.metaStorage.CollectExpiredGSYMs(ctx, ttl, pagination)
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

func (s *GSYMStorage) Delete(
	ctx context.Context,
	IDs []string,
) error {
	err := s.metaStorage.RemoveGSYMs(ctx, IDs)
	if err != nil {
		return err
	}

	blobErr := s.blobStorage.DeleteObjects(ctx, IDs)
	if blobErr != nil {
		s.logger.Error(ctx, "Failed to delete GSYM blobs", log.Error(blobErr))
	}

	return blobErr
}
