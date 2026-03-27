package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/unwindtable"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/parser"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func main() {
	l, err := xlog.ForCLI(xlog.CLIConfig{
		Level: log.DebugLevel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()
	if err := run(ctx, l); err != nil {
		l.Fatal(ctx, "Failed to analyze binary", log.Error(err))
	}
}

func run(ctx context.Context, l xlog.Logger) error {
	r := mock.NewRegistry(nil)

	binaryParserOptions := &parse.BinaryAnalysisOptions{}
	var preferSframe bool
	flag.BoolVar(&preferSframe, "prefer-sframe", false, "")
	if preferSframe {
		binaryParserOptions.PreferredUnwindInfoSource = parse.UnwindInfoSource_Sframe
	}
	unwindTableManager, err := unwindtable.NewBPFManager(l.Logger(), r, nil)
	if err != nil {
		return err
	}

	m, err := binary.NewBPFBinaryManager(l.Logger(), r, nil /* state */, unwindTableManager)
	if err != nil {
		return err
	}

	binaryParser, err := parser.NewBinaryParser(l, r, binaryParserOptions)
	if err != nil {
		return err
	}

	f, err := os.Open(flag.Args()[0])
	if err != nil {
		return err
	}
	defer f.Close()

	analysis, err := binaryParser.Parse(ctx, f)
	if err != nil {
		return err
	}

	a, err := m.Add(ctx, "nobuildid", 0, analysis)
	if err != nil {
		return err
	}

	l.Info(ctx, "Analyzed binary", log.Any("allocation", a))
	return nil
}
