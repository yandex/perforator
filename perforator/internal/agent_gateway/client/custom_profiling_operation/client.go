package custom_profiling_operation

import (
	"context"

	"google.golang.org/grpc"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type Client struct {
	client cpo_proto.CustomProfilingOperationServiceClient
	logger xlog.Logger
}

func NewClient(conn *grpc.ClientConn, l xlog.Logger) *Client {
	return &Client{
		client: cpo_proto.NewCustomProfilingOperationServiceClient(conn),
		logger: l,
	}
}

func (c *Client) UpdateOperationStatus(ctx context.Context, id string, status *cpo_proto.OperationStatus) error {
	_, err := c.client.UpdateOperationStatus(ctx, &cpo_proto.UpdateOperationStatusRequest{
		ID:     id,
		Status: status,
	})
	return err
}

type LongPoller struct {
	client              *Client
	lastLongPollingData *cpo_proto.LongPollingData
	logger              xlog.Logger
}

func (c *Client) CreateLongPoller() *LongPoller {
	return &LongPoller{
		client: c,
		logger: c.logger,
	}
}

// Not thread-safe.
func (p *LongPoller) PollOperations(ctx context.Context, nodeID string, podNames []string) ([]*cpo_proto.Operation, error) {
	p.logger.Debug(ctx, "Started PollOperations", log.String("nodeID", nodeID), log.Strings("podNames", podNames))

	resp, err := p.client.client.PollOperations(ctx, &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host: nodeID,
			Pods: podNames,
		},
		LongPollingData: p.lastLongPollingData,
	})
	if err != nil {
		return nil, err
	}
	p.lastLongPollingData = resp.NextLongPollingData

	p.logger.Debug(ctx, "Finished PollOperations", log.Any("response", resp))

	return resp.Operations, nil
}
