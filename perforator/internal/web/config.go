package service

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/pkg/s3"
	"github.com/yandex/perforator/perforator/pkg/tracing"
)

type RenderedProfilesStorageConfig struct {
	S3Bucket string `yaml:"bucket"`
}

type PortsConfig struct {
	MetricsPort uint `yaml:"metrics_port"`
	HTTPPort    uint `yaml:"http_port"`
	GRPCPort    uint `yaml:"grpc_port"`
}

type Config struct {
	S3Config                      *s3.Config                     `yaml:"s3"`
	PortsConfig                   *PortsConfig                   `yaml:"ports"`
	ClientConfig                  *ClientConfig                  `yaml:"proxy_client"`
	RenderedProfilesStorageConfig *RenderedProfilesStorageConfig `yaml:"rendered_profiles"`
	Tracing                       *tracing.Config                `yaml:"tracing"`
}

func (c *Config) fillDefault() {
	if c.RenderedProfilesStorageConfig == nil {
		c.RenderedProfilesStorageConfig = &RenderedProfilesStorageConfig{
			S3Bucket: "perforator-task-results",
		}
	}

	if c.ClientConfig == nil {
		c.ClientConfig = &ClientConfig{}
	}

	if c.PortsConfig == nil {
		c.PortsConfig = &PortsConfig{
			MetricsPort: 85,
			HTTPPort:    80,
			GRPCPort:    81,
		}
	}

	if c.Tracing == nil {
		c.Tracing = tracing.NewDefaultConfig()
	}
}

func ParseConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't open config file: %w", err)
	}

	var conf Config

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	err = dec.Decode(&conf)
	if err != nil {
		return nil, fmt.Errorf("can't parse config: %s, with error: %w", configPath, err)
	}

	conf.fillDefault()

	if conf.ClientConfig.HTTPHost == "" {
		return nil, fmt.Errorf("perforator proxy http host is required")
	}

	if conf.ClientConfig.GRPCHost == "" {
		return nil, fmt.Errorf("perforator proxy grpc host is required")
	}

	return &conf, nil
}
