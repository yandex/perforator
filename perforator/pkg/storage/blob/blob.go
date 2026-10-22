package blob

import (
	"context"
	"fmt"
	"io"
)

type ErrNotExist struct {
	Err error
	Key string
}

func (e *ErrNotExist) Error() string {
	return fmt.Sprintf("key %s does not exist: %s", e.Key, e.Err.Error())
}

func (e *ErrNotExist) Unwrap() error {
	return e.Err
}

type Storage interface {
	Put(ctx context.Context, key string, src io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Size(ctx context.Context, key string) (uint64, error)
	Delete(ctx context.Context, key string) error
	DeleteObjects(ctx context.Context, keys []string) error

	// List returns up to limit keys in ascending order, starting strictly after the supplied key.
	List(ctx context.Context, after string, limit uint32) ([]string, error)
}
