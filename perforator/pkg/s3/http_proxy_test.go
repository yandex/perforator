package s3

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func runMockServer(
	ctx context.Context,
	t *testing.T,
	endpoint string,
	initialAddress string,
	addressUpdates <-chan string,
	port chan<- uint16,
	addressUpdatesDone chan<- struct{},
) {
	logger := xlog.ForTest(t)
	mux := http.NewServeMux()
	currentAddress := atomic.Pointer[string]{}
	currentAddress.Store(&initialAddress)
	mux.HandleFunc("GET "+endpoint, func(rw http.ResponseWriter, r *http.Request) {
		addr := *currentAddress.Load()
		logger.Info(ctx, "Got request", log.String("response", addr))
		_, err := rw.Write([]byte(addr))
		assert.NoError(t, err)
	})
	srv := &http.Server{
		Handler: mux,
	}
	listener, err := net.Listen("tcp", ":0")
	if !assert.NoError(t, err) {
		return
	}
	port <- uint16(listener.Addr().(*net.TCPAddr).Port)
	wg, wgCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		return srv.Serve(listener)
	})
	wg.Go(func() error {
		<-wgCtx.Done()
		return srv.Shutdown(wgCtx)
	})
	wg.Go(func() error {
		for {
			select {
			case <-wgCtx.Done():
				return nil
			case newAddress := <-addressUpdates:
				currentAddress.Store(&newAddress)
				logger.Info(ctx, "Reconfigured", log.String("address", newAddress))
				addressUpdatesDone <- struct{}{}
			}
		}
	})
	err = wg.Wait()
	assert.ErrorIs(t, err, http.ErrServerClosed)
}

func TestSimple(t *testing.T) {
	t.Parallel()
	logger := xlog.ForTest(t)
	portChan := make(chan uint16, 1)
	testCtx, cancel := context.WithTimeoutCause(context.Background(), 15*time.Second, fmt.Errorf("test timeout exceeded"))
	defer cancel()
	wg, ctx := errgroup.WithContext(testCtx)
	errDone := errors.New("test done")
	wg.Go(func() error {
		runMockServer(ctx, t, "/proxy", "somehost:993", nil, portChan, nil)
		return errors.New("should not propagate")
	})

	wg.Go(func() error {
		var port uint16
		select {
		case port = <-portChan:
		case <-ctx.Done():
			return errDone
		}
		dp, err := newDynamicProxy(ctx, ctx, logger, mock.NewRegistry(nil), &DynamicHTTPProxyConfig{
			UpdateInterval:        time.Second,
			Kind:                  dynamicHTTPProxyKindHostname,
			ConfigurationEndpoint: fmt.Sprintf("http://localhost:%d/proxy", port),
		})
		if !assert.NoError(t, err) {
			return errDone
		}
		defer dp.stop()

		url := dp.proxy()
		assert.Equal(t, "https://somehost:993", url.String())
		return errDone
	})
	err := wg.Wait()
	assert.ErrorIs(t, err, errDone)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	logger := xlog.ForTest(t)
	addrChan := make(chan string, 1)
	portChan := make(chan uint16, 1)
	addrUpdatesDone := make(chan struct{}, 1)
	testCtx, cancel := context.WithTimeoutCause(context.Background(), 15*time.Second, fmt.Errorf("test timeout exceeded"))
	defer cancel()
	wg, ctx := errgroup.WithContext(testCtx)
	errDone := errors.New("test done")
	wg.Go(func() error {
		runMockServer(ctx, t, "/proxy", "somehost:993", addrChan, portChan, addrUpdatesDone)
		return errors.New("should not propagate")
	})
	wg.Go(func() error {
		var port uint16
		select {
		case port = <-portChan:
		case <-ctx.Done():
			return errDone
		}
		updateInterval := time.Second
		dp, err := newDynamicProxy(ctx, ctx, logger, mock.NewRegistry(nil), &DynamicHTTPProxyConfig{
			UpdateInterval:        updateInterval,
			Kind:                  dynamicHTTPProxyKindHostname,
			ConfigurationEndpoint: fmt.Sprintf("http://localhost:%d/proxy", port),
		})
		if !assert.NoError(t, err) {
			return errDone
		}
		defer dp.stop()

		url := dp.proxy()
		assert.Equal(t, "https://somehost:993", url.String())
		addrChan <- "otherhost:1007"
		<-addrUpdatesDone
		time.Sleep(2 * updateInterval)
		url = dp.proxy()
		assert.Equal(t, "https://otherhost:1007", url.String())
		return errDone
	})
	err := wg.Wait()
	assert.ErrorIs(t, err, errDone)
}

func TestE2E(t *testing.T) {
	t.Parallel()
	logger := xlog.ForTest(t)

	addrChan := make(chan string, 1)
	portChan := make(chan uint16, 1)
	addrUpdatesDone := make(chan struct{}, 1)
	testCtx, cancel := context.WithTimeoutCause(context.Background(), 15*time.Second, fmt.Errorf("test timeout exceeded"))
	defer cancel()
	wg, ctx := errgroup.WithContext(testCtx)
	errDone := errors.New("test done")
	wg.Go(func() error {
		runMockServer(ctx, t, "/proxy", "someproxyhost:993", addrChan, portChan, addrUpdatesDone)
		return errors.New("should not propagate")
	})

	wg.Go(func() error {
		var port uint16
		select {
		case port = <-portChan:
		case <-ctx.Done():
			return errDone
		}
		err := os.Setenv("SOMEVAR", "somevalue")
		if !assert.NoError(t, err) {
			return errDone
		}

		client, err := NewClient(ctx, ctx, logger, &Config{
			Endpoint:     "yandex.net",
			SecretKeyEnv: "SOMEVAR",
			AccessKeyEnv: "SOMEVAR",
			MaxRetries:   ptr.Uint32(0),
			DynamicHTTPProxy: &DynamicHTTPProxyConfig{
				UpdateInterval:        300 * time.Second,
				Kind:                  dynamicHTTPProxyKindHostname,
				ConfigurationEndpoint: fmt.Sprintf("http://localhost:%d/proxy", port),
				MaxErrorUpdates:       1,
			},
		}, mock.NewRegistry(&mock.RegistryOpts{
			AllowLoadRegisteredMetrics: true,
		}))
		if !assert.NoError(t, err) {
			return errDone
		}
		defer client.Close()

		addrChan <- "otherproxyhost:1007"
		<-addrUpdatesDone

		_, err = client.GetBucketAcl(&s3.GetBucketAclInput{
			Bucket: ptr.String("bucket"),
		})
		if !assert.Error(t, err) {
			return errDone
		}
		assert.Contains(t, err.Error(), "someproxyhost")

		// dynamicProxy should update endpoint after error
		time.Sleep(time.Second)

		_, err = client.GetBucketAcl(&s3.GetBucketAclInput{
			Bucket: ptr.String("bucket"),
		})
		if !assert.Error(t, err) {
			return errDone
		}
		assert.Contains(t, err.Error(), "otherproxyhost")

		return errDone
	})
	err := wg.Wait()
	assert.ErrorIs(t, err, errDone)
}
