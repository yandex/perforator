package binarymeta

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

var (
	ErrUploadInProgress = errors.New("another upload is in progress")
	ErrAlreadyUploaded  = errors.New("already uploaded")
)

const DefaultUploadClaimStaleAfter = 5 * time.Minute

type UploadStatus string

const (
	Uploaded   UploadStatus = "uploaded"
	InProgress UploadStatus = "in_progress"
	NotStarted UploadStatus = "not_started"
)

type UploadClaim interface {
	Commit(ctx context.Context, blobInfo *storage.BlobInfo) error
	Ping(ctx context.Context) error
	Abort(ctx context.Context) error
}

type CompressionOption struct {
	Method                     compressionpb.CompressionMethod
	UnverifiedUncompressedSize uint64
}

type AttributesOption struct {
	Attributes *perforatorstorage.BinaryAttributes
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

func WithAttributes(attrs *perforatorstorage.BinaryAttributes) Option {
	return attributesOptionSetter{Attributes: proto.CloneOf(attrs)}
}

type (
	BinaryMeta struct {
		BuildID           string
		BlobInfo          *storage.BlobInfo
		GSYMBlobInfo      *storage.BlobInfo
		Timestamp         time.Time
		LastUsedTimestamp time.Time
		Status            UploadStatus
		Attributes        *perforatorstorage.BinaryAttributes
		Compression       compressionpb.CompressionMethod
		UncompressedSize  uint64
	}
)

type Storage interface {
	BeginUpload(
		ctx context.Context,
		buildID string,
		timestamp time.Time,
		opts ...Option,
	) (UploadClaim, error)

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
