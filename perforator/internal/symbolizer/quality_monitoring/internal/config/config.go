package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

type Config struct {
	Client                client.Config `yaml:"client"`
	CheckQualityInterval  time.Duration `yaml:"check_quality_interval"`
	IterationSplay        time.Duration `yaml:"iteration_splay"`
	ServicesNumberToCheck uint64        `yaml:"services_number_to_check"`
	ServicesOffset        uint64        `yaml:"services_offset"`
	MaxSamplesToMerge     uint32        `yaml:"max_samples_to_merge"`

	ServicesCheckingConcurrency      int           `yaml:"services_checking_concurrency"`
	SleepAfterFailedServicesChecking time.Duration `yaml:"sleep_after_failed_services_checking"`

	Tracing *tracing.Config `yaml:"tracing"`
}

func (c *Config) fillDefault() {
	if c == nil {
		c = &Config{}
	}

	if c.CheckQualityInterval == 0 {
		c.CheckQualityInterval = 30 * time.Minute
	}
	if c.ServicesCheckingConcurrency == 0 {
		c.ServicesCheckingConcurrency = 1
	}
	if c.ServicesNumberToCheck == 0 {
		c.ServicesNumberToCheck = 100
	}
	if c.MaxSamplesToMerge == 0 {
		c.MaxSamplesToMerge = 10
	}
	if c.Client.MaxReceiveMessageSize == 0 {
		c.Client.MaxReceiveMessageSize = 100 * 1024 * 1024 // 100M
	}

	if c.Tracing == nil {
		c.Tracing = tracing.NewDefaultConfig()
	}
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't open config file: %s", configPath)
	}

	var conf Config

	err = yaml.NewDecoder(file).Decode(&conf)
	if err != nil {
		return nil, fmt.Errorf("can't parse config: %s, with error: %w", configPath, err)
	}

	conf.fillDefault()

	return &conf, nil
}
