package symbolize

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

// intended for single thread use
// first acquire needed binaries then wait for them
type CachedBinariesBatch struct {
	l                xlog.Logger
	binaryProvider   binaryprovider.BinaryProvider
	mutex            sync.RWMutex
	acquiredBinaries map[string]binaryprovider.FileHandle // guarded by mutex
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

// Acquire blocks until the binary is downloaded and ready (or fails). A failure
// is logged, not fatal: symbolization proceeds with whatever binaries it got.
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
	return ctx.Err()
}

func (b *CachedBinariesBatch) PathByBuildID(buildID string) string {
	if handle := b.acquiredBinaries[buildID]; handle != nil {
		return handle.Path()
	}

	return ""
}

func (b *CachedBinariesBatch) Count() int {
	return len(b.acquiredBinaries)
}

func (b *CachedBinariesBatch) Release() {
	for _, binary := range b.acquiredBinaries {
		binary.Close()
	}
}
