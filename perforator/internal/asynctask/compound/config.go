package compound

import (
	"time"
)

type TasksStorageType string

const (
	Postgres TasksStorageType = "postgres"
	InMemory TasksStorageType = "inmemory"
)

type TasksConfig struct {
	StorageType TasksStorageType `yaml:"type,omitempty"`
	PingPeriod  time.Duration    `yaml:"ping_period,omitempty"`
	PingTimeout time.Duration    `yaml:"ping_timeout,omitempty"`
	MaxAttempts int              `yaml:"max_attempts,omitempty"`
}
