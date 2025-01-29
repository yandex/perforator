package app

import (
	"context"

	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
)

type S3BinaryFetcher struct {
	binaryProvider binaryprovider.BinaryProvider
}

func NewS3BinaryFetcher(binaryProvider binaryprovider.BinaryProvider) (*S3BinaryFetcher, error) {
	return &S3BinaryFetcher{
		binaryProvider: binaryProvider,
	}, nil
}

func (f *S3BinaryFetcher) FetchBinary(ctx context.Context, binaryID string) (binaryprovider.FileHandle, error) {
	handle, err := f.binaryProvider.Acquire(ctx, binaryID)
	if err != nil {
		return nil, err
	}

	err = handle.WaitStored(ctx)
	if err != nil {
		return nil, err
	}

	return handle, nil
}
