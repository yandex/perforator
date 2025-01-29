package binaryprocessing

// #include "ehframe.h"
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"unsafe"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
)

func analyzeBinary(path string, analysis *parse.BinaryAnalysis) (*BinaryAnalysisStats, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	res := C.build_binary_analysis(cpath)
	if res.err != nil {
		defer C.binary_analysis_free_error(res.err)
		text := C.binary_analysis_error_text(res.err)
		return nil, fmt.Errorf("failed to build binary analysis: %s", C.GoString(text))
	}
	if res.buf == nil {
		return nil, fmt.Errorf("failed to build binary analysis: no error")
	}
	defer C.binary_analysis_free(res.buf)

	buf := C.GoBytes(unsafe.Pointer(res.buf), res.len)

	return LoadBinaryAnalysis(bytes.NewBuffer(buf), analysis)
}

func loadBinaryAnalysis(r io.Reader, analysis *parse.BinaryAnalysis) (*BinaryAnalysisStats, error) {
	cr := countingReader{r, 0}

	zr, err := zstd.NewReader(&cr)
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(buf, analysis)
	if err != nil {
		return nil, err
	}

	err = integrateDeltaEncodedTable(analysis.UnwindTable)
	if err != nil {
		return nil, err
	}

	return &BinaryAnalysisStats{
		UnwindTableStats: UnwindTableStats{
			NumRows:              len(analysis.UnwindTable.GetStartPc()),
			NumBytesUncompressed: len(buf),
			NumBytesCompressed:   cr.Count(),
		},
	}, nil
}

func integrateDeltaEncodedTable(table *unwind.UnwindTable) error {
	pc := uint64(0)
	for i := range table.GetStartPc() {
		table.GetStartPc()[i] += pc
		pc = table.GetStartPc()[i] + table.GetPcRange()[i]
	}
	return nil
}
