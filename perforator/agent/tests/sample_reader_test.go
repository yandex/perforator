package profiler_test

import (
	"bytes"
	"context"
	"encoding/binary"
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
			storage = append(storage, sample)
		}
	}
	_, p := setupProfilerWithCallback(&simpleConfig, callback)
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

func TestParseFunctions(t *testing.T) {
	rawSamples := ensureRawSamples(t)
	var sampleParse, sampleBinaryRead, sampleUnmarshalUnsafe unwinder.RecordSample
	for j := 0; j < len(rawSamples); j++ {
		err := sampleParse.UnmarshalBinary(rawSamples[j])
		if err != nil {
			panic(err)
		}
		err = binary.Read(bytes.NewReader(rawSamples[j]), binary.LittleEndian, &sampleBinaryRead)
		if err != nil {
			panic(err)
		}
		if sampleParse != sampleBinaryRead {
			panic("Not correct")
		}
		err = sampleUnmarshalUnsafe.UnmarshalBinaryUnsafe(rawSamples[j])
		if err != nil {
			panic(err)
		}
		if sampleParse != sampleUnmarshalUnsafe {
			panic("field-by-field and unsafe cast parsing produced different results")
		}
	}
}

func BenchmarkParseRawSamplesTrivial(b *testing.B) {
	rawSamples := ensureRawSamples(b)
	var sample unwinder.RecordSample
	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < len(rawSamples); j++ {
			err := binary.Read(bytes.NewReader(rawSamples[j]), binary.LittleEndian, &sample)
			if err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkParseRawSamplesOptimized(b *testing.B) {
	rawSamples := ensureRawSamples(b)
	var sample unwinder.RecordSample
	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < len(rawSamples); j++ {
			err := sample.UnmarshalBinary(rawSamples[j])
			if err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkParseRawSamplesUnsafe(b *testing.B) {
	rawSamples := ensureRawSamples(b)
	var sample unwinder.RecordSample
	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < len(rawSamples); j++ {
			err := sample.UnmarshalBinaryUnsafe(rawSamples[j])
			if err != nil {
				panic(err)
			}
		}
	}
}
