package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/semaphore"

	"github.com/yandex/perforator/library/go/core/log"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var _ storage.Storage = (*ProfileStorage)(nil)
var _ Storage = (*ProfileStorage)(nil)

type ProfileStorage struct {
	MetaStorage meta.Storage
	BlobStorage blob.Storage

	downloadSemaphore *semaphore.Weighted

	decompressor *zstd.Decoder

	log xlog.Logger
}

func (s *ProfileStorage) putBlob(ctx context.Context, id string, bytes []byte) error {
	writer, err := s.BlobStorage.Put(ctx, id)
	if err != nil {
		return err
	}
	_, err = writer.Write(bytes)
	if err != nil {
		return err
	}

	_, err = writer.Commit()
	return err
}

// implements profilestorage.Storage
func (s *ProfileStorage) StoreProfile(ctx context.Context, metas []*meta.ProfileMetadata, body []byte) (meta.ProfileID, error) {
	if len(metas) == 0 {
		return "", errors.New("no profile metas is specified")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	for _, meta := range metas {
		meta.ID = id.String()
	}

	s.log.Debug(ctx, "Store profile", log.Array("metas", metas))

	err = s.putBlob(ctx, id.String(), body)
	if err != nil {
		return "", err
	}

	s.log.Debug(ctx, "Successfully inserted profile blob",
		log.String("id", id.String()),
	)

	var joinedErr error
	for _, meta := range metas {
		err = s.MetaStorage.StoreProfile(ctx, meta)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}

	return id.String(), joinedErr
}

// implements profilestorage.Storage
func (s *ProfileStorage) ListServices(ctx context.Context, query *meta.ServiceQuery) ([]*meta.ServiceMetadata, error) {
	return s.MetaStorage.ListServices(ctx, query)
}

// implements profilestorage.Storage
func (s *ProfileStorage) ListSuggestions(
	ctx context.Context,
	query *meta.SuggestionsQuery,
) ([]*meta.Suggestion, error) {
	return s.MetaStorage.ListSuggestions(ctx, query)
}

func (s *ProfileStorage) uncompressZstd(byteString []byte, compression string) ([]byte, error) {
	result, err := s.decompressor.DecodeAll(byteString, []byte{})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ProfileStorage) uncompressIfNeeded(bytes []byte, compression string) ([]byte, error) {
	if strings.HasPrefix(compression, "zstd") {
		return s.uncompressZstd(bytes, compression)
	}

	return bytes, nil
}

func validateFiltersProfileQuery(q *meta.ProfileQuery) error {
	if len(q.Selector.Matchers) == 0 {
		return errors.New("at least one filter must be set: node id, pod id, build id, cpu, profile id or service")
	}

	return nil
}

func (s *ProfileStorage) getBlob(ctx context.Context, key meta.ProfileID) (ProfileData, error) {
	if err := s.downloadSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer s.downloadSemaphore.Release(1)

	buf := util.NewWriteAtBuffer(nil)

	err := s.BlobStorage.Get(ctx, string(key), buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// implements profilestorage.Storage
func (s *ProfileStorage) SelectProfiles(ctx context.Context, filters *meta.ProfileQuery) ([]*meta.ProfileMetadata, error) {
	s.log.Debug(ctx,
		"Select profiles",
		log.String("selector", filters.Selector.Repr()),
		log.UInt64("limit", filters.Limit),
		log.UInt64("offset", filters.Offset),
		log.UInt64("max_samples", filters.MaxSamples),
	)

	err := validateFiltersProfileQuery(filters)
	if err != nil {
		return nil, err
	}

	metas, err := s.MetaStorage.SelectProfiles(ctx, filters)
	if err != nil {
		return nil, err
	}

	return metas, nil
}

// implements profilestorage.Storage
func (s *ProfileStorage) FetchProfile(ctx context.Context, meta *meta.ProfileMetadata) (ProfileData, error) {
	data, err := s.getBlob(ctx, meta.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile %q blob: %w", meta.ID, err)
	}

	codec := meta.Attributes[CompressionLabel]
	data, err = s.uncompressIfNeeded(data, codec)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress profile %s, compression `%s`: %w", meta.ID, codec, err)
	}

	return data, nil
}

// implements profilestorage.Storage
func (s *ProfileStorage) CollectExpired(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
	shardParams *storage.ShardParams,
) ([]*storage.ObjectMeta, error) {
	profiles, err := s.MetaStorage.CollectExpiredProfiles(ctx, ttl, pagination, *shardParams)
	if err != nil {
		return nil, err
	}

	result := make([]*storage.ObjectMeta, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, &storage.ObjectMeta{
			ID: profile.ID,
			BlobInfo: &storage.BlobInfo{
				ID: profile.ID,
			},
			LastUsedTimestamp: profile.LastUsedTimestamp,
		})
	}

	return result, nil
}

// implements profilestorage.Storage
func (s *ProfileStorage) Delete(ctx context.Context, IDs []string) error {
	metas, err := s.MetaStorage.GetProfiles(ctx, IDs)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(metas))
	for _, meta := range metas {
		keys = append(keys, meta.ID)
	}

	err = s.BlobStorage.DeleteObjects(ctx, keys)
	if err != nil {
		return err
	}

	return s.MetaStorage.RemoveProfiles(ctx, IDs)
}

func NewStorage(
	logger xlog.Logger,
	metaStorage meta.Storage,
	blobStorage blob.Storage,
	blobDownloadConcurrency uint32,
) (*ProfileStorage, error) {
	if blobDownloadConcurrency == 0 {
		blobDownloadConcurrency = 32
	}

	decompressor, err := zstd.NewReader(nil)

	if err != nil {
		return nil, err
	}

	return &ProfileStorage{
		MetaStorage:       metaStorage,
		BlobStorage:       blobStorage,
		downloadSemaphore: semaphore.NewWeighted(int64(blobDownloadConcurrency)),
		log:               logger,
		decompressor:      decompressor,
	}, nil
}
