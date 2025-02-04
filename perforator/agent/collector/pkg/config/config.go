package config

import (
	"time"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	storage "github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/upload"
	"github.com/yandex/perforator/perforator/internal/linguist/python/symbolizer"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/storage/client"
)

type ProcessDiscoveryConfig struct {
	// Number of process analyzer goroutines.
	// More concurrency means faster process analyzing and more accurate stacks,
	// But increases consumption of system resources.
	Concurrency int `yaml:"concurrency"`

	// Do not discover and analyze unrelated processes.
	// Can lead to lost stacks when process joins profiled cgroup,
	// But significantly reduces profiler CPU & RAM usage.
	// It makes sense to have this option disabled for daemonized system-wide profiler
	// And enable for ad-hoc one-shot profiles.
	IgnoreUnrelatedProcesses bool `yaml:"ignore_unrelated_processes"`
}

type EgressConfig struct {
	Interval time.Duration `yaml:"interval"`
}

type PerfEventConfig struct {
	// Type of the event to collect.
	// perfevent.CPUCycles by default.
	Type perfevent.Type `yaml:"type"`

	// Number of events to collect per second.
	// Mutually exclusive with Rate.
	// 99HZ by default.
	Frequency *uint64 `yaml:"frequency"`

	// Sample rate to collect events at.
	// Mutually exclusive with Frequency.
	SampleRate *uint64 `yaml:"sample_rate"`
}

type SampleConsumerConfig struct {
	// Byte size of percpu perf buffer to read samples from.
	PerfBufferPerCPUSize *int `yaml:"perfbuf_percpu_size"`

	// Number of bytes to generate wakeup notifications.
	// Higher values can lead to lower profiler overhead, but higher number of lost samples.
	// See man perf_event_open(2) for more info.
	PerfBufferWatermark *int `yaml:"perfbuf_watermark"`

	// Whitelist of environment variables to save in profiles.
	EnvWhitelist []string `yaml:"env_whitelist"`
}

type WhiteListEntry struct {
	Pattern string `yaml:"pattern"`
}

type BlackListEntry struct {
	Pattern string `yaml:"pattern"`
	Reason  string `yaml:"reason"`
}

type PodProfileOptions struct {
	WhiteList []WhiteListEntry `yaml:"whitelist,omitempty"`
	BlackList []BlackListEntry `yaml:"blacklist,omitempty"`

	Default bool `yaml:"default,omitempty"`
}

type KubernetesConfig struct {
	TopologyLableKey  string `yaml:"topology_lable_key,omitempty"`
	KubeletCgroupRoot string `yaml:"kubelet_cgroup_root,omitempty"`
}

type PodsDeploySystemConfig struct {
	DeploySystem        string            `yaml:"deploy_system,omitempty"`
	PodOptions          PodProfileOptions `yaml:"pod_options,omitempty"`
	UpdateCgroupsPeriod time.Duration     `yaml:"update_cgroups_period"`
	Labels              map[string]string `yaml:"labels,omitempty"`
	KubernetesConfig    KubernetesConfig  `yaml:"kubernetes,omitempty"`
}

type SymbolizerConfig struct {
	Python symbolizer.SymbolizerConfig `yaml:"python"`
}

type Config struct {
	Debug bool           `yaml:"debug"`
	BPF   machine.Config `yaml:"bpf"`

	ProcessDiscovery ProcessDiscoveryConfig `yaml:"process_discovery"`

	Egress EgressConfig `yaml:"egress"`

	StorageClientConfig *client.Config                 `yaml:"storage,omitempty"`
	LocalStorageConfig  *storage.LocalStorageConfig    `yaml:"local_storage,omitempty"`
	InMemoryStorage     *storage.InMemoryStorageConfig `yaml:"inmemory_storage,omitempty"`

	UploadSchedulerConfig upload.SchedulerConfig `yaml:"upload_scheduler,omitempty"`
	SampleConsumer        SampleConsumerConfig   `yaml:"sample_consumer,omitempty"`
	Symbolizer            SymbolizerConfig       `yaml:"symbolizer,omitempty"`

	PerfEvents []PerfEventConfig `yaml:"perf_events"`
	Signals    []string          `yaml:"signals"`

	EnableLBRDeprecated *bool `yaml:"enable_lbr"`
	EnablePerfMaps      *bool `yaml:"enable_perf_map"`
	EnablePerfMapsJVM   *bool `yaml:"enable_perf_maps_jvm"`

	PodsDeploySystemConfig *PodsDeploySystemConfig `yaml:"pods_deploy_system,omitempty"`
	Cgroups                cgroups.TrackerConfig   `yaml:"cgroups,omitempty"`
}

var (
	defaultTracedSignals = []string{
		"SIGINT",
		"SIGQUIT",
		"SIGILL",
		"SIGTRAP",
		"SIGABRT",
		"SIGBUS",
		"SIGFPE",
		"SIGKILL",
		"SIGSEGV",
		"SIGTERM",
		"SIGPIPE",
		"SIGALRM",
		"SIGXCPU",
		"SIGXFSZ",
		"SIGVTALRM",
	}
)

func defaultValue[T comparable](ptr *T, value T) {
	var zero T
	if *ptr == zero {
		*ptr = value
	}
}

func defaultPointer[T any](ptr **T, value T) {
	if *ptr == nil {
		*ptr = &value
	}
}

func defaultSlice[T any](ptr *[]T, value ...T) {
	if *ptr == nil || len(*ptr) == 0 {
		*ptr = value
	}
}

func (c *Config) FillDefault() {
	if c.Egress.Interval == 0 {
		c.Egress.Interval = time.Minute
	}

	if c.BPF.EnablePageTableScaling == nil {
		c.BPF.EnablePageTableScaling = ptr.T(true)
	}
	if c.BPF.PageTableScaleFactorGB == nil {
		c.BPF.PageTableScaleFactorGB = ptr.T(240)
	}
	if !c.BPF.Debug {
		c.BPF.Debug = c.Debug
	}

	if (c.BPF.TraceSignals == nil || !*c.BPF.TraceSignals) && len(c.PerfEvents) == 0 {
		c.PerfEvents = []PerfEventConfig{{
			Type:      perfevent.CPUCycles,
			Frequency: ptr.Uint64(99),
		}}
	}

	if c.BPF.TraceLBR == nil {
		c.BPF.TraceLBR = ptr.Bool(true)
	}
	if c.BPF.TraceWallTime == nil {
		c.BPF.TraceWallTime = ptr.Bool(true)
	}
	if c.BPF.TraceSignals == nil {
		c.BPF.TraceSignals = ptr.Bool(true)
	}
	if c.BPF.TracePython == nil {
		c.BPF.TracePython = ptr.Bool(true)
	}
	if c.EnableLBRDeprecated != nil {
		c.BPF.TraceLBR = c.EnableLBRDeprecated
	}

	if *c.BPF.TraceSignals {
		defaultSlice(&c.Signals, defaultTracedSignals...)
	}

	if len(c.SampleConsumer.EnvWhitelist) == 0 {
		c.SampleConsumer.EnvWhitelist = []string{"YT_OPERATION_ID"}
	}

	if c.PodsDeploySystemConfig != nil {
		if c.PodsDeploySystemConfig.UpdateCgroupsPeriod == time.Duration(0) {
			c.PodsDeploySystemConfig.UpdateCgroupsPeriod = 1 * time.Minute
		}
	}

	defaultPointer(&c.SampleConsumer.PerfBufferPerCPUSize, 16*1024*1024)
	defaultPointer(&c.SampleConsumer.PerfBufferWatermark, 100*2048)
}
