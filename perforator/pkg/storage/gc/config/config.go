package config

import (
	"errors"
	"time"
)

const (
	DefaultLeaseName = "gc"
	DefaultLeaseTTL  = 30 * time.Second
)

type StorageType string

const (
	Profile StorageType = "profile"
	Binary  StorageType = "binary"
	GSYM    StorageType = "gsym"
)

type StorageConfig struct {
	Type           StorageType   `yaml:"type"`
	TTL            time.Duration `yaml:"ttl"`
	DeletePageSize uint32        `yaml:"delete_page_size,omitempty"`
}

type Config struct {
	Storages  []StorageConfig `yaml:"storages,omitempty"`
	LeaseName string          `yaml:"lease_name,omitempty"`
	LeaseTTL  time.Duration   `yaml:"lease_ttl,omitempty"`
}

func (c *Config) FillDefault() {
	if c.LeaseName == "" {
		c.LeaseName = DefaultLeaseName
	}
	if c.LeaseTTL == 0 {
		c.LeaseTTL = DefaultLeaseTTL
	}
}

func (c Config) Validate() error {
	if c.LeaseTTL <= 0 {
		return errors.New("GC lease TTL must be positive")
	}
	return nil
}
