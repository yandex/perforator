package cluster_top

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/pkg/filecache"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
)

type BinaryProviderConfig struct {
	FileCache                *filecache.Config `yaml:"file_cache"`
	MaxSimultaneousDownloads uint32            `yaml:"max_simultaneous_downloads"`
}

type JobLeaseConfig struct {
	TTL               time.Duration `yaml:"ttl"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	OperationTimeout  time.Duration `yaml:"operation_timeout"`
}

func defaultJobLeaseConfig() JobLeaseConfig {
	return JobLeaseConfig{
		TTL:               5 * time.Minute,
		HeartbeatInterval: 30 * time.Second,
		OperationTimeout:  10 * time.Second,
	}
}

func (c JobLeaseConfig) Validate() error {
	if c.TTL <= 0 || c.HeartbeatInterval <= 0 || c.OperationTimeout <= 0 {
		return fmt.Errorf("job lease intervals must be positive")
	}
	if c.HeartbeatInterval >= c.TTL || c.OperationTimeout >= c.TTL-c.HeartbeatInterval {
		return fmt.Errorf("job lease TTL must exceed heartbeat interval plus operation timeout")
	}
	return nil
}

type WorkerConfig struct {
	Lease           JobLeaseConfig `yaml:"lease"`
	SkippedServices []string       `yaml:"skipped_services"`
}

type Config struct {
	Storage        bundle.Config        `yaml:"storage"`
	BinaryProvider BinaryProviderConfig `yaml:"binary_provider"`
	Worker         WorkerConfig         `yaml:"worker"`

	GsymS3Bucket string `yaml:"gsym_s3_bucket"`
}

func ParseConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	conf := &Config{Worker: WorkerConfig{Lease: defaultJobLeaseConfig()}}
	err = yaml.NewDecoder(file).Decode(conf)
	if err != nil {
		return nil, err
	}

	if err := conf.Worker.Lease.Validate(); err != nil {
		return nil, err
	}
	return conf, nil
}
