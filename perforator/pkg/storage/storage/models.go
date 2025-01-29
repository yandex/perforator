package storage

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

type ObjectID string

type BlobInfo struct {
	ID   string
	Size uint64
}

type ObjectMeta struct {
	ID                string
	BlobInfo          *BlobInfo
	LastUsedTimestamp time.Time
}

type ShardParams struct {
	NumShards  uint32
	ShardIndex uint32
}

type Storage interface {
	CollectExpired(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
		shardParams *ShardParams,
	) ([]*ObjectMeta, error)

	Delete(ctx context.Context, IDs []string) error
}
