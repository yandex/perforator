package symbolize

import (
	"context"

	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
)

func (s *Symbolizer) scheduleBinaryDownloads(
	ctx context.Context,
	buildIDs []string,
	binaryProvider binaryprovider.BinaryProvider,
	logErrorOnFailedAcquire bool,
) (binaries *CachedBinariesBatch, err error) {
	binaries = NewCachedBinariesBatch(s.logger, binaryProvider, logErrorOnFailedAcquire)

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
