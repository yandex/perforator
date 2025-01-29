package binaryprocessing

import (
	"io"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
)

type UnwindTableStats struct {
	NumRows              int
	NumBytesCompressed   int
	NumBytesUncompressed int
}

type BinaryAnalysisStats struct {
	UnwindTableStats UnwindTableStats
}

func BuildBinaryAnalysis(path string, analysis *parse.BinaryAnalysis) (*BinaryAnalysisStats, error) {
	return analyzeBinary(path, analysis)
}

func LoadBinaryAnalysis(r io.Reader, analysis *parse.BinaryAnalysis) (*BinaryAnalysisStats, error) {
	return loadBinaryAnalysis(r, analysis)
}
