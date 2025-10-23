package servicediscovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	dnsDiscovererName = "dns"
	serviceProto      = "tcp"
)

func init() {
	discovererProducers[dnsDiscovererName] =
		discovererProducerFunc(
			func(c *Config, l xlog.Logger) (Discoverer, error) {
				return newDNSDiscoverer(c, l), nil
			},
		)
}

// fields are named in terms of DNS SRV record format
type dnsServiceConfig struct {
	Name    *string
	Service *string
}

var ErrDNSServiceDiscoveryError = errors.New("DNS service discovery error")
var ErrDNSServiceConfigError = errors.New("DNS service config error")

func newMandatoryFieldError(fieldName string) error {
	return fmt.Errorf("`%s` is mandatory field", fieldName)
}

func newDNSServiceDiscoveryError(err error) error {
	return fmt.Errorf("%w: %w", ErrDNSServiceDiscoveryError, err)
}

func (c *dnsServiceConfig) Validate() error {
	var errs []error
	if c.Name == nil {
		errs = append(errs, newMandatoryFieldError("name"))
	}
	if c.Service == nil {
		errs = append(errs, newMandatoryFieldError("service"))
	}

	if errs != nil {
		return fmt.Errorf("%w: %w", ErrDNSServiceConfigError, errors.Join(errs...))
	}

	return nil
}

type dnsDiscoverer struct {
	l xlog.Logger
	c *Config
	r *net.Resolver
}

func newDNSDiscoverer(c *Config, l xlog.Logger) *dnsDiscoverer {
	return &dnsDiscoverer{
		c: c,
		l: l.WithName(fmt.Sprintf("%s-discoverer", dnsDiscovererName)),
		r: &net.Resolver{},
	}
}

// Discover implements Discoverer.
func (d *dnsDiscoverer) Discover(ctx context.Context) (discoveredEndpoints []string, err error) {
	type lookupResponse struct {
		endpoints []string
		error
	}

	respChan := make(chan *lookupResponse)
	wg := &sync.WaitGroup{}
	for _, service := range d.c.DNSRecords {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, endpoints, err := d.r.LookupSRV(ctx, *service.Service, serviceProto, *service.Name)
			if err != nil {
				respChan <- &lookupResponse{
					error: newDNSServiceDiscoveryError(fmt.Errorf("failed to perform SRV lookup: %w", err)),
				}
				return
			}

			res := make([]string, len(endpoints))
			for i, endpoint := range endpoints {
				res[i] = fmt.Sprintf("%s:%v", endpoint.Target, endpoint.Port)
			}

			respChan <- &lookupResponse{
				endpoints: res,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(respChan)
	}()

	var errs []error
	for resp := range respChan {
		if resp.error != nil {
			errs = append(errs, resp.error)
			continue
		}

		discoveredEndpoints = append(discoveredEndpoints, resp.endpoints...)
	}

	return discoveredEndpoints, errors.Join(errs...)
}

var _ Discoverer = &dnsDiscoverer{}
