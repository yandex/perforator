package servicediscovery

import (
	"errors"
	"fmt"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	discovererProducers  map[DiscovererType]discovererProducer = make(map[DiscovererType]discovererProducer)
	errUnknownDiscoverer error                                 = errors.New("unknown discoverer name")
)

func newUnknownDiscovererError(discoverer DiscovererType) error {
	return fmt.Errorf("%s: %w", discoverer, errUnknownDiscoverer)
}

func NewDiscoverer(c *Config, l xlog.Logger) (Discoverer, error) {
	producer, ok := discovererProducers[c.Discoverer]
	if !ok {
		return nil, newUnknownDiscovererError(c.Discoverer)
	}

	return producer.New(c, l)
}
