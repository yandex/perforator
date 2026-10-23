package cluster_top

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/symbolize/symbolizec.h>
// #include <perforator/symbolizer/lib/cluster_top/cluster_top_c.h>
import "C"
import (
	"context"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTopSymbolizer struct {
	l xlog.Logger

	gsymDownloader binaryprovider.BinaryProvider
}

func NewClusterTopSymbolizer(l xlog.Logger, gsymDownloader binaryprovider.BinaryProvider) (*ClusterTopSymbolizer, error) {
	return &ClusterTopSymbolizer{
		l:              l,
		gsymDownloader: gsymDownloader,
	}, nil
}

func (s *ClusterTopSymbolizer) DownloadAllGSYMs(
	ctx context.Context,
	buildIDs []string,
) (binaries *symbolize.CachedBinariesBatch, err error) {
	return symbolize.DownloadBinaries(ctx, s.l, buildIDs, s.gsymDownloader, false)
}

type PerfTopAggregator struct {
	symbolizer *ClusterTopSymbolizer

	gsyms *symbolize.CachedBinariesBatch

	aggregator unsafe.Pointer
}

func (s *ClusterTopSymbolizer) NewPerfTopAggregator() (*PerfTopAggregator, error) {
	aggregator := C.MakePerfTopAggregator()

	return &PerfTopAggregator{
		symbolizer: s,
		// gsyms is initialized later in InitializeSymbolizers
		aggregator: aggregator,
	}, nil
}

func (a *PerfTopAggregator) Destroy() {
	C.DestroyPerfTopAggregator(a.aggregator)
}

func (a *PerfTopAggregator) InitializeSymbolizers(ctx context.Context, buildIDs []string) error {
	gsyms, err := a.symbolizer.DownloadAllGSYMs(ctx, buildIDs)
	if err != nil {
		return err
	}

	a.InitializeSymbolizersWithGSYMs(gsyms, buildIDs)

	return nil
}

func (a *PerfTopAggregator) InitializeSymbolizersWithGSYMs(
	gsyms *symbolize.CachedBinariesBatch,
	buildIDs []string,
) {
	a.gsyms = gsyms

	for _, buildID := range buildIDs {
		path := gsyms.PathByBuildID(buildID)
		if path != "" {
			cBuildID := C.CString(buildID)
			defer C.free(unsafe.Pointer(cBuildID))

			cPath := C.CString(path)
			defer C.free(unsafe.Pointer(cPath))

			C.InitializeSymbolizerForPerfTopAggregator(
				a.aggregator,
				cBuildID, C.ui64(len(buildID)),
				cPath, C.ui64(len(path)),
			)
		}
	}
}

func (a *PerfTopAggregator) AddYaprofProfiles(
	ctx context.Context,
	profiles [][]byte,
) error {
	for _, profile := range profiles {
		if len(profile) == 0 {
			continue
		}

		profileErr := C.AddYaprofIntoPerfTopAggregator(
			a.aggregator,
			(*C.char)(unsafe.Pointer(&profile[0])),
			C.ui64(len(profile)),
		)

		if profileErr != nil {
			message := C.GoString(C.PerforatorErrorString(profileErr))
			C.PerforatorErrorDispose(profileErr)
			return fmt.Errorf("failed to aggregate profile: %s", message)
		}

		if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (a *PerfTopAggregator) MergeWith(otherA *PerfTopAggregator) {
	C.MergePerfTopAggregators(a.aggregator, otherA.aggregator)
}

func (a *PerfTopAggregator) Extract() []Function {
	var cEntries C.ui64
	var cFunctions *C.TClusterTopFunction

	C.FinalizePerfTopAggregator(
		a.aggregator,
		&cEntries,
		&cFunctions,
	)

	functionsArray := unsafe.Slice(cFunctions, int(cEntries))
	defer C.FreePerfTopAggregatorResult(cEntries, cFunctions)

	functions := make([]Function, 0, int(cEntries))
	for _, function := range functionsArray {
		var selfCycles, cumulativeCycles big.Int

		selfCycles.SetBytes(unsafe.Slice((*byte)(unsafe.Pointer(&function.SelfCycles[0])), 16))
		cumulativeCycles.SetBytes(unsafe.Slice((*byte)(unsafe.Pointer(&function.CumulativeCycles[0])), 16))

		functions = append(functions, Function{
			Name:             C.GoString(function.Name),
			BuildID:          C.GoString(function.BuildId),
			SourceFile:       C.GoString(function.SourceFile),
			SourceLine:       uint32(function.SourceLine),
			SourceColumn:     uint32(function.SourceColumn),
			SelfCycles:       selfCycles,
			CumulativeCycles: cumulativeCycles,
		})
	}

	return functions
}
