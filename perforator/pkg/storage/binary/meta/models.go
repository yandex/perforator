package binarymeta

import (
	"context"
	"errors"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

var (
	ErrUploadInProgress = errors.New("another upload is in progress")
	ErrAlreadyUploaded  = errors.New("already uploaded")
)

type UploadStatus string

const (
	Uploaded   UploadStatus = "uploaded"
	InProgress UploadStatus = "in_progress"
	NotStarted UploadStatus = "not_started"
)

type Commiter interface {
	Commit(ctx context.Context, blobInfo *storage.BlobInfo) error
	Ping(ctx context.Context) error
	Abort(ctx context.Context) error
}

type CompressionOption struct {
	Method                     compressionpb.CompressionMethod
	UnverifiedUncompressedSize uint64
}

type AttributesOption struct {
	Attributes map[string]string
}

type Option interface {
	Apply(*BinaryMetaOptions)
}

type BinaryMetaOptions struct {
	Compression CompressionOption
	Attributes  AttributesOption
}

func DefaultBinaryMetaOptions() *BinaryMetaOptions {
	return &BinaryMetaOptions{
		Compression: CompressionOption{
			Method:                     compressionpb.CompressionMethod_None,
			UnverifiedUncompressedSize: 0,
		},
		Attributes: AttributesOption{},
	}
}

type compressionOptionSetter CompressionOption

func (o compressionOptionSetter) apply(opts *BinaryMetaOptions) {
	opts.Compression = CompressionOption(o)
}

func (o compressionOptionSetter) Apply(opts *BinaryMetaOptions) {
	o.apply(opts)
}

func WithCompression(method compressionpb.CompressionMethod, unverifiedUncompressedSize uint64) Option {
	return compressionOptionSetter{Method: method, UnverifiedUncompressedSize: unverifiedUncompressedSize}
}

type attributesOptionSetter AttributesOption

func (o attributesOptionSetter) apply(opts *BinaryMetaOptions) {
	opts.Attributes = AttributesOption(o)
}

func (o attributesOptionSetter) Apply(opts *BinaryMetaOptions) {
	o.apply(opts)
}

func WithAttributes(attrs map[string]string) Option {
	return attributesOptionSetter{Attributes: attrs}
}

type (
	BinaryMeta struct {
		BuildID           string
		BlobInfo          *storage.BlobInfo
		GSYMBlobInfo      *storage.BlobInfo
		Timestamp         time.Time
		LastUsedTimestamp time.Time
		Status            UploadStatus
		Attributes        map[string]string
		Compression       compressionpb.CompressionMethod
		UncompressedSize  uint64
	}
)

type Storage interface {
	StoreBinary(
		ctx context.Context,
		buildID string,
		timestamp time.Time,
		opts ...Option,
	) (Commiter, error)

	GetBinaries(
		ctx context.Context,
		buildIDs []string,
	) ([]*BinaryMeta, error)

	// no shard support here
	CollectExpiredBinaries(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
	) ([]*BinaryMeta, error)

	RemoveBinaries(
		ctx context.Context,
		buildIDs []string,
	) error
}
