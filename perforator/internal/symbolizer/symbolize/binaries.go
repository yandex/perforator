package symbolize

import (
	"context"

	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func ScheduleBinaryDownloads(
	ctx context.Context,
	l xlog.Logger,
	buildIDs []string,
	binaryProvider binaryprovider.BinaryProvider,
	logErrorOnFailedAcquire bool,
) (binaries *CachedBinariesBatch, err error) {
	binaries = NewCachedBinariesBatch(l, binaryProvider, logErrorOnFailedAcquire)

	ctx, span := otel.Tracer("Symbolizer").Start(
		ctx, "symbolize.(*Symbolizer).prepareBinaries",
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	defer func() {
		if err != nil {
			binaries.Release()
		}
	}()

	for _, buildID := range buildIDs {
		binaries.AcquireAsync(ctx, buildID)
	}

	err = binaries.WaitAllDownloads(ctx)
	if err != nil {
		return
	}

	return
}
