package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/ctxlog"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type dynamicProxyParser = func(response []byte) (*url.URL, error)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// freshConnHttpClientchedHttpClient is an http client which never reuses connections between requests.
// We use it because sticking requests to one configuration server can make resulting
// load distribution worse.
type freshConnHttpClient struct {
}

func (c *freshConnHttpClient) Do(req *http.Request) (*http.Response, error) {
	tempClient := &http.Client{
		Transport: &http.Transport{},
	}
	return tempClient.Do(req)
}

type dynamicProxy struct {
	logger         xlog.Logger
	parser         dynamicProxyParser
	client         httpClient
	configEndpoint string
	errChan        chan struct{}
	currentProxyMu sync.RWMutex
	currentProxy   *url.URL
	canceled       bool

	periodicUpdates      metrics.Counter
	periodicUpdateErrors metrics.Counter
	errorUpdates         metrics.Counter
	errorUpdateErrors    metrics.Counter
	throttledErrors      metrics.Counter
}

func (dp *dynamicProxy) proxy() *url.URL {
	dp.currentProxyMu.RLock()
	defer dp.currentProxyMu.RUnlock()
	if dp.canceled {
		dp.logger.Warn(context.TODO(), "Using http proxy after updater shutdown")
	}
	return dp.currentProxy
}

func (dp *dynamicProxy) updateProxy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dp.configEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create configuration endpoint request: %w", err)
	}
	resp, err := dp.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to query configuration endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read configuration endpoint response: %w", err)
	}
	var proxyURL *url.URL
	proxyURL, err = dp.parser(body)
	if err != nil {
		return err
	}

	dp.logger.Info(ctx, "Applying new http proxy configuration", log.String("proxy", proxyURL.String()))
	dp.currentProxyMu.Lock()
	defer dp.currentProxyMu.Unlock()
	dp.currentProxy = proxyURL
	return nil
}

func (dp *dynamicProxy) updateLoop(ctx context.Context, updateInterval time.Duration, maxErrorUpdates int) {
	ticker := time.Tick(updateInterval)
	currentErrorUpdates := 0
	for {
		var err error
		select {
		case <-ctx.Done():
			dp.logger.Debug(ctx, "Stopping http proxy background update")
			dp.currentProxyMu.Lock()
			dp.canceled = true
			dp.currentProxyMu.Unlock()
			return
		case <-ticker:
			dp.logger.Debug(ctx, "Starting periodic http proxy configuration update")
			dp.periodicUpdates.Inc()
			err = dp.updateProxy(ctxlog.WithFields(ctx, log.String("reason", "periodic")))
			if err == nil {
				currentErrorUpdates = 0
			} else {
				dp.periodicUpdateErrors.Inc()
			}
		case <-dp.errChan:
			if currentErrorUpdates < maxErrorUpdates {
				currentErrorUpdates++
				dp.logger.Info(
					ctx,
					"Updating http proxy configuration after transport error",
					log.Int("errorCount", currentErrorUpdates),
					log.Int("maxUpdates", maxErrorUpdates),
				)
				dp.errorUpdates.Inc()
				err = dp.updateProxy(ctxlog.WithFields(ctx, log.String("reason", "error")))
				if err != nil {
					dp.errorUpdateErrors.Inc()
				}
			} else {
				dp.logger.Debug(
					ctx,
					"Not updating http proxy configuration after error, because update limit is reached",
					log.Int("limit", maxErrorUpdates),
				)
				dp.throttledErrors.Inc()
			}
		}
		if err != nil {
			dp.logger.Warn(ctx, "Failed to update http proxy configuration", log.Error(err))
		}
	}
}

func (dp *dynamicProxy) onError() {
	select {
	case dp.errChan <- struct{}{}:
	default:
	}
}

func newDynamicProxy(ctx context.Context, refreshCtx context.Context, logger xlog.Logger, reg metrics.Registry, conf *DynamicHTTPProxyConfig) (*dynamicProxy, error) {
	if conf.UpdateInterval <= 0 {
		return nil, errors.New("update_interval must be positive")
	}
	dp := &dynamicProxy{
		logger:         logger.WithName("s3_http_proxy_updater"),
		client:         &freshConnHttpClient{},
		configEndpoint: conf.ConfigurationEndpoint,
		errChan:        make(chan struct{}, 1),

		periodicUpdates:      reg.WithTags(map[string]string{"reason": "periodic"}).Counter("configuration_update_attempts"),
		periodicUpdateErrors: reg.WithTags(map[string]string{"reason": "periodic"}).Counter("configuration_update_errors"),
		errorUpdates:         reg.WithTags(map[string]string{"reason": "error"}).Counter("configuration_update_attempts"),
		errorUpdateErrors:    reg.WithTags(map[string]string{"reason": "error"}).Counter("configuration_update_errors"),
		throttledErrors:      reg.Counter("throttled_error_update_attempts"),
	}
	switch conf.Kind {
	case "hostname":
		scheme := "https"
		if conf.OverrideScheme != "" {
			scheme = conf.OverrideScheme
		}
		port := conf.OverridePort
		dp.parser = func(response []byte) (*url.URL, error) {
			host := string(response)
			u, err := url.Parse(scheme + "://" + host)
			if err != nil {
				return nil, err
			}
			if u.Port() == "" && port != nil {
				u, err = url.Parse(fmt.Sprintf("%s://%s:%d", scheme, host, *port))
				if err != nil {
					return nil, err
				}
			}
			return u, nil
		}
	// other ways to dynamically configure http proxy can be added here
	default:
		return nil, fmt.Errorf("unknown kind: %q", conf.Kind)
	}
	err := dp.updateProxy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial proxy configuration: %w", err)
	}
	go dp.updateLoop(refreshCtx, conf.UpdateInterval, int(conf.MaxErrorUpdates))
	return dp, nil
}
