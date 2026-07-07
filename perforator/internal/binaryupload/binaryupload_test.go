package binaryupload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type fakeBinaryWriter struct {
	buf       bytes.Buffer
	committed bool
	aborted   bool
}

func (w *fakeBinaryWriter) Write(p []byte) (int, error)     { return w.buf.Write(p) }
func (w *fakeBinaryWriter) Commit(context.Context) error    { w.committed = true; return nil }
func (w *fakeBinaryWriter) Abort(ctx context.Context) error { w.aborted = true; return nil }

type fakeBinaryStorage struct {
	metas    []*binarymeta.BinaryMeta
	storeErr error
	getErr   error
	getCalls int
	writers  map[string]*fakeBinaryWriter
}

func (f *fakeBinaryStorage) StoreBinary(_ context.Context, buildID string, _ time.Time, _ ...binarymeta.Option) (binarystorage.TransactionalWriter, error) {
	if f.storeErr != nil {
		return nil, f.storeErr
	}
	w := &fakeBinaryWriter{}
	if f.writers == nil {
		f.writers = map[string]*fakeBinaryWriter{}
	}
	f.writers[buildID] = w
	return w, nil
}

func (f *fakeBinaryStorage) LoadBinary(context.Context, string, io.WriterAt) (*binarymeta.BinaryMeta, error) {
	panic("unused")
}

func (f *fakeBinaryStorage) GetBinaries(context.Context, []string) ([]*binarymeta.BinaryMeta, error) {
	f.getCalls++
	return f.metas, f.getErr
}

func (f *fakeBinaryStorage) CollectExpired(context.Context, time.Duration, *util.Pagination, *storage.ShardParams) ([]*storage.ObjectMeta, error) {
	return nil, nil
}

func (f *fakeBinaryStorage) Delete(context.Context, []string) error { return nil }

type fakePushStream struct {
	grpc.ServerStream
	reqs     []*perforatorstorage.PushBinaryRequest
	response *perforatorstorage.PushBinaryResponse
	closeErr error
}

func (s *fakePushStream) Context() context.Context { return context.Background() }

func (s *fakePushStream) Recv() (*perforatorstorage.PushBinaryRequest, error) {
	if len(s.reqs) == 0 {
		return nil, io.EOF
	}
	req := s.reqs[0]
	s.reqs = s.reqs[1:]
	return req, nil
}

func (s *fakePushStream) SendAndClose(resp *perforatorstorage.PushBinaryResponse) error {
	if s.closeErr != nil {
		return s.closeErr
	}
	s.response = resp
	return nil
}

func newTestService(t *testing.T, fake *fakeBinaryStorage, opts Options) *Service {
	return NewService(xlog.ForTest(t), metricsmock.NewRegistry(nil), fake, opts)
}

func headChunk(buildID string, compression compressionpb.CompressionMethod, uncompressedSize uint64) *perforatorstorage.PushBinaryRequest {
	return &perforatorstorage.PushBinaryRequest{
		Chunk: &perforatorstorage.PushBinaryRequest_HeadChunk{
			HeadChunk: &perforatorstorage.PushBinaryRequestHead{
				BuildID:          buildID,
				Compression:      compression,
				UncompressedSize: uncompressedSize,
			},
		},
	}
}

func bodyChunk(data []byte) *perforatorstorage.PushBinaryRequest {
	return &perforatorstorage.PushBinaryRequest{
		Chunk: &perforatorstorage.PushBinaryRequest_BodyChunk{
			BodyChunk: &perforatorstorage.PushBinaryRequestBody{Binary: data},
		},
	}
}

func TestPushBinary_Simple(t *testing.T) {
	fake := &fakeBinaryStorage{}

	stream := &fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_Zstd, 100),
		bodyChunk([]byte("hello ")),
		bodyChunk([]byte("world")),
	}}
	svc := newTestService(t, fake, Options{})
	require.NoError(t, svc.PushBinary(stream))
	require.NotNil(t, stream.response)

	w := fake.writers["abc"]
	require.NotNil(t, w)
	require.True(t, w.committed)
	require.False(t, w.aborted)
	require.Equal(t, "hello world", w.buf.String())
}

func TestPushBinary_HeadValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		head *perforatorstorage.PushBinaryRequest
	}{
		{"missing build id", headChunk("", compressionpb.CompressionMethod_None, 0)},
		{"compressed without size", headChunk("abc", compressionpb.CompressionMethod_Zstd, 0)},
		{"body first", bodyChunk([]byte("data"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newTestService(t, &fakeBinaryStorage{}, Options{}).PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{tc.head}})
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestPushBinary_AlreadyUploaded(t *testing.T) {
	err := newTestService(t, &fakeBinaryStorage{storeErr: binarymeta.ErrAlreadyUploaded}, Options{}).PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestPushBinary_WriteErrorAborts(t *testing.T) {
	fake := &fakeBinaryStorage{}

	stream := &fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
		headChunk("abc", compressionpb.CompressionMethod_None, 0), // head instead of body
	}}
	err := newTestService(t, fake, Options{}).PushBinary(stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	w := fake.writers["abc"]
	require.NotNil(t, w)
	require.False(t, w.committed)
	require.True(t, w.aborted)
}

func TestAnnounceBinaries(t *testing.T) {
	fake := &fakeBinaryStorage{metas: []*binarymeta.BinaryMeta{
		{BuildID: "uploaded", Status: binarymeta.Uploaded, LastUsedTimestamp: time.Now()},
		{BuildID: "fresh-in-progress", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now()},
		{BuildID: "stale-in-progress", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now().Add(-time.Hour)},
	}}

	svc := newTestService(t, fake, Options{})
	resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
		AvailableBuildIDs: []string{"uploaded", "fresh-in-progress", "stale-in-progress", "missing"},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"stale-in-progress", "missing"}, resp.UnknownBuildIDs)
}

func TestPushBinary_DenyWrites(t *testing.T) {
	svc := newTestService(t, &fakeBinaryStorage{}, Options{DenyWrites: true})
	err := svc.PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestAnnounceBinaries_LookupError(t *testing.T) {
	fake := &fakeBinaryStorage{getErr: errors.New("pg down")}

	svc := newTestService(t, fake, Options{})
	_, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
		AvailableBuildIDs: []string{"a", "b"},
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestAnnounceBinaries_KnownCache(t *testing.T) {
	fake := &fakeBinaryStorage{metas: []*binarymeta.BinaryMeta{
		{BuildID: "a", Status: binarymeta.Uploaded, LastUsedTimestamp: time.Now()},
	}}
	svc := newTestService(t, fake, Options{KnownCacheTTL: time.Minute, KnownCacheSize: 100})

	for i := 0; i < 2; i++ {
		resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
			AvailableBuildIDs: []string{"a"},
		})
		require.NoError(t, err)
		require.Empty(t, resp.UnknownBuildIDs)
	}
	require.Equal(t, 1, fake.getCalls) // second announce answered from the known cache
}

func TestPushBinary_ResponseFailureKeepsCommit(t *testing.T) {
	fake := &fakeBinaryStorage{}
	svc := newTestService(t, fake, Options{})

	stream := &fakePushStream{
		reqs: []*perforatorstorage.PushBinaryRequest{
			headChunk("abc", compressionpb.CompressionMethod_None, 0),
			bodyChunk([]byte("data")),
		},
		closeErr: errors.New("stream torn down"),
	}
	require.Error(t, svc.PushBinary(stream))

	w := fake.writers["abc"]
	require.NotNil(t, w)
	require.True(t, w.committed)
	require.False(t, w.aborted) // the upload is stored; a response failure must not destroy it
}

func TestPushBinary_EmptyBodyRejected(t *testing.T) {
	fake := &fakeBinaryStorage{}
	svc := newTestService(t, fake, Options{})

	err := svc.PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	w := fake.writers["abc"]
	require.NotNil(t, w)
	require.False(t, w.committed)
	require.True(t, w.aborted)
}

func TestAnnounceBinaries_FreshInProgressNotCached(t *testing.T) {
	fake := &fakeBinaryStorage{metas: []*binarymeta.BinaryMeta{
		{BuildID: "a", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now()},
	}}
	svc := newTestService(t, fake, Options{KnownCacheTTL: time.Minute, KnownCacheSize: 100})

	for i := 0; i < 2; i++ {
		resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
			AvailableBuildIDs: []string{"a"},
		})
		require.NoError(t, err)
		require.Empty(t, resp.UnknownBuildIDs)
	}
	require.Equal(t, 2, fake.getCalls) // fresh in-progress must be re-checked, not cached
}
