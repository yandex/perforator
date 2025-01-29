package config

import (
	"time"
)

type StorageType string

const (
	Profile StorageType = "profile"
	Binary  StorageType = "binary"
)

type TTLConfig struct {
	TTL          time.Duration            `yaml:"ttl"`
	ServicesTTLs map[string]time.Duration `yaml:"services_ttls"`
}

type ConcurrencyConfig struct {
	Concurrency uint32 `yaml:"concurrency,omitempty"`
	Shards      uint32 `yaml:"shards,omitempty"`
}

type StorageConfig struct {
	Type           StorageType        `yaml:"type"`
	TTL            TTLConfig          `yaml:"ttl_config"`
	DeletePageSize uint32             `yaml:"delete_page_size,omitempty"`
	Concurrency    *ConcurrencyConfig `yaml:"concurrency,omitempty"`
}

type Config struct {
	Storages []StorageConfig `yaml:"storages,omitempty"`
}

func (c *Config) FillDefault() {
	for _, conf := range c.Storages {
		if conf.DeletePageSize == 0 {
			conf.DeletePageSize = 500
		}
	}
}
