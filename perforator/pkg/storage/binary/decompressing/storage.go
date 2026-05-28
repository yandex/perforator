package decompressing

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"

	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/xio/ringstream"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

const ringBufferHeadroomBytes = 64 * 1024 * 1024

var ErrInvalidDownloadConfig = errors.New("invalid parallel download config")

type Storage struct {
	binarystorage.Storage

	bufferSize        int
	writerConcurrency int
}

func calculateMinRingBufferBytes(parallelDownloadConfig models.ParallelDownloadConfig) int {
	return 2 * parallelDownloadConfig.Concurrency * int(parallelDownloadConfig.PartSize)
}

func calculateOptimalRingBufferSize(parallelDownloadConfig models.ParallelDownloadConfig) (int, error) {
	minRingBufferBytes := calculateMinRingBufferBytes(parallelDownloadConfig)
	if minRingBufferBytes <= 0 {
		return 0, ErrInvalidDownloadConfig
	}
	return minRingBufferBytes + ringBufferHeadroomBytes, nil
}

func New(inner binarystorage.Storage, parallelDownloadConfig models.ParallelDownloadConfig) (*Storage, error) {
	bufferSize, err := calculateOptimalRingBufferSize(parallelDownloadConfig)
	if err != nil {
		return nil, err
	}

	return &Storage{
		Storage:           inner,
		bufferSize:        bufferSize,
		writerConcurrency: parallelDownloadConfig.Concurrency,
	}, nil
}

func (s *Storage) LoadBinary(
	ctx context.Context,
	buildID string,
	writer io.WriterAt,
) (*binarymeta.BinaryMeta, error) {
	metas, err := s.GetBinaries(ctx, []string{buildID})
	if err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, binarystorage.ErrNotFound
	}
	meta := metas[0]

	switch meta.Compression {
	case compressionpb.CompressionMethod_Unknown:
		return nil, fmt.Errorf("binary %s: compression is unknown", buildID)
	case compressionpb.CompressionMethod_None:
		return s.Storage.LoadBinary(ctx, buildID, writer)
	case compressionpb.CompressionMethod_Zstd:
		if meta.UncompressedSize == 0 {
			return nil, fmt.Errorf("binary %s: compression=zstd but uncompressed_size is zero", buildID)
		}
		return s.decompressZstd(ctx, buildID, meta, writer)
	default:
		return nil, fmt.Errorf("unsupported compression method: %s", meta.Compression)
	}
}

func (s *Storage) decompressZstd(
	ctx context.Context,
	buildID string,
	meta *binarymeta.BinaryMeta,
	writer io.WriterAt,
) (*binarymeta.BinaryMeta, error) {
	ring := ringstream.NewAdapter(
		s.bufferSize,
		ringstream.WithDeadlockDetection(s.writerConcurrency),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	loadErrCh := make(chan error, 1)
	go func() {
		_, err := s.Storage.LoadBinary(ctx, buildID, ring)
		ring.CloseWithError(err)
		loadErrCh <- err
		close(loadErrCh)
	}()

	decoder, err := zstd.NewReader(ring)
	if err != nil {
		cancel()
		ring.CloseWithError(err)
		<-loadErrCh
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer decoder.Close()

	outputWriter := io.NewOffsetWriter(newLimitWriterAt(writer, int64(meta.UncompressedSize)), 0)

	_, copyErr := io.Copy(outputWriter, decoder)
	if copyErr != nil {
		cancel()
	}

	loadErr := <-loadErrCh

	err = errors.Join(loadErr, copyErr)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

type limitWriterAt struct {
	w   io.WriterAt
	max int64
}

func newLimitWriterAt(w io.WriterAt, max int64) *limitWriterAt {
	return &limitWriterAt{w: w, max: max}
}

func (l *limitWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off+int64(len(p)) > l.max {
		return 0, fmt.Errorf("decompressed size exceeds expected %d bytes", l.max)
	}
	return l.w.WriteAt(p, off)
}
