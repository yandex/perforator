package clickhouse

import (
	"time"
)

type BatchingConfig struct {
	Size     uint32        `yaml:"size"`
	Interval time.Duration `yaml:"interval"`
}

type Config struct {
	Batching           BatchingConfig `yaml:"batching"`
	ReadRequestRetries uint32         `yaml:"read_request_retries"`
}
