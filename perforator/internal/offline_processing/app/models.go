package app

import (
	"context"

	"github.com/yandex/perforator/perforator/internal/offline_processing/models"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
)

type BinaryTranscationHandler interface {
	GetBinaryID() string

	Finalize(ctx context.Context, processingErr error)

	models.ProcessingTransaction
}

type BinarySelector interface {
	SelectBinary(ctx context.Context) (BinaryTranscationHandler, error)

	GetQueuedBinariesCount(ctx context.Context) (uint64, error)
}

type BinaryFetcher interface {
	FetchBinary(ctx context.Context, binaryID string) (binaryprovider.FileHandle, error)
}

type BinaryProcessor interface {
	ProcessBinary(ctx context.Context, trx models.ProcessingTransaction, binaryID string, binaryPath string) error
}
