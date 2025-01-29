package server

import (
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/tracing"
)

type ProfileBlacklist struct {
	ProfilerVersions []string `yaml:"profiler_version"`
}

type ServerConfig struct {
	Insecure bool `yaml:"insecure"`
}

type RenderedProfiles struct {
	URLPrefix string `yaml:"url_prefix"`
	S3Bucket  string `yaml:"bucket"`
}

type BinaryProviderConfig struct {
	FileCache                *asyncfilecache.Config `yaml:"file_cache"`
	MaxSimultaneousDownloads uint32                 `yaml:"max_simultaneous_downloads"`
}

type MicroscopeThrottle struct {
	LimitPerUser uint32        `yaml:"microscopes_per_user_limit"`
	LimitWindow  time.Duration `yaml:"limit_window"`
}

type MicroscopeConfig struct {
	Throttle *MicroscopeThrottle `yaml:"throttle"`
}

type ListServicesSettings struct {
	DefaultMaxStaleAge time.Duration `yaml:"default_max_timestamp_prune_interval"`
}

type PGOConfig struct {
	CreateLLVMProfBinaryPath string `yaml:"create_llvm_prof_path"`
	LlvmBoltBinaryPath       string `yaml:"llvm-bolt_path"`
}

type TasksConfig struct {
	ConcurrencyLimit int64 `yaml:"concurrency_limit,omitempty"`
}

type SymbolizationConfig struct {
	UseGSYM bool `yaml:"use_gsym"`
}

type Config struct {
	StorageConfig        bundle.Config         `yaml:"storage"`
	BinaryProvider       BinaryProviderConfig  `yaml:"binary_provider"`
	Server               ServerConfig          `yaml:"server"`
	Tasks                TasksConfig           `yaml:"tasks"`
	RenderedProfiles     *RenderedProfiles     `yaml:"rendered_profiles"`
	ProfileBlacklist     *ProfileBlacklist     `yaml:"profile_blacklist"`
	MicroscopeConfig     *MicroscopeConfig     `yaml:"microscope"`
	ListServicesSettings *ListServicesSettings `yaml:"list_services_settings"`
	PGOConfig            *PGOConfig            `yaml:"pgo_config"`
	Tracing              *tracing.Config       `yaml:"tracing"`
	SymbolizationConfig  SymbolizationConfig   `yaml:"symbolization"`
}

func ParseConfig(path string) (conf *Config, err error) {
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	conf = &Config{}
	err = yaml.NewDecoder(file).Decode(conf)
	return
}

func (c *Config) FillDefault() {
	if c.Tasks.ConcurrencyLimit == 0 {
		c.Tasks.ConcurrencyLimit = math.MaxInt64
	}
	if c.Tracing == nil {
		c.Tracing = tracing.NewDefaultConfig()
	}
}
