package binaryprovider

import (
	"context"
)

// FileHandle is a ready binary on disk. Acquire blocks until it is ready, so the
// handle is usable immediately; Close releases it (and unpins the cache entry).
type FileHandle interface {
	Close()
	Path() string
}

type BinaryProvider interface {
	Acquire(ctx context.Context, buildID string) (FileHandle, error)
}
