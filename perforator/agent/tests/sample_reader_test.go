package profiler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

var (
	sampleLimit  = 1000
	simpleConfig = config.Config{
		Debug: false,
		BPF: machine.Config{
			PageTableSizeKB: &PageTableSizeKB,
		},
	}
)

func setupProfilerWithCallback(c *config.Config, sampleCallback machine.RawSampleCallback) (log.Logger, *profiler.Profiler) {
	lconf := zap.KVConfig(log.DebugLevel)
	lconf.OutputPaths = []string{"stderr"}
	l := zap.Must(lconf)

	r := nop.Registry{}

	p, err := profiler.NewProfiler(c, l, r.WithPrefix("profiler"), profiler.WithRawSampleCallback(sampleCallback))

	if err != nil {
		panic(err)
	}

	return l, p
}

func startProfilerAndCollectRawSamples() [][]byte {
	var storage [][]byte
	callback := func(sample []byte) {
		if len(storage) < sampleLimit {
			// The slice is backed by a buffer that is reused by the perf reader
			// on the next read, so we must copy the bytes before storing it.
			cp := make([]byte, len(sample))
			copy(cp, sample)
			storage = append(storage, cp)
		}
	}
	_, p := setupProfilerWithCallback(&simpleConfig, callback)
	defer p.Close()
	err := p.TraceWholeSystem(nil)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	err = p.Run(ctx)
	if !errors.Is(err, context.Cause(ctx)) {
		panic(err)
	}
	return storage
}

var (
	collectSamplesOnce sync.Once
	collectedSamples   [][]byte
)

func ensureRawSamples(t testing.TB) [][]byte {
	t.Helper()
	collectSamplesOnce.Do(func() {
		collectedSamples = startProfilerAndCollectRawSamples()
	})
	if len(collectedSamples) == 0 {
		t.Fatal("no raw samples were collected")
	}
	return collectedSamples
}

func TestParsePackedSample(t *testing.T) {
	rawSamples := ensureRawSamples(t)
	parsed := unwinder.NewRecordSampleParsed()
	for j := 0; j < len(rawSamples); j++ {
		err := unwinder.ParsePackedSample(rawSamples[j], parsed)
		if err != nil {
			t.Fatalf("failed to parse packed sample %d: %v", j, err)
		}
	}
}

func BenchmarkParsePackedSample(b *testing.B) {
	rawSamples := ensureRawSamples(b)
	parsed := unwinder.NewRecordSampleParsed()
	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < len(rawSamples); j++ {
			err := unwinder.ParsePackedSample(rawSamples[j], parsed)
			if err != nil {
				panic(err)
			}
		}
	}
}
