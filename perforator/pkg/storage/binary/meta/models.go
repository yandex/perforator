package binarymeta

import (
	"context"
	"errors"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
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
	Abort(ctx context.Context) error // no-op after successful commit
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
	}
)

type Storage interface {
	StoreBinary(
		ctx context.Context,
		binaryMeta *BinaryMeta,
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
