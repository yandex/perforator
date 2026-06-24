package client

import (
	"context"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/consistenthash"
	_ "github.com/yandex/perforator/perforator/pkg/grpcutil/grpcreg"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	symbolizerproto "github.com/yandex/perforator/perforator/proto/symbolizer"
)

const maxRecvMsgSize = 1024 * 1024 * 1024 // 1G

type Client struct {
	l             xlog.Logger
	conn          *grpc.ClientConn
	stub          symbolizerproto.SymbolizerClient
	maxConcurrent int
}

func NewClient(c *Config, l xlog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		c.Target,
		grpc.WithDefaultServiceConfig(consistenthash.ServiceConfig),
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
	batch []*symbolizerproto.PerBinaryRequest,
) (results []*symbolizerproto.PerBinaryResponse, failed []*symbolizerproto.PerBinaryRequest) {
	perBinary := make([]*symbolizerproto.PerBinaryResponse, len(batch))

	g := errgroup.Group{}
	g.SetLimit(c.maxConcurrent)
	for i, req := range batch {
		g.Go(func() error {
			resp, err := c.stub.Symbolize(
				consistenthash.WithKey(ctx, req.BuildID),
				&symbolizerproto.SymbolizeRequest{Batch: []*symbolizerproto.PerBinaryRequest{req}},
			)
			if err != nil {
				c.l.Warn(ctx, "Remote symbolization failed for binary",
					log.String("buildID", req.BuildID), log.Error(err))
				return nil
			}
			if len(resp.GetBatch()) > 0 {
				perBinary[i] = resp.Batch[0]
			}
			return nil
		})
	}
	_ = g.Wait()

	results = make([]*symbolizerproto.PerBinaryResponse, 0, len(batch))
	for i, req := range batch {
		if perBinary[i] == nil {
			failed = append(failed, req)
			continue
		}
		results = append(results, perBinary[i])
	}
	return results, failed
}
