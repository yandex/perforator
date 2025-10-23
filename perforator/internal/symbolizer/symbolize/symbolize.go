package symbolize

import (
	"context"
	"errors"
	"sync"
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

var errUnknownBinary = errors.New("Unknown binary")

type localSymbolizationPathProvider struct{}

func (*localSymbolizationPathProvider) Path(mapping *pprof.Mapping) string {
	return mapping.File
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

// `location` must belong to given `profile` - since this function
// is used to apply symbolization result to profile
func AddLine(profile *pprof.Profile, location *pprof.Location, lineInfo *symbolizer.Line, opts *perforator.SymbolizeOptions) {
	function := &pprof.Function{
		ID:         uint64(len(profile.Function)) + 1,
		Name:       lineInfo.DemangledFunctionName,
		SystemName: lineInfo.FunctionName,
		Filename:   lineInfo.Filename,
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

func getUnsymbolizedUniqueBuildIDs(ctx context.Context, profile *pprof.Profile, logger xlog.Logger) []string {
	uniqueBuildIDs := make(map[string]struct{})

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

		uniqueBuildIDs[mapping.BuildID] = struct{}{}
	}

	buildIDs := make([]string, 0, len(uniqueBuildIDs))
	for buildID := range uniqueBuildIDs {
		buildIDs = append(buildIDs, buildID)
	}

	unsymbolizedBuildIDs := make(map[string]struct{})
	for _, loc := range profile.Location {
		if loc.Mapping != nil && len(loc.Line) == 0 {
			unsymbolizedBuildIDs[loc.Mapping.BuildID] = struct{}{}
		}
	}

	toRemove := make([]string, 0)
	for buildID := range uniqueBuildIDs {
		if _, ok := unsymbolizedBuildIDs[buildID]; !ok {
			toRemove = append(toRemove, buildID)
		}
	}

	for _, buildID := range toRemove {
		delete(uniqueBuildIDs, buildID)
		logger.Debug(ctx, "already symbolized", log.String("buildID", buildID))
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

func (s *Symbolizer) acquireBinaries(ctx context.Context, buildIDs []string) (
	gsymCachedBinaries *CachedBinariesBatch,
	cachedBinaries *CachedBinariesBatch,
	err error,
) {
	withGSYMLogger := s.logger.WithName("WithGSYM")
	gsymCachedBinaries = NewCachedBinariesBatch(withGSYMLogger, s.gsymBinaryProvider, false)
	if s.symbolizationMode == SymbolizationModeGSYMPreferred {
		gsymCachedBinaries, err = ScheduleBinaryDownloads(ctx, withGSYMLogger, buildIDs, s.gsymBinaryProvider, false)
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
	cachedBinaries, err = ScheduleBinaryDownloads(ctx, s.logger.WithName("WithoutGSYM"), buildIDsWithoutGSYM, s.binaryProvider, true)
	if err != nil {
		gsymCachedBinaries.Release()
		return nil, nil, err
	}

	return gsymCachedBinaries, cachedBinaries, nil
}

func (s *Symbolizer) SymbolizeStorageProfile(
	ctx context.Context,
	profile *pprof.Profile,
	opts *perforator.SymbolizeOptions,
) (*pprof.Profile, error) {
	buildIDs := getUnsymbolizedUniqueBuildIDs(ctx, profile, s.logger)

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

func getUniqueBuildIDsFromBatch(batch []*symbolizer.PerBinaryRequest) (buildIDs []string, buildIDOffsets map[string][]uint64) {
	buildIDOffsets = make(map[string][]uint64)
	for _, binaryRequest := range batch {
		buildIDOffsets[binaryRequest.BuildID] = append(buildIDOffsets[binaryRequest.BuildID], binaryRequest.Offsets...)
	}

	buildIDs = make([]string, 0, len(buildIDOffsets))
	for buildID := range buildIDOffsets {
		buildIDs = append(buildIDs, buildID)
	}

	return buildIDs, buildIDOffsets
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

	return "", false, errUnknownBinary
}

func (s *Symbolizer) traceUnknownBinary(ctx context.Context, buildID string, originalFile string) {
	s.logger.Trace(ctx, "Unknown binary",
		log.String("buildid", buildID),
		log.String("original_file", originalFile),
	)
	s.metrics.unknownBinaries.Inc()
}

func (s *Symbolizer) SymbolizeBatch(
	ctx context.Context,
	batch []*symbolizer.PerBinaryRequest,
) ([]*symbolizer.PerBinaryResponse, error) {
	buildIDs, buildIDOffsets := getUniqueBuildIDsFromBatch(batch)

	s.logger.Debug(ctx, "unique buildIDs", log.Array("buildIDs", buildIDs))

	res := make([]*symbolizer.PerBinaryResponse, len(buildIDs))

	_, span := otel.Tracer("Symbolizer").Start(ctx, "symbolize.(*Symbolizer).acquireSymbolizerLock")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer span.End()

	gsymCachedBinaries, cachedBinaries, err := s.acquireBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}
	defer gsymCachedBinaries.Release()
	defer cachedBinaries.Release()

	for i, buildID := range buildIDs {
		res[i] = &symbolizer.PerBinaryResponse{
			BuildID: buildID,
		}

		path, useGSYM, err := s.provideCachePath(ctx, gsymCachedBinaries, cachedBinaries, buildID, "")
		if err != nil {
			res[i].Error = err.Error()
			continue
		}

		lineInfoCnt := 0
		for _, offset := range buildIDOffsets[buildID] {
			loc := &symbolizer.LocationSymbolizationResult{
				AddressType: &symbolizer.LocationSymbolizationResult_ELFOffset{
					ELFOffset: offset,
				},
			}

			lineInfos, err := s.symbolizeLocation(ctx, buildID, offset, path, useGSYM)
			if err != nil {
				loc.Error = err.Error()
			} else {
				for _, lineInfo := range lineInfos {
					loc.Lines = append(loc.Lines, lineInfo.ProtoLine)
				}
				lineInfoCnt += len(lineInfos)
			}

			res[i].Locations = append(res[i].Locations, loc)
		}

		s.logger.Debug(ctx, "Symbolization stats", log.String("build_id", buildID), log.Int("line_count", lineInfoCnt))
	}

	return res, nil
}
