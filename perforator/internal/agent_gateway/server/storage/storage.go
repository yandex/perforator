package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/binaryupload"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/kafka/producer"
	profilebundle "github.com/yandex/perforator/perforator/pkg/profile/bundle"
	"github.com/yandex/perforator/perforator/pkg/profile_event"
	"github.com/yandex/perforator/perforator/pkg/profile_event/async_publisher"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	profilemeta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/pprofprofile"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
	"github.com/yandex/perforator/perforator/util/go/tsformat"
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

	timeToDatabaseHist metrics.Timer
}

type Service struct {
	conf *ServiceConfig
	opts *options

	reg     xmetrics.Registry
	metrics *serviceMetrics
	logger  xlog.Logger

	profileSamplerByTypes map[string]*moduloSampler
	mutex                 sync.RWMutex

	profileStorage profilestorage.Storage

	microscopeFilter *filter.PullingFilter

	*binaryupload.Service

	profileCommentProcessors map[string]func(string, *profilemeta.ProfileMetadata) error

	signalPublisher *async_publisher.AsyncSignalProfileEventPublisher
	signalAllow     map[string]struct{}
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

	var (
		asyncPublisher *async_publisher.AsyncSignalProfileEventPublisher
		signalAllow    map[string]struct{}
	)
	if conf.ProfileSignalEvents != nil && conf.ProfileSignalEvents.Kafka != nil {
		if len(conf.ProfileSignalEvents.AllowedSignals) == 0 {
			return nil, errors.New("init kafka publisher: there should be at least one signal in \"allowed_signals\"")
		}
		signalAllow = makeStringSet(conf.ProfileSignalEvents.AllowedSignals)

		kp, err := producer.NewKafkaProducer(logger, conf.ProfileSignalEvents.Kafka)
		if err != nil {
			return nil, fmt.Errorf("init kafka producer: %w", err)
		}
		asyncPublisher = async_publisher.NewAsyncSignalProfileEventPublisher(kp, logger, reg, conf.ProfileSignalEvents.Config)
	}

	service := &Service{
		logger: logger,
		conf:   conf,
		reg:    reg,
		opts:   opts,
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
			timeToDatabaseHist: reg.DurationHistogram(
				"profile_time_to_database.seconds",
				metrics.MakeExponentialDurationBuckets(time.Minute, 1.1, 30),
			),
		},
		profileSamplerByTypes: make(map[string]*moduloSampler),
		profileStorage:        storageBundle.ProfileStorage,
		microscopeFilter:      microscopeFilter,
		Service: binaryupload.NewService(logger, reg, storageBundle.BinaryStorage, binaryupload.Options{
			MaxConcurrentUploads: 1,
			KnownCacheSize:       int64(opts.maxBuildIDCacheEntries),
			DenyWrites:           !opts.pushBinaryWriteAbility,
		}),
		profileCommentProcessors: make(map[string]func(string, *profilemeta.ProfileMetadata) error),
		signalPublisher:          asyncPublisher,
		signalAllow:              signalAllow,
	}

	service.initProfileCommentProcessors()
	return service, nil
}

func (s *Service) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	if s.microscopeFilter != nil {
		g.Go(func() error {
			s.microscopeFilter.Run(ctx)
			s.logger.Warn(ctx, "Stopped pulling microscopes")
			return nil
		})
	}

	if s.signalPublisher != nil {
		g.Go(func() error {
			return s.signalPublisher.Run(ctx)
		})
	}

	return g.Wait()
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

	meta, err := s.createProfileMetaFromLabels(ctx, labels)
	if err != nil {
		return nil, err
	}

	if profile.TimeNanos != 0 {
		meta.Timestamp = time.Unix(0, profile.TimeNanos)
	}

	return meta, nil
}

func (s *Service) extractProfileBytesMeta(
	ctx context.Context,
	req *perforatorstorage.PushProfileRequest,
) (pprofBody []byte, yaprofBody []byte, meta *profilemeta.ProfileMetadata, err error) {
	// Legacy path: labels are embedded inside the protobuf Profile message.
	// Deprecated: new agents should send labels via req.GetLabels() and raw bytes via ProfileBytes/YaprofBytes.
	if legacyProfile := req.GetProfile(); legacyProfile != nil {
		meta, err = s.getMetadataFromProfile(ctx, legacyProfile)
		if err != nil {
			return
		}

		pprofBody, err = proto.Marshal(legacyProfile)
		if err != nil {
			return
		}

		yaprofBody = req.GetYaprofBytes()
		s.applyCommonMeta(req, meta)
		return
	}

	// Normal path: labels come from req.GetLabels(), profile bytes from dedicated fields.
	meta, err = s.createProfileMetaFromLabels(ctx, req.GetLabels())
	if err != nil {
		return
	}

	pprofBody = req.GetProfileBytes()
	yaprofBody = req.GetYaprofBytes()

	if len(pprofBody) == 0 && len(yaprofBody) == 0 {
		return nil, nil, nil, errors.New("request does not contain profile")
	}

	s.applyCommonMeta(req, meta)
	return
}

func (s *Service) applyCommonMeta(req *perforatorstorage.PushProfileRequest, meta *profilemeta.ProfileMetadata) {
	if req.StartTimestamp != nil && !req.StartTimestamp.AsTime().IsZero() {
		meta.Timestamp = req.StartTimestamp.AsTime()
	}

	meta.BuildIDs = slices.Clone(req.GetBuildIDs())
	meta.Envs = slices.Clone(req.GetEnvs())
	meta.CustomProfilingOperationID = req.GetCPOID()
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
	if meta.Timestamp.IsZero() {
		meta.Timestamp = time.Now()
	}
}

type pushProfileAdmitResult int

const (
	notAllowed pushProfileAdmitResult = iota
	passedSampling
	passedMicroscopes
)

func (s *Service) samplerForTypes(eventTypes []string) *moduloSampler {
	modulo := s.minModuloForTypes(eventTypes)
	key := typesKey(eventTypes)

	s.mutex.RLock()
	sampler := s.profileSamplerByTypes[key]
	s.mutex.RUnlock()

	if sampler != nil {
		return sampler
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	sampler = s.profileSamplerByTypes[key]
	if sampler == nil {
		sampler = newModuloSampler(modulo)
		s.profileSamplerByTypes[key] = sampler
	}

	return sampler
}

func (s *Service) moduloForEvent(eventType string) uint64 {
	modulo, ok := s.opts.samplingModuloByEvent[eventType]
	if !ok {
		modulo = s.opts.samplingModulo
	}
	if modulo == 0 {
		modulo = 1
	}

	return modulo
}

func (s *Service) minModuloForTypes(eventTypes []string) uint64 {
	var modulo uint64
	for _, eventType := range eventTypes {
		m := s.moduloForEvent(eventType)
		if modulo == 0 || m < modulo {
			modulo = m
		}
	}
	if modulo == 0 {
		modulo = 1
	}

	return modulo
}

func typesKey(eventTypes []string) string {
	sorted := slices.Clone(eventTypes)
	slices.Sort(sorted)
	return strings.Join(sorted, "\x00")
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

	if req.GetProfileRepresentation() == nil && len(req.GetYaprofBytes()) == 0 {
		return nil, errors.New("missing profile field")
	}

	pprofBody, yaprofBody, meta, err := s.extractProfileBytesMeta(ctx, req)
	if err != nil {
		return nil, err
	}
	s.fixupMissingMetadataFields(meta)

	eventTypes := fixupEventTypes(req.EventTypes)
	metas := createMetasWithEventType(meta, eventTypes)

	if req.CPOID == "" {
		// Do not sample CPO profiles
		metas = s.sampleProfiles(ctx, l, metas)
		if len(metas) == 0 {
			return &perforatorstorage.PushProfileResponse{ID: ""}, nil
		}
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
		profilebundle.NewBundle(pprofBody, yaprofBody),
		profilemeta.WithPersistCallback(func(m *profilemeta.ProfileMetadata) {
			if !m.Timestamp.IsZero() {
				s.metrics.timeToDatabaseHist.RecordDuration(time.Since(m.Timestamp))
			}
		}),
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

	totalSize := len(pprofBody) + len(yaprofBody)
	s.metrics.profilesBytesCount.Add(int64(totalSize))
	s.metrics.profilesBytesSizes.RecordValue(float64(totalSize))

	l.Info(ctx,
		"Pushed profile",
		log.String("service", meta.Service),
		log.Time("timestamp", meta.Timestamp),
		log.String("profile_id", profileID),
	)

	if s.signalPublisher != nil && s.shouldPublishSignals(req.GetSignalTypes()) {
		if s.conf.ProfileSignalEvents.SamplingRate > 0 && rand.Float64() <= s.conf.ProfileSignalEvents.SamplingRate {
			if slices.Contains(eventTypes, sampletype.SampleTypeSignalCount) {
				ev := &profile_event.SignalProfileEvent{
					ProfileID:   profileID,
					Service:     meta.Service,
					Cluster:     meta.Cluster,
					NodeID:      meta.NodeID,
					PodID:       meta.PodID,
					Timestamp:   meta.Timestamp.UTC(),
					BuildIDs:    meta.BuildIDs,
					MainEvent:   sampletype.SampleTypeSignalCount,
					SignalTypes: req.GetSignalTypes(),
				}

				s.signalPublisher.TryEnqueueForPublish(ctx, ev)
			} else {
				l.Warn(ctx,
					"Missing proper event type",
					log.String("service", meta.Service),
					log.Time("timestamp", meta.Timestamp),
					log.String("profile_id", profileID),
				)
			}
		}
	}

	return &perforatorstorage.PushProfileResponse{ID: profileID}, nil
}

func (s *Service) sampleProfiles(
	ctx context.Context,
	l xlog.Logger,
	metas []*profilemeta.ProfileMetadata,
) []*profilemeta.ProfileMetadata {
	sampler := s.samplerForTypes(metas[0].AllEventTypes)
	sampled := sampler.Sample()
	sampledWeight := sampler.modulo

	count := 0
	for _, meta := range metas {
		var (
			admitResult   pushProfileAdmitResult
			profileWeight uint64
		)
		switch {
		case sampled:
			admitResult, profileWeight = passedSampling, sampledWeight
		case s.microscopeFilter != nil && s.microscopeFilter.Filter(meta):
			admitResult, profileWeight = passedMicroscopes, 1
		default:
			admitResult = notAllowed
		}

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

// ///////////////////////////////////////////////////////////////////////////////////////////
func makeStringSet(strs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(strs))
	for _, x := range strs {
		m[x] = struct{}{}
	}
	return m
}

func (s *Service) shouldPublishSignals(signalTypes []string) bool {
	if len(signalTypes) == 0 || s.signalAllow == nil {
		return false
	}

	for _, sig := range signalTypes {
		if _, ok := s.signalAllow[sig]; ok {
			return true
		}
	}
	return false
}
