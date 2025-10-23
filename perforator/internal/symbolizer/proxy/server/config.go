package server

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	bpclient "github.com/yandex/perforator/perforator/internal/binaryprocessor/client"
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
	Pool             string `yaml:"pool"`
	ConcurrencyLimit int64  `yaml:"concurrency_limit,omitempty"`
}

type SymbolizationConfig struct {
	UseGSYM bool `yaml:"use_gsym"`
}

type ACLConfig struct {
	RecordRemoteUsers []string `yaml:"record_remote_users"`
}

type ProfileMergerConfig struct {
	ThreadCount uint32 `yaml:"thread_count"`
}

type FeaturesConfig struct {
	EnableCPOExperimental     *bool `yaml:"enable_custom_profiling_operation"`
	EnableNewProfileMerger    *bool `yaml:"enable_new_profile_merger"`
	EnableRemoteSymbolization *bool `yaml:"enable_remote_symbolization"`
}

func (c *FeaturesConfig) FillDefault() {
	if c.EnableCPOExperimental == nil {
		c.EnableCPOExperimental = ptr.Bool(false)
	}
	if c.EnableNewProfileMerger == nil {
		c.EnableNewProfileMerger = ptr.Bool(false)
	}
	if c.EnableRemoteSymbolization == nil {
		c.EnableRemoteSymbolization = ptr.Bool(false)
	}
}

type Config struct {
	StorageConfig               bundle.Config         `yaml:"storage"`
	BinaryProvider              BinaryProviderConfig  `yaml:"binary_provider"`
	Server                      ServerConfig          `yaml:"server"`
	Tasks                       TasksConfig           `yaml:"tasks"`
	RenderedProfiles            *RenderedProfiles     `yaml:"rendered_profiles"`
	ProfileBlacklist            *ProfileBlacklist     `yaml:"profile_blacklist"`
	MicroscopeConfig            *MicroscopeConfig     `yaml:"microscope"`
	ListServicesSettings        *ListServicesSettings `yaml:"list_services_settings"`
	PGOConfig                   *PGOConfig            `yaml:"pgo_config"`
	Tracing                     *tracing.Config       `yaml:"tracing"`
	ProfileMerger               ProfileMergerConfig   `yaml:"profile_merger"`
	SymbolizationConfig         SymbolizationConfig   `yaml:"symbolization"`
	ACL                         ACLConfig             `yaml:"acl"`
	FeaturesConfig              FeaturesConfig        `yaml:"features"`
	BinaryProcessorClientConfig *bpclient.Config      `yaml:"binary_processor_client"`
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
	if c.ProfileMerger.ThreadCount == 0 {
		c.ProfileMerger.ThreadCount = 16
	}
	c.FeaturesConfig.FillDefault()
	if *c.FeaturesConfig.EnableRemoteSymbolization &&
		c.BinaryProcessorClientConfig != nil {
		c.BinaryProcessorClientConfig.FillDefault()
	}
}

func newInvalidFieldError(fieldName string, err error) error {
	return fmt.Errorf("field `%s` is invalid: %w", fieldName, err)
}

func (c *Config) Validate() error {
	var errs []error
	if c.BinaryProcessorClientConfig != nil {
		err := c.BinaryProcessorClientConfig.Validate()
		if err != nil {
			errs = append(errs, newInvalidFieldError("binary_processor_client", err))
		}
	}

	return errors.Join(errs...)
}
