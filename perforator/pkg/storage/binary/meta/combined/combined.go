package combined

import (
	"context"
	"errors"
	"time"

	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

var (
	_ binarymeta.Storage = (*CombinedStorage)(nil)
)

type CombinedStorage struct {
	readStorage  binarymeta.Storage
	writeStorage binarymeta.Storage
}

func NewCombinedStorage(
	readStorage binarymeta.Storage,
	writeStorage binarymeta.Storage,
) *CombinedStorage {
	return &CombinedStorage{
		readStorage:  readStorage,
		writeStorage: writeStorage,
	}
}

func (s *CombinedStorage) StoreBinary(
	ctx context.Context,
	binaryMeta *binarymeta.BinaryMeta,
) (binarymeta.Commiter, error) {
	return s.writeStorage.StoreBinary(ctx, binaryMeta)
}

func removeDuplicates(metas []*binarymeta.BinaryMeta) []*binarymeta.BinaryMeta {
	seen := make(map[string]bool)
	result := make([]*binarymeta.BinaryMeta, 0, len(metas))
	for _, item := range metas {
		if !seen[item.BuildID] {
			seen[item.BuildID] = true
			result = append(result, item)
		}
	}
	return result
}

func (s *CombinedStorage) GetBinaries(
	ctx context.Context,
	buildIDs []string,
) (res []*binarymeta.BinaryMeta, err error) {
	metas, err := s.readStorage.GetBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}
	res = append(res, metas...)

	metas, err = s.writeStorage.GetBinaries(ctx, buildIDs)
	if err != nil {
		return nil, err
	}
	res = append(res, metas...)

	res = removeDuplicates(res)
	return
}

func (s *CombinedStorage) CollectExpiredBinaries(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
) ([]*binarymeta.BinaryMeta, error) {
	return nil, errors.New("collect expired is unsupported in combined storage")
}

func (s *CombinedStorage) RemoveBinaries(
	ctx context.Context,
	buildIDs []string,
) error {
	return errors.New("remove binaries is unsupported in combined storage")
}
