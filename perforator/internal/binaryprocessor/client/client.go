package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/serialx/hashring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/servicediscovery"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	symbolizerService "github.com/yandex/perforator/perforator/proto/symbolizer"
)

type client struct {
	clientConn *grpc.ClientConn
	bpClient   symbolizerService.SymbolizerClient
}

func (c *client) close() error {
	if c == nil {
		return nil
	}

	return c.clientConn.Close()
}

func newClient(target string) (*client, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithDefaultCallOptions(
			grpc.MaxRecvMsgSizeCallOption{
				MaxRecvMsgSize: int(1024 * 1024 * 1024 /*1G*/),
			}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &client{
		clientConn: conn,
		bpClient:   symbolizerService.NewSymbolizerClient(conn),
	}, nil
}

type DistributingClient struct {
	l xlog.Logger
	c *Config

	// guarded by mutex section
	mu       sync.Locker
	hashring *hashring.HashRing
	// endpoint is an <hostname>:<port> URL
	endpoints map[string]*client

	discoverer servicediscovery.Discoverer
}

func NewDistributingClient(
	c *Config,
	l xlog.Logger,
) (*DistributingClient, error) {
	discoverer, err := servicediscovery.NewDiscoverer(&c.ServiceDiscoveryConfig, l)
	if err != nil {
		return nil, err
	}

	return &DistributingClient{
		l:          l,
		c:          c,
		endpoints:  make(map[string]*client),
		discoverer: discoverer,
		hashring:   hashring.New([]string{}),
		mu:         &sync.Mutex{},
	}, nil
}

func (r *DistributingClient) getEndpointsUnlocked() []string {
	endpoints := make([]string, 0, len(r.endpoints))
	for endpoint := range r.endpoints {
		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

func (r *DistributingClient) getEndpointClient(endpoint string) (*client, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	client, ok := r.endpoints[endpoint]
	return client, ok
}

type endpointClient struct {
	endpoint string
	*client
}

func (r *DistributingClient) registerNewEndpoints(endpointClients []*endpointClient) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, endpointClient := range endpointClients {
		r.endpoints[endpointClient.endpoint] = endpointClient.client
		r.hashring = r.hashring.AddNode(endpointClient.endpoint)
	}
}

func (r *DistributingClient) removeEndpoint(endpoint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.removeEndpointUnlocked(endpoint)
}

func (r *DistributingClient) removeEndpointUnlocked(endpoint string) error {
	client := r.endpoints[endpoint]

	delete(r.endpoints, endpoint)
	r.hashring = r.hashring.RemoveNode(endpoint)

	return client.close()
}

func (r *DistributingClient) excludeUndiscoveredEndpoints(ctx context.Context, discoveredEndpoints []string) {
	discovered := make(map[string]struct{})
	for _, endpoint := range discoveredEndpoints {
		discovered[endpoint] = struct{}{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	endpoints := r.getEndpointsUnlocked()

	for _, endpoint := range endpoints {
		_, ok := discovered[endpoint]
		if !ok {
			r.l.Warn(ctx, "Removing undiscovered endpoint", log.String("endpoint", endpoint))
			err := r.removeEndpointUnlocked(endpoint)
			if err != nil {
				r.l.Error(ctx, "Endpoint removal error", log.Error(err))
			}
		}
	}
}

func (r *DistributingClient) Run(ctx context.Context) error {
	for {
		discoverCtx, discoverCancel := context.WithTimeoutCause(
			ctx,
			r.c.ServiceDiscoveryConfig.DiscoverTimeout,
			errors.New("binary processor distributing client: service discovery attempt's timeout exceeded"),
		)
		defer discoverCancel()
		endpoints, err := r.discoverer.Discover(discoverCtx)

		if err != nil {
			r.l.Error(ctx, "Error during service discovery", log.Error(err))
		}

		if endpoints != nil {
			// add new endpoints
			endpointClients := make([]*endpointClient, 0)
			for _, endpoint := range endpoints {
				_, ok := r.getEndpointClient(endpoint)
				if !ok {
					r.l.Info(ctx, "Discovered new endpoint", log.String("endpoint", endpoint))
					client, err := newClient(endpoint)
					if err != nil {
						r.l.Error(ctx, "Failed to create endpoint client", log.Error(err))
						err = nil
						continue
					}

					endpointClients = append(
						endpointClients,
						&endpointClient{
							endpoint: endpoint,
							client:   client,
						},
					)
				}
			}

			r.registerNewEndpoints(endpointClients)

			// remove faded away endpoints
			r.excludeUndiscoveredEndpoints(ctx, endpoints)
		}

		select {
		case <-time.After(r.c.ServiceDiscoveryConfig.DiscoverInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *DistributingClient) getHashRing() *hashring.HashRing {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.hashring
}

func getEndpoint(r *hashring.HashRing, key string, skipCount int) (node string, ok bool) {
	nodes, ok := r.GetNodes(key, skipCount+1)
	if !ok {
		return "", ok
	}

	return nodes[skipCount], ok
}

func (r *DistributingClient) distributeRequestsByEndpoints(
	batch []*symbolizerService.PerBinaryRequest,
	skipCount int,
) (
	endpointBatch map[string][]*symbolizerService.PerBinaryRequest,
	orphanBatch []*symbolizerService.PerBinaryRequest,
) {
	hr := r.getHashRing()
	endpointBatch = make(map[string][]*symbolizerService.PerBinaryRequest)
	for _, request := range batch {
		endpoint, ok := getEndpoint(hr, request.BuildID, skipCount)
		if !ok {
			orphanBatch = append(orphanBatch, request)
			continue
		}

		endpointBatch[endpoint] = append(endpointBatch[endpoint], request)
	}

	return endpointBatch, orphanBatch
}

type remoteSymbolizationResult struct {
	// batch to send to an endpoint.
	// if getting an endpoint fails return these batch
	// to perform symbolization locally
	batch []*symbolizerService.PerBinaryRequest

	// not nil - performed remote symbolization successfully
	response *symbolizerService.SymbolizeResponse
}

func (r *DistributingClient) symbolizeAtEndpoint(
	ctx context.Context,
	endpoint string,
	batch []*symbolizerService.PerBinaryRequest,
	hopsMade int,
) (res *remoteSymbolizationResult, redistribute bool, hopsMadeNext int) {
	res = &remoteSymbolizationResult{
		batch: batch,
	}

	if hopsMade >= r.c.MaxRetries {
		r.l.Warn(ctx, "Hops limit reached", log.String("endpoint", endpoint))
		return res, false, hopsMade
	}

	client, ok := r.getEndpointClient(endpoint)

	// seems like endpoint died in between mutex locks.
	// redirect to another endpoints
	if !ok {
		r.l.Warn(ctx, "Endpoint found dead", log.String("endpoint", endpoint))
		return res, true, hopsMade
	}

	r.l.Debug(ctx, "Redirecting symbolize to an endpoint", log.String("endpoint", endpoint))
	var err error
	res.response, err = client.bpClient.Symbolize(ctx, &symbolizerService.SymbolizeRequest{
		Batch: batch,
	})

	hopsMade += 1

	if err != nil {
		r.l.Warn(ctx, "Remote symbolization failed", log.String("endpoint", endpoint), log.Error(err))

		return res, true, hopsMade
	}

	return res, false, hopsMade
}

func (r *DistributingClient) symbolizeAtEndpoints(
	ctx context.Context,
	wg *sync.WaitGroup,
	endpointBatch map[string][]*symbolizerService.PerBinaryRequest,
	hopsMade int,
	responses chan *remoteSymbolizationResult,
) {
	for endpoint, batch := range endpointBatch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, redistribute, hopsMadeNext := r.symbolizeAtEndpoint(ctx, endpoint, batch, hopsMade)
			if redistribute {
				r.l.Debug(ctx, "Redistribute batch", log.String("endpoint", endpoint))
				endpointBatch, orphanBatch := r.distributeRequestsByEndpoints(batch, hopsMadeNext)
				r.symbolizeAtEndpoints(ctx, wg, endpointBatch, hopsMadeNext, responses)
				if orphanBatch != nil {
					responses <- &remoteSymbolizationResult{
						batch: orphanBatch,
					}
				}

				return
			}

			responses <- resp
		}()
	}
}

func (r *DistributingClient) DistributeSymbolize(
	ctx context.Context,
	batch []*symbolizerService.PerBinaryRequest,
) (
	response *symbolizerService.SymbolizeResponse,
	failedReqs []*symbolizerService.PerBinaryRequest,
	err error,
) {
	endpointBatch, failedReqs := r.distributeRequestsByEndpoints(batch, 0)

	responses := make(chan *remoteSymbolizationResult)

	wg := &sync.WaitGroup{}
	r.symbolizeAtEndpoints(ctx, wg, endpointBatch, 0, responses)

	go func() {
		wg.Wait()
		close(responses)
	}()

	resResp := &symbolizerService.SymbolizeResponse{}
	for resp := range responses {
		if resp.response != nil {
			resResp.Batch = append(resResp.Batch, resp.response.Batch...)
			continue
		}

		failedReqs = append(failedReqs, resp.batch...)
	}

	return resResp, failedReqs, nil
}
