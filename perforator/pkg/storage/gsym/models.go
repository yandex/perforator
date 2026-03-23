package gsym

import (
	"context"
	"errors"
	"io"
	"time"

	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

var (
	errNotFound = errors.New("gsym not found")
)

type Storage interface {
	LoadGSYM(
		ctx context.Context,
		buildID string,
		writer io.WriterAt,
	) (*gsymmeta.GSYMMeta, error)

	GetGSYMs(
		ctx context.Context,
		buildIDs []string,
	) ([]*gsymmeta.GSYMMeta, error)

	CollectExpired(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
		shardParams *storage.ShardParams,
	) ([]*storage.ObjectMeta, error)

	Delete(ctx context.Context, IDs []string) error
}
