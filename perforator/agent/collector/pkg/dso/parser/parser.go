package parser

import (
	"context"
	"fmt"
	"os"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/preprocessing/lib/go/binaryprocessing"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type parserMetrics struct {
	okAnalyzedBinaries     metrics.Counter
	failedAnalyzedBinaries metrics.Counter
}

type BinaryParser struct {
	l xlog.Logger
	r metrics.Registry

	metrics *parserMetrics
	options *parse.BinaryAnalysisOptions
}

func NewBinaryParser(l xlog.Logger, r metrics.Registry, options *parse.BinaryAnalysisOptions) (*BinaryParser, error) {
	if options == nil {
		l.Info(context.Background(), "Create a new BinaryParser with default options, as the user has not provided a custom one")
		options = &parse.BinaryAnalysisOptions{}
	}

	return &BinaryParser{
		l: l,
		r: r,
		metrics: &parserMetrics{
			okAnalyzedBinaries:     r.WithTags(map[string]string{"status": "ok"}).Counter("analyzed_binaries.count"),
			failedAnalyzedBinaries: r.WithTags(map[string]string{"status": "failed"}).Counter("analyzed_binaries.count"),
		},
		options: options,
	}, nil
}

func (p *BinaryParser) Parse(ctx context.Context, f *os.File) (res *parse.BinaryAnalysis, err error) {
	defer func() {
		if err != nil {
			p.metrics.failedAnalyzedBinaries.Inc()
		} else {
			p.metrics.okAnalyzedBinaries.Inc()
		}
	}()

	res = &parse.BinaryAnalysis{}
	stats, err := binaryprocessing.BuildBinaryAnalysis(
		fmt.Sprintf("/proc/self/fd/%d", f.Fd()),
		res,
		p.options,
	)
	if err != nil {
		p.l.Debug(ctx, "Failed to analyze binary", log.Error(err))
		return
	}

	p.l.Debug(
		ctx,
		"Analyzed binary",
		log.Int("unwtable_rows", stats.UnwindTableStats.NumRows),
		log.Int("unwtable_compressed", stats.UnwindTableStats.NumBytesCompressed),
		log.Int("unwtable_uncompressed", stats.UnwindTableStats.NumBytesUncompressed),
		log.Any("python_config", res.PythonConfig),
		log.Any("pthread_config", res.PthreadConfig),
		log.Any("php_config", res.PhpConfig),
	)

	return
}
