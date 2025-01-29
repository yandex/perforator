package multi

import (
	"context"
	"errors"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

type MultiStorage struct {
	// Supports read-write operations.
	primary meta.Storage
	// Supports write-only operations.
	secondary meta.Storage
}

func NewStorage(primary, secondary meta.Storage) *MultiStorage {
	return &MultiStorage{primary, secondary}
}

var _ meta.Storage = (*MultiStorage)(nil)

// CollectExpiredProfiles implements meta.Storage.
func (s *MultiStorage) CollectExpiredProfiles(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
	shardParams storage.ShardParams,
) ([]*meta.ProfileMetadata, error) {
	return s.primary.CollectExpiredProfiles(ctx, ttl, pagination, shardParams)
}

// GetProfiles implements meta.Storage.
func (s *MultiStorage) GetProfiles(ctx context.Context, profileIDs []string) ([]*meta.ProfileMetadata, error) {
	return s.primary.GetProfiles(ctx, profileIDs)
}

// ListServices implements meta.Storage.
func (s *MultiStorage) ListServices(ctx context.Context, query *meta.ServiceQuery) ([]*meta.ServiceMetadata, error) {
	return s.primary.ListServices(ctx, query)
}

// ListSuggestions implements meta.Storage.
func (s *MultiStorage) ListSuggestions(ctx context.Context, query *meta.SuggestionsQuery) ([]*meta.Suggestion, error) {
	return s.primary.ListSuggestions(ctx, query)
}

// RemoveProfiles implements meta.Storage.
func (s *MultiStorage) RemoveProfiles(ctx context.Context, profileIDs []string) error {
	return s.primary.RemoveProfiles(ctx, profileIDs)
}

// SelectProfiles implements meta.Storage.
func (s *MultiStorage) SelectProfiles(ctx context.Context, query *meta.ProfileQuery) ([]*meta.ProfileMetadata, error) {
	return s.primary.SelectProfiles(ctx, query)
}

// StoreProfile implements meta.Storage.
func (s *MultiStorage) StoreProfile(ctx context.Context, meta *meta.ProfileMetadata) error {
	return errors.Join(
		s.primary.StoreProfile(ctx, meta),
		s.secondary.StoreProfile(ctx, meta),
	)
}
