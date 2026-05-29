package storage

import (
	"context"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type CompressionFunction func([]byte) ([]byte, error)

func compressZstd(byteString []byte, level int) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	result := []byte{}
	return encoder.EncodeAll(byteString, result), nil
}

func getZstdCompressionFunction(level int) CompressionFunction {
	return func(byteString []byte) ([]byte, error) {
		return compressZstd(byteString, level)
	}
}

func compressionFunctionFromString(compression string) (CompressionFunction, error) {
	if strings.HasPrefix(compression, "zstd") {
		level := int(6)
		_, err := fmt.Sscanf(compression, "zstd_%d", &level)
		if err != nil {
			return nil, err
		}

		return getZstdCompressionFunction(level), nil
	}

	if compression == "" {
		return nil, nil
	}

	return nil, fmt.Errorf("unrecognized compression codec %s", compression)
}

type pushBinaryParams struct {
	compression      compressionpb.CompressionMethod
	uncompressedSize uint64
}

func defaultPushBinaryParams() *pushBinaryParams {
	return &pushBinaryParams{
		compression:      compressionpb.CompressionMethod_None,
		uncompressedSize: 0,
	}
}

type PushBinaryOption func(*pushBinaryParams)

func WithCompression(codec compressionpb.CompressionMethod, size uint64) PushBinaryOption {
	return func(p *pushBinaryParams) {
		if codec != compressionpb.CompressionMethod_None {
			p.compression = codec
			p.uncompressedSize = size
		}
	}
}

type binaryGRPCClientWriter struct {
	io.WriteCloser
	client   perforatorstorage.PerforatorStorage_PushBinaryClient
	cancelFn func()
}

func newBinaryGRPCClientWriter(
	client perforatorstorage.PerforatorStorage_PushBinaryClient,
	cancelFn func(),
) *binaryGRPCClientWriter {
	return &binaryGRPCClientWriter{
		client:   client,
		cancelFn: cancelFn,
	}
}

func (w *binaryGRPCClientWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	err := w.client.Send(
		&perforatorstorage.PushBinaryRequest{
			Chunk: &perforatorstorage.PushBinaryRequest_BodyChunk{
				BodyChunk: &perforatorstorage.PushBinaryRequestBody{
					Binary: p,
				},
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *binaryGRPCClientWriter) Close() error {
	if w.cancelFn != nil {
		w.cancelFn()
	}
	_, err := w.client.CloseAndRecv()
	return err
}

// TODO: remove because of retryConfig for grpc.ClientConn ?
type Timeouts struct {
	PushBinaryTimeout       time.Duration `yaml:"push_binary"`
	PushProfileTimeout      time.Duration `yaml:"push_profile"`
	AnnounceBinariesTimeout time.Duration `yaml:"announce_binaries"`
}

func (t *Timeouts) fillDefault() {
	if t.PushBinaryTimeout == time.Duration(0) {
		t.PushBinaryTimeout = 15 * time.Minute
	}
	if t.PushProfileTimeout == time.Duration(0) {
		t.PushProfileTimeout = time.Minute
	}
	if t.AnnounceBinariesTimeout == time.Duration(0) {
		t.AnnounceBinariesTimeout = 10 * time.Second
	}
}

type Config struct {
	ProfileCompression      string   `yaml:"profile_compression,omitempty"`
	EnableBinaryCompression *bool    `yaml:"enable_binary_compression,omitempty"`
	RPCTimeouts             Timeouts `yaml:"timeouts"`
}

func (c *Config) IsBinaryCompressionEnabled() bool {
	return c.EnableBinaryCompression != nil && *c.EnableBinaryCompression
}

func (c *Config) fillDefault() {
	c.RPCTimeouts.fillDefault()
	if c.EnableBinaryCompression == nil {
		c.EnableBinaryCompression = ptr.Bool(false)
	}
}

type Client struct {
	conf             Config
	compressionFunc  CompressionFunction
	compressionCodec string
	client           perforatorstorage.PerforatorStorageClient
	logger           xlog.Logger
}

func NewClient(conf *Config, l xlog.Logger, conn *grpc.ClientConn) (*Client, error) {
	l = l.WithName("PerforatorStorage.Client")
	conf.fillDefault()

	compressFunc, err := compressionFunctionFromString(conf.ProfileCompression)
	if err != nil {
		return nil, err
	}

	return &Client{
		conf:             *conf,
		compressionFunc:  compressFunc,
		compressionCodec: conf.ProfileCompression,
		client:           perforatorstorage.NewPerforatorStorageClient(conn),
		logger:           l,
	}, nil
}

type Profile struct {
	Raw                        []byte
	YaprofRaw                  []byte
	Labels                     map[string]string
	BuildIDs                   []string
	Envs                       []string
	EventTypes                 []string
	SignalTypes                []string
	CustomProfilingOperationID string
	StartTimestamp             time.Time
	Duration                   time.Duration
}

// return pushed profile size and error.
func (c *Client) PushProfile(
	ctx context.Context,
	profile *Profile,
) (uint64, error) {
	var err error
	if c.compressionFunc != nil {
		if len(profile.Raw) > 0 {
			profile.Raw, err = c.compressionFunc(profile.Raw)
			if err != nil {
				return 0, fmt.Errorf("failed to compress pprof profile: %w", err)
			}
		}
		if len(profile.YaprofRaw) > 0 {
			profile.YaprofRaw, err = c.compressionFunc(profile.YaprofRaw)
			if err != nil {
				return 0, fmt.Errorf("failed to compress yaprof profile: %w", err)
			}
		}
		newLabels := make(map[string]string, len(profile.Labels)+1)
		maps.Copy(newLabels, profile.Labels)
		newLabels[profilestorage.CompressionLabel] = string(c.compressionCodec)
		profile.Labels = newLabels
	}

	totalSize := len(profile.Raw) + len(profile.YaprofRaw)
	c.logger.Debug(ctx, "Pushing profile", log.Int("size", totalSize))

	ctx, cancel := context.WithTimeout(ctx, c.conf.RPCTimeouts.PushProfileTimeout)
	defer cancel()

	req := &perforatorstorage.PushProfileRequest{
		Labels:      profile.Labels,
		BuildIDs:    profile.BuildIDs,
		Envs:        profile.Envs,
		EventTypes:  profile.EventTypes,
		SignalTypes: profile.SignalTypes,
		CPOID:       profile.CustomProfilingOperationID,
	}

	if len(profile.Raw) > 0 {
		req.ProfileRepresentation = &perforatorstorage.PushProfileRequest_ProfileBytes{
			ProfileBytes: profile.Raw,
		}
	}
	if len(profile.YaprofRaw) > 0 {
		req.YaprofBytes = profile.YaprofRaw
	}

	if !profile.StartTimestamp.IsZero() {
		req.StartTimestamp = timestamppb.New(profile.StartTimestamp)
	}
	if profile.Duration > 0 {
		req.Duration = durationpb.New(profile.Duration)
	}

	res, err := c.client.PushProfile(ctx, req)
	if err != nil {
		c.logger.Error(ctx, "Failed to push profile", log.Error(err))
		return 0, err
	}

	c.logger.Debug(ctx, "Successfully pushed profile", log.String("id", res.ID))
	return uint64(totalSize), err
}

func (c *Client) AnnounceBinaries(ctx context.Context, availableBuildIDs []string) ([]string, error) {
	l := c.logger.With(log.Array("available_build_ids", availableBuildIDs))
	l.Debug(ctx, "Announcing binaries")

	ctx, cancel := context.WithTimeout(ctx, c.conf.RPCTimeouts.AnnounceBinariesTimeout)
	defer cancel()

	resp, err := c.client.AnnounceBinaries(
		ctx,
		&perforatorstorage.AnnounceBinariesRequest{
			AvailableBuildIDs: availableBuildIDs,
		},
	)
	if err != nil {
		l.Error(ctx, "Failed announce binaries")
		return nil, err
	}

	l.Debug(ctx, "Announced binaries", log.Array("unknown_build_ids", resp.UnknownBuildIDs))
	return resp.UnknownBuildIDs, nil
}

func (c *Client) PushBinary(ctx context.Context, buildID string, opts ...PushBinaryOption) (io.WriteCloser, error) {
	l := c.logger.With(log.String("build_id", buildID))
	l.Debug(ctx, "Pushing binary")

	params := defaultPushBinaryParams()
	for _, opt := range opts {
		opt(params)
	}

	if params.compression != compressionpb.CompressionMethod_None && params.uncompressedSize == 0 {
		return nil, fmt.Errorf("uncompressed_size is required when compression is set")
	}

	if params.compression != compressionpb.CompressionMethod_None && params.compression != compressionpb.CompressionMethod_Zstd {
		return nil, fmt.Errorf("unsupported compression codec %s", params.compression.String())
	}

	var err error
	ctx, cancel := context.WithTimeout(ctx, c.conf.RPCTimeouts.PushBinaryTimeout)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	var pushBinaryClient perforatorstorage.PerforatorStorage_PushBinaryClient
	pushBinaryClient, err = c.client.PushBinary(ctx)
	if err != nil {
		l.Error(ctx, "Failed to initialize binary upload")
		return nil, err
	}

	err = pushBinaryClient.Send(
		&perforatorstorage.PushBinaryRequest{
			Chunk: &perforatorstorage.PushBinaryRequest_HeadChunk{
				HeadChunk: &perforatorstorage.PushBinaryRequestHead{
					BuildID:          buildID,
					Compression:      params.compression,
					UncompressedSize: params.uncompressedSize,
				},
			},
		},
	)
	if err != nil {
		l.Error(ctx, "Failed to send binary upload header", log.Error(err))
		return nil, err
	}

	var writer io.WriteCloser = newBinaryGRPCClientWriter(pushBinaryClient, cancel)
	defer func() {
		if err != nil {
			closeErr := writer.Close()
			if closeErr != nil {
				l.Error(ctx, "Failed to close push binary writer on initialization error", log.Error(closeErr))
			}
		}
	}()

	switch params.compression {
	case compressionpb.CompressionMethod_None:
		break
	case compressionpb.CompressionMethod_Zstd:
		writer, err = zstd.NewWriter(writer, zstd.WithEncoderConcurrency(1))
		if err != nil {
			l.Error(ctx, "Failed to create compression writer", log.Error(err))
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported compression codec %s", params.compression.String())
	}

	l.Debug(ctx, "Successfully created push binary writer")
	return writer, nil
}
