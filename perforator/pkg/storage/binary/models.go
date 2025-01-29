package binary

import (
	"context"
	"errors"
	"io"
	"time"

	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

var (
	ErrNotFound = errors.New("binary not found")
)

type TransactionalWriter interface {
	io.Writer
	Commit(ctx context.Context) error
	Abort(ctx context.Context) error
}

type Storage interface {
	StoreBinary(
		ctx context.Context,
		binaryMeta *binarymeta.BinaryMeta,
	) (TransactionalWriter, error)

	LoadBinary(
		ctx context.Context,
		buildID string,
		writer io.WriterAt,
	) (*binarymeta.BinaryMeta, error)

	GetBinaries(
		ctx context.Context,
		buildIDs []string,
	) ([]*binarymeta.BinaryMeta, error)

	CollectExpired(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
		shardParams *storage.ShardParams,
	) ([]*storage.ObjectMeta, error)

	Delete(ctx context.Context, IDs []string) error
}

type StorageSelector interface {
	Binary() Storage

	GSYM() Storage
}
