package meta

import (
	"context"
	"time"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

type (
	ServiceID = string
	ProfileID = string

	ProfileMetadata struct {
		ID                         ProfileID
		System                     string
		MainEventType              string
		AllEventTypes              []string
		Cluster                    string
		Service                    ServiceID
		PodID                      string
		NodeID                     string
		Timestamp                  time.Time
		BuildIDs                   []string
		Attributes                 map[string]string
		LastUsedTimestamp          time.Time
		Envs                       []string
		CustomProfilingOperationID string
		BlobSize                   uint64
	}

	ProfileQuery struct {
		util.Pagination
		util.SortOrder
		Selector   *querylang.Selector
		MaxSamples uint64
	}

	ServiceMetadata struct {
		Service      ServiceID
		LastUpdate   time.Time
		ProfileCount uint64
	}

	ServiceQuery struct {
		util.Pagination
		util.SortOrder
		Regex       *string
		MaxStaleAge *time.Duration
	}

	SuggestionsQuery struct {
		Field    string
		Regex    *string
		Selector *querylang.Selector
		util.Pagination
	}

	Suggestion struct {
		Value string
	}
)

// StoreOption is a functional option for StoreProfile.
type StoreOption func(*StoreOptions)

// StoreOptions holds optional configuration for StoreProfile.
type StoreOptions struct {
	PersistCallback func(*ProfileMetadata)
}

// WithPersistCallback sets a callback that will be called after the profile
// is actually persisted to the database. For async implementations (e.g.
// batching ClickHouse writer), this fires after the batch is flushed,
// not when StoreProfile returns.
func WithPersistCallback(cb func(*ProfileMetadata)) StoreOption {
	return func(o *StoreOptions) {
		o.PersistCallback = cb
	}
}

// BuildStoreOptions collects options into StoreOptions.
func BuildStoreOptions(opts []StoreOption) StoreOptions {
	var o StoreOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

type Storage interface {
	StoreProfile(ctx context.Context, meta *ProfileMetadata, opts ...StoreOption) error

	ListServices(ctx context.Context, query *ServiceQuery) ([]*ServiceMetadata, error)

	ListSuggestions(ctx context.Context, query *SuggestionsQuery) ([]*Suggestion, error)

	SelectProfiles(ctx context.Context, query *ProfileQuery) ([]*ProfileMetadata, error)

	GetProfiles(ctx context.Context, profileIDs []ProfileID) ([]*ProfileMetadata, error)

	CollectExpiredProfiles(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
		shardParams storage.ShardParams,
	) ([]*ProfileMetadata, error)

	RemoveProfiles(ctx context.Context, profileIDs []ProfileID) error
}
