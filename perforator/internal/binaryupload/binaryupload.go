// Package binaryupload implements the binary-upload subset of the
// PerforatorStorage service (AnnounceBinaries + PushBinary) over a binary
// storage. The same Service runs in the storage (for agents) and in the proxy
// (for uploads from outside the infrastructure).
package binaryupload

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/karlseguin/ccache/v3"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

const (
	defaultKnownCacheTTL  = 10 * time.Minute
	defaultKnownCacheSize = 100000

	abortTimeout = 6 * time.Second

	// staleUploadAge is how long an in-progress upload is trusted before its
	// binary is announced as unknown again (so a crashed upload can be retried).
	staleUploadAge = 5 * time.Minute
)

type Options struct {
	MaxConcurrentUploads int64
	// KnownCacheTTL/KnownCacheSize bound how long and how many buildIDs are
	// remembered as present without asking the meta storage.
	KnownCacheTTL  time.Duration
	KnownCacheSize int64
	DenyWrites     bool
}

type serviceMetrics struct {
	uploadsSucceeded metrics.Counter
	uploadsFailed    metrics.Counter
	uploadsRaced     metrics.Counter
	uploadsRejected  metrics.Counter
	uploadedBytes    metrics.Counter
	uploadTimer      metrics.Timer
	abortsSucceeded  metrics.Counter
	abortsFailed     metrics.Counter

	announcesSucceeded metrics.Counter
	announcesFailed    metrics.Counter
	announceCacheHits  metrics.Counter
	announceLookups    metrics.Counter
}

// Every sensor counts one unit: uploads.count and announces.count count
// requests, announce_cache_hits.count counts build ids. rejected_uploads.count
// is its own sensor because rejection happens before the artifact kind is
// known and must not change the label set of uploads.count.
func newServiceMetrics(reg metrics.Registry) serviceMetrics {
	uploads := func(status string) metrics.Counter {
		return reg.WithTags(map[string]string{"status": status}).Counter("uploads.count")
	}
	announces := func(status string) metrics.Counter {
		return reg.WithTags(map[string]string{"status": status}).Counter("announces.count")
	}
	return serviceMetrics{
		uploadsSucceeded: uploads("success"),
		uploadsFailed:    uploads("failed"),
		uploadsRaced:     uploads("raced"),
		uploadsRejected:  reg.Counter("rejected_uploads.count"),
		uploadedBytes:    reg.Counter("uploaded_bytes.count"),
		uploadTimer:      reg.Timer("upload_duration.timer"),
		abortsSucceeded:  reg.WithTags(map[string]string{"status": "success"}).Counter("aborts.count"),
		abortsFailed:     reg.WithTags(map[string]string{"status": "failed"}).Counter("aborts.count"),

		announcesSucceeded: announces("success"),
		announcesFailed:    announces("failed"),
		announceCacheHits:  reg.Counter("announce_cache_hits.count"),
		announceLookups:    reg.Counter("announce_lookups.count"),
	}
}

type Service struct {
	perforatorstorage.UnimplementedPerforatorStorageServer

	l       xlog.Logger
	storage binarystorage.Storage
	limiter *semaphore.Weighted
	known   *ccache.Cache[bool]
	opts    Options
	metrics serviceMetrics
}

func NewService(
	l xlog.Logger,
	reg metrics.Registry,
	storage binarystorage.Storage,
	opts Options,
) *Service {
	if opts.MaxConcurrentUploads <= 0 {
		opts.MaxConcurrentUploads = 1
	}
	if opts.KnownCacheTTL <= 0 {
		opts.KnownCacheTTL = defaultKnownCacheTTL
	}
	if opts.KnownCacheSize <= 0 {
		opts.KnownCacheSize = defaultKnownCacheSize
	}
	known := ccache.New(ccache.Configure[bool]().MaxSize(opts.KnownCacheSize))

	return &Service{
		l:       l.WithName("BinaryUpload"),
		storage: storage,
		limiter: semaphore.NewWeighted(opts.MaxConcurrentUploads),
		known:   known,
		opts:    opts,
		metrics: newServiceMetrics(reg.WithPrefix("binary_upload")),
	}
}

func (s *Service) markKnown(buildID string) {
	s.known.Set(buildID, true, s.opts.KnownCacheTTL)
}

func (s *Service) isKnown(buildID string) bool {
	item := s.known.Get(buildID)
	return item != nil && !item.Expired()
}

func (s *Service) unknownBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	binaries, err := s.storage.GetBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(binaries))
	for _, binary := range binaries {
		if binary.Status == binarymeta.InProgress {
			if time.Since(binary.LastUsedTimestamp) > staleUploadAge {
				continue
			}
			// Fresh in-progress uploads must stay uncached so they are
			// re-checked once staleUploadAge passes.
			known[binary.BuildID] = true
			continue
		}
		known[binary.BuildID] = true
		s.markKnown(binary.BuildID)
	}

	unknown := make([]string, 0, len(buildIDs)-len(known))
	for _, buildID := range buildIDs {
		if !known[buildID] {
			unknown = append(unknown, buildID)
		}
	}
	return unknown, nil
}

func (s *Service) AnnounceBinaries(
	ctx context.Context,
	req *perforatorstorage.AnnounceBinariesRequest,
) (*perforatorstorage.AnnounceBinariesResponse, error) {
	var lookup []string
	for _, buildID := range req.GetAvailableBuildIDs() {
		if !s.isKnown(buildID) {
			lookup = append(lookup, buildID)
		}
	}
	s.metrics.announceCacheHits.Add(int64(len(req.GetAvailableBuildIDs()) - len(lookup)))

	var unknown []string
	if len(lookup) > 0 {
		s.metrics.announceLookups.Inc()
		var err error
		unknown, err = s.unknownBinaries(ctx, lookup)
		if err != nil {
			s.metrics.announcesFailed.Inc()
			s.l.Error(ctx, "Failed to look up announced binaries", log.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to look up binaries: %v", err)
		}
	}

	s.metrics.announcesSucceeded.Inc()
	return &perforatorstorage.AnnounceBinariesResponse{UnknownBuildIDs: unknown}, nil
}

func (s *Service) PushBinary(stream perforatorstorage.PerforatorStorage_PushBinaryServer) error {
	if s.opts.DenyWrites {
		s.metrics.uploadsRejected.Inc()
		return status.Error(codes.FailedPrecondition, "this replica is not allowed to upload binaries")
	}
	if !s.limiter.TryAcquire(1) {
		s.metrics.uploadsRejected.Inc()
		return status.Error(codes.ResourceExhausted, "too many concurrent binary uploads")
	}
	defer s.limiter.Release(1)

	return s.push(stream)
}

func isCompressed(m compressionpb.CompressionMethod) bool {
	return m != compressionpb.CompressionMethod_None && m != compressionpb.CompressionMethod_Unknown
}

func receiveHead(stream perforatorstorage.PerforatorStorage_PushBinaryServer) (*perforatorstorage.PushBinaryRequestHead, error) {
	first, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	head := first.GetHeadChunk()
	switch {
	case head == nil:
		return nil, status.Error(codes.InvalidArgument, "first chunk must be a head chunk")
	case head.GetBuildID() == "":
		return nil, status.Error(codes.InvalidArgument, "build id is missing")
	case isCompressed(head.GetCompression()) && head.GetUncompressedSize() == 0:
		return nil, status.Errorf(codes.InvalidArgument, "uncompressed size is required when compression is %s", head.GetCompression())
	}
	return head, nil
}

func copyBody(writer binarystorage.TransactionalWriter, stream perforatorstorage.PerforatorStorage_PushBinaryServer) (uint64, error) {
	var bytesTransmitted uint64
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return bytesTransmitted, nil
		}
		if err != nil {
			return bytesTransmitted, err
		}

		body := chunk.GetBodyChunk()
		if body == nil {
			return bytesTransmitted, status.Error(codes.InvalidArgument, "chunks after the first must be body chunks")
		}

		written, err := writer.Write(body.GetBinary())
		if err != nil {
			return bytesTransmitted, status.Errorf(codes.Internal, "failed to write chunk: %v", err)
		}
		bytesTransmitted += uint64(written)
	}
}

// push aborts the pending upload on any error before the commit; a response
// failure after the commit leaves the binary stored.
func (s *Service) push(stream perforatorstorage.PerforatorStorage_PushBinaryServer) (err error) {
	ctx := stream.Context()

	start := time.Now()
	committed := false
	var buildID string
	var compression compressionpb.CompressionMethod
	var bytes uint64
	defer func() {
		switch {
		case committed:
			s.markKnown(buildID)
			s.metrics.uploadsSucceeded.Inc()
			s.metrics.uploadedBytes.Add(int64(bytes))
			s.metrics.uploadTimer.RecordDuration(time.Since(start))
			s.l.Info(ctx, "Uploaded binary",
				log.String("build_id", buildID),
				log.String("compression", compression.String()),
				log.UInt64("bytes", bytes),
			)
		case status.Code(err) == codes.AlreadyExists || status.Code(err) == codes.Aborted:
			s.metrics.uploadsRaced.Inc()
		default:
			s.metrics.uploadsFailed.Inc()
		}
		if err != nil {
			s.l.Warn(ctx, "Failed to upload binary", log.String("build_id", buildID), log.Error(err))
		}
	}()

	head, err := receiveHead(stream)
	if err != nil {
		return err
	}
	buildID, compression = head.GetBuildID(), head.GetCompression()

	var opts []binarymeta.Option
	if isCompressed(compression) {
		opts = append(opts, binarymeta.WithCompression(compression, head.GetUncompressedSize()))
	}

	writer, err := s.storage.StoreBinary(ctx, buildID, start, opts...)
	if err != nil {
		switch {
		case errors.Is(err, binarymeta.ErrAlreadyUploaded):
			return status.Errorf(codes.AlreadyExists, "binary %s is already uploaded", buildID)
		case errors.Is(err, binarymeta.ErrUploadInProgress):
			return status.Errorf(codes.Aborted, "binary %s upload is already in progress", buildID)
		}
		return status.Errorf(codes.Internal, "failed to start binary upload: %v", err)
	}
	defer func() {
		if err == nil || committed {
			return
		}
		abortCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), abortTimeout)
		defer cancel()
		if abortErr := writer.Abort(abortCtx); abortErr != nil {
			s.metrics.abortsFailed.Inc()
			err = errors.Join(err, abortErr)
		} else {
			s.metrics.abortsSucceeded.Inc()
		}
	}()

	bytes, err = copyBody(writer, stream)
	if err != nil {
		return err
	}
	if bytes == 0 {
		return status.Error(codes.InvalidArgument, "no binary data received")
	}

	if err = writer.Commit(ctx); err != nil {
		return status.Errorf(codes.Internal, "failed to commit binary: %v", err)
	}
	committed = true

	return stream.SendAndClose(&perforatorstorage.PushBinaryResponse{})
}
