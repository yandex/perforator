package servicediscovery

import (
	"context"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const predefinedDiscovererName DiscovererType = "predefined"

func init() {
	discovererProducers[predefinedDiscovererName] =
		discovererProducerFunc(
			func(c *Config, l xlog.Logger) (Discoverer, error) {
				return newPredefinedDiscoverer(c), nil
			},
		)
}

type predefinedDiscoverer struct {
	endpoints []string
}

func newPredefinedDiscoverer(c *Config) *predefinedDiscoverer {
	return &predefinedDiscoverer{
		endpoints: c.Endpoints,
	}
}

// Discover implements ServiceDiscoverer.
func (p *predefinedDiscoverer) Discover(ctx context.Context) (discoveredEndpoints []string, err error) {
	return p.endpoints, nil
}

var _ Discoverer = &predefinedDiscoverer{}
