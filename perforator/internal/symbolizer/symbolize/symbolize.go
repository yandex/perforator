package symbolize

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/symbolize/symbolizec.h>
import "C"
import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/otel"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

const (
	UnknownLine = "<unknown>"
)

type BinaryPathProvider interface {
	Path(mapping *pprof.Mapping) string
}

type localSymbolizationPathProvider struct{}

func (*localSymbolizationPathProvider) Path(mapping *pprof.Mapping) string {
	return mapping.File
}

type nilPathProvider struct{}

func (*nilPathProvider) Path(*pprof.Mapping) string {
	return ""
}

type symbolizerMetrics struct {
	symbolizationTimer      metrics.Timer
	unknownBinaries         metrics.Counter
	unsymbolizableLocations metrics.Counter
}

type SymbolizationMode int

const (
	SymbolizationModeDWARF = iota
	SymbolizationModeGSYMPreferred
)

type Symbolizer struct { // thread-safe
	logger            xlog.Logger
	symbolizationMode SymbolizationMode

	binaryProvider     binaryprovider.BinaryProvider
	gsymBinaryProvider binaryprovider.BinaryProvider
	symbolizer         unsafe.Pointer
	metrics            *symbolizerMetrics

	mutex sync.Mutex
}

func NewSymbolizer(
	logger xlog.Logger,
	reg metrics.Registry,
	binaryProvider binaryprovider.BinaryProvider,
	gsymBinaryProvider binaryprovider.BinaryProvider,
	symbolizationMode SymbolizationMode,
) (*Symbolizer, error) {
	var errPtr *C.char = nil
	var symbolizer unsafe.Pointer = C.MakeSymbolizer(&errPtr)
	if errPtr != nil {
		return nil, errors.New(C.GoString(errPtr))
	}

	reg = reg.WithPrefix("symbolizer")

	return &Symbolizer{
		logger:             logger,
		symbolizationMode:  symbolizationMode,
		binaryProvider:     binaryProvider,
		gsymBinaryProvider: gsymBinaryProvider,
		symbolizer:         symbolizer,
		metrics: &symbolizerMetrics{
			symbolizationTimer:      reg.Timer("symbolization.timer"),
			unknownBinaries:         reg.Counter("unknown_binaries.count"),
			unsymbolizableLocations: reg.Counter("unsymbolizable_locations.count"),
		},
	}, nil
}

func (s *Symbolizer) Destroy() {
	C.DestroySymbolizer(s.symbolizer)
}

func addLine(profile *pprof.Profile, location *pprof.Location, lineInfo *C.TLineInfo, opts *perforator.SymbolizeOptions) {
	function := &pprof.Function{
		ID:         uint64(len(profile.Function)) + 1,
		Name:       C.GoString(lineInfo.DemangledFunctionName),
		SystemName: C.GoString(lineInfo.FunctionName),
		Filename:   C.GoString(lineInfo.FileName),
		StartLine:  int64(lineInfo.StartLine),
	}

	// Do not demangle function names, if requested.
	if d := opts.Demangle; d != nil && !*d {
		function.Name = function.SystemName
	}

	profile.Function = append(
		profile.Function,
		function,
	)

	line := uint64(lineInfo.Line)
	if opts != nil && opts.GetEmbedDwarfDiscriminators() {
		line |= uint64(lineInfo.Discriminator) << 32
	}

	location.Line = append(
		location.Line,
		pprof.Line{
			Function: function,
			Line:     int64(line),
		},
	)
}

func getUniqueBuildIDs(ctx context.Context, profile *pprof.Profile, logger xlog.Logger) []string {
	uniqueBuildIDS := map[string]struct{}{}

	for i, mapping := range profile.Mapping {
		if mapping == nil {
			continue
		}

		l := logger.With(
			log.Int("i", i),
			log.String("build_id", mapping.BuildID),
			log.String("path", mapping.File),
			log.UInt64("start", mapping.Start),
			log.UInt64("limit", mapping.Limit),
			log.UInt64("offset", mapping.Offset),
		)

		l.Debug(ctx, "Found mapping")

		if mapping.BuildID == "" {
			continue
		}

		uniqueBuildIDS[mapping.BuildID] = struct{}{}
	}

	buildIDs := make([]string, 0, len(uniqueBuildIDS))
	for buildID, _ := range uniqueBuildIDS {
		buildIDs = append(buildIDs, buildID)
	}

	return buildIDs
}

func (s *Symbolizer) symbolize(
	ctx context.Context,
	profile *pprof.Profile,
	pathProvider BinaryPathProvider,
	gsymPathProvider BinaryPathProvider,
	opts *perforator.SymbolizeOptions,
) error {
	start := time.Now()
	defer func() {
		C.PruneCaches(s.symbolizer)
		s.metrics.symbolizationTimer.RecordDuration(time.Since(start))
	}()

	s.logger.Debug(ctx, "Start symbolize")
	for _, location := range profile.Location {
		s.symbolizeLocation(ctx, location, profile, pathProvider, gsymPathProvider, opts)
	}
	return nil
}

func (s *Symbolizer) symbolizeLocation(
	ctx context.Context,
	location *pprof.Location,
	profile *pprof.Profile,
	pathProvider BinaryPathProvider,
	gsymPathProvider BinaryPathProvider,
	opts *perforator.SymbolizeOptions,
) {
	// Skip symbolized code.
	if len(location.Line) > 0 {
		return
	}

	if location.Mapping == nil {
		return
	}

	path := pathProvider.Path(location.Mapping)
	address := location.Address + location.Mapping.Offset - location.Mapping.Start

	useGsym := C.ui32(0)
	gsymPath := gsymPathProvider.Path(location.Mapping)
	if gsymPath != "" {
		path = gsymPath
		useGsym = C.ui32(1)
	}

	if path == "" {
		s.logger.Trace(ctx, "Unknown binary",
			log.String("buildid", location.Mapping.BuildID),
			log.String("address", fmt.Sprintf("%x", location.Address)),
		)
		s.metrics.unknownBinaries.Inc()
		return
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	linesCount := C.ui64(0)
	var errPtr *C.char = nil

	lines := C.Symbolize(
		s.symbolizer,
		cpath,
		C.ulong(len(path)),
		C.ui64(address),
		&linesCount,
		&errPtr,
		useGsym,
	)
	if errPtr != nil {
		s.logger.Error(ctx, "Failed to symbolize code", log.String("error", C.GoString(errPtr)))
		s.metrics.unsymbolizableLocations.Inc()
		return
	}
	defer C.DestroySymbolizeResult(lines, linesCount)

	if linesCount == 0 {
		return
	}

	location.Line = []pprof.Line{}
	linesSlice := unsafe.Slice(lines, linesCount)
	for _, lineInfo := range linesSlice {
		addLine(profile, location, &lineInfo, opts)
	}
}

// inplace symbolization using local binaries paths
func (s *Symbolizer) SymbolizeLocalProfile(ctx context.Context, profile *pprof.Profile) error {
	return s.symbolize(ctx, profile, &localSymbolizationPathProvider{}, &nilPathProvider{}, nil)
}

func (s *Symbolizer) SymbolizeStorageProfile(
	ctx context.Context,
	profile *pprof.Profile,
	opts *perforator.SymbolizeOptions,
) (*pprof.Profile, error) {
	buildIDs := getUniqueBuildIDs(ctx, profile, s.logger)
	var err error

	gsymCachedBinaries := NewCachedBinariesBatch(s.logger, s.gsymBinaryProvider, false)
	if s.symbolizationMode == SymbolizationModeGSYMPreferred {
		gsymCachedBinaries, err = s.scheduleBinaryDownloads(ctx, buildIDs, s.gsymBinaryProvider, false)
		if err != nil {
			return nil, err
		}
	}
	defer gsymCachedBinaries.Release()

	buildIDsWithoutGSYM := make([]string, 0)
	for _, buildID := range buildIDs {
		if gsymCachedBinaries.PathByBuildID(buildID) == "" {
			buildIDsWithoutGSYM = append(buildIDsWithoutGSYM, buildID)
		}
	}
	cachedBinaries, err := s.scheduleBinaryDownloads(ctx, buildIDsWithoutGSYM, s.binaryProvider, true)
	if err != nil {
		return nil, err
	}
	defer cachedBinaries.Release()

	_, span := otel.Tracer("Symbolizer").Start(ctx, "symbolize.(*Symbolizer).acquireSymbolizerLock")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	span.End()

	err = s.symbolize(ctx, profile, cachedBinaries, gsymCachedBinaries, opts)
	if err != nil {
		return nil, err
	}

	return profile, nil
}
