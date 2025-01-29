package service

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

////////////////////////////////////////////////////////////////////////////////

type ClientConfig struct {
	HTTPHost string `yaml:"http_host"`
	GRPCHost string `yaml:"grpc_host"`

	// TODO: add OAuth token with perforator:api scope and tls support.
	//Insecure bool
	//Token    string
}

type Client struct {
	l xlog.Logger

	connection        *grpc.ClientConn
	client            perforator.PerforatorClient
	microscopesClient perforator.MicroscopeServiceClient
	taskClient        perforator.TaskServiceClient
}

func NewClient(cfg *ClientConfig, l xlog.Logger) (*Client, error) {
	if cfg.GRPCHost == "" {
		return nil, errors.New("grpc host is not set")
	}

	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: 30 * time.Second,
		}),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024 /*1G*/)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(cfg.GRPCHost, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		l:                 l,
		connection:        conn,
		client:            perforator.NewPerforatorClient(conn),
		microscopesClient: perforator.NewMicroscopeServiceClient(conn),
		taskClient:        perforator.NewTaskServiceClient(conn),
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) ListServices(ctx context.Context, req *perforator.ListServicesRequest) (*perforator.ListServicesResponse, error) {
	return c.client.ListServices(ctx, req)
}

func (c *Client) ListSuggestions(
	ctx context.Context,
	req *perforator.ListSuggestionsRequest,
) (*perforator.ListSuggestionsResponse, error) {
	return c.client.ListSuggestions(ctx, req)
}

func (c *Client) ListProfiles(ctx context.Context, req *perforator.ListProfilesRequest) (*perforator.ListProfilesResponse, error) {
	return c.client.ListProfiles(ctx, req)
}

func (c *Client) GetProfile(ctx context.Context, req *perforator.GetProfileRequest) (*perforator.GetProfileResponse, error) {
	return c.client.GetProfile(ctx, req)
}

func (c *Client) MergeProfiles(ctx context.Context, req *perforator.MergeProfilesRequest) (*perforator.MergeProfilesResponse, error) {
	return c.client.MergeProfiles(ctx, req)
}

func (c *Client) UploadProfile(ctx context.Context, req *perforator.UploadProfileRequest) (*perforator.UploadProfileResponse, error) {
	return c.client.UploadProfile(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) ListMicroscopes(ctx context.Context, req *perforator.ListMicroscopesRequest) (*perforator.ListMicroscopesResponse, error) {
	return c.microscopesClient.ListMicroscopes(ctx, req)
}

func (c *Client) SetMicroscope(ctx context.Context, req *perforator.SetMicroscopeRequest) (*perforator.SetMicroscopeResponse, error) {
	return c.microscopesClient.SetMicroscope(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) GetTask(ctx context.Context, req *perforator.GetTaskRequest) (*perforator.GetTaskResponse, error) {
	return c.taskClient.GetTask(ctx, req)
}

func (c *Client) StartTask(ctx context.Context, req *perforator.StartTaskRequest) (*perforator.StartTaskResponse, error) {
	return c.taskClient.StartTask(ctx, req)
}

func (c *Client) ListTasks(ctx context.Context, req *perforator.ListTasksRequest) (*perforator.ListTasksResponse, error) {
	return c.taskClient.ListTasks(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////
