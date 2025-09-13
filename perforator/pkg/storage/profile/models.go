package profile

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

const (
	CompressionLabel = "compression"
	ServiceLabel     = "service"
	TimestampLabel   = "timestamp"
)

type ProfileData = []byte

type Storage interface {
	StoreProfile(
		ctx context.Context,
		metas []*meta.ProfileMetadata,
		body ProfileData,
	) (meta.ProfileID, error)

	ListServices(
		ctx context.Context,
		query *meta.ServiceQuery,
	) ([]*meta.ServiceMetadata, error)

	ListSuggestions(
		ctx context.Context,
		query *meta.SuggestionsQuery,
	) ([]*meta.Suggestion, error)

	SelectProfiles(
		ctx context.Context,
		query *meta.ProfileQuery,
	) ([]*meta.ProfileMetadata, error)

	FetchProfile(
		ctx context.Context,
		meta *meta.ProfileMetadata,
	) (ProfileData, error)

	CollectExpired(
		ctx context.Context,
		ttl time.Duration,
		pagination *util.Pagination,
		shardParams *storage.ShardParams,
	) ([]*storage.ObjectMeta, error)

	Delete(
		ctx context.Context,
		ids []string,
	) error
}
