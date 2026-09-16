package config

import (
	"fmt"
	"time"
)

type ClusterTopConfig struct {
	Enabled          bool
	TTL              time.Duration
	Interval         time.Duration
	JobsBatchSize    uint32
	JobsInterval     time.Duration
	OperationTimeout time.Duration
}

func DefaultClusterTopConfig() ClusterTopConfig {
	return ClusterTopConfig{
		TTL:              30 * 24 * time.Hour,
		Interval:         time.Hour,
		JobsBatchSize:    5000,
		JobsInterval:     100 * time.Millisecond,
		OperationTimeout: time.Minute,
	}
}

func (c ClusterTopConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.TTL <= 0 || c.Interval <= 0 || c.JobsBatchSize == 0 || c.JobsInterval <= 0 || c.OperationTimeout <= 0 {
		return fmt.Errorf("Cluster Top GC TTL, intervals, batch size and operation timeout must be positive")
	}
	return nil
}
