package servicediscovery

import (
	"errors"
	"time"

	"github.com/yandex/perforator/perforator/pkg/endpointsetresolver"
)

type DiscovererType string

type Config struct {
	EndpointSetConfig []endpointsetresolver.EndpointSetConfig `yaml:"endpoint_set"`
	Endpoints         []string                                `yaml:"endpoints"`
	DiscoverInterval  time.Duration                           `yaml:"discover_interval"`
	DiscoverTimeout   time.Duration                           `yaml:"discover_timeout"`
	DNSRecords        []*dnsServiceConfig                     `yaml:"dns_records"`
	Discoverer        DiscovererType                          `yaml:"discoverer"`
}

func (c *Config) FillDefault() {
	if c.DiscoverInterval == 0 {
		c.DiscoverInterval = 5 * time.Second
	}
	if c.DiscoverTimeout == 0 {
		c.DiscoverTimeout = 30 * time.Second
	}
}

func (c *Config) Validate() error {
	var errs []error
	for _, dnsC := range c.DNSRecords {
		errs = append(errs, dnsC.Validate())
	}

	return errors.Join(errs...)
}
