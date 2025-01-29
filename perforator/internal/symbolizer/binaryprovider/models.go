package binaryprovider

import (
	"context"
)

type BinaryInfo struct {
	BuildID string
	Size    uint64
}

type FileHandle interface {
	Close()
	Path() string
	WaitStored(ctx context.Context) error
}

type BinaryProvider interface {
	Acquire(ctx context.Context, buildID string) (FileHandle, error)
}
