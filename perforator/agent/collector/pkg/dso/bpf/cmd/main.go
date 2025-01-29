package main

import (
	"context"
	"os"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/parser"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func main() {
	l := zap.Must(zap.TSKVConfig(log.DebugLevel))

	if err := run(context.Background(), xlog.New(l)); err != nil {
		l.Fatal("Failed to analyze binary", log.Error(err))
	}
}

func run(ctx context.Context, l xlog.Logger) error {
	r := mock.NewRegistry(nil)

	m, err := binary.NewBPFBinaryManager(l.Logger(), r, nil)
	if err != nil {
		return err
	}

	binaryParser, err := parser.NewBinaryParser(l, r)
	if err != nil {
		return err
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer f.Close()

	analysis, err := binaryParser.Parse(ctx, f)
	if err != nil {
		return err
	}

	a, err := m.Add("nobuildid", 0, analysis)
	if err != nil {
		return err
	}

	l.Info(ctx, "Analyzed binary", log.Any("allocation", a))
	return nil
}
