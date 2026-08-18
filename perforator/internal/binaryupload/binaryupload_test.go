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
	blobmodels "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type fakeCommiter struct {
	committed bool
	aborted   bool
}

func (c *fakeCommiter) Commit(context.Context, *storage.BlobInfo) error {
	c.committed = true
	return nil
}
func (c *fakeCommiter) Ping(context.Context) error  { return nil }
func (c *fakeCommiter) Abort(context.Context) error { c.aborted = true; return nil }

type fakeMetaStorage struct {
	metas     []*binarymeta.BinaryMeta
	storeErr  error
	getErr    error
	getCalls  int
	commiters map[string]*fakeCommiter
}

func (f *fakeMetaStorage) StoreBinary(_ context.Context, buildID string, _ time.Time, _ ...binarymeta.Option) (binarymeta.Commiter, error) {
	if f.storeErr != nil {
		return nil, f.storeErr
	}
	c := &fakeCommiter{}
	if f.commiters == nil {
		f.commiters = map[string]*fakeCommiter{}
	}
	f.commiters[buildID] = c
	return c, nil
}

func (f *fakeMetaStorage) GetBinaries(context.Context, []string) ([]*binarymeta.BinaryMeta, error) {
	f.getCalls++
	return f.metas, f.getErr
}

func (f *fakeMetaStorage) CollectExpiredBinaries(context.Context, time.Duration, *util.Pagination) ([]*binarymeta.BinaryMeta, error) {
	return nil, nil
}

func (f *fakeMetaStorage) RemoveBinaries(context.Context, []string) error { return nil }

type fakeBlobWriter struct {
	key string
	buf bytes.Buffer
}

func (w *fakeBlobWriter) Write(p []byte) (int, error) { return w.buf.Write(p) }
func (w *fakeBlobWriter) Commit() (string, error)     { return w.key, nil }

type fakeBlobStorage struct {
	writers map[string]*fakeBlobWriter
}

func (f *fakeBlobStorage) Put(_ context.Context, key string) (blobmodels.Writer, error) {
	w := &fakeBlobWriter{key: key}
	if f.writers == nil {
		f.writers = map[string]*fakeBlobWriter{}
	}
	f.writers[key] = w
	return w, nil
}

func (f *fakeBlobStorage) Get(context.Context, string) (io.ReadCloser, error) { panic("unused") }

func (f *fakeBlobStorage) Size(context.Context, string) (uint64, error) { panic("unused") }

func (f *fakeBlobStorage) Delete(context.Context, string) error { return nil }

func (f *fakeBlobStorage) DeleteObjects(context.Context, []string) error { return nil }

func (f *fakeBlobStorage) List(context.Context, *blobmodels.Pagination, *storage.ShardParams) ([]string, error) {
	panic("unused")
}

type fakeStorage struct {
	meta  *fakeMetaStorage
	blobs *fakeBlobStorage
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{meta: &fakeMetaStorage{}, blobs: &fakeBlobStorage{}}
}

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

func newTestService(t *testing.T, fake *fakeStorage, opts Options) *Service {
	st := binarystorage.NewStorage(
		fake.meta,
		fake.blobs,
		xlog.ForTest(t),
		metricsmock.NewRegistry(nil),
	)
	return NewService(xlog.ForTest(t), metricsmock.NewRegistry(nil), st, opts)
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
	fake := newFakeStorage()

	stream := &fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_Zstd, 100),
		bodyChunk([]byte("hello ")),
		bodyChunk([]byte("world")),
	}}
	svc := newTestService(t, fake, Options{})
	require.NoError(t, svc.PushBinary(stream))
	require.NotNil(t, stream.response)

	c := fake.meta.commiters["abc"]
	require.NotNil(t, c)
	require.True(t, c.committed)
	require.False(t, c.aborted)
	require.Equal(t, "hello world", fake.blobs.writers["abc"].buf.String())
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
			err := newTestService(t, newFakeStorage(), Options{}).PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{tc.head}})
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestPushBinary_AlreadyUploaded(t *testing.T) {
	fake := newFakeStorage()
	fake.meta.storeErr = binarymeta.ErrAlreadyUploaded

	err := newTestService(t, fake, Options{}).PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestPushBinary_WriteErrorAborts(t *testing.T) {
	fake := newFakeStorage()

	stream := &fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
		headChunk("abc", compressionpb.CompressionMethod_None, 0), // head instead of body
	}}
	err := newTestService(t, fake, Options{}).PushBinary(stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	c := fake.meta.commiters["abc"]
	require.NotNil(t, c)
	require.False(t, c.committed)
	require.True(t, c.aborted)
}

func TestAnnounceBinaries(t *testing.T) {
	fake := newFakeStorage()
	fake.meta.metas = []*binarymeta.BinaryMeta{
		{BuildID: "uploaded", Status: binarymeta.Uploaded, LastUsedTimestamp: time.Now()},
		{BuildID: "fresh-in-progress", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now()},
		{BuildID: "stale-in-progress", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now().Add(-time.Hour)},
	}

	svc := newTestService(t, fake, Options{})
	resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
		AvailableBuildIDs: []string{"uploaded", "fresh-in-progress", "stale-in-progress", "missing"},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"stale-in-progress", "missing"}, resp.UnknownBuildIDs)
}

func TestPushBinary_DenyWrites(t *testing.T) {
	svc := newTestService(t, newFakeStorage(), Options{DenyWrites: true})
	err := svc.PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestAnnounceBinaries_LookupError(t *testing.T) {
	fake := newFakeStorage()
	fake.meta.getErr = errors.New("pg down")

	svc := newTestService(t, fake, Options{})
	_, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
		AvailableBuildIDs: []string{"a", "b"},
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestAnnounceBinaries_KnownCache(t *testing.T) {
	fake := newFakeStorage()
	fake.meta.metas = []*binarymeta.BinaryMeta{
		{BuildID: "a", Status: binarymeta.Uploaded, LastUsedTimestamp: time.Now()},
	}
	svc := newTestService(t, fake, Options{KnownCacheTTL: time.Minute, KnownCacheSize: 100})

	for i := 0; i < 2; i++ {
		resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
			AvailableBuildIDs: []string{"a"},
		})
		require.NoError(t, err)
		require.Empty(t, resp.UnknownBuildIDs)
	}
	require.Equal(t, 1, fake.meta.getCalls) // second announce answered from the known cache
}

func TestPushBinary_ResponseFailureKeepsCommit(t *testing.T) {
	fake := newFakeStorage()
	svc := newTestService(t, fake, Options{})

	stream := &fakePushStream{
		reqs: []*perforatorstorage.PushBinaryRequest{
			headChunk("abc", compressionpb.CompressionMethod_None, 0),
			bodyChunk([]byte("data")),
		},
		closeErr: errors.New("stream torn down"),
	}
	require.Error(t, svc.PushBinary(stream))

	c := fake.meta.commiters["abc"]
	require.NotNil(t, c)
	require.True(t, c.committed)
	require.False(t, c.aborted) // the upload is stored; a response failure must not destroy it
}

func TestPushBinary_EmptyBodyRejected(t *testing.T) {
	fake := newFakeStorage()
	svc := newTestService(t, fake, Options{})

	err := svc.PushBinary(&fakePushStream{reqs: []*perforatorstorage.PushBinaryRequest{
		headChunk("abc", compressionpb.CompressionMethod_None, 0),
	}})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	c := fake.meta.commiters["abc"]
	require.NotNil(t, c)
	require.False(t, c.committed)
	require.True(t, c.aborted)
}

func TestAnnounceBinaries_FreshInProgressNotCached(t *testing.T) {
	fake := newFakeStorage()
	fake.meta.metas = []*binarymeta.BinaryMeta{
		{BuildID: "a", Status: binarymeta.InProgress, LastUsedTimestamp: time.Now()},
	}
	svc := newTestService(t, fake, Options{KnownCacheTTL: time.Minute, KnownCacheSize: 100})

	for i := 0; i < 2; i++ {
		resp, err := svc.AnnounceBinaries(context.Background(), &perforatorstorage.AnnounceBinariesRequest{
			AvailableBuildIDs: []string{"a"},
		})
		require.NoError(t, err)
		require.Empty(t, resp.UnknownBuildIDs)
	}
	require.Equal(t, 2, fake.meta.getCalls) // fresh in-progress must be re-checked, not cached
}
