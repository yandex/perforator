package processors

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/klauspost/compress/zstd"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/offline_processing/gsym"
	"github.com/yandex/perforator/perforator/internal/offline_processing/models"
	blob_storage "github.com/yandex/perforator/perforator/pkg/storage/blob/s3"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const kZstdCompressionLevel = 6

func compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(kZstdCompressionLevel)))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	result := []byte{}
	return encoder.EncodeAll(data, result), nil
}

type GsymProcessor struct {
	s3 *blob_storage.S3Storage
}

func NewGsymProcessor(l xlog.Logger, reg metrics.Registry, s3Client *s3.S3, s3Bucket string) (*GsymProcessor, error) {
	if s3Client == nil {
		return nil, fmt.Errorf("s3client is nil")
	}

	gsymStorage, err := blob_storage.NewS3Storage(
		l,
		reg.WithPrefix("gsym_storage"),
		s3Client,
		s3Bucket,
	)
	if err != nil {
		return nil, err
	}

	return &GsymProcessor{
		s3: gsymStorage,
	}, nil
}

func (p *GsymProcessor) ProcessBinary(ctx context.Context, trx models.ProcessingTransaction, binaryID string, binaryPath string) error {
	output, err := os.CreateTemp("", "*.gsym")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())

	err = gsym.ConvertDWARFToGsym(binaryPath, output.Name(), 4 /* TODO: make it configurable, PERFORATOR-426 */)
	if err != nil {
		return err
	}

	gsym, err := os.ReadFile(output.Name())
	if err != nil {
		return err
	}

	compressedGsym, err := compressZstd(gsym)
	if err != nil {
		return err
	}

	err = p.uploadGsym(ctx, binaryID, compressedGsym)
	if err != nil {
		return err
	}

	err = trx.SetGSYMSizes(ctx, uint64(len(gsym)), uint64(len(compressedGsym)))
	if err != nil {
		return err
	}

	return nil
}

func (p *GsymProcessor) uploadGsym(ctx context.Context, id string, gsym []byte) error {
	w, err := p.s3.Put(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to start s3 gsym writer: %w", err)
	}

	_, err = w.Write(gsym)
	if err != nil {
		return fmt.Errorf("failed to upload gsym to s3: %w", err)
	}

	_, err = w.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit s3 gsym upload: %w", err)
	}

	return nil
}
