package profile

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	profileproto "github.com/yandex/perforator/perforator/proto/profile"
)

var _ storage.Storage = (*ProfileStorage)(nil)
var _ Storage = (*ProfileStorage)(nil)

type Options struct {
	WriteInContainerFormat bool
}

type ProfileStorage struct {
	MetaStorage meta.Storage
	BlobStorage blob.Storage

	downloadSemaphore *semaphore.Weighted

	decompressor *zstd.Decoder

	opts Options

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
func (s *ProfileStorage) StoreProfile(ctx context.Context, metas []*meta.ProfileMetadata, profile *bundle.ProfileBundle, opts ...meta.StoreOption) (meta.ProfileID, error) {
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

	var blobData []byte
	if s.opts.WriteInContainerFormat {
		blobData, err = s.wrapInContainer(profile.GetPprof(), profile.GetYaprof())
		if err != nil {
			return "", fmt.Errorf("failed to wrap profile in container: %w", err)
		}
		for _, meta := range metas {
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]string)
			}
			meta.Attributes[BlobFormatLabel] = BlobFormatContainer
		}
	} else {
		blobData, err = profile.GetOrConvertPprof()
		if err != nil {
			return "", fmt.Errorf("failed to get pprof profile: %w", err)
		}
	}

	err = s.putBlob(ctx, id.String(), blobData)
	if err != nil {
		return "", err
	}

	s.log.Debug(ctx, "Successfully inserted profile blob",
		log.String("id", id.String()),
	)

	var joinedErr error
	for _, meta := range metas {
		err = s.MetaStorage.StoreProfile(ctx, meta, opts...)
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

func (s *ProfileStorage) uncompressZstd(byteString []byte) ([]byte, error) {
	result, err := s.decompressor.DecodeAll(byteString, []byte{})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ProfileStorage) uncompressIfNeeded(bytes []byte, compression string) ([]byte, error) {
	if strings.HasPrefix(compression, "zstd") {
		return s.uncompressZstd(bytes)
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
func (s *ProfileStorage) FetchProfile(ctx context.Context, meta *meta.ProfileMetadata) (*bundle.ProfileBundle, error) {
	data, err := s.getBlob(ctx, meta.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile %q blob: %w", meta.ID, err)
	}

	switch blobFormat := meta.Attributes[BlobFormatLabel]; blobFormat {
	case BlobFormatContainer:
		container := &profileproto.ProfileContainer{}
		if err := proto.Unmarshal(data, container); err != nil {
			return nil, fmt.Errorf("failed to unmarshal container for profile %s: %w", meta.ID, err)
		}
		profileBundle, err := s.bundleFromContainer(container, meta.ID)
		if err != nil {
			return nil, err
		}
		return profileBundle, nil
	case BlobFormatLegacyPprof:
		// Legacy pprof is the default: absent blob_format or blob_format with empty value.
		codec := meta.Attributes[CompressionLabel]
		data, err = s.uncompressIfNeeded(data, codec)
		if err != nil {
			return nil, fmt.Errorf("failed to uncompress profile %s, compression `%s`: %w", meta.ID, codec, err)
		}

		return bundle.NewPprofBundle(data), nil
	default:
		return nil, fmt.Errorf("unsupported profile blob format %q for profile %s", blobFormat, meta.ID)
	}
}

func (s *ProfileStorage) bundleFromContainer(container *profileproto.ProfileContainer, profileID meta.ProfileID) (*bundle.ProfileBundle, error) {
	var pprofData []byte
	var yaprofData []byte

	if container.Pprof != nil {
		var err error
		pprofData, err = s.uncompressPayload(container.Pprof, profileID)
		if err != nil {
			return nil, fmt.Errorf("failed to uncompress pprof payload for profile %s: %w", profileID, err)
		}
	}

	if container.Yaprof != nil {
		var err error
		yaprofData, err = s.uncompressPayload(container.Yaprof, profileID)
		if err != nil {
			return nil, fmt.Errorf("failed to uncompress yaprof payload for profile %s: %w", profileID, err)
		}
	}

	if pprofData == nil && yaprofData == nil {
		return nil, fmt.Errorf("profile container %s has no payload", profileID)
	}

	if pprofData != nil && yaprofData != nil {
		return bundle.NewBundle(pprofData, yaprofData), nil
	}
	if pprofData != nil {
		return bundle.NewPprofBundle(pprofData), nil
	}
	return bundle.NewYaprofBundle(yaprofData), nil
}

func (s *ProfileStorage) uncompressPayload(payload *profileproto.ProfileContainer_Payload, profileID meta.ProfileID) (ProfileData, error) {
	switch payload.CompressionMethod {
	case compressionpb.CompressionMethod_None:
		return payload.Data, nil
	case compressionpb.CompressionMethod_Zstd:
		return s.uncompressZstd(payload.Data)
	case compressionpb.CompressionMethod_Gzip:
		return s.uncompressGzip(payload.Data)
	default:
		return nil, fmt.Errorf("profile %s: unsupported compression method %v", profileID, payload.CompressionMethod)
	}
}

func (s *ProfileStorage) uncompressGzip(byteString []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(byteString))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

var zstdMagic = []byte{0x28, 0xb5, 0x2f, 0xfd}

var gzipMagic = []byte{0x1f, 0x8b}

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
			ID:                profile.ID,
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
	opts Options,
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
		opts:              opts,
	}, nil
}

func detectCompressionMethod(data []byte) compressionpb.CompressionMethod {
	if bytes.HasPrefix(data, zstdMagic) {
		return compressionpb.CompressionMethod_Zstd
	}
	if bytes.HasPrefix(data, gzipMagic) {
		return compressionpb.CompressionMethod_Gzip
	}
	return compressionpb.CompressionMethod_None
}

func (s *ProfileStorage) wrapInContainer(pprofBody []byte, yaprofBody []byte) ([]byte, error) {
	container := &profileproto.ProfileContainer{}

	if len(pprofBody) > 0 {
		container.Pprof = &profileproto.ProfileContainer_Payload{
			CompressionMethod: detectCompressionMethod(pprofBody),
			Data:              pprofBody,
		}
	}

	if len(yaprofBody) > 0 {
		container.Yaprof = &profileproto.ProfileContainer_Payload{
			CompressionMethod: detectCompressionMethod(yaprofBody),
			Data:              yaprofBody,
		}
	}

	return proto.Marshal(container)
}
