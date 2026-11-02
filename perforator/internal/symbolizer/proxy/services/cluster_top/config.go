package cluster_top

import (
	"errors"
	"time"
)

type Config struct {
	QueryTimeout time.Duration `yaml:"query_timeout"`
}

func (c *Config) FillDefault() {
	if c.QueryTimeout == 0 {
		c.QueryTimeout = 5 * time.Second
	}
}

func (c Config) Validate() error {
	if c.QueryTimeout < 0 {
		return errors.New("Cluster Top query timeout must not be negative")
	}
	return nil
}
