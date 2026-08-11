package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/require"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

// fakeS3 serves ranged GetObject requests from an in-memory blob, optionally
// interrupting the first response bodies to exercise the downloader's retry.
type fakeS3 struct {
	s3iface.S3API

	blob          []byte
	interruptions int
	invalidRange  bool
	omitETag      bool
	shiftRange    int64
	unrangedCalls int
	headSize      *int64

	mu          sync.Mutex
	calls       []string
	inflight    int
	maxInflight int
}

type flakyBody struct {
	io.Reader
	fail bool
}

func (b *flakyBody) Read(p []byte) (int, error) {
	if b.fail {
		half := len(p) / 2
		if half == 0 {
			half = 1
		}
		n, _ := b.Reader.Read(p[:half])
		if n > 0 {
			b.fail = false
			return n, fmt.Errorf("connection reset mid-body")
		}
	}
	return b.Reader.Read(p)
}

func (b *flakyBody) Close() error { return nil }

// MaxRetries feeds the downloader's partBodyMaxRetries; the real client
// implements it via its retryer.
func (f *fakeS3) MaxRetries() int { return 3 }

func (f *fakeS3) HeadObjectWithContext(_ aws.Context, _ *awss3.HeadObjectInput, _ ...request.Option) (*awss3.HeadObjectOutput, error) {
	size := int64(len(f.blob))
	if f.headSize != nil {
		size = *f.headSize
	}
	return &awss3.HeadObjectOutput{ContentLength: aws.Int64(size)}, nil
}

func (f *fakeS3) GetObjectWithContext(_ aws.Context, in *awss3.GetObjectInput, _ ...request.Option) (*awss3.GetObjectOutput, error) {
	var etag *string
	if !f.omitETag {
		etag = aws.String("test-etag")
	}
	if f.invalidRange && aws.StringValue(in.Range) != "" {
		return nil, awserr.New("InvalidRange", "requested range not satisfiable", nil)
	}
	f.mu.Lock()
	f.calls = append(f.calls, aws.StringValue(in.Range))
	f.inflight++
	if f.inflight > f.maxInflight {
		f.maxInflight = f.inflight
	}
	interrupt := f.interruptions > 0
	if interrupt {
		f.interruptions--
	}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.inflight--
		f.mu.Unlock()
	}()

	if aws.StringValue(in.Range) == "" {
		f.unrangedCalls++
		return &awss3.GetObjectOutput{
			Body:          &flakyBody{Reader: strings.NewReader(string(f.blob))},
			ContentLength: aws.Int64(int64(len(f.blob))),
		}, nil
	}
	var start, end int64
	_, err := fmt.Sscanf(aws.StringValue(in.Range), "bytes=%d-%d", &start, &end)
	if err != nil {
		return nil, fmt.Errorf("unexpected range %q", aws.StringValue(in.Range))
	}
	end = min(end, int64(len(f.blob))-1)
	body := f.blob[start : end+1]

	return &awss3.GetObjectOutput{
		Body:          &flakyBody{Reader: strings.NewReader(string(body)), fail: interrupt},
		ContentLength: aws.Int64(int64(len(body))),
		ContentRange:  aws.String(fmt.Sprintf("bytes %d-%d/%d", start+f.shiftRange, end+f.shiftRange, len(f.blob))),
		ETag:          etag,
	}, nil
}

func newTestS3Storage(fake *fakeS3) *S3Storage {
	reg := metricsmock.NewRegistry(nil)
	return &S3Storage{
		bucket: "test",
		client: fake,
		l:      xlog.NewNop(),
		downloader: s3manager.NewDownloaderWithClient(fake, func(d *s3manager.Downloader) {
			d.Concurrency = 1
		}),
		downloadCfg: defaultParallelDownloadConfig(),
		metrics: &mdsStorageMetrics{
			bytesDownloaded:   reg.Counter("downloaded.bytes"),
			bytesUploaded:     reg.Counter("uploaded.bytes"),
			degradedDownloads: reg.CounterVec("degraded_downloads.count", []string{"reason"}),
		},
	}
}

// TestFetchPart_SingleRangedCall pins that an explicit Range keeps the SDK
// downloader to exactly one call for exactly the requested range — no derived
// subranges, no internal fan-out.
func TestFetchPart_SingleRangedCall(t *testing.T) {
	fake := &fakeS3{blob: make([]byte, 1<<16)}
	s := newTestS3Storage(fake)

	part, err := s.fetchPart(context.Background(), "k", nil, 1024, 2047)
	require.NoError(t, err)
	require.Len(t, part, 1024)
	require.Equal(t, []string{"bytes=1024-2047"}, fake.calls)
	require.Equal(t, 1, fake.maxInflight)
}

// TestFetchPart_RetriesInterruptedBody pins the property the custom reader
// relies on: a body interrupted mid-read is retried by the SDK with the same
// range, not surfaced as a short part.
func TestFetchPart_RetriesInterruptedBody(t *testing.T) {
	blob := make([]byte, 8192)
	for i := range blob {
		blob[i] = byte(i % 251)
	}
	fake := &fakeS3{blob: blob, interruptions: 1}
	s := newTestS3Storage(fake)

	part, err := s.fetchPart(context.Background(), "k", nil, 0, 8191)
	require.NoError(t, err)
	require.Equal(t, blob, part)
	require.GreaterOrEqual(t, len(fake.calls), 2)
	for _, r := range fake.calls {
		require.Equal(t, "bytes=0-8191", r)
	}
}

// TestGet_InvalidRangeEmptyBlob pins the InvalidRange fallback: an empty
// object yields an empty stream through the sequential path.
func TestGet_InvalidRangeEmptyBlob(t *testing.T) {
	fake := &fakeS3{blob: nil, invalidRange: true}
	s := newTestS3Storage(fake)

	r, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Empty(t, data)
}

// TestGet_InvalidRangeFallsBackSequential pins that an unexpected
// InvalidRange degrades to a full sequential download instead of failing or
// faking emptiness.
func TestGet_InvalidRangeFallsBackSequential(t *testing.T) {
	blob := []byte("sequential fallback data")
	fake := &fakeS3{blob: blob, invalidRange: true}
	s := newTestS3Storage(fake)

	r, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, blob, data)
}

// TestFetchPart_ShortRangeFails pins the length validation: a server that
// ignores part of the range must not produce a silently short part.
func TestFetchPart_ShortRangeFails(t *testing.T) {
	fake := &fakeS3{blob: make([]byte, 100)}
	s := newTestS3Storage(fake)

	_, err := s.fetchPart(context.Background(), "k", nil, 0, 999)
	require.ErrorContains(t, err, "expected 1000 bytes")
}

// TestGet_NoETagFallsBackSequential pins that without an ETag the parallel
// path is not used: parts could not be pinned to one object version.
func TestGet_NoETagFallsBackSequential(t *testing.T) {
	blob := bytes.Repeat([]byte("v"), 3*1024)
	fake := &fakeS3{blob: blob, omitETag: true}
	s := newTestS3Storage(fake)
	s.downloadCfg.PartSize = 1024

	r, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, blob, data)
	require.Equal(t, 1, fake.unrangedCalls)
}

// TestGet_WrongIntervalFallsBackSequential pins that a response answering a
// different interval than requested is not spliced in as part zero.
func TestGet_WrongIntervalFallsBackSequential(t *testing.T) {
	blob := bytes.Repeat([]byte("w"), 3*1024)
	fake := &fakeS3{blob: blob, shiftRange: 1}
	s := newTestS3Storage(fake)
	s.downloadCfg.PartSize = 1024

	r, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, blob, data)
	require.Equal(t, 1, fake.unrangedCalls)
}
