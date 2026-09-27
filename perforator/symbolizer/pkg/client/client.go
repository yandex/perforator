package client

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/endpointsetresolver"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
	"github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/proto/profile"
)

////////////////////////////////////////////////////////////////////////////////

type FlamegraphOptions = perforator.FlamegraphOptions
type TextProfileOptions = perforator.TextProfileOptions
type RenderFormat = perforator.RenderFormat

type FormatOptions struct {
	Flamegraph  *FlamegraphOptions
	TextProfile *TextProfileOptions
}

type Config struct {
	MaxReceiveMessageSize uint64

	// One of `EndpointSet` or `URL` should be provided.
	EndpointSet endpointsetresolver.EndpointSetConfig
	URL         string

	// OAuth token with perforator:api scope.
	Insecure bool
	// TokenEnv and Token are mutually exclusive.
	TokenEnv string
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
	host   string
	port   uint16
	secure bool
}

const (
	MaxMessageSize = 256 * 1024 * 1024
)

// If you are patching Perforator, you can initialize this in a `func init()`
var requireToken bool = false

func NewClient(ctx context.Context, c *Config, l xlog.Logger) (*Client, error) {
	if c.URL == "" && c.EndpointSet.ID == "" {
		endpoint, err := getDefaultPerforatorEndpoint()
		if err != nil {
			return nil, fmt.Errorf("no perforator endpoint found: %w", err)
		}
		c.URL = fmt.Sprintf("%s:%d", endpoint.host, endpoint.port)
		c.Insecure = !endpoint.secure
	}
	if c.Token != "" && c.TokenEnv != "" {
		return nil, fmt.Errorf("both token and token_env are set")
	}
	hasToken := (c.Token != "") || (c.TokenEnv != "")

	if requireToken && !hasToken && os.Getenv("PERFORATOR_FORCE_DISABLE_TOKEN") != "yes" {
		return nil, fmt.Errorf("no OAuth token found")
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
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		grpc.WithUserAgent(makeUserAgentString()),
	}

	if hasToken {
		var token string
		if c.TokenEnv != "" {
			token = os.Getenv(c.TokenEnv)
			if token == "" {
				return nil, fmt.Errorf("token_env %s is not set", c.TokenEnv)
			}
		} else {
			token = c.Token
		}
		l.Debug(context.Background(), "Using provided OAuth token")
		opts = append(opts, grpc.WithPerRPCCredentials(oauth.TokenSource{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: token,
			}),
		}))
	} else if !c.Insecure {
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

	l.Debug(ctx, "Running health check on a new API client")
	healthClient := grpc_health_v1.NewHealthClient(conn)
	health, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		l.Warn(
			ctx,
			"Health check failed when creating API client",
			log.Error(err),
			log.String("hint", fmt.Sprintf("API address was deduced as %q, is it right?", target)),
		)
	} else if health.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		l.Warn(
			ctx,
			"Health check returned unexpected status when creating API client",
			log.String("status", health.Status.String()),
		)
	} else {
		l.Debug(
			ctx,
			"Health check succeeded",
		)
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
		Paginated: &pagination.Paginated{
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
				TimeInterval: &time_interval.TimeInterval{
					From: timestamppb.New(filters.FromTS),
					To:   timestamppb.New(filters.ToTS),
				},
			},
			Paginated: &pagination.Paginated{
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
		Paginated: &pagination.Paginated{
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

func (c *Client) DeleteMicroscope(
	ctx context.Context,
	id string,
) error {
	ctx, span := c.tracer.Start(ctx, "DeleteMicroscope")
	defer span.End()

	c.l.Info(
		ctx,
		"Delete microscope",
		log.String("id", id),
	)

	_, err := c.microscopesClient.DeleteMicroscope(ctx, &perforator.DeleteMicroscopeRequest{
		ID: id,
	})
	return err
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
	MaxSamples   uint32
	Format       *RenderFormat
	MergeOptions *profile.MergeOptions
	Experimental *perforator.MergeExperimentalOptions
}

func (c *Client) GetProfileByURL(profileURL string) ([]byte, error) {
	return c.fetchResult(nil, profileURL, false)
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
	taskAnnotation string,
) (string, []byte, []*perforator.ProfileMeta, error) {
	taskID, res, err := c.MergeProfilesProto(ctx, request, taskAnnotation)
	if err != nil {
		return "", nil, nil, err
	}

	buf, err := c.fetchResult(res.GetProfile(), res.GetProfileURL(), asURL)
	if err != nil {
		return "", nil, nil, err
	}

	return taskID, buf, res.GetProfileMeta(), nil
}

func (c *Client) MergeProfilesProto(
	ctx context.Context,
	request *perforator.MergeProfilesRequest,
	taskAnnotation string,
) (taskID string, res *perforator.MergeProfilesResponse, err error) {
	id, result, err := c.runTask(ctx, &perforator.TaskSpec{
		Kind: &perforator.TaskSpec_MergeProfiles{MergeProfiles: request},
	}, taskAnnotation)
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
	taskAnnotation string,
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
			TimeInterval: &time_interval.TimeInterval{
				From: timestamppb.New(request.FromTS),
				To:   timestamppb.New(request.ToTS),
			},
		},
		MaxSamples:   request.MaxSamples,
		MergeOptions: request.MergeOptions,
		Experimental: request.Experimental,
	}

	taskID, buf, profilesMeta, err := c.doMergeProfiles(ctx, req, asURL, taskAnnotation)
	c.l.Info(
		ctx,
		"Merged profile task id",
		log.String("task.id", taskID),
	)
	return buf, profilesMeta, err
}

func (c *Client) GetPGOProfile(
	ctx context.Context,
	selector string,
	format *perforator.PGOProfileFormat,
	asURL bool,
	taskAnnotation string,
) ([]byte, *perforator.PGOMeta, error) {
	_, span := c.tracer.Start(ctx, "GetPGOProfile")
	defer span.End()

	_, result, err := c.runTask(ctx, &perforator.TaskSpec{
		Kind: &perforator.TaskSpec_GeneratePGOProfile{
			GeneratePGOProfile: &perforator.GeneratePGOProfileRequest{
				Query: &perforator.ProfileQuery{
					Selector: selector,
				},
				Format: format,
			},
		},
	}, taskAnnotation)
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
	taskAnnotation string,
) ([]byte, error) {
	_, res, err := c.DiffProfilesProto(ctx, req, taskAnnotation)
	if err != nil {
		return nil, err
	}

	return c.fetchResult(res.GetProfile(), res.GetProfileURL(), asURL)
}

func (c *Client) DiffProfilesProto(
	ctx context.Context,
	req *perforator.DiffProfilesRequest,
	taskAnnotation string,
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
	}, taskAnnotation)

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

// pass one of flamegraphOptions or textProfileOptions, the other can be nil.
func (c *Client) UploadRenderedProfile(
	ctx context.Context,
	meta *perforator.ProfileMeta,
	formatOptions FormatOptions,
	profile *pprof.Profile,
	taskAnnotation string,
) (profileID string, taskID string, err error) {
	profileID, err = c.UploadProfile(ctx, meta, profile)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload profile: %w", err)
	}

	// Derive event type from profile for the selector.
	// The server requires event_type to be specified, otherwise it defaults to cpu.cycles.
	eventType := sampletype.SampleTypeCPUCycles
	if len(profile.SampleType) > 0 {
		eventType = sampletype.SampleTypeToString(profile.SampleType[0])
	}

	// Render the profile.
	req := &perforator.MergeProfilesRequest{
		Format: &perforator.RenderFormat{
			Symbolize: &perforator.SymbolizeOptions{
				Symbolize: ptr.Bool(false),
			},
		},
		Query: &perforator.ProfileQuery{
			Selector: fmt.Sprintf(`{system_name = "uploads", id="%s", event_type="%s"}`, profileID, eventType),
			TimeInterval: &time_interval.TimeInterval{
				From: timestamppb.New(meta.GetTimestamp().AsTime().Add(-time.Minute)),
			},
		},
	}
	if formatOptions.Flamegraph != nil {
		req.Format.Format = &perforator.RenderFormat_JSONFlamegraph{
			JSONFlamegraph: formatOptions.Flamegraph,
		}
	} else {
		req.Format.Format = &perforator.RenderFormat_TextProfile{
			TextProfile: formatOptions.TextProfile,
		}
	}

	// ClickHouse async_insert: profile may not be visible immediately after upload.
	// Poll with exponential backoff (100ms → 200ms → 400ms → … ≤ 5 s total)
	// instead of an unconditional 5 s sleep, so fast inserts return early.
	const maxWait = 5 * time.Second
	delay := 100 * time.Millisecond
	deadline := time.Now().Add(maxWait)
	for {
		taskID, _, err = c.MergeProfilesProto(ctx, req, taskAnnotation)
		if err == nil {
			break
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return "", "", fmt.Errorf("failed to render uploaded profile: %w", err)
		}
		if delay > remaining {
			delay = remaining
		}
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
	}
	c.l.Info(ctx,
		"Uploaded profile",
		log.String("task.id", taskID),
		log.String("profile.id", profileID),
	)

	return profileID, taskID, nil
}

func (c *Client) runTask(
	ctx context.Context,
	spec *perforator.TaskSpec,
	taskAnnotation string,
) (string, *perforator.TaskResult, error) {
	res, err := c.taskclient.StartTask(ctx, &perforator.StartTaskRequest{
		Spec:       spec,
		Annotation: taskAnnotation,
	})
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
			log.String("task.id", id),
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
