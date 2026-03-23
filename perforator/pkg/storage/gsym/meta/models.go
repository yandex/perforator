package gsymmeta

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

type GSYMMeta struct {
	BuildID           string
	CompressedSize    uint64
	UncompressedSize  uint64
	LastUsedTimestamp time.Time
}

type Storage interface {
	GetGSYMs(
		ctx context.Context,
		buildIDs []string,
	) ([]*GSYMMeta, error)

	CollectExpiredGSYMs(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
	) ([]*GSYMMeta, error)

	RemoveGSYMs(
		ctx context.Context,
		buildIDs []string,
	) error
}
