package decompressing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	mock_binary "github.com/yandex/perforator/perforator/pkg/storage/binary/mock"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

type verifyWriterAt struct {
	mu     sync.Mutex
	chunks map[int64][]byte
}

func newVerifyWriterAt() *verifyWriterAt {
	return &verifyWriterAt{chunks: make(map[int64][]byte)}
}

func (w *verifyWriterAt) WriteAt(p []byte, off int64) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.chunks[off] = append([]byte{}, p...)
	return len(p), nil
}

func (w *verifyWriterAt) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.chunks) == 0 {
		return nil
	}
	var offsets []int64
	for off := range w.chunks {
		offsets = append(offsets, off)
	}
	sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })
	var result []byte
	for _, off := range offsets {
		result = append(result, w.chunks[off]...)
	}
	return result
}

type failWriterAt struct {
	err error
}

func (w *failWriterAt) WriteAt(_ []byte, _ int64) (int, error) {
	return 0, w.err
}

func compressZstd(data []byte) []byte {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil
	}
	_, _ = enc.Write(data)
	_ = enc.Close()
	return buf.Bytes()
}

func testDownloadConfig() models.ParallelDownloadConfig {
	return models.ParallelDownloadConfig{
		Concurrency: 20,
		PartSize:    s3manager.DefaultDownloadPartSize,
	}
}

func expectGetBinaries(
	mock *mock_binary.MockStorage,
	buildID string,
	metas []*binarymeta.BinaryMeta,
) {
	mock.EXPECT().
		GetBinaries(gomock.Any(), []string{buildID}).
		Return(metas, nil)
}

func expectLoadBinaryWriteAt(
	mock *mock_binary.MockStorage,
	buildID string,
	fn func(ctx context.Context, buildID string, w io.WriterAt) error,
) {
	mock.EXPECT().
		LoadBinary(gomock.Any(), buildID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, id string, w io.WriterAt) (*binarymeta.BinaryMeta, error) {
			if err := fn(ctx, id, w); err != nil {
				return nil, err
			}
			return &binarymeta.BinaryMeta{BuildID: id}, nil
		})
}

func expectLoadBinaryWrite(
	mock *mock_binary.MockStorage,
	buildID string,
	data []byte,
) {
	expectLoadBinaryWriteAt(mock, buildID, func(_ context.Context, _ string, w io.WriterAt) error {
		if len(data) == 0 {
			return nil
		}
		_, err := w.WriteAt(data, 0)
		return err
	})
}

func TestStorage_PassthroughUncompressed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build1"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:     buildID,
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 5},
		},
	})

	var calledWith io.WriterAt
	expectLoadBinaryWriteAt(mock, buildID, func(_ context.Context, _ string, w io.WriterAt) error {
		calledWith = w
		_, err := w.WriteAt([]byte("hello"), 0)
		return err
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	meta, err := s.LoadBinary(ctx, buildID, w)
	require.NoError(t, err)
	require.Equal(t, buildID, meta.BuildID)
	require.Equal(t, []byte("hello"), w.Bytes())
	require.NotNil(t, calledWith)
}

func TestStorage_ZstdSequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build2"

	original := []byte("hello world this is test data for zstd compression")
	compressed := compressZstd(original)

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: uint64(len(original)),
			BlobInfo:         &storage.BlobInfo{Size: uint64(len(compressed))},
		},
	})
	expectLoadBinaryWrite(mock, buildID, compressed)

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	meta, err := s.LoadBinary(ctx, buildID, w)
	require.NoError(t, err)
	require.Equal(t, buildID, meta.BuildID)
	require.Equal(t, original, w.Bytes())
}

func TestStorage_ZstdOutOfOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build3"

	ctx := t.Context()

	original := make([]byte, 1024*1024)
	for i := range original {
		original[i] = byte(i % 256)
	}
	compressed := compressZstd(original)
	partSize := 64 * 1024

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: uint64(len(original)),
			BlobInfo:         &storage.BlobInfo{Size: uint64(len(compressed))},
		},
	})
	expectLoadBinaryWriteAt(mock, buildID, func(_ context.Context, _ string, w io.WriterAt) error {
		type chunk struct {
			off  int
			data []byte
		}

		var chunks []chunk
		for off := 0; off < len(compressed); off += partSize {
			end := off + partSize
			if end > len(compressed) {
				end = len(compressed)
			}
			chunks = append(chunks, chunk{
				off:  off,
				data: compressed[off:end],
			})
		}

		// Simulate s3manager: parts are requested in order, but each
		// download completes after its own network delay, so WriteAt
		// calls may finish out of order.
		g := errgroup.Group{}
		for i, chunk := range chunks {
			chunk := chunk
			// Later parts get shorter delays so completion order differs from offset order.
			delay := time.Duration(len(chunks)-i) * time.Millisecond
			g.Go(func() error {
				time.Sleep(delay)
				_, err := w.WriteAt(chunk.data, int64(chunk.off))
				return err
			})
		}
		return g.Wait()
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	meta, err := s.LoadBinary(ctx, buildID, w)
	require.NoError(t, err)
	require.Equal(t, buildID, meta.BuildID)
	require.Equal(t, original, w.Bytes())
}

func TestStorage_InvalidDownloadConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mock_binary.NewMockStorage(ctrl)

	_, err := New(mock, models.ParallelDownloadConfig{})
	require.ErrorIs(t, err, ErrInvalidDownloadConfig)
}

func TestStorage_DecompressedExceedsLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build6"

	original := []byte("short")
	compressed := compressZstd(original)

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: 3,
			BlobInfo:         &storage.BlobInfo{Size: uint64(len(compressed))},
		},
	})
	expectLoadBinaryWrite(mock, buildID, compressed)

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decompressed size exceeds expected")
	require.LessOrEqual(t, len(w.Bytes()), 3)
}

func TestStorage_ZstdUncompressedSizeZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build7"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: 0,
			BlobInfo:         &storage.BlobInfo{Size: 100},
		},
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.Contains(t, err.Error(), "uncompressed_size is zero")
}

func TestStorage_UnsupportedCompression(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build8"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:     buildID,
			Compression: compressionpb.CompressionMethod_Gzip,
			BlobInfo:    &storage.BlobInfo{Size: 100},
		},
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported compression method")
}

func TestStorage_LoadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build9"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:     buildID,
			Compression: compressionpb.CompressionMethod_None,
			BlobInfo:    &storage.BlobInfo{Size: 10},
		},
	})
	mock.EXPECT().
		LoadBinary(gomock.Any(), buildID, gomock.Any()).
		Return(nil, errors.New("inner load failed"))

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.Contains(t, err.Error(), "inner load failed")
}

func TestStorage_UnknownCompressionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build10"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:     buildID,
			Compression: compressionpb.CompressionMethod_Unknown,
			BlobInfo:    &storage.BlobInfo{Size: 5},
		},
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.Contains(t, err.Error(), "compression is unknown")
}

func TestStorage_ZstdReaderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build-zstd-reader-error"

	compressedPayload := []byte("not a zstd stream")

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: 100,
			BlobInfo:         &storage.BlobInfo{Size: uint64(len(compressedPayload))},
		},
	})

	var innerCtx context.Context
	expectLoadBinaryWriteAt(mock, buildID, func(loadCtx context.Context, _ string, w io.WriterAt) error {
		innerCtx = loadCtx
		_, err := w.WriteAt(compressedPayload, 0)
		return err
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	w := newVerifyWriterAt()
	_, err = s.LoadBinary(ctx, buildID, w)
	require.Error(t, err)
	require.NotNil(t, innerCtx)
	require.ErrorIs(t, innerCtx.Err(), context.Canceled)
}

func TestStorage_WriterError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build-writer-error"

	original := []byte("hello world this is test data for zstd compression")
	compressed := compressZstd(original)
	writerErr := errors.New("target writer failed")

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: uint64(len(original)),
			BlobInfo:         &storage.BlobInfo{Size: uint64(len(compressed))},
		},
	})

	var innerCtx context.Context
	expectLoadBinaryWriteAt(mock, buildID, func(loadCtx context.Context, _ string, w io.WriterAt) error {
		innerCtx = loadCtx
		_, err := w.WriteAt(compressed, 0)
		return err
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	_, err = s.LoadBinary(ctx, buildID, &failWriterAt{err: writerErr})
	require.Error(t, err)
	require.ErrorIs(t, err, writerErr)
	require.NotNil(t, innerCtx)
	require.ErrorIs(t, innerCtx.Err(), context.Canceled)
}

func TestStorage_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mock_binary.NewMockStorage(ctrl)
	buildID := "build-context-canceled"

	expectGetBinaries(mock, buildID, []*binarymeta.BinaryMeta{
		{
			BuildID:          buildID,
			Compression:      compressionpb.CompressionMethod_Zstd,
			UncompressedSize: 100,
			BlobInfo:         &storage.BlobInfo{Size: 100},
		},
	})

	loadStarted := make(chan struct{})
	expectLoadBinaryWriteAt(mock, buildID, func(loadCtx context.Context, _ string, _ io.WriterAt) error {
		close(loadStarted)
		<-loadCtx.Done()
		return loadCtx.Err()
	})

	s, err := New(mock, testDownloadConfig())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := s.LoadBinary(ctx, buildID, newVerifyWriterAt())
		errCh <- err
	}()

	<-loadStarted
	cancel()

	err = <-errCh
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}
