package models

import "context"

type ProcessingTransaction interface {
	SetGSYMSizes(ctx context.Context, uncompressedSize uint64, compressedSize uint64) error
}
