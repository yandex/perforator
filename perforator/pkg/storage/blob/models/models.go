package models

import (
	"context"
	"fmt"
	"io"

	"github.com/yandex/perforator/perforator/pkg/storage/storage"
)

type ErrNoExist struct {
	Err error
	Key string
}

func (e *ErrNoExist) Error() string {
	return fmt.Sprintf("key %s does not exist: %s", e.Key, e.Err.Error())
}

func (e *ErrNoExist) Unwrap() error {
	return e.Err
}

type Writer interface {
	io.Writer
	Commit() (string, error)
}

type Pagination struct {
	KeyFrom string

	// s3 supports up to 1000
	Limit uint32
}

type Storage interface {
	Put(ctx context.Context, key string) (Writer, error)

	Get(ctx context.Context, key string, w io.WriterAt) error

	Size(ctx context.Context, key string) (uint64, error)

	Delete(ctx context.Context, key string) error

	DeleteObjects(ctx context.Context, keys []string) error

	List(ctx context.Context, pagination *Pagination, shards *storage.ShardParams) ([]string, error)
}
