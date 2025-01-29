package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/karlseguin/ccache/v3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	profilemeta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	storagetvm "github.com/yandex/perforator/perforator/pkg/storage/tvm"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/pprofprofile"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
	"github.com/yandex/perforator/perforator/util/go/tsformat"
)

const (
	cacheItemTTL = 10 * time.Minute
)

type storageMetrics struct {
	droppedProfiles      metrics.Counter
	microscopedProfiles  metrics.Counter
	storedProfiles       metrics.Counter
	storedProfilesErrors metrics.Counter
	profilesBytesCount   metrics.Counter
	profilesBytesSizes   metrics.Histogram

	pushProfileInProgress   metrics.IntGauge
	successPushProfileTimer metrics.Timer
	failPushProfileTimer    metrics.Timer

	storedBinaries       metrics.Counter
	storedBinariesErrors metrics.Counter
	droppedBinaryUploads metrics.Counter
	binariesBytesCount   metrics.Counter
	binariesUploadTimer  metrics.Timer

	failedAbortBinariesUploads  metrics.Counter
	successAbortBinariesUploads metrics.Counter

	successAnnounceBinaries   metrics.Counter
	failedAnnounceBinaries    metrics.Counter
	announceBinariesCacheHit  metrics.Counter
	announceBinariesCacheMiss metrics.Counter
}

type StorageOptions struct {
	ClusterName            string
	SamplingModulo         uint64
	MaxBuildIDCacheEntries uint64
	PushProfileTimeout     time.Duration
	PushBinaryWriteAbility bool
}

type StorageServer struct {
	conf       *Config
	opts       *StorageOptions
	reg        xmetrics.Registry
	grpcServer *grpc.Server
	metrics    *storageMetrics
	logger     xlog.Logger

	binaryUploadLimiter *semaphore.Weighted
	profileSampler      Sampler

	profileStorage profilestorage.Storage
	binaryStorage  binarystorage.Storage

	microscopeFilter *filter.PullingFilter

	buildIDCache *ccache.Cache[bool]

	profileCommentProcessors map[string]func(string, *profilemeta.ProfileMetadata) error
}

func (s *StorageServer) initProfileCommentProcessors() {
	s.profileCommentProcessors[profilestorage.ServiceLabel] = func(value string, metadata *profilemeta.ProfileMetadata) error {
		metadata.Service = value
		return nil
	}
	s.profileCommentProcessors[profilestorage.TimestampLabel] = func(value string, metadata *profilemeta.ProfileMetadata) error {
		ts, err := time.Parse(tsformat.TimestampStringFormat, value)
		if err != nil {
			return err
		}
		metadata.Timestamp = ts
		return nil
	}
}

func (s *StorageServer) createProfileMetaFromLabels(ctx context.Context, labels map[string]string) (*profilemeta.ProfileMetadata, error) {
	result := profilemeta.ProfileMetadata{
		Attributes: make(map[string]string),
	}

	for k, v := range labels {
		processor, present := s.profileCommentProcessors[k]
		if !present {
			result.Attributes[k] = v
			continue
		}

		err := processor(v, &result)
		if err != nil {
			s.logger.Warn(ctx,
				"Failed to process profile label",
				log.String("key", k),
				log.String("value", v),
				log.Error(err),
			)
		}
	}

	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	return &result, nil
}

func (s *StorageServer) getMetadataFromProfile(ctx context.Context, profile *pprofprofile.Profile) (*profilemeta.ProfileMetadata, error) {
	labels := map[string]string{}

	for _, strID := range profile.Comment {
		parts := bytes.SplitN(profile.StringTable[strID], []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}

		labels[string(parts[0])] = string(parts[1])
	}

	return s.createProfileMetaFromLabels(ctx, labels)
}

func (s *StorageServer) extractProfileBytesMeta(
	ctx context.Context,
	req *perforatorstorage.PushProfileRequest,
) (body []byte, meta *profilemeta.ProfileMetadata, err error) {
	switch req.ProfileRepresentation.(type) {
	case *perforatorstorage.PushProfileRequest_ProfileBytes:
		meta, err = s.createProfileMetaFromLabels(ctx, req.GetLabels())
		if err != nil {
			return
		}

		body = req.GetProfileBytes()

	case *perforatorstorage.PushProfileRequest_Profile:
		meta, err = s.getMetadataFromProfile(ctx, req.GetProfile())
		if err != nil {
			return
		}

		body, err = proto.Marshal(req.GetProfile())
		if err != nil {
			return
		}

	default:
		return nil, nil, errors.New("request does not contain profile")
	}

	meta.BuildIDs = slices.Clone(req.GetBuildIDs())
	meta.Envs = slices.Clone(req.GetEnvs())
	return
}

func (s *StorageServer) fixupMissingMetadataFields(meta *profilemeta.ProfileMetadata) {
	if meta.System == "" {
		meta.System = "perforator"
	}
	if meta.Cluster == "" {
		if s.opts.ClusterName != "" {
			meta.Cluster = s.opts.ClusterName
		} else {
			meta.Cluster = meta.Attributes["cluster"]
		}
	}
	if meta.NodeID == "" {
		meta.NodeID = meta.Attributes["host"]
	}
	if meta.PodID == "" {
		meta.PodID = meta.Attributes["pod"]
	}
}

type PushProfileAdmitResult int

const (
	NotAllowed PushProfileAdmitResult = iota
	PassedSampling
	PassedMicroscopes
)

func (s *StorageServer) admitPushProfile(meta *profilemeta.ProfileMetadata) (PushProfileAdmitResult, error) {
	if s.profileSampler.Sample() {
		return PassedSampling, nil
	}

	if s.microscopeFilter != nil && s.microscopeFilter.Filter(meta) {
		return PassedMicroscopes, nil
	}

	return NotAllowed, nil
}

func fixupEventTypes(eventTypes []string) []string {
	if len(eventTypes) == 0 {
		return []string{sampletype.SampleTypeCPUCycles}
	}

	return eventTypes
}

func createMetasWithEventType(commonMeta *profilemeta.ProfileMetadata, eventTypes []string) []*profilemeta.ProfileMetadata {
	metas := make([]*profilemeta.ProfileMetadata, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		newMeta := *commonMeta
		newMeta.MainEventType = eventType
		newMeta.AllEventTypes = eventTypes
		metas = append(metas, &newMeta)
	}

	return metas
}

// implements storage.PerforatorStorage/PushProfile
func (s *StorageServer) PushProfile(ctx context.Context, req *perforatorstorage.PushProfileRequest) (*perforatorstorage.PushProfileResponse, error) {
	s.metrics.pushProfileInProgress.Add(1)
	defer func() {
		s.metrics.pushProfileInProgress.Add(-1)
	}()

	l := s.logger.With(log.Any("labels", req.Labels))

	ts := time.Now()
	var err error
	defer func() {
		if err != nil {
			s.metrics.failPushProfileTimer.RecordDuration(time.Since(ts))
		} else {
			s.metrics.successPushProfileTimer.RecordDuration(time.Since(ts))
		}
	}()

	if req.GetProfileRepresentation() == nil {
		return nil, errors.New("missing profile field")
	}

	body, meta, err := s.extractProfileBytesMeta(ctx, req)
	if err != nil {
		return nil, err
	}
	s.fixupMissingMetadataFields(meta)

	admitResult, err := s.admitPushProfile(meta)
	if err != nil {
		return nil, err
	}

	var profileWeight uint64
	switch admitResult {
	case PassedMicroscopes:
		l.Debug(ctx, "Passed microscope")
		profileWeight = 1
	case PassedSampling:
		l.Debug(ctx, "Passed sampling")
		profileWeight = s.opts.SamplingModulo
	case NotAllowed:
		l.Debug(ctx, "Dropped profile")
		s.metrics.droppedProfiles.Inc()
		return &perforatorstorage.PushProfileResponse{ID: ""}, nil
	}

	meta.Attributes[profilequerylang.WeightLabel] = fmt.Sprintf("%d", profileWeight)

	defer func() {
		if err == nil {
			s.metrics.storedProfiles.Inc()
		} else {
			s.metrics.storedProfilesErrors.Inc()
		}
	}()

	storeProfileCtx := ctx
	var cancel context.CancelFunc
	if s.opts.PushProfileTimeout != time.Duration(0) {
		storeProfileCtx, cancel = context.WithTimeout(ctx, s.opts.PushProfileTimeout)
		defer cancel()
	}

	eventTypes := fixupEventTypes(req.EventTypes)
	metas := createMetasWithEventType(meta, eventTypes)

	var profileID string
	profileID, err = s.profileStorage.StoreProfile(
		storeProfileCtx,
		metas,
		body,
	)
	if err != nil {
		l.Error(ctx,
			"Failed to push profile",
			log.String("service", meta.Service),
			log.Array("event_types", eventTypes),
			log.Error(err),
		)
		return nil, err
	}

	if admitResult == PassedMicroscopes {
		s.metrics.microscopedProfiles.Inc()
	}
	s.metrics.profilesBytesCount.Add(int64(len(body)))
	s.metrics.profilesBytesSizes.RecordValue(float64(len(body)))
	l.Info(ctx,
		"Pushed profile",
		log.String("service", meta.Service),
		log.Time("timestamp", meta.Timestamp),
		log.String("profile_id", profileID),
	)

	return &perforatorstorage.PushProfileResponse{ID: profileID}, nil
}

func (s *StorageServer) doAnnounceBinaries(
	ctx context.Context,
	lookupBinaries []string,
) ([]string, error) {
	var err error
	defer func() {
		if err == nil {
			s.metrics.successAnnounceBinaries.Inc()
		} else {
			s.metrics.failedAnnounceBinaries.Inc()
		}
	}()

	existentBuildIDs := map[string]bool{}
	binaries, err := s.binaryStorage.GetBinaries(ctx, lookupBinaries)
	if err != nil {
		return nil, err
	}

	for _, binary := range binaries {
		if binary.Status == binarymeta.InProgress && time.Since(binary.LastUsedTimestamp) > 5*time.Minute {
			continue
		}

		existentBuildIDs[binary.BuildID] = true
		s.buildIDCache.Set(binary.BuildID, true, cacheItemTTL)
	}

	unknownBinaries := make([]string, 0, len(lookupBinaries)-len(existentBuildIDs))

	for _, buildID := range lookupBinaries {
		if !existentBuildIDs[buildID] {
			unknownBinaries = append(unknownBinaries, buildID)
		}
	}

	return unknownBinaries, nil
}

// implemenets storage.PerforatorStorage/AnnounceBinaries
func (s *StorageServer) AnnounceBinaries(
	ctx context.Context,
	req *perforatorstorage.AnnounceBinariesRequest,
) (*perforatorstorage.AnnounceBinariesResponse, error) {
	if req.AvailableBuildIDs == nil {
		return nil, errors.New("missing available build ids")
	}

	lookupBinaries := make([]string, 0)
	unknownBinaries := make([]string, 0)
	for _, buildID := range req.AvailableBuildIDs {
		item := s.buildIDCache.Get(buildID)
		if item == nil || item.Expired() {
			lookupBinaries = append(lookupBinaries, buildID)
			continue
		}

		if !item.Value() {
			unknownBinaries = append(unknownBinaries, buildID)
		}
	}

	if len(lookupBinaries) > 0 {
		s.metrics.announceBinariesCacheMiss.Inc()
		var unknownLookedUpBinaries []string = nil
		unknownLookedUpBinaries, err := s.doAnnounceBinaries(ctx, lookupBinaries)
		if err != nil {
			// temporary fix to avoid extra binary uploads from agents on errors
			unknownLookedUpBinaries = []string{}
			s.logger.Error(ctx, "Failed to announce binaries", log.Array("lookup_binaries", lookupBinaries), log.Error(err))
			// return nil, err
		}

		unknownBinaries = append(unknownBinaries, unknownLookedUpBinaries...)
	} else {
		s.metrics.announceBinariesCacheHit.Inc()
	}

	return &perforatorstorage.AnnounceBinariesResponse{
		UnknownBuildIDs: unknownBinaries,
	}, nil
}

func (s *StorageServer) pushBinaryPreamble(reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (buildID string, err error) {
	firstChunk, err := reqStream.Recv()
	if err != nil {
		return "", err
	}

	reqHead, ok := firstChunk.Chunk.(*perforatorstorage.PushBinaryRequest_HeadChunk)
	if !ok {
		return "", errors.New("first chunk must be head chunk")
	}
	if reqHead.HeadChunk.BuildID == "" {
		return "", errors.New("build id is missing")
	}

	return reqHead.HeadChunk.BuildID, nil
}

func (s *StorageServer) pushBinaryProcessStream(writer binarystorage.TransactionalWriter, reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (bytesTransmitted uint64, err error) {
	bytesTransmitted = 0

	for {
		chunk, err := reqStream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return bytesTransmitted, fmt.Errorf("failed push binary recv chunk: %w", err)
		}

		bodyChunk, okBodyChunk := chunk.Chunk.(*perforatorstorage.PushBinaryRequest_BodyChunk)
		if !okBodyChunk {
			return bytesTransmitted, errors.New("chunks after first must be body chunks")
		}

		var written int
		written, err = writer.Write(bodyChunk.BodyChunk.Binary)
		if err != nil {
			return bytesTransmitted, fmt.Errorf("failed push binary write chunk: %w", err)
		}

		bytesTransmitted += uint64(written)
	}

	return bytesTransmitted, nil
}

func (s *StorageServer) pushBinaryPerformUpload(buildID string, reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) error {
	start := time.Now()

	writer, err := s.binaryStorage.StoreBinary(
		reqStream.Context(),
		&binarymeta.BinaryMeta{
			BuildID:   buildID,
			Timestamp: start,
		},
	)
	if err != nil {
		if !errors.Is(err, binarymeta.ErrAlreadyUploaded) && !errors.Is(err, binarymeta.ErrUploadInProgress) {
			s.metrics.storedBinariesErrors.Inc()
		}
		return fmt.Errorf("failed to store binary in meta storage: %w", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if err != nil {
			s.metrics.storedBinariesErrors.Inc()
			err = writer.Abort(ctx)
			if err != nil {
				s.metrics.failedAbortBinariesUploads.Inc()
			} else {
				s.metrics.successAbortBinariesUploads.Inc()
			}
		}
	}()

	bytesTransmitted, err := s.pushBinaryProcessStream(writer, reqStream)
	if err != nil {
		return err
	}

	err = writer.Commit(reqStream.Context())
	if err != nil {
		return fmt.Errorf("failed to commit binary: %w", err)
	}

	s.metrics.storedBinaries.Inc()
	s.metrics.binariesUploadTimer.RecordDuration(time.Since(start))
	s.metrics.binariesBytesCount.Add(int64(bytesTransmitted))
	s.buildIDCache.Set(buildID, true, cacheItemTTL)

	s.logger.Info(reqStream.Context(), "Uploaded binary", log.String("build_id", buildID))

	return nil
}

func (s *StorageServer) pushBinaryImpl(reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (buildID string, err error) {
	if !s.binaryUploadLimiter.TryAcquire(1) {
		return "", errors.New("failed to acquire binary upload semaphore")
	}
	defer s.binaryUploadLimiter.Release(1)

	buildID, err = s.pushBinaryPreamble(reqStream)
	if err != nil {
		return buildID, fmt.Errorf("failed preambule: %w", err)
	}

	err = s.pushBinaryPerformUpload(buildID, reqStream)
	if err != nil {
		return buildID, fmt.Errorf("failed to perform upload: %w", err)
	}

	err = reqStream.SendAndClose(&perforatorstorage.PushBinaryResponse{})
	if err != nil {
		return buildID, fmt.Errorf("failed to send and close: %w", err)
	}

	return buildID, nil
}

// implements storage.PerforatorStorage/PushBinary
func (s *StorageServer) PushBinary(
	reqStream perforatorstorage.PerforatorStorage_PushBinaryServer,
) error {
	if !s.opts.PushBinaryWriteAbility {
		s.metrics.droppedBinaryUploads.Inc()
		return errors.New("this replica is not allowed to upload binaries")
	}

	buildID, err := s.pushBinaryImpl(reqStream)
	if err != nil {
		s.logger.Warn(reqStream.Context(), "Failed to push binary", log.String("build_id", buildID), log.Error(err))
	}
	return err
}

func (s *StorageServer) runGrpcServer(ctx context.Context) error {
	s.logger.Info(ctx, "Starting profile storage server", log.UInt32("port", s.conf.Port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.conf.Port))
	if err != nil {
		return err
	}

	if err = s.grpcServer.Serve(lis); err != nil {
		s.logger.Error(ctx, "Failed to grpc server", log.Error(err))
	}

	return err
}

func (s *StorageServer) runMetricsServer(ctx context.Context) error {
	http.Handle("/metrics", s.reg.HTTPHandler(ctx, s.logger))
	port := s.conf.MetricsPort
	if port == 0 {
		port = 85
	}
	s.logger.Info(ctx, "Starting metrics server", log.UInt32("port", port))

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *StorageServer) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.runGrpcServer(ctx)
	})

	if s.microscopeFilter != nil {
		g.Go(func() error {
			s.microscopeFilter.Run(ctx)
			s.logger.Warn(ctx, "Stopped pulling microscopes")
			return nil
		})
	}

	g.Go(func() error {
		return s.runMetricsServer(ctx)
	})

	return g.Wait()
}

func NewStorageServer(
	conf *Config,
	logger xlog.Logger,
	registry xmetrics.Registry,
	opts *StorageOptions,
) (*StorageServer, error) {
	if opts == nil {
		opts = &StorageOptions{
			ClusterName:            os.Getenv("DEPLOY_NODE_DC"),
			SamplingModulo:         1,
			MaxBuildIDCacheEntries: 14000000,
			PushProfileTimeout:     10 * time.Second,
			PushBinaryWriteAbility: true,
		}
	}

	initCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	storageBundle, err := bundle.NewStorageBundle(initCtx, logger, registry, &conf.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage bundle: %w", err)
	}

	var microscopeFilter *filter.PullingFilter
	if conf.MicroscopePullerConfig != nil {
		if storageBundle.MicroscopeStorage == nil {
			return nil, errors.New("microscope storage must be specified in config")
		}

		microscopeFilter, err = filter.NewPullingFilter(
			logger,
			registry,
			*conf.MicroscopePullerConfig,
			storageBundle.MicroscopeStorage,
		)
		if err != nil {
			return nil, err
		}
	}

	cache := ccache.New[bool](ccache.Configure[bool]().MaxSize(int64(opts.MaxBuildIDCacheEntries)))

	server := &StorageServer{
		logger: logger,
		conf:   conf,
		opts:   opts,
		reg:    registry,
		metrics: &storageMetrics{
			pushProfileInProgress:   registry.IntGauge("push_profile.in_progress.gauge"),
			successPushProfileTimer: registry.WithTags(map[string]string{"kind": "success"}).Timer("push_profile.timer"),
			failPushProfileTimer:    registry.WithTags(map[string]string{"kind": "fail"}).Timer("push_profile.timer"),
			droppedProfiles:         registry.WithTags(map[string]string{"kind": "dropped"}).Counter("profiles.count"),
			storedProfiles:          registry.WithTags(map[string]string{"kind": "stored"}).Counter("profiles.count"),
			microscopedProfiles:     registry.WithTags(map[string]string{"kind": "microscoped"}).Counter("profiles.count"),
			storedProfilesErrors:    registry.WithTags(map[string]string{"kind": "failed_store"}).Counter("profiles.count"),
			profilesBytesCount:      registry.WithTags(map[string]string{"kind": "profiles"}).Counter("bytes.uploaded"),
			profilesBytesSizes: registry.WithTags(map[string]string{"kind": "profile"}).Histogram(
				"size.bytes",
				metrics.MakeLinearBuckets(0, 1024*100, 10),
			),
			storedBinaries:              registry.WithTags(map[string]string{"kind": "stored"}).Counter("binaries.count"),
			storedBinariesErrors:        registry.WithTags(map[string]string{"kind": "failed_store"}).Counter("binaries.count"),
			droppedBinaryUploads:        registry.Counter("binaries.dropped_uploads"),
			binariesBytesCount:          registry.WithTags(map[string]string{"kind": "binaries"}).Counter("bytes.uploaded"),
			binariesUploadTimer:         registry.Timer("binaries.upload_timer"),
			failedAbortBinariesUploads:  registry.WithTags(map[string]string{"status": "failed"}).Counter("binary_upload_aborts.count"),
			successAbortBinariesUploads: registry.WithTags(map[string]string{"status": "success"}).Counter("binary_upload_aborts.count"),
			successAnnounceBinaries:     registry.WithTags(map[string]string{"kind": "success"}).Counter("announce_binaries.count"),
			failedAnnounceBinaries:      registry.WithTags(map[string]string{"kind": "failed"}).Counter("announce_binaries.count"),
			announceBinariesCacheHit:    registry.WithTags(map[string]string{"kind": "hit"}).Counter("announce_binaries.count"),
			announceBinariesCacheMiss:   registry.WithTags(map[string]string{"kind": "miss"}).Counter("announce_binaries.count"),
		},
		profileSampler:           NewModuloSampler(opts.SamplingModulo),
		binaryUploadLimiter:      semaphore.NewWeighted(1),
		profileStorage:           storageBundle.ProfileStorage,
		binaryStorage:            storageBundle.BinaryStorage.Binary(),
		microscopeFilter:         microscopeFilter,
		buildIDCache:             cache,
		profileCommentProcessors: make(map[string]func(string, *profilemeta.ProfileMetadata) error),
	}

	creds, err := credentials.NewServerTLSFromFile(
		conf.TLSConfig.CertificateFile,
		conf.TLSConfig.KeyFile,
	)
	if err != nil {
		return nil, err
	}

	credsInterceptor, err := getInterceptor(conf, logger)
	if err != nil {
		return nil, err
	}

	metricsInterceptor := grpcmetrics.NewMetricsInterceptor(registry)

	unaryServerInterceptors := []grpc.UnaryServerInterceptor{metricsInterceptor.UnaryServer()}
	streamServerInterceptors := []grpc.StreamServerInterceptor{metricsInterceptor.StreamServer()}
	if credsInterceptor != nil {
		unaryServerInterceptors = append(unaryServerInterceptors, credsInterceptor.Unary())
		streamServerInterceptors = append(streamServerInterceptors, credsInterceptor.Stream())
	}

	server.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*1024 /* 1 GB */),
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			unaryServerInterceptors...,
		),
		grpc.ChainStreamInterceptor(
			streamServerInterceptors...,
		),
	)
	perforatorstorage.RegisterPerforatorStorageServer(server.grpcServer, server)
	reflection.Register(server.grpcServer)

	server.initProfileCommentProcessors()

	return server, nil
}

func getInterceptor(conf *Config, logger xlog.Logger) (creds.ServerInterceptor, error) {
	if conf.TvmAuth != nil {
		return storagetvm.NewTVMServerInterceptor(
			conf.TvmAuth.ID,
			os.Getenv(conf.TvmAuth.SecretEnvName),
			logger,
		)
	}
	return nil, nil
}
