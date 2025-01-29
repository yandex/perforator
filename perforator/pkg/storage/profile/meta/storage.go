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
		ID                ProfileID
		BlobID            ProfileID
		System            string
		MainEventType     string
		AllEventTypes     []string
		Cluster           string
		Service           ServiceID
		PodID             string
		NodeID            string
		Timestamp         time.Time
		BuildIDs          []string
		Attributes        map[string]string
		LastUsedTimestamp time.Time
		Envs              []string
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

type Storage interface {
	StoreProfile(ctx context.Context, meta *ProfileMetadata) error

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
