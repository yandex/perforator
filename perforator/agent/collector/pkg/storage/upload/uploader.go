package upload

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/karlseguin/ccache/v3"
	"golang.org/x/sync/semaphore"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
)

const (
	DefaultUploadedBinariesCacheSize          = uint64(10000)
	DefaultUploadedBinariesCacheItemTTL       = 10 * time.Minute
	DefaultUploadedBinariesCacheItemMaxJitter = 20 * time.Minute
)

var (
	ErrBinariesQueueFull = errors.New("binaries queue is full")
)

type UploadedBinaryCacheConfig struct {
	ExpirationMaxJitter time.Duration `yaml:"expiration_max_jitter"`
	ItemTimeout         time.Duration `yaml:"base_expiration_timeout"`
	MaxSize             uint64        `yaml:"size"`
}

func (c *UploadedBinaryCacheConfig) fillDefault() {
	if c.MaxSize == 0 {
		c.MaxSize = DefaultUploadedBinariesCacheSize
	}
	if c.ItemTimeout == time.Duration(0) {
		c.ItemTimeout = DefaultUploadedBinariesCacheItemTTL
	}
	if c.ExpirationMaxJitter == time.Duration(0) {
		c.ExpirationMaxJitter = DefaultUploadedBinariesCacheItemMaxJitter
	}
}

type SchedulerConfig struct {
	MaxClosedBinariesQueue    uint32                    `yaml:"max_closed_binaries_queue"`
	MaxSimultaneousUploads    uint32                    `yaml:"max_simultaneous_uploads"`
	UploadedBinaryCacheConfig UploadedBinaryCacheConfig `yaml:"uploaded_binaries_cache"`
}

func (c *SchedulerConfig) fillDefault() {
	if c.MaxClosedBinariesQueue == 0 {
		c.MaxClosedBinariesQueue = 1000000
	}
	if c.MaxSimultaneousUploads == 0 {
		c.MaxSimultaneousUploads = 30
	}
	c.UploadedBinaryCacheConfig.fillDefault()
}

type ClosedBinary struct {
	BuildID string
	Handle  binary.SealedFile
}

type uploadSchedulerMetrics struct {
	binariesEnqueuedForUpload         metrics.Counter
	closedBinariesQueueSize           metrics.FuncIntGauge
	announceBinariesRequestsPerformed metrics.Counter

	uploadedBinariesCacheHits   metrics.Counter
	uploadedBinariesCacheMisses metrics.Counter
}

type Scheduler struct {
	conf SchedulerConfig

	// cache of binaries that are present in storage with the purpose of reducing AnnounceBinaries load on backend
	uploadedBinariesCache *ccache.Cache[string]

	binariesQueue chan ClosedBinary

	simultaneousUploadsSem *semaphore.Weighted

	storage client.BinaryStorage
	l       log.Logger

	randomSource rand.Source

	metrics *uploadSchedulerMetrics
}

func NewUploadScheduler(
	conf SchedulerConfig,
	storage client.BinaryStorage,
	l log.Logger,
	r metrics.Registry,
) (*Scheduler, error) {
	conf.fillDefault()

	l.Info("New upload scheduler", log.Any("config", conf))

	if conf.MaxSimultaneousUploads == 0 {
		return nil, errors.New("max simultaneous uploads must be positive")
	}

	scheduler := &Scheduler{
		conf:                   conf,
		uploadedBinariesCache:  ccache.New(ccache.Configure[string]().MaxSize(int64(conf.UploadedBinaryCacheConfig.MaxSize))),
		binariesQueue:          make(chan ClosedBinary, conf.MaxClosedBinariesQueue),
		simultaneousUploadsSem: semaphore.NewWeighted(int64(conf.MaxSimultaneousUploads)),
		storage:                storage,
		randomSource:           rand.NewSource(time.Now().UnixNano()),
		l:                      l.WithName("uploader"),
	}

	scheduler.metrics = &uploadSchedulerMetrics{
		binariesEnqueuedForUpload: r.WithTags(map[string]string{"kind": "enqueued_for_upload"}).Counter("binaries.count"),
		closedBinariesQueueSize: r.WithTags(map[string]string{"kind": "closed"}).FuncIntGauge("binary_upload_queue.size", func() int64 {
			return int64(len(scheduler.binariesQueue))
		}),
		uploadedBinariesCacheHits:         r.WithTags(map[string]string{"kind": "hit"}).Counter("binaries.announce_binaries.cache"),
		uploadedBinariesCacheMisses:       r.WithTags(map[string]string{"kind": "miss"}).Counter("binaries.announce_binaries.cache"),
		announceBinariesRequestsPerformed: r.WithTags(map[string]string{"kind": "performed"}).Counter("binaries.announce_binaries.requests"),
	}

	return scheduler, nil
}

// thread safe
func (u *Scheduler) ScheduleBinary(buildID string, handle binary.SealedFile) error {
	if item := u.uploadedBinariesCache.Get(buildID); item != nil && !item.Expired() {
		u.metrics.uploadedBinariesCacheHits.Inc()
		return nil
	}

	select {
	case u.binariesQueue <- ClosedBinary{BuildID: buildID, Handle: handle}:
	default:
		return ErrBinariesQueueFull
	}

	u.metrics.binariesEnqueuedForUpload.Inc()

	return nil
}

func (u *Scheduler) uploadBinaryImpl(ctx context.Context, buildID string, closedBinary *ClosedBinary) {
	err := u.simultaneousUploadsSem.Acquire(ctx, 1)
	if err != nil { // context was cancelled
		return
	}
	defer u.simultaneousUploadsSem.Release(1)

	err = u.storage.StoreBinary(ctx, buildID, closedBinary.Handle)
	if err != nil {
		u.l.Error("Failed to upload binary", log.String("build_id", buildID), log.Error(err))
		return
	}
}

func (u *Scheduler) getRandomJitter(cap time.Duration) time.Duration {
	generator := rand.New(u.randomSource)
	maxNanos := int64(cap)
	nanos := generator.Int63n(maxNanos + 1)
	return time.Duration(nanos)
}

func (u *Scheduler) uploadBinary(ctx context.Context, buildID string, binary *ClosedBinary) {
	go u.uploadBinaryImpl(ctx, buildID, binary)
}

func (u *Scheduler) announceBinariesWithCache(ctx context.Context, buildIDs []string) (unknownBuildIDs []string, err error) {
	unknownBuildIDs = []string{}
	askBuildIDs := make([]string, 0)
	for _, buildID := range buildIDs {
		item := u.uploadedBinariesCache.Get(buildID)
		if item == nil || item.Expired() {
			u.metrics.uploadedBinariesCacheMisses.Inc()
			askBuildIDs = append(askBuildIDs, buildID)
			continue
		}

		u.metrics.uploadedBinariesCacheHits.Inc()
	}

	if len(askBuildIDs) == 0 {
		return
	}

	u.metrics.announceBinariesRequestsPerformed.Inc()
	unknownBuildIDs, err = u.storage.AnnounceBinaries(ctx, askBuildIDs)
	if err != nil {
		return
	}

	unknownBinaries := make(map[string]bool, len(unknownBuildIDs))
	for _, buildID := range unknownBuildIDs {
		unknownBinaries[buildID] = true
	}

	for _, buildID := range askBuildIDs {
		if unknownBinaries[buildID] {
			continue
		}

		u.uploadedBinariesCache.Set(
			buildID,
			"",
			u.conf.UploadedBinaryCacheConfig.ItemTimeout+u.getRandomJitter(u.conf.UploadedBinaryCacheConfig.ExpirationMaxJitter),
		)
	}
	return
}

func (u *Scheduler) announceAndUpload(
	ctx context.Context,
	binaries map[string]*ClosedBinary,
) error {
	buildIDs := make([]string, 0, len(binaries))
	for buildID := range binaries {
		buildIDs = append(buildIDs, buildID)
	}

	unknownBuildIDs, err := u.announceBinariesWithCache(ctx, buildIDs)
	if err != nil {
		u.l.Warn("Failed announce binaries", log.Error(err), log.Array("build_ids", buildIDs))
		return err
	}

	for _, buildID := range unknownBuildIDs {
		u.uploadBinary(ctx, buildID, binaries[buildID])
	}

	return nil
}

func (u *Scheduler) runUploadScheduler(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case closedBinary := <-u.binariesQueue:
			err := u.announceAndUpload(ctx, map[string]*ClosedBinary{closedBinary.BuildID: &closedBinary})
			if err != nil {
				u.l.Error(
					"Failed to upload closed binary",
					log.String("build_id", closedBinary.BuildID),
					log.Error(err),
				)
			}
		}
	}
}

func (u *Scheduler) RunWorker(ctx context.Context) error {
	return u.runUploadScheduler(ctx)
}
