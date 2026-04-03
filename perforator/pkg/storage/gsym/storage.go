package gsym

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type gsymStorage struct {
	logger      xlog.Logger
	metaStorage gsymmeta.Storage
	blobStorage blob.Storage
}

func NewStorage(
	metaStorage gsymmeta.Storage,
	blobStorage blob.Storage,
	logger xlog.Logger,
) Storage {
	return &gsymStorage{
		metaStorage: metaStorage,
		blobStorage: blobStorage,
		logger:      logger,
	}
}

func (s *gsymStorage) loadGSYMMeta(
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

func (s *gsymStorage) LoadGSYM(
	ctx context.Context,
	buildID string,
	writer io.WriterAt,
) (*gsymmeta.GSYMMeta, error) {
	meta, err := s.loadGSYMMeta(ctx, buildID)
	if err != nil {
		return nil, err
	}

	err = s.blobStorage.Get(ctx, buildID, writer)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (s *gsymStorage) GetGSYMs(
	ctx context.Context,
	buildIDs []string,
) ([]*gsymmeta.GSYMMeta, error) {
	return s.metaStorage.GetGSYMs(ctx, buildIDs)
}

func (s *gsymStorage) CollectExpired(
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

func (s *gsymStorage) Delete(
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
