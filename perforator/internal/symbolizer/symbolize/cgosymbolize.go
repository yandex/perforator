package symbolize

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/symbolize/symbolizec.h>
import "C"
import (
	"context"
	"errors"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func newLineInfo(buildID string, addr uint64, lineInfo *C.TLineInfo) *LineInfo {
	return &LineInfo{
		BuildID: buildID,
		Address: addr,
		ProtoLine: &ProtoLine{
			DemangledFunctionName: C.GoString(lineInfo.DemangledFunctionName),
			FunctionName:          C.GoString(lineInfo.FunctionName),
			Filename:              C.GoString(lineInfo.FileName),
			StartLine:             uint64(lineInfo.StartLine),
			Line:                  uint64(lineInfo.Line),
			Column:                uint64(lineInfo.Column),
			Discriminator:         uint64(lineInfo.Discriminator),
		},
	}
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
			symbolizationTimer:        reg.Timer("symbolization.timer"),
			unknownBinaries:           reg.Counter("unknown_binaries.count"),
			unsymbolizableLocations:   reg.Counter("unsymbolizable_locations.count"),
			binariesWithDWARFFallback: reg.Counter("binaries_with_dwarf_fallback.count"),
			binariesWithGSYM:          reg.Counter("binaries_with_gsym.count"),
		},
	}, nil
}

func (s *Symbolizer) Destroy() {
	C.DestroySymbolizer(s.symbolizer)
}

func (s *Symbolizer) pruneCaches() {
	C.PruneCaches(s.symbolizer)
}

func (s *Symbolizer) symbolizeLocation(
	ctx context.Context,
	buildID string,
	address uint64,
	path string,
	useGSYM bool,
) ([]*LineInfo, error) {
	cUseGSYM := C.ui32(0)
	if useGSYM {
		cUseGSYM = C.ui32(1)
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
		cUseGSYM,
	)
	if errPtr != nil {
		errStr := C.GoString(errPtr)
		s.logger.Error(ctx, "Failed to symbolize code",
			log.String("error", errStr),
			log.String("build_id", buildID),
			log.String("path", path),
			log.UInt64("address", address),
			log.Bool("useGSYM", useGSYM))
		s.metrics.unsymbolizableLocations.Inc()
		return nil, errors.New(errStr)
	}
	defer C.DestroySymbolizeResult(lines, linesCount)

	if linesCount == 0 {
		return nil, nil
	}

	res := make([]*LineInfo, linesCount)
	linesSlice := unsafe.Slice(lines, linesCount)
	for i, lineInfo := range linesSlice {
		res[i] = newLineInfo(buildID, address, &lineInfo)
	}

	return res, nil
}
