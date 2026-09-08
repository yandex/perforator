package config

import (
	"time"
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
	Storages []StorageConfig `yaml:"storages,omitempty"`
}
