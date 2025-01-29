package symbolize

import (
	"context"
	"errors"
	"sync"

	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrBuildIDAcquired = errors.New("build id releaser is already acquired")
)

// intended for single thread use
// first acquire needed binaries then wait for them
type CachedBinariesBatch struct {
	l                xlog.Logger
	binaryProvider   binaryprovider.BinaryProvider
	mutex            sync.RWMutex
	acquiredBinaries map[string]binaryprovider.FileHandle // guarded by mutex
	downloaded       map[string]bool                      // not guarded
	acquireGroup     sync.WaitGroup

	logErrorOnFailedAcquire bool
}

func NewCachedBinariesBatch(
	l xlog.Logger,
	provider binaryprovider.BinaryProvider,
	logErrorOnFailedAcquire bool,
) *CachedBinariesBatch {
	return &CachedBinariesBatch{
		l:                       l.WithName("CachedBinariesBatch"),
		binaryProvider:          provider,
		acquiredBinaries:        map[string]binaryprovider.FileHandle{},
		downloaded:              map[string]bool{},
		logErrorOnFailedAcquire: logErrorOnFailedAcquire,
	}
}

func (b *CachedBinariesBatch) addAcquired(buildID string, handle binaryprovider.FileHandle) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	prevBinary := b.acquiredBinaries[buildID]
	if prevBinary != nil {
		// no need to store more than one acquired ref to binary
		handle.Close()
		return
	}

	b.acquiredBinaries[buildID] = handle
}

func (b *CachedBinariesBatch) Acquire(ctx context.Context, buildID string) {
	acquiredFile, err := b.binaryProvider.Acquire(ctx, buildID)
	if err != nil {
		logFn := b.l.Info
		if b.logErrorOnFailedAcquire {
			logFn = b.l.Error
		}
		logFn(ctx, "Failed to acquire binary", log.String("build_id", buildID), log.Error(err))
		return
	}

	b.addAcquired(buildID, acquiredFile)
}

func (b *CachedBinariesBatch) AcquireAsync(ctx context.Context, buildID string) {
	b.acquireGroup.Add(1)
	go func() {
		defer b.acquireGroup.Done()
		b.Acquire(ctx, buildID)
	}()
}

func (b *CachedBinariesBatch) waitDownload(ctx context.Context, buildID string) error {
	if b.downloaded[buildID] {
		return nil
	}

	binary := b.acquiredBinaries[buildID]
	if binary == nil {
		return nil
	}

	err := binary.WaitStored(ctx)
	if err != nil {
		binary.Close()
		delete(b.acquiredBinaries, buildID)
		return err
	}

	b.downloaded[buildID] = true
	return nil
}

func (b *CachedBinariesBatch) WaitAllDownloads(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("Symbolizer").Start(
		ctx, "cachedbinaries.(*CachedBinariesBatch).WaitAllDownloads",
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	b.acquireGroup.Wait()

	buildIDs := make(map[string]bool, len(b.acquiredBinaries))
	for buildID := range b.acquiredBinaries {
		buildIDs[buildID] = true
	}

	for buildID := range buildIDs {
		err = b.waitDownload(ctx, buildID)

		if err != nil {
			b.l.Warn(
				ctx, "Failed to download binary",
				log.String("build_id", buildID),
				log.Error(err),
			)
		}

		if err != nil && errors.Is(err, context.Canceled) {
			// avoid waiting other downloads if context was cancelled
			break
		} else {
			err = nil
		}
	}

	return
}

func (b *CachedBinariesBatch) Path(mapping *pprof.Mapping) string {
	return b.PathByBuildID(mapping.BuildID)
}

func (b *CachedBinariesBatch) PathByBuildID(buildID string) string {
	if b.downloaded[buildID] {
		return b.acquiredBinaries[buildID].Path()
	}

	return ""
}

func (b *CachedBinariesBatch) Release() {
	for _, binary := range b.acquiredBinaries {
		binary.Close()
	}
}
