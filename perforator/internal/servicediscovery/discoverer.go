package servicediscovery

import (
	"context"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type Discoverer interface {
	Discover(ctx context.Context) (discoveredEndpoints []string, err error)
}

type discovererProducer interface {
	New(c *Config, l xlog.Logger) (Discoverer, error)
}

type discovererProducerFunc func(c *Config, l xlog.Logger) (Discoverer, error)

func (f discovererProducerFunc) New(c *Config, l xlog.Logger) (Discoverer, error) {
	return f(c, l)
}
