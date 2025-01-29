package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/endpointsetresolver"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

////////////////////////////////////////////////////////////////////////////////

type FlamegraphOptions = perforator.FlamegraphOptions
type RenderFormat = perforator.RenderFormat

type Config struct {
	MaxReceiveMessageSize uint64

	// One of `EndpointSet` or `URL` should be provided.
	EndpointSet endpointsetresolver.EndpointSetConfig
	URL         string

	// OAuth token with perforator:api scope.
	Insecure bool
	Token    string
}

type Client struct {
	l      xlog.Logger
	tracer trace.Tracer

	md                metadata.MD
	connection        *grpc.ClientConn
	client            perforator.PerforatorClient
	microscopesClient perforator.MicroscopeServiceClient
	taskclient        perforator.TaskServiceClient
	httpclient        *resty.Client
}

type endpoint struct {
	url    string
	secure bool
}

const (
	MaxMessageSize = 256 * 1024 * 1024
)

func NewClient(c *Config, l xlog.Logger) (*Client, error) {
	if c.URL == "" && c.EndpointSet.ID == "" {
		endpoint, err := getDefaultPerforatorEndpoint()
		if err != nil {
			return nil, fmt.Errorf("no perforator endpoint found: %w", err)
		}
		c.URL = endpoint.url
		c.Insecure = !endpoint.secure
	}

	if !c.Insecure && c.Token == "" {
		return nil, errors.New("no OAuth token found")
	}

	transportDialOption, err := newTransportCredentialsDialOption(!c.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transport creds: %w", err)
	}

	opts := []grpc.DialOption{
		transportDialOption,
		grpc.WithDefaultCallOptions(grpc.MaxRecvMsgSizeCallOption{
			MaxRecvMsgSize: int(c.MaxReceiveMessageSize),
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: 30 * time.Second,
		}),
		grpc.WithMaxMsgSize(MaxMessageSize),
		grpc.WithChainUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		grpc.WithUserAgent(makeUserAgentString()),
	}

	if c.Token != "" {
		l.Debug(context.Background(), "Using provided OAuth token")
		opts = append(opts, grpc.WithPerRPCCredentials(oauth.TokenSource{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: c.Token,
			}),
		}))
	} else {
		l.Warn(context.Background(), "No OAuth token found")
	}

	var target string
	if c.URL != "" {
		target = c.URL

		l.Debug(
			context.Background(),
			"Connecting to the storage via url",
			log.String("url", target),
		)
	} else if c.EndpointSet.ID != "" {
		endpointSetTarget, resolverOpts, err := endpointsetresolver.GetGrpcTargetAndResolverOpts(c.EndpointSet, l)
		if err != nil {
			return nil, err
		}
		target = endpointSetTarget
		opts = append(opts, resolverOpts...)
	} else {
		return nil, fmt.Errorf("no perforator endpoint defined")
	}

	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		l:                 l,
		tracer:            otel.Tracer("Perforator proxy client"),
		connection:        conn,
		client:            perforator.NewPerforatorClient(conn),
		microscopesClient: perforator.NewMicroscopeServiceClient(conn),
		taskclient:        perforator.NewTaskServiceClient(conn),
		httpclient:        resty.New().SetTimeout(time.Hour).SetRetryCount(3),
	}, nil
}

func (c *Client) ListServices(
	ctx context.Context,
	offset, limit uint64,
	regex *string,
	pruneInterval *time.Duration,
	order string,
) ([]*perforator.ServiceMeta, error) {
	ctx, span := c.tracer.Start(ctx, "ListServices")
	defer span.End()

	c.l.Info(
		ctx,
		"List services",
		log.UInt64("offset", offset),
		log.UInt64("limit", limit),
		log.String("order", order),
	)

	var ord *perforator.ListServicesOrderByClause
	switch order {
	case "services":
		ord = perforator.ListServicesOrderByClause_Services.Enum()
	case "profiles":
		ord = perforator.ListServicesOrderByClause_ProfileCount.Enum()
	default:
		return nil, fmt.Errorf("unknown order: expected one of [services, profiles] got: %s", order)
	}

	var interval *durationpb.Duration
	if pruneInterval != nil {
		interval = durationpb.New(*pruneInterval)
	}

	req := &perforator.ListServicesRequest{
		Paginated: &perforator.Paginated{
			Offset: offset,
			Limit:  limit,
		},
		Regex:       regex,
		OrderBy:     ord,
		MaxStaleAge: interval,
	}

	resp, err := c.client.ListServices(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Services, nil
}

type ProfileFilters struct {
	FromTS   time.Time
	ToTS     time.Time
	Selector string
}

func (c *Client) ListProfiles(
	ctx context.Context,
	filters *ProfileFilters,
	offset,
	limit uint64,
) ([]*perforator.ProfileMeta, error) {
	ctx, span := c.tracer.Start(ctx, "ListProfiles")
	defer span.End()

	c.l.Info(
		ctx,
		"List profiles",
		log.Any("filters", filters),
		log.UInt64("offset", offset),
		log.UInt64("limit", limit),
	)

	resp, err := c.client.ListProfiles(
		ctx,
		&perforator.ListProfilesRequest{
			Query: &perforator.ProfileQuery{
				Selector: filters.Selector,
				TimeInterval: &perforator.TimeInterval{
					From: timestamppb.New(filters.FromTS),
					To:   timestamppb.New(filters.ToTS),
				},
			},
			Paginated: &perforator.Paginated{
				Offset: offset,
				Limit:  limit,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.Profiles, nil
}

type MicroscopesFilters struct {
	User        string
	StartsAfter *time.Time
}

func (c *Client) ListMicroscopes(
	ctx context.Context,
	filters *MicroscopesFilters,
	offset,
	limit uint64,
) ([]*perforator.Microscope, error) {
	ctx, span := c.tracer.Start(ctx, "ListMicroscopes")
	defer span.End()

	c.l.Info(
		ctx,
		"List microscopes",
		log.Any("filters", filters),
		log.UInt64("offset", offset),
		log.UInt64("limit", limit),
	)

	req := &perforator.ListMicroscopesRequest{
		Paginated: &perforator.Paginated{
			Offset: offset,
			Limit:  limit,
		},
		User: filters.User,
	}
	if filters.StartsAfter != nil {
		req.StartsAfter = timestamppb.New(*filters.StartsAfter)
	}

	resp, err := c.microscopesClient.ListMicroscopes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Microscopes, nil
}

func (c *Client) CreateMicroscope(
	ctx context.Context,
	selector string,
) (string, error) {
	ctx, span := c.tracer.Start(ctx, "CreateMicroscope")
	defer span.End()

	c.l.Info(
		ctx,
		"Create microscope",
		log.String("selector", selector),
	)

	resp, err := c.microscopesClient.SetMicroscope(ctx, &perforator.SetMicroscopeRequest{
		Selector: selector,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *Client) GetProfile(
	ctx context.Context,
	profileID string,
	format *RenderFormat,
) ([]byte, *perforator.ProfileMeta, error) {
	ctx, span := c.tracer.Start(ctx, "GetProfile")
	defer span.End()

	c.l.Info(ctx,
		"Get profile",
		log.String("profile_id", profileID),
	)

	resp, err := c.client.GetProfile(
		ctx,
		&perforator.GetProfileRequest{
			ProfileID: profileID,
			Format:    format,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return resp.Profile, resp.ProfileMeta, nil
}

type MergeProfilesRequest struct {
	ProfileFilters
	MaxSamples uint32
	Format     *RenderFormat
}

func (c *Client) fetchResult(profileBytes []byte, profileURL string, asURL bool) ([]byte, error) {
	buf := profileBytes
	if profileURL != "" {
		if asURL {
			return []byte(profileURL), nil
		}
		c.l.Info(context.Background(), "Downloading symbolization result", log.String("URL", profileURL))
		rsp, err := c.httpclient.R().Get(profileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch rendered profile: %w", err)
		}
		if !rsp.IsSuccess() {
			return nil, fmt.Errorf("failed to fetch rendered profile: got HTTP status %s", rsp.Status())
		}
		buf = rsp.Body()
	}

	return buf, nil
}

func (c *Client) doMergeProfiles(
	ctx context.Context,
	request *perforator.MergeProfilesRequest,
	asURL bool,
) ([]byte, []*perforator.ProfileMeta, error) {
	_, res, err := c.MergeProfilesProto(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	buf, err := c.fetchResult(res.GetProfile(), res.GetProfileURL(), asURL)
	if err != nil {
		return nil, nil, err
	}

	return buf, res.GetProfileMeta(), nil
}

func (c *Client) MergeProfilesProto(
	ctx context.Context,
	request *perforator.MergeProfilesRequest,
) (taskID string, res *perforator.MergeProfilesResponse, err error) {
	id, result, err := c.runTask(ctx, &perforator.TaskSpec{
		Kind: &perforator.TaskSpec_MergeProfiles{MergeProfiles: request},
	})
	if err != nil {
		return id, nil, err
	}

	switch v := result.GetKind().(type) {
	case *perforator.TaskResult_MergeProfiles:
		return id, v.MergeProfiles, nil
	default:
		return id, nil, fmt.Errorf("failed to parse async task response: unsuppported kind %+v", v)
	}
}

func (c *Client) MergeProfiles(
	ctx context.Context,
	request *MergeProfilesRequest,
	asURL bool,
) ([]byte, []*perforator.ProfileMeta, error) {
	ctx, span := c.tracer.Start(ctx, "MergeProfiles")
	defer span.End()

	c.l.Info(
		ctx,
		"Merging profiles",
		log.Any("filters", request.ProfileFilters),
		log.Any("format", request.Format),
	)

	req := &perforator.MergeProfilesRequest{
		Format: request.Format,
		Query: &perforator.ProfileQuery{
			Selector: request.Selector,
			TimeInterval: &perforator.TimeInterval{
				From: timestamppb.New(request.FromTS),
				To:   timestamppb.New(request.ToTS),
			},
		},
		MaxSamples: request.MaxSamples,
	}

	return c.doMergeProfiles(ctx, req, asURL)
}

func (c *Client) GetPGOProfile(
	ctx context.Context,
	service string,
	format *perforator.PGOProfileFormat,
	asURL bool,
) ([]byte, *perforator.PGOMeta, error) {
	_, span := c.tracer.Start(ctx, "GetPGOProfile")
	defer span.End()

	c.l.Info(
		ctx,
		"Get sPGO profile",
		log.Any("service", service),
	)

	_, result, err := c.runTask(ctx, &perforator.TaskSpec{
		Kind: &perforator.TaskSpec_GeneratePGOProfile{
			GeneratePGOProfile: &perforator.GeneratePGOProfileRequest{
				Service: service,
				Format:  format,
			},
		},
	})
	if err != nil {
		return nil, nil, err
	}

	res := result.GetGeneratePGOProfile()
	if res == nil {
		return nil, nil, fmt.Errorf("failed to parse async task response: unsuppported kind %T", result.GetKind())
	}

	buf, err := c.fetchResult(res.GetProfile(), res.GetProfileURL(), asURL)
	if err != nil {
		return nil, nil, err
	}

	return buf, res.GetPGOMeta(), nil
}

func (c *Client) DiffProfiles(
	ctx context.Context,
	req *perforator.DiffProfilesRequest,
	asURL bool,
) ([]byte, error) {
	_, res, err := c.DiffProfilesProto(ctx, req)
	if err != nil {
		return nil, err
	}

	return c.fetchResult(res.GetProfile(), res.GetProfileURL(), asURL)
}

func (c *Client) DiffProfilesProto(
	ctx context.Context,
	req *perforator.DiffProfilesRequest,
) (
	taskID string,
	rsp *perforator.DiffProfilesResponse,
	err error,
) {
	ctx, span := c.tracer.Start(ctx, "DiffProfiles")
	defer span.End()

	c.l.Info(
		ctx,
		"Diff profiles",
		log.Any("baseline_query", req.GetBaselineQuery()),
		log.Any("diff_query", req.GetDiffQuery()),
	)

	taskID, result, err := c.runTask(ctx, &perforator.TaskSpec{
		Kind: &perforator.TaskSpec_DiffProfiles{DiffProfiles: req},
	})

	if err != nil {
		return
	}

	res := result.GetDiffProfiles()
	if res == nil {
		err = fmt.Errorf("failed to parse async task response: unsuppported kind %T", result.GetKind())
		return
	}

	return taskID, res, err
}

func (c *Client) UploadProfile(ctx context.Context, meta *perforator.ProfileMeta, profile *pprof.Profile) (string, error) {
	var buf bytes.Buffer
	err := profile.Write(&buf)
	if err != nil {
		return "", fmt.Errorf("failed to serialize profile: %w", err)
	}

	res, err := c.client.UploadProfile(ctx, &perforator.UploadProfileRequest{
		Profile:     buf.Bytes(),
		ProfileMeta: meta,
	})
	if err != nil {
		return "", err
	}

	return res.GetProfileID(), nil
}

func (c *Client) UploadRenderedProfile(ctx context.Context, meta *perforator.ProfileMeta, profile *pprof.Profile) (profileID string, taskID string, err error) {
	profileID, err = c.UploadProfile(ctx, meta, profile)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload profile: %w", err)
	}

	// FIXME(sskvor): We store profiles in the Clickhouse using async_insert,
	// so UploadProfile may return when the profile is not available yet for reading.
	// Temporary kludge until we have a better way to synchronously upload profile.
	time.Sleep(time.Second * 5)

	// Render the profile.
	taskID, _, err = c.MergeProfilesProto(ctx, &perforator.MergeProfilesRequest{
		Format: &perforator.RenderFormat{
			Symbolize: &perforator.SymbolizeOptions{
				Symbolize: ptr.Bool(false),
			},
			Format: &perforator.RenderFormat_Flamegraph{
				Flamegraph: &perforator.FlamegraphOptions{
					MaxDepth:  ptr.Uint32(256),
					MinWeight: ptr.T(1e-10),
				},
			},
		},
		Query: &perforator.ProfileQuery{
			Selector: fmt.Sprintf(`{id="%s"}`, profileID),
			TimeInterval: &perforator.TimeInterval{
				From: timestamppb.New(meta.GetTimestamp().AsTime().Add(-time.Minute)),
			},
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to render uploaded profile: %w", err)
	}
	c.l.Info(ctx,
		"Uploaded profile",
		log.String("task.id", taskID),
		log.String("profile.id", profileID),
	)

	return profileID, taskID, nil
}

func (c *Client) runTask(ctx context.Context, spec *perforator.TaskSpec) (string, *perforator.TaskResult, error) {
	res, err := c.taskclient.StartTask(ctx, &perforator.StartTaskRequest{Spec: spec})
	if err != nil {
		return "", nil, err
	}
	id := res.GetTaskID()

	c.l.Debug(
		ctx,
		"Started async task",
		log.String("id", id),
	)

	ticker := time.NewTicker(time.Second)
	var task *perforator.GetTaskResponse
	var prevstate = perforator.TaskState_Created

	for {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-ticker.C:
		}

		task, err = c.taskclient.GetTask(ctx, &perforator.GetTaskRequest{TaskID: id})
		if err != nil {
			c.l.Warn(ctx, "Failed to fetch task status", log.Error(err), log.String("id", id))
		}

		state := task.GetStatus().GetState()

		attempts := task.GetStatus().GetAttempts()

		var (
			executor  string
			starttime time.Time
		)
		if n := len(attempts); n > 0 {
			attempt := attempts[n-1]
			executor = attempt.GetExecutor()
			starttime = time.UnixMicro(attempt.GetStartTime())
		}

		if state != prevstate {
			c.l.Debug(ctx, "Task state changed",
				log.Any("from", prevstate),
				log.Any("to", state),
			)
			prevstate = state
		}

		c.l.Debug(ctx, "Fetched task",
			log.Any("state", state),
			log.String("executor", executor),
			log.Duration("runtime", time.Since(starttime)),
			log.Int("attempt", len(attempts)),
		)

		if finalstatus[state] {
			break
		}
	}

	if task.GetStatus().GetState() == perforator.TaskState_Failed {
		return id, nil, fmt.Errorf("task failed: %s", task.GetStatus().GetError())
	}

	return id, task.GetResult(), nil
}

var finalstatus = map[perforator.TaskState]bool{
	perforator.TaskState_Finished: true,
	perforator.TaskState_Failed:   true,
}

////////////////////////////////////////////////////////////////////////////////
