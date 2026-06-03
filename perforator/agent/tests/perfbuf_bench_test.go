package profiler_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	storage "github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

type perfbufBenchConfig struct {
	name                 string
	perfBufferPerCPUSize int
	perfBufferWatermark  int
	concurrency          int
}

var (
	concurrenciesToTry = []int{4, 8, 16}
)

var perfbufBaseBenchConfigs = [3]perfbufBenchConfig{
	{
		name:                 "BufSize_16MB/Watermark_50KB",
		perfBufferPerCPUSize: 16 * 1024 * 1024,
		perfBufferWatermark:  50 * 1024,
	},
	{
		name:                 "BufSize_16MB/Watermark_200KB",
		perfBufferPerCPUSize: 16 * 1024 * 1024,
		perfBufferWatermark:  200 * 1024,
	},
	{
		name:                 "BufSize_16MB/Watermark_1MB",
		perfBufferPerCPUSize: 16 * 1024 * 1024,
		perfBufferWatermark:  1024 * 1024,
	},
}

var perfBufBenchConfigs []perfbufBenchConfig

func init() {
	for _, bc := range perfbufBaseBenchConfigs {
		for _, concurrency := range concurrenciesToTry {
			perfBufBenchConfigs = append(perfBufBenchConfigs, perfbufBenchConfig{
				name:                 bc.name + fmt.Sprintf("/Concurrency_%d", concurrency),
				perfBufferPerCPUSize: bc.perfBufferPerCPUSize,
				perfBufferWatermark:  bc.perfBufferWatermark,
				concurrency:          concurrency,
			})
		}
	}
}

const (
	collectDuration = 10 * time.Second
	profilesDir     = "bench_profiles"
)

func profilesDirPath() string {
	return yatest.OutputPath(profilesDir)
}

func BenchmarkPerfbufThroughput(b *testing.B) {
	prepareEnv(b)

	binaryPath, err := yatest.BinaryPath("perforator/agent/tests/dummies/perfbuf_bench/perfbuf_bench")
	require.NoError(b, err)

	require.NoError(b, os.MkdirAll(profilesDirPath(), 0755))

	for _, bc := range perfBufBenchConfigs {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			runPerfbufBenchmark(b, binaryPath, bc)
		})
	}
}

func runPerfbufBenchmark(b *testing.B, binaryPath string, bc perfbufBenchConfig) {
	b.Helper()

	// Start CPU profiling for this sub-benchmark.
	sanitized := strings.NewReplacer("/", "_", " ", "_").Replace(bc.name)
	profPath := filepath.Join(profilesDirPath(), sanitized+".cpu.prof")
	profFile, err := os.Create(profPath)
	require.NoError(b, err)
	require.NoError(b, pprof.StartCPUProfile(profFile))
	defer func() {
		pprof.StopCPUProfile()
		closeErr := profFile.Close()
		if closeErr != nil {
			b.Logf("Failed to close pprof file: %v", closeErr)
		}
		b.Logf("CPU profile saved to %s", profPath)
	}()

	cfg := config.Config{
		InMemoryStorage: &storage.InMemoryStorageConfig{
			Watermark: 1000,
		},
		BPF: machine.Config{
			TraceLBR:      ptr.Bool(false),
			TraceWallTime: ptr.Bool(false),
			TraceSignals:  ptr.Bool(false),
		},
		PerfEvents: []config.PerfEventConfig{},
		SampleConsumer: config.SampleConsumerConfig{
			PerfBufferPerCPUSize: ptr.Int(bc.perfBufferPerCPUSize),
			PerfBufferWatermark:  ptr.Int(bc.perfBufferWatermark),
		},
	}

	l, r, _, _, p := setupProfiler(b, cfg)
	defer p.Close()

	// Launch the load generator binary.
	subprocess := exec.Command(binaryPath, "--duration", "120", "--concurrency", fmt.Sprintf("%d", bc.concurrency))
	subprocess.Stderr = os.Stderr

	require.NoError(b, subprocess.Start())
	defer subprocess.Process.Kill()

	ctx := b.Context()

	_, err = p.TracePid(linux.CurrentNamespacePID(subprocess.Process.Pid))
	require.NoError(b, err)

	err = p.Start(ctx)
	require.NoError(b, err)

	// Dynamically create and attach uprobe after profiler has started.
	u := p.UprobeManager().Create(uprobe.Config{
		Path:   binaryPath,
		Symbol: "target_func",
		Pid:    linux.CurrentNamespacePID(subprocess.Process.Pid),
	})
	err = u.Attach()
	require.NoError(b, err)
	defer u.Close()

	// Collect samples for a fixed duration.
	time.Sleep(collectDuration)

	err = p.Stop(ctx)
	require.NoError(b, err)

	// Read metrics.
	collected := metric(r, "profiler.bpf.perfbuf.Samples.samples.count", tags{"status": "collected"})
	lost := metric(r, "profiler.bpf.perfbuf.Samples.samples.count", tags{"status": "lost"})
	total := collected + lost

	var lossPct float64
	if total > 0 {
		lossPct = lost / total * 100
	}

	throughput := collected / collectDuration.Seconds()

	l.Info(ctx, "Perfbuf benchmark results",
		log.Float64("collected", collected),
		log.Float64("lost", lost),
		log.Float64("total", total),
		log.Float64("loss_pct", lossPct),
		log.Float64("throughput_samples_per_sec", throughput),
		log.String("config", fmt.Sprintf("buf=%dMB wm=%dKB",
			bc.perfBufferPerCPUSize/(1024*1024),
			bc.perfBufferWatermark/1024)),
	)

	b.ReportMetric(collected, "samples_collected")
	b.ReportMetric(lost, "samples_lost")
	b.ReportMetric(lossPct, "loss_pct")
	b.ReportMetric(throughput, "samples/sec")
}
