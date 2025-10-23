package client

import (
	"github.com/yandex/perforator/perforator/internal/servicediscovery"
)

type Config struct {
	ServiceDiscoveryConfig servicediscovery.Config `yaml:"service_discovery"`
	MaxRetries             int                     `yaml:"max_retries"`
}

const defaultHopsLimit = 2

func (c *Config) FillDefault() {
	c.ServiceDiscoveryConfig.FillDefault()
	if c.MaxRetries == 0 {
		c.MaxRetries = defaultHopsLimit
	}
}

func (c *Config) Validate() error {
	return c.ServiceDiscoveryConfig.Validate()
}
