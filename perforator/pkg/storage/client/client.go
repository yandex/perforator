package client

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/certifi"
	"github.com/yandex/perforator/perforator/pkg/endpointsetresolver"
	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	storagetvm "github.com/yandex/perforator/perforator/pkg/storage/tvm"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

const (
	MaxSendMessageSize = 1024 * 1024 * 1024 // 1 GB
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

type BinaryGRPCClientWriter struct {
	io.WriteCloser
	client perforatorstorage.PerforatorStorage_PushBinaryClient
}

func NewBinaryGRPCClientWriter(
	client perforatorstorage.PerforatorStorage_PushBinaryClient,
) *BinaryGRPCClientWriter {
	return &BinaryGRPCClientWriter{
		client: client,
	}
}

func (w *BinaryGRPCClientWriter) Write(p []byte) (int, error) {
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

func (w *BinaryGRPCClientWriter) Close() error {
	_, err := w.client.CloseAndRecv()
	return err
}

type TvmConfig struct {
	SecretVar        string `yaml:"tvm_secret_var"`
	ServiceFromTvmID uint32 `yaml:"from_service_id"`
	ServiceToTvmID   uint32 `yaml:"to_service_id"`
	CacheDir         string `yaml:"cache_dir"`
}

type GRPCConfig struct {
	MaxSendMessageSize uint32 `yaml:"max_send_message_size"`
}

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
	TvmConfig          *TvmConfig                            `yaml:"tvm"`
	GRPCConfig         GRPCConfig                            `yaml:"grpc,omitempty"`
	EndpointSet        endpointsetresolver.EndpointSetConfig `yaml:"endpoint_set,omitempty"`
	Host               string                                `yaml:"host,omitempty"`
	Port               uint32                                `yaml:"port,omitempty"`
	CertificateName    string                                `yaml:"name,omitempty"`
	CACertPath         string                                `yaml:"ca_cert_path,omitempty"`
	ProfileCompression string                                `yaml:"profile_compression,omitempty"`
	RPCTimeouts        Timeouts                              `yaml:"timeouts"`
}

type Client struct {
	conf             Config
	compressionFunc  CompressionFunction
	compressionCodec string
	creds            creds.DestroyablePerRPCCredentials
	connection       *grpc.ClientConn
	client           perforatorstorage.PerforatorStorageClient
	logger           xlog.Logger
}

func NewStorageClient(conf *Config, l xlog.Logger) (*Client, error) {
	l = l.WithName("storage.Client")

	if conf.Host == "" && conf.EndpointSet.ID == "" {
		return nil, errors.New("endpointset or host must be specified")
	}

	compressFunc, err := compressionFunctionFromString(conf.ProfileCompression)
	if err != nil {
		return nil, err
	}

	creds, err := getCreds(conf, l)
	if err != nil {
		return nil, err
	}

	var maxSendMsgSize uint32 = MaxSendMessageSize
	if conf.GRPCConfig.MaxSendMessageSize != 0 {
		maxSendMsgSize = conf.GRPCConfig.MaxSendMessageSize
	}

	var certPool *x509.CertPool
	if conf.CACertPath != "" {
		cert, err := os.ReadFile(conf.CACertPath)
		if err != nil {
			return nil, err
		}

		certPool = x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("failed to add server CA's certificate: %s", conf.CACertPath)
		}
	} else {
		certPool, err = certifi.NewDefaultCertPool()
		if err != nil {
			return nil, err
		}
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(
				certPool,
				conf.CertificateName,
			),
		),
		grpc.WithDefaultCallOptions(
			grpc.MaxSendMsgSizeCallOption{
				MaxSendMsgSize: int(maxSendMsgSize),
			},
		),
	}
	if creds != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(creds))
	}

	var target string
	if conf.Host != "" {
		target = conf.Host
	} else {
		endpointSetTarget, resolverOpts, err := endpointsetresolver.GetGrpcTargetAndResolverOpts(conf.EndpointSet, l)
		if err != nil {
			return nil, err
		}
		target = endpointSetTarget
		opts = append(opts, resolverOpts...)
	}

	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, err
	}

	configCopy := *conf
	configCopy.RPCTimeouts.fillDefault()

	return &Client{
		conf:             configCopy,
		compressionFunc:  compressFunc,
		compressionCodec: conf.ProfileCompression,
		creds:            creds,
		connection:       conn,
		client:           perforatorstorage.NewPerforatorStorageClient(conn),
		logger:           l,
	}, nil
}

func getCreds(conf *Config, l xlog.Logger) (creds.DestroyablePerRPCCredentials, error) {
	if conf.TvmConfig != nil {
		return storagetvm.NewTVMCredentials(
			conf.TvmConfig.ServiceFromTvmID,
			conf.TvmConfig.ServiceToTvmID,
			os.Getenv(conf.TvmConfig.SecretVar),
			conf.TvmConfig.CacheDir,
			l,
		)
	}
	return nil, nil
}

// return pushed profile size and error
func (c *Client) PushProfile(
	ctx context.Context,
	profileBytes []byte,
	labels map[string]string,
	buildIDs []string,
	envs []string,
	eventTypes []string,
) (uint64, error) {
	var err error
	if c.compressionFunc != nil {
		profileBytes, err = c.compressionFunc(profileBytes)
		if err != nil {
			return 0, fmt.Errorf("failed to compress profile: %w", err)
		}
		labels[profilestorage.CompressionLabel] = string(c.compressionCodec)
	}

	c.logger.Debug(ctx, "Pushing profile", log.Int("size", len(profileBytes)))

	ctx, cancel := context.WithTimeout(ctx, c.conf.RPCTimeouts.PushProfileTimeout)
	defer cancel()

	res, err := c.client.PushProfile(
		ctx,
		&perforatorstorage.PushProfileRequest{
			ProfileRepresentation: &perforatorstorage.PushProfileRequest_ProfileBytes{
				ProfileBytes: profileBytes,
			},
			Labels:     labels,
			BuildIDs:   buildIDs,
			Envs:       envs,
			EventTypes: eventTypes,
		},
	)
	if err != nil {
		c.logger.Error(ctx, "Failed to push profile", log.Error(err))
		return 0, err
	}

	c.logger.Debug(ctx, "Successfully pushed profile", log.String("id", res.ID))
	return uint64(len(profileBytes)), err
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

func (c *Client) PushBinary(ctx context.Context, buildID string) (io.WriteCloser, context.CancelFunc, error) {
	l := c.logger.With(log.String("build_id", buildID))
	l.Debug(ctx, "Pushing binary")

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
		l.Error(ctx, "failed push binary start client")
		return nil, nil, err
	}

	err = pushBinaryClient.Send(
		&perforatorstorage.PushBinaryRequest{
			Chunk: &perforatorstorage.PushBinaryRequest_HeadChunk{
				HeadChunk: &perforatorstorage.PushBinaryRequestHead{
					BuildID: buildID,
				},
			},
		},
	)
	if err != nil {
		l.Error(ctx, "failed push binary send buildid chunk")
		return nil, nil, err
	}

	writer := NewBinaryGRPCClientWriter(pushBinaryClient)
	l.Debug(ctx, "Successfully created push binary writer")
	return writer, cancel, nil
}

func (c *Client) Destroy() {
	_ = c.connection.Close()
	if c.creds != nil {
		c.creds.Destroy()
	}
}
