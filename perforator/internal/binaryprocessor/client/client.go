package client

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/consistenthash"
	_ "github.com/yandex/perforator/perforator/pkg/grpcutil/grpcreg"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	symbolizerproto "github.com/yandex/perforator/perforator/proto/symbolizer"
)

const maxRecvMsgSize = 1024 * 1024 * 1024 // 1G

var serviceConfig = fmt.Sprintf(`{
	"loadBalancingConfig": [{%q: {}}],
	"methodConfig": [{
		"name": [{"service": "NPerforator.NProto.NSymbolizer.Symbolizer"}],
		"retryPolicy": {
			"maxAttempts": 2,
			"initialBackoff": "0.1s",
			"maxBackoff": "1s",
			"backoffMultiplier": 2,
			"retryableStatusCodes": ["UNAVAILABLE"]
		}
	}]
}`, consistenthash.Name)

type Client struct {
	l             xlog.Logger
	conn          *grpc.ClientConn
	stub          symbolizerproto.SymbolizerClient
	maxConcurrent int
}

func NewClient(c *Config, l xlog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		c.Target,
		grpc.WithDefaultServiceConfig(serviceConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		l:             l,
		conn:          conn,
		stub:          symbolizerproto.NewSymbolizerClient(conn),
		maxConcurrent: c.MaxConcurrentRequests,
	}, nil
}

func (c *Client) Run(ctx context.Context) error {
	c.conn.Connect()
	<-ctx.Done()
	return c.conn.Close()
}

func (c *Client) Symbolize(
	ctx context.Context,
	requests []*symbolizerproto.SymbolizeRequest,
) ([]*symbolizerproto.SymbolizeResponse, error) {
	responses := make([]*symbolizerproto.SymbolizeResponse, len(requests))
	errs := make([]error, len(requests))

	g := errgroup.Group{}
	g.SetLimit(c.maxConcurrent)
	for i, req := range requests {
		g.Go(func() error {
			resp, err := c.stub.Symbolize(consistenthash.WithKey(ctx, req.BuildID), req)
			if err != nil {
				c.l.Warn(ctx, "Remote symbolization failed for binary",
					log.String("buildID", req.BuildID), log.Error(err))
				// NotFound (binary absent) and FailedPrecondition (present but
				// unsymbolizable) are binary-level failures local symbolization
				// cannot fix; any other code warrants a fallback, so record it.
				if code := status.Code(err); code != codes.NotFound && code != codes.FailedPrecondition {
					errs[i] = err
				}
				return nil
			}
			responses[i] = resp
			return nil
		})
	}
	_ = g.Wait()

	// A non-nil (joined) error signals the caller to fall back to local symbolization.
	return responses, errors.Join(errs...)
}
