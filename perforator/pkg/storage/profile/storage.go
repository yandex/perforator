package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/errgroup"
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

type BlobID string

// Sometimes we request a lot of profiles from the storage, and profiles from different services
// might drastically differ in size. For example, a service instance running on 4000%CPU could
// generate ~x6 more profile data than a service instance running on 700%.
// We can't really know upfront whether N profiles would fit in memory, so this limit acts as
// "16Gb of profiling data should be enough for everyone".
const DefaultBatchDownloadTotalSizeSoftLimit uint64 = 16 * 1024 * 1024 * 1024

type ProfileStorage struct {
	MetaStorage meta.Storage
	BlobStorage blob.Storage

	downloadSemaphore *semaphore.Weighted

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

func (s *ProfileStorage) getBlob(ctx context.Context, key string) ([]byte, error) {
	buf := util.NewWriteAtBuffer(nil)

	err := s.BlobStorage.Get(ctx, string(key), buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
		meta.BlobID = id.String()
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

func uncompressZstd(byteString []byte, compression string) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	result, err := decoder.DecodeAll(byteString, []byte{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func uncompressIfNeeded(bytes []byte, compression string) ([]byte, error) {
	if strings.HasPrefix(compression, "zstd") {
		return uncompressZstd(bytes, compression)
	}

	return bytes, nil
}

func (s *ProfileStorage) fillProfileBlobFields(ctx context.Context, profile *Profile) error {
	var err error

	profile.Body, err = s.getBlob(ctx, profile.Meta.BlobID)
	if err != nil {
		return err
	}

	codec := profile.Meta.Attributes[CompressionLabel]
	profile.Body, err = uncompressIfNeeded(profile.Body, codec)
	if err != nil {
		return fmt.Errorf(
			"failed to uncompress profile %s, compression `%s`: %w",
			profile.Meta.ID,
			codec,
			err,
		)
	}

	return nil
}

func (s *ProfileStorage) downloadProfileBlobs(ctx context.Context, profiles []*Profile, batchDownloadTotalSizeSoftLimit uint64) error {
	var downloadedSizeApprox atomic.Uint64
	var droppedProfilesCount atomic.Uint64

	g, ctx := errgroup.WithContext(ctx)
	for _, profile := range profiles {
		profileCopy := profile
		g.Go(func() error {
			err := s.downloadSemaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer s.downloadSemaphore.Release(1)

			if downloadedSizeApprox.Load() >= batchDownloadTotalSizeSoftLimit {
				droppedProfilesCount.Add(1)
				return nil
			}

			err = s.fillProfileBlobFields(ctx, profileCopy)
			noExistErr := &blob.ErrNoExist{}
			if err != nil && !errors.As(err, &noExistErr) {
				return err
			}

			downloadedSizeApprox.Add(uint64(len(profileCopy.Body)))

			return nil
		})
	}

	droppedProfiles := droppedProfilesCount.Load()
	if droppedProfiles != 0 {
		s.log.Warn(
			ctx,
			"Some profiles were not loaded due to memory limits",
			log.UInt64("droppedProfiles", droppedProfiles),
			log.UInt64("downloadedSize", downloadedSizeApprox.Load()),
		)
	}

	return g.Wait()
}

func validateFiltersProfileQuery(q *meta.ProfileQuery) error {
	if len(q.Selector.Matchers) == 0 {
		return errors.New("at least one filter must be set: node id, pod id, build id, cpu, profile id or service")
	}

	return nil
}

func (s *ProfileStorage) doSelectProfiles(
	ctx context.Context,
	filters *meta.ProfileQuery,
	onlyMetadata bool,
	batchDownloadTotalSizeSoftLimit uint64,
) ([]*Profile, error) {
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

	result := make([]*Profile, 0, len(metas))
	for _, meta := range metas {
		result = append(result, &Profile{Meta: meta})
	}

	if onlyMetadata {
		return result, nil
	}

	err = s.downloadProfileBlobs(ctx, result, batchDownloadTotalSizeSoftLimit)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// implements profilestorage.Storage
func (s *ProfileStorage) SelectProfiles(ctx context.Context, filters *meta.ProfileQuery, onlyMetadata bool) ([]*Profile, error) {
	return s.doSelectProfiles(ctx, filters, onlyMetadata, DefaultBatchDownloadTotalSizeSoftLimit)
}

// implements profilestorage.Storage
func (s *ProfileStorage) SelectProfilesLimited(
	ctx context.Context,
	filters *meta.ProfileQuery,
	batchDownloadTotalSizeSoftLimit uint64,
) ([]*Profile, error) {
	return s.doSelectProfiles(ctx, filters, false /* onlyMetadata */, batchDownloadTotalSizeSoftLimit)
}

// implements profilestorage.Storage
func (s *ProfileStorage) GetProfiles(ctx context.Context, profileIDs []meta.ProfileID, onlyMetadata bool) ([]*Profile, error) {
	metas, err := s.MetaStorage.GetProfiles(ctx, profileIDs)
	if err != nil {
		return nil, err
	}

	res := make([]*Profile, 0, len(profileIDs))
	for _, meta := range metas {
		res = append(res, &Profile{Meta: meta})
	}

	if onlyMetadata {
		return res, nil
	}

	err = s.downloadProfileBlobs(ctx, res, DefaultBatchDownloadTotalSizeSoftLimit)
	if err != nil {
		return nil, err
	}

	return res, nil
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
				ID: profile.BlobID,
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
		keys = append(keys, meta.BlobID)
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
) *ProfileStorage {
	if blobDownloadConcurrency == 0 {
		blobDownloadConcurrency = 32
	}

	return &ProfileStorage{
		MetaStorage:       metaStorage,
		BlobStorage:       blobStorage,
		downloadSemaphore: semaphore.NewWeighted(int64(blobDownloadConcurrency)),
		log:               logger,
	}
}
