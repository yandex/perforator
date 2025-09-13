package cluster_top

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/symbolize/symbolizec.h>
// #include <perforator/symbolizer/lib/cluster_top/cluster_top_c.h>
import "C"
import (
	"context"
	"math/big"
	"unsafe"

	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTopSymbolizer struct {
	l xlog.Logger

	gsymDownloader *downloader.GSYMDownloader
}

func NewClusterTopSymbolizer(l xlog.Logger, gsymDownloader *downloader.GSYMDownloader) (*ClusterTopSymbolizer, error) {
	return &ClusterTopSymbolizer{
		l:              l,
		gsymDownloader: gsymDownloader,
	}, nil
}

func (s *ClusterTopSymbolizer) DownloadAllGSYMs(
	ctx context.Context,
	buildIDs []string,
) (binaries *symbolize.CachedBinariesBatch, err error) {
	return symbolize.ScheduleBinaryDownloads(ctx, s.l, buildIDs, s.gsymDownloader, false)
}

type ServicePerfTopAggregator struct {
	symbolizer  *ClusterTopSymbolizer
	serviceName string

	gsyms *symbolize.CachedBinariesBatch

	aggregator unsafe.Pointer
}

func (s *ClusterTopSymbolizer) NewServicePerfTopAggregator(serviceName string) (*ServicePerfTopAggregator, error) {
	aggregator := C.MakeServicePerfTopAggregator()

	return &ServicePerfTopAggregator{
		symbolizer:  s,
		serviceName: serviceName,
		// gsyms is initialized later in InitializeSymbolizers
		aggregator: aggregator,
	}, nil
}

func (a *ServicePerfTopAggregator) Destroy() {
	C.DestroyServicePerfTopAggregator(a.aggregator)
}

func (a *ServicePerfTopAggregator) InitializeSymbolizers(ctx context.Context, buildIDs []string) error {
	gsyms, err := a.symbolizer.DownloadAllGSYMs(ctx, buildIDs)
	if err != nil {
		return err
	}

	a.InitializeSymbolizersWithGSYMs(gsyms, buildIDs)

	return nil
}

func (a *ServicePerfTopAggregator) InitializeSymbolizersWithGSYMs(
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

			C.InitializeSymbolizerForServicePerfTopAggregator(
				a.aggregator,
				cBuildID, C.ui64(len(buildID)),
				cPath, C.ui64(len(path)),
			)
		}
	}
}

func (a *ServicePerfTopAggregator) AddProfiles(
	ctx context.Context,
	profiles []profile.ProfileData,
) error {
	cServiceName := C.CString(a.serviceName)
	defer C.free(unsafe.Pointer(cServiceName))

	for _, profile := range profiles {
		if len(profile) == 0 {
			continue
		}

		C.AddProfileIntoServicePerfTopAggregator(
			a.aggregator,
			cServiceName,
			C.ui64(len(a.serviceName)),
			(*C.char)(unsafe.Pointer(&profile[0])),
			C.ui64(len(profile)),
		)

		if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (a *ServicePerfTopAggregator) MergeWith(otherA *ServicePerfTopAggregator) {
	C.MergeServicePerfTopAggregators(a.aggregator, otherA.aggregator)
}

func (a *ServicePerfTopAggregator) Extract() []Function {
	var cEntries C.ui64
	var cFunctions **C.char
	var cSelfCycles, cCumulativeCycles *C.char

	C.FinalizeServicePerfTopAggregator(
		a.aggregator,
		&cEntries,
		&cFunctions,
		&cSelfCycles,
		&cCumulativeCycles,
	)

	functionsArray := (*[1 << 30]*C.char)(unsafe.Pointer(cFunctions))
	selfCyclesArray := (*[1 << 30]byte)(unsafe.Pointer(cSelfCycles))
	cumulativeCyclesArray := (*[1 << 30]byte)(unsafe.Pointer(cCumulativeCycles))
	defer func() {
		for i := uint64(0); i < uint64(cEntries); i += 1 {
			C.free(unsafe.Pointer(functionsArray[i]))
		}
		C.free(unsafe.Pointer(cFunctions))

		C.free(unsafe.Pointer(cSelfCycles))
		C.free(unsafe.Pointer(cCumulativeCycles))
	}()

	functions := make([]Function, 0, int(cEntries))
	for i := uint64(0); i < uint64(cEntries); i += 1 {
		var selfCycles, cumulativeCycles big.Int

		selfCycles.SetBytes(selfCyclesArray[i*16 : (i+1)*16])
		cumulativeCycles.SetBytes(cumulativeCyclesArray[i*16 : (i+1)*16])

		functions = append(functions, Function{
			Name:             C.GoString(functionsArray[i]),
			SelfCycles:       selfCycles,
			CumulativeCycles: cumulativeCycles,
		})
	}

	return functions
}
