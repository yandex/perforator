package postgrestaskservice

import "time"

type Config struct {
	PingPeriod  time.Duration `yaml:"ping_period"`
	PingTimeout time.Duration `yaml:"ping_timeout"`
	MaxAttempts int           `yaml:"max_attempts"`
}

func (c *Config) fillDefault() {
	if c.PingPeriod == 0 {
		c.PingPeriod = time.Second
	}
	if c.PingTimeout == 0 {
		c.PingTimeout = time.Second * 10
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 3
	}
}
