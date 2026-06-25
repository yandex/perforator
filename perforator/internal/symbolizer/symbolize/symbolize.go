package symbolize

import (
	"context"
	"errors"
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
	"github.com/yandex/perforator/perforator/proto/symbolizer"
)

const (
	UnknownLine = "<unknown>"
)

var ErrUnknownBinary = errors.New("unknown binary")

// ErrSymbolization marks a binary that is present but could not be symbolized
// (e.g. an unparseable ELF/DWARF). Local fallback shares the same binary storage,
// so it cannot help — callers should treat it like ErrUnknownBinary, not retry it.
var ErrSymbolization = errors.New("symbolization failed")

type localSymbolizationPathProvider struct{}

func (*localSymbolizationPathProvider) Path(mapping *pprof.Mapping) string {
	return mapping.File
}

type symbolizerMetrics struct {
	symbolizationTimer        metrics.Timer
	unknownBinaries           metrics.Counter
	unsymbolizableLocations   metrics.Counter
	binariesWithGSYM          metrics.Counter
	binariesWithDWARFFallback metrics.Counter
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

// `location` must belong to given `profile` - since this function
// is used to apply symbolization result to profile
func AddLine(profile *pprof.Profile, location *pprof.Location, lineInfo *symbolizer.Frame, opts *perforator.SymbolizeOptions) {
	function := &pprof.Function{
		ID:         uint64(len(profile.Function)) + 1,
		Name:       lineInfo.DemangledFunctionName,
		SystemName: lineInfo.FunctionName,
		Filename:   lineInfo.FileName,
		StartLine:  int64(lineInfo.StartLine),
	}

	// Do not demangle function names, if requested.
	if opts != nil && opts.Demangle != nil && !*opts.Demangle {
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

func SkewedOffset(location *pprof.Location) uint64 {
	return location.Address + location.Mapping.Offset - location.Mapping.Start
}

func VisitUnsymbolizedLocations(profile *pprof.Profile, visit func(loc *pprof.Location, buildID string, offset uint64)) {
	for _, loc := range profile.Location {
		if len(loc.Line) > 0 || loc.Mapping == nil || loc.Mapping.BuildID == "" {
			continue
		}
		visit(loc, loc.Mapping.BuildID, SkewedOffset(loc))
	}
}

func getUnsymbolizedUniqueBuildIDs(profile *pprof.Profile) []string {
	unsymbolized := make(map[string]struct{})
	VisitUnsymbolizedLocations(profile, func(_ *pprof.Location, buildID string, _ uint64) {
		unsymbolized[buildID] = struct{}{}
	})

	buildIDs := make([]string, 0, len(unsymbolized))
	for buildID := range unsymbolized {
		buildIDs = append(buildIDs, buildID)
	}
	return buildIDs
}

// inplace symbolization using local binaries paths
func (s *Symbolizer) SymbolizeLocalProfile(
	ctx context.Context,
	profile *pprof.Profile,
	binaryPathProvider BinaryPathProvider,
	gsymBinaryPathProvider BinaryPathProvider,
) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.symbolizePprof(ctx, profile, binaryPathProvider, gsymBinaryPathProvider, nil)
}

func (s *Symbolizer) symbolizePprof(
	ctx context.Context,
	profile *pprof.Profile,
	pathProvider BinaryPathProvider,
	gsymPathProvider BinaryPathProvider,
	opts *perforator.SymbolizeOptions,
) error {
	start := time.Now()
	defer func() {
		s.pruneCaches()
		s.metrics.symbolizationTimer.RecordDuration(time.Since(start))
	}()

	s.logger.Debug(ctx, "Start symbolize")
	binaryErrors := make(map[string]error)
	VisitUnsymbolizedLocations(profile, func(location *pprof.Location, buildID string, offset uint64) {
		path, useGSYM, err := s.provideCachePath(ctx, gsymPathProvider, pathProvider, buildID, location.Mapping.File)
		if err != nil {
			return // unknown binary, already traced
		}

		frames, err := s.symbolizeLocation(ctx, buildID, offset, path, useGSYM)
		if err != nil {
			binaryErrors[buildID] = err // one bad binary must not block the rest
			return
		}
		for _, frame := range frames {
			AddLine(profile, location, frame, opts)
		}
	})

	if len(binaryErrors) > 0 {
		errs := make([]error, 0, len(binaryErrors))
		for _, err := range binaryErrors {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	return nil
}

func (s *Symbolizer) acquireBinaries(ctx context.Context, buildIDs []string) (
	gsymCachedBinaries *CachedBinariesBatch,
	cachedBinaries *CachedBinariesBatch,
	err error,
) {
	withGSYMLogger := s.logger.WithName("WithGSYM")
	gsymCachedBinaries = NewCachedBinariesBatch(withGSYMLogger, s.gsymBinaryProvider, false)
	if s.symbolizationMode == SymbolizationModeGSYMPreferred {
		gsymCachedBinaries, err = DownloadBinaries(ctx, withGSYMLogger, buildIDs, s.gsymBinaryProvider, false)
		if err != nil {
			return nil, nil, err
		}
	}

	buildIDsWithoutGSYM := make([]string, 0)
	for _, buildID := range buildIDs {
		if gsymCachedBinaries.PathByBuildID(buildID) == "" {
			buildIDsWithoutGSYM = append(buildIDsWithoutGSYM, buildID)
		}
	}
	cachedBinaries, err = DownloadBinaries(ctx, s.logger.WithName("WithoutGSYM"), buildIDsWithoutGSYM, s.binaryProvider, true)
	if err != nil {
		gsymCachedBinaries.Release()
		return nil, nil, err
	}
	s.metrics.binariesWithGSYM.Add(int64(len(buildIDs) - len(buildIDsWithoutGSYM)))
	s.metrics.binariesWithDWARFFallback.Add(int64(cachedBinaries.Count()))

	return gsymCachedBinaries, cachedBinaries, nil
}

func (s *Symbolizer) SymbolizeStorageProfile(
	ctx context.Context,
	profile *pprof.Profile,
	opts *perforator.SymbolizeOptions,
) (*pprof.Profile, error) {
	buildIDs := getUnsymbolizedUniqueBuildIDs(profile)

	gsymCachedBinaries, cachedBinaries, err := s.acquireBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}
	defer gsymCachedBinaries.Release()
	defer cachedBinaries.Release()

	_, span := otel.Tracer("Symbolizer").Start(ctx, "symbolize.(*Symbolizer).acquireSymbolizerLock")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer span.End()

	err = s.symbolizePprof(ctx, profile, cachedBinaries, gsymCachedBinaries, opts)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Symbolizer) provideCachePath(
	ctx context.Context,
	gsymCachedBinaries BinaryPathProvider,
	cachedBinaries BinaryPathProvider,
	buildID string,
	originalFile string,
) (path string, useGSYM bool, err error) {
	gsymPath := gsymCachedBinaries.PathByBuildID(buildID)
	if gsymPath != "" {
		return gsymPath, true, nil
	}

	path = cachedBinaries.PathByBuildID(buildID)
	if path != "" {
		return path, false, nil
	}

	s.traceUnknownBinary(ctx, buildID, originalFile)

	return "", false, ErrUnknownBinary
}

func (s *Symbolizer) traceUnknownBinary(ctx context.Context, buildID string, originalFile string) {
	s.logger.Trace(ctx, "Unknown binary",
		log.String("buildid", buildID),
		log.String("original_file", originalFile),
	)
	s.metrics.unknownBinaries.Inc()
}

// Symbolize returns results positional with addresses (empty Frames = unresolvable);
// the binary is acquired before the lock so downloads stay concurrent.
func (s *Symbolizer) Symbolize(ctx context.Context, buildID string, addresses []uint64) ([]*symbolizer.AddressResult, error) {
	gsymCachedBinaries, cachedBinaries, err := s.acquireBinaries(ctx, []string{buildID})
	if err != nil {
		return nil, err
	}
	defer gsymCachedBinaries.Release()
	defer cachedBinaries.Release()

	_, span := otel.Tracer("Symbolizer").Start(ctx, "symbolize.(*Symbolizer).Symbolize")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer span.End()

	path, useGSYM, err := s.provideCachePath(ctx, gsymCachedBinaries, cachedBinaries, buildID, "")
	if err != nil {
		return nil, err
	}

	lineInfoCnt := 0
	results := make([]*symbolizer.AddressResult, len(addresses))
	backing := make([]symbolizer.AddressResult, len(addresses))
	for i, offset := range addresses {
		frames, serr := s.symbolizeLocation(ctx, buildID, offset, path, useGSYM)
		if serr != nil {
			return nil, errors.Join(ErrSymbolization, serr)
		}
		backing[i].Frames = frames
		lineInfoCnt += len(frames)
		results[i] = &backing[i]
	}

	s.logger.Debug(ctx, "Symbolization stats", log.String("build_id", buildID), log.Int("line_count", lineInfoCnt))
	return results, nil
}
