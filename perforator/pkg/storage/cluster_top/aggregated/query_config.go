package aggregated

import clickhousego "github.com/ClickHouse/clickhouse-go/v2"

type QueryConfig struct {
	MaxThreads     uint64 `yaml:"max_threads"`
	MaxMemoryUsage uint64 `yaml:"max_memory_usage"`
}

func (c *QueryConfig) FillDefault() {
	if c.MaxThreads == 0 {
		c.MaxThreads = 2
	}
	if c.MaxMemoryUsage == 0 {
		c.MaxMemoryUsage = 1 << 30
	}
}

// Query duration is bounded by the caller context and the read-retry timeout.
// clickhouse-go derives max_execution_time from the resulting context deadline.
func (c QueryConfig) settings() clickhousego.Settings {
	return clickhousego.Settings{
		"max_threads":            c.MaxThreads,
		"max_memory_usage":       c.MaxMemoryUsage,
		"timeout_overflow_mode":  "throw",
		"group_by_overflow_mode": "throw",
		"read_overflow_mode":     "throw",
		"result_overflow_mode":   "throw",
	}
}
