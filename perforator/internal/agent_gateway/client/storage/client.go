package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

func compressZstd(byteString []byte, level int) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	result := []byte{}
	return encoder.EncodeAll(byteString, result), nil
}

type compressionConfig struct {
	codec     compressionpb.CompressionMethod
	codecName string
	zstdLevel int
}

func (c *compressionConfig) compressBytes(data []byte) ([]byte, error) {
	switch c.codec {
	case compressionpb.CompressionMethod_Zstd:
		return compressZstd(data, c.zstdLevel)
	default:
		return nil, fmt.Errorf("unsupported compression codec %s", c.codec.String())
	}
}

func (c *compressionConfig) newWriter(w io.Writer) (io.WriteCloser, error) {
	switch c.codec {
	case compressionpb.CompressionMethod_Zstd:
		return zstd.NewWriter(
			w,
			zstd.WithEncoderConcurrency(1),
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(c.zstdLevel)),
		)
	default:
		return nil, fmt.Errorf("unsupported compression codec %s", c.codec.String())
	}
}

func compressionConfigFromString(compression string) (*compressionConfig, error) {
	if compression == "" {
		return nil, nil
	}

	if compression == "zstd" || strings.HasPrefix(compression, "zstd_") {
		level := 6
		if compression != "zstd" {
			_, err := fmt.Sscanf(compression, "zstd_%d", &level)
			if err != nil {
				return nil, err
			}
		}

		return &compressionConfig{
			codec:     compressionpb.CompressionMethod_Zstd,
			codecName: compression,
			zstdLevel: level,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized compression codec %s", compression)
}

type pushBinaryParams struct {
	uncompressedSize uint64
	attributes       map[string]string
}

type PushBinaryOption func(*pushBinaryParams)

func WithUncompressedSize(size uint64) PushBinaryOption {
	return func(p *pushBinaryParams) {
		p.uncompressedSize = size
	}
}

func WithBinaryAttributes(attributes map[string]string) PushBinaryOption {
	return func(p *pushBinaryParams) {
		p.attributes = maps.Clone(attributes)
	}
}

type BinaryWriter interface {
	io.Writer
	Close() error
	Abort()
}

type binaryGRPCClientWriter struct {
	stream          perforatorstorage.PerforatorStorage_PushBinaryClient
	cancelUploadCtx context.CancelFunc
	finishOnce      sync.Once
	recvErr         error
}

func newBinaryGRPCClientWriter(
	stream perforatorstorage.PerforatorStorage_PushBinaryClient,
	cancelUploadCtx context.CancelFunc,
) *binaryGRPCClientWriter {
	return &binaryGRPCClientWriter{
		stream:          stream,
		cancelUploadCtx: cancelUploadCtx,
	}
}

func (w *binaryGRPCClientWriter) Close() error {
	w.finishOnce.Do(func() {
		_, w.recvErr = w.stream.CloseAndRecv()
		if w.cancelUploadCtx != nil {
			w.cancelUploadCtx()
			w.cancelUploadCtx = nil
		}
	})
	return w.recvErr
}

func (w *binaryGRPCClientWriter) Abort() {
	w.finishOnce.Do(func() {
		if w.cancelUploadCtx != nil {
			w.cancelUploadCtx()
			w.cancelUploadCtx = nil
		}
	})
}

func (w *binaryGRPCClientWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	err := w.stream.Send(
		&perforatorstorage.PushBinaryRequest{
			Chunk: &perforatorstorage.PushBinaryRequest_BodyChunk{
				BodyChunk: &perforatorstorage.PushBinaryRequestBody{
					Binary: p,
				},
			},
		},
	)
	if errors.Is(err, io.EOF) {
		// grpc client stream contract:
		// if server closed the stream, Write returns io.EOF and aborts the stream.
		// Error can be obtained from CloseAndRecv.
		if recvErr := w.Close(); recvErr != nil {
			return 0, recvErr
		}
		// Could not obtain the server error - treat as error still
		return 0, errors.New("server closed PushBinary stream before upload finished")
	}
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// compressingBinaryWriter wraps a compression encoder on top of the gRPC stream
// writer. Its Close must flush the encoder AND finish the underlying gRPC stream:
// closing only the encoder flushes the compressed tail but never calls
// CloseAndRecv, so the server keeps blocking on Recv and never commits the upload.
type compressingBinaryWriter struct {
	encoder    io.WriteCloser
	grpcStream *binaryGRPCClientWriter
}

func (w *compressingBinaryWriter) Write(p []byte) (int, error) {
	return w.encoder.Write(p)
}

func (w *compressingBinaryWriter) Close() error {
	return errors.Join(w.encoder.Close(), w.grpcStream.Close())
}

func (w *compressingBinaryWriter) Abort() {
	w.grpcStream.Abort()
	_ = w.encoder.Close()
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
	ProfileCompression string   `yaml:"profile_compression,omitempty"`
	BinaryCompression  string   `yaml:"binary_compression,omitempty"`
	RPCTimeouts        Timeouts `yaml:"timeouts"`
}

func (c *Config) fillDefault() {
	c.RPCTimeouts.fillDefault()
}

type Client struct {
	conf               Config
	profileCompression *compressionConfig
	binaryCompression  *compressionConfig
	client             perforatorstorage.PerforatorStorageClient
	logger             xlog.Logger
}

func NewClient(conf *Config, l xlog.Logger, conn *grpc.ClientConn) (*Client, error) {
	l = l.WithName("PerforatorStorage.Client")
	conf.fillDefault()

	profileCompression, err := compressionConfigFromString(conf.ProfileCompression)
	if err != nil {
		return nil, err
	}

	binaryCompression, err := compressionConfigFromString(conf.BinaryCompression)
	if err != nil {
		return nil, err
	}

	return &Client{
		conf:               *conf,
		profileCompression: profileCompression,
		binaryCompression:  binaryCompression,
		client:             perforatorstorage.NewPerforatorStorageClient(conn),
		logger:             l,
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
	if c.profileCompression != nil {
		if len(profile.Raw) > 0 {
			profile.Raw, err = c.profileCompression.compressBytes(profile.Raw)
			if err != nil {
				return 0, fmt.Errorf("failed to compress pprof profile: %w", err)
			}
		}
		if len(profile.YaprofRaw) > 0 {
			profile.YaprofRaw, err = c.profileCompression.compressBytes(profile.YaprofRaw)
			if err != nil {
				return 0, fmt.Errorf("failed to compress yaprof profile: %w", err)
			}
		}
		newLabels := make(map[string]string, len(profile.Labels)+1)
		maps.Copy(newLabels, profile.Labels)
		newLabels[profilestorage.CompressionLabel] = c.profileCompression.codecName
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

func (c *Client) PushBinary(ctx context.Context, buildID string, opts ...PushBinaryOption) (BinaryWriter, error) {
	l := c.logger.With(log.String("build_id", buildID))
	l.Debug(ctx, "Pushing binary")

	params := &pushBinaryParams{}
	for _, opt := range opts {
		opt(params)
	}

	compression := compressionpb.CompressionMethod_None
	var uncompressedSize uint64
	if c.binaryCompression != nil {
		if params.uncompressedSize == 0 {
			return nil, fmt.Errorf("uncompressed_size is required when compression is set")
		}
		compression = c.binaryCompression.codec
		uncompressedSize = params.uncompressedSize
	}

	var err error
	uploadCtx, cancelUploadCtx := context.WithTimeout(ctx, c.conf.RPCTimeouts.PushBinaryTimeout)
	defer func() {
		if err != nil {
			cancelUploadCtx()
		}
	}()

	var pushBinaryClient perforatorstorage.PerforatorStorage_PushBinaryClient
	pushBinaryClient, err = c.client.PushBinary(uploadCtx)
	if err != nil {
		l.Error(ctx, "Failed to initialize binary upload")
		return nil, err
	}

	err = pushBinaryClient.Send(
		&perforatorstorage.PushBinaryRequest{
			Chunk: &perforatorstorage.PushBinaryRequest_HeadChunk{
				HeadChunk: &perforatorstorage.PushBinaryRequestHead{
					BuildID:          buildID,
					Compression:      compression,
					UncompressedSize: uncompressedSize,
					Attributes:       params.attributes,
				},
			},
		},
	)
	if err != nil {
		l.Error(ctx, "Failed to send binary upload header", log.Error(err))
		return nil, err
	}

	grpcWriter := newBinaryGRPCClientWriter(pushBinaryClient, cancelUploadCtx)
	var writer BinaryWriter = grpcWriter
	defer func() {
		if err != nil {
			writer.Abort()
		}
	}()

	if c.binaryCompression != nil {
		var encoder io.WriteCloser
		encoder, err = c.binaryCompression.newWriter(grpcWriter)
		if err != nil {
			l.Error(ctx, "Failed to create compression writer", log.Error(err))
			return nil, err
		}
		writer = &compressingBinaryWriter{encoder: encoder, grpcStream: grpcWriter}
	}

	l.Debug(ctx, "Successfully created push binary writer")
	return writer, nil
}
