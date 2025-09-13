package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/karlseguin/ccache/v3"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	profilemeta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/pprofprofile"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
	"github.com/yandex/perforator/perforator/util/go/tsformat"
)

const (
	cacheItemTTL = 10 * time.Minute
)

type serviceMetrics struct {
	receivedProfiles     metrics.Counter
	droppedProfiles      metrics.Counter
	sampledProfiles      metrics.Counter
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

type Service struct {
	conf *ServiceConfig
	opts *options

	reg     xmetrics.Registry
	metrics *serviceMetrics
	logger  xlog.Logger

	binaryUploadLimiter   *semaphore.Weighted
	defaultProfileSampler *moduloSampler
	profileSamplerByEvent map[string]*moduloSampler

	profileStorage profilestorage.Storage
	binaryStorage  binarystorage.Storage

	microscopeFilter *filter.PullingFilter

	buildIDCache *ccache.Cache[bool]

	profileCommentProcessors map[string]func(string, *profilemeta.ProfileMetadata) error
}

func NewService(
	conf *ServiceConfig,
	logger xlog.Logger,
	reg xmetrics.Registry,
	storageBundle *bundle.StorageBundle,
	optAppliers ...Option,
) (*Service, error) {
	opts := defaultOpts()
	for _, optApplier := range optAppliers {
		optApplier(opts)
	}

	var microscopeFilter *filter.PullingFilter
	var err error
	if conf.MicroscopePullerConfig != nil {
		if storageBundle.MicroscopeStorage == nil {
			return nil, errors.New("microscope storage must be specified in config")
		}

		microscopeFilter, err = filter.NewPullingFilter(
			logger,
			reg,
			*conf.MicroscopePullerConfig,
			storageBundle.MicroscopeStorage,
		)
		if err != nil {
			return nil, err
		}
	}

	cache := ccache.New[bool](ccache.Configure[bool]().MaxSize(int64(opts.maxBuildIDCacheEntries)))

	service := &Service{
		logger: logger,
		conf:   conf,
		reg:    reg,
		metrics: &serviceMetrics{
			pushProfileInProgress:   reg.IntGauge("push_profile.in_progress.gauge"),
			successPushProfileTimer: reg.WithTags(map[string]string{"kind": "success"}).Timer("push_profile.timer"),
			failPushProfileTimer:    reg.WithTags(map[string]string{"kind": "fail"}).Timer("push_profile.timer"),
			receivedProfiles:        reg.Counter("profiles.received.count"),
			droppedProfiles:         reg.WithTags(map[string]string{"kind": "dropped"}).Counter("profiles.count"),
			sampledProfiles:         reg.WithTags(map[string]string{"kind": "sampled"}).Counter("profiles.count"),
			storedProfiles:          reg.WithTags(map[string]string{"kind": "stored"}).Counter("profiles.count"),
			microscopedProfiles:     reg.WithTags(map[string]string{"kind": "microscoped"}).Counter("profiles.count"),
			storedProfilesErrors:    reg.WithTags(map[string]string{"kind": "failed_store"}).Counter("profiles.count"),
			profilesBytesCount:      reg.WithTags(map[string]string{"kind": "profiles"}).Counter("bytes.uploaded"),
			profilesBytesSizes: reg.WithTags(map[string]string{"kind": "profile"}).Histogram(
				"size.bytes",
				metrics.MakeLinearBuckets(0, 1024*100, 10),
			),
			storedBinaries:              reg.WithTags(map[string]string{"kind": "stored"}).Counter("binaries.count"),
			storedBinariesErrors:        reg.WithTags(map[string]string{"kind": "failed_store"}).Counter("binaries.count"),
			droppedBinaryUploads:        reg.Counter("binaries.dropped_uploads"),
			binariesBytesCount:          reg.WithTags(map[string]string{"kind": "binaries"}).Counter("bytes.uploaded"),
			binariesUploadTimer:         reg.Timer("binaries.upload_timer"),
			failedAbortBinariesUploads:  reg.WithTags(map[string]string{"status": "failed"}).Counter("binary_upload_aborts.count"),
			successAbortBinariesUploads: reg.WithTags(map[string]string{"status": "success"}).Counter("binary_upload_aborts.count"),
			successAnnounceBinaries:     reg.WithTags(map[string]string{"kind": "success"}).Counter("announce_binaries.count"),
			failedAnnounceBinaries:      reg.WithTags(map[string]string{"kind": "failed"}).Counter("announce_binaries.count"),
			announceBinariesCacheHit:    reg.WithTags(map[string]string{"kind": "hit"}).Counter("announce_binaries.count"),
			announceBinariesCacheMiss:   reg.WithTags(map[string]string{"kind": "miss"}).Counter("announce_binaries.count"),
		},
		defaultProfileSampler:    newModuloSampler(opts.samplingModulo),
		profileSamplerByEvent:    make(map[string]*moduloSampler),
		binaryUploadLimiter:      semaphore.NewWeighted(1),
		profileStorage:           storageBundle.ProfileStorage,
		binaryStorage:            storageBundle.BinaryStorage.Binary(),
		microscopeFilter:         microscopeFilter,
		buildIDCache:             cache,
		profileCommentProcessors: make(map[string]func(string, *profilemeta.ProfileMetadata) error),
	}

	for typ, modulo := range opts.samplingModuloByEvent {
		service.profileSamplerByEvent[typ] = newModuloSampler(modulo)
	}

	service.initProfileCommentProcessors()
	return service, nil
}

func (s *Service) Run(ctx context.Context) error {
	if s.microscopeFilter != nil {
		s.microscopeFilter.Run(ctx)
		s.logger.Warn(ctx, "Stopped pulling microscopes")
	}

	return nil
}

func (s *Service) initProfileCommentProcessors() {
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

func (s *Service) createProfileMetaFromLabels(ctx context.Context, labels map[string]string) (*profilemeta.ProfileMetadata, error) {
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

func (s *Service) getMetadataFromProfile(ctx context.Context, profile *pprofprofile.Profile) (*profilemeta.ProfileMetadata, error) {
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

func (s *Service) extractProfileBytesMeta(
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

func (s *Service) fixupMissingMetadataFields(meta *profilemeta.ProfileMetadata) {
	if meta.System == "" {
		meta.System = "perforator"
	}
	if meta.Cluster == "" {
		// We want to prioritize cluster label received from agent
		// If it is empty, we fall back to user-provided cluster name on storage side
		if val := meta.Attributes["cluster"]; val != "" {
			meta.Cluster = val
		} else if s.opts.clusterName != "" {
			meta.Cluster = s.opts.clusterName
		}
	}
	if meta.NodeID == "" {
		meta.NodeID = meta.Attributes["host"]
	}
	if meta.PodID == "" {
		meta.PodID = meta.Attributes["pod"]
	}
}

type pushProfileAdmitResult int

const (
	notAllowed pushProfileAdmitResult = iota
	passedSampling
	passedMicroscopes
)

func (s *Service) sampleProfile(meta *profilemeta.ProfileMetadata) (pushProfileAdmitResult, uint64) {
	sampler := s.profileSamplerByEvent[meta.MainEventType]
	if sampler == nil {
		sampler = s.defaultProfileSampler
	}
	if sampler.Sample() {
		return passedSampling, sampler.modulo
	}

	if s.microscopeFilter != nil && s.microscopeFilter.Filter(meta) {
		return passedMicroscopes, 1
	}

	return notAllowed, 0
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

// implements PerforatorStorage/PushProfile
func (s *Service) PushProfile(ctx context.Context, req *perforatorstorage.PushProfileRequest) (*perforatorstorage.PushProfileResponse, error) {
	s.metrics.pushProfileInProgress.Add(1)
	defer func() {
		s.metrics.pushProfileInProgress.Add(-1)
	}()
	s.metrics.receivedProfiles.Inc()

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

	eventTypes := fixupEventTypes(req.EventTypes)
	metas := createMetasWithEventType(meta, eventTypes)

	metas = s.sampleProfiles(ctx, l, metas)
	if len(metas) == 0 {
		return &perforatorstorage.PushProfileResponse{ID: ""}, nil
	}

	defer func() {
		if err == nil {
			s.metrics.storedProfiles.Inc()
		} else {
			s.metrics.storedProfilesErrors.Inc()
		}
	}()

	storeProfileCtx := ctx
	var cancel context.CancelFunc
	if s.opts.pushProfileTimeout != time.Duration(0) {
		storeProfileCtx, cancel = context.WithTimeout(ctx, s.opts.pushProfileTimeout)
		defer cancel()
	}

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

func (s *Service) sampleProfiles(
	ctx context.Context,
	l xlog.Logger,
	metas []*profilemeta.ProfileMetadata,
) []*profilemeta.ProfileMetadata {
	count := 0
	for _, meta := range metas {
		admitResult, profileWeight := s.sampleProfile(meta)

		switch admitResult {
		case passedMicroscopes:
			l.Debug(ctx, "Passed microscope")
			s.metrics.microscopedProfiles.Inc()
		case passedSampling:
			l.Debug(ctx, "Passed sampling")
			s.metrics.sampledProfiles.Inc()
		case notAllowed:
			l.Debug(ctx, "Dropped profile")
			s.metrics.droppedProfiles.Inc()
			continue
		}

		meta.Attributes[profilequerylang.WeightLabel] = fmt.Sprintf("%d", profileWeight)

		metas[count] = meta
		count++
	}

	return metas[:count]
}

func (s *Service) doAnnounceBinaries(
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

// implements PerforatorStorage/AnnounceBinaries
func (s *Service) AnnounceBinaries(
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

func (s *Service) pushBinaryPreamble(reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (buildID string, err error) {
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

func (s *Service) pushBinaryProcessStream(writer binarystorage.TransactionalWriter, reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (bytesTransmitted uint64, err error) {
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

func (s *Service) pushBinaryPerformUpload(buildID string, reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) error {
	start := time.Now()

	writer, err := s.binaryStorage.StoreBinary(
		reqStream.Context(),
		&binarymeta.BinaryMeta{
			BuildID:   buildID,
			Timestamp: start,
		},
	)
	if err != nil {
		if !errors.Is(err, binarymeta.ErrAlreadyUploaded) &&
			!errors.Is(err, binarymeta.ErrUploadInProgress) {
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

func (s *Service) pushBinaryImpl(reqStream perforatorstorage.PerforatorStorage_PushBinaryServer) (buildID string, err error) {
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

// implements PerforatorStorage/PushBinary
func (s *Service) PushBinary(
	reqStream perforatorstorage.PerforatorStorage_PushBinaryServer,
) error {
	if !s.opts.pushBinaryWriteAbility {
		s.metrics.droppedBinaryUploads.Inc()
		return errors.New("this replica is not allowed to upload binaries")
	}

	buildID, err := s.pushBinaryImpl(reqStream)
	if err != nil {
		s.logger.Warn(reqStream.Context(), "Failed to push binary", log.String("build_id", buildID), log.Error(err))
	}
	return err
}
