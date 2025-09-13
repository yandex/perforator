package custom_profiling_operation

import "time"

type ServiceConfig struct {
	PollInterval             time.Duration `yaml:"poll_interval"`
	PrefetchInterval         time.Duration `yaml:"prefetch_interval"`
	LongPollingTimeout       time.Duration `yaml:"long_polling_timeout"`
	CollectOperationsTimeout time.Duration `yaml:"collect_operations_timeout"`
}

func (c *ServiceConfig) FillDefault() {
	if c.PollInterval == time.Duration(0) {
		c.PollInterval = 10 * time.Second
	}

	if c.PrefetchInterval == time.Duration(0) {
		c.PrefetchInterval = 10 * time.Minute
	}

	if c.LongPollingTimeout == time.Duration(0) {
		c.LongPollingTimeout = 15 * time.Second
	}

	if c.CollectOperationsTimeout == time.Duration(0) {
		c.CollectOperationsTimeout = 10 * time.Second
	}
}
