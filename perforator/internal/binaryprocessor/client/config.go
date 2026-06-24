package client

import (
	"errors"
)

const defaultMaxConcurrentRequests = 32

type Config struct {
	Target                string `yaml:"target"`
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests"`
}

func (c *Config) FillDefault() {
	if c.MaxConcurrentRequests == 0 {
		c.MaxConcurrentRequests = defaultMaxConcurrentRequests
	}
}

func (c *Config) Validate() error {
	if c.Target == "" {
		return errors.New("target is required")
	}
	return nil
}
