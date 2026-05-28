package blob

import (
	"errors"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/fs"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/s3"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewStorage(l xlog.Logger, reg metrics.Registry, opts ...Option) (Handle, error) {
	reg = reg.WithPrefix("blob")

	options := defaultOpts()
	for _, opt := range opts {
		opt(options)
	}

	switch {
	case options.s3client != nil:
		storage, err := s3.NewS3Storage(l, reg, options.s3client, options.s3bucket)
		if err != nil {
			return Handle{}, err
		}
		return Handle{
			Storage:  storage,
			Download: storage.ParallelDownloadConfig(),
		}, nil
	case options.fsPath != "":
		storage, err := fs.NewFSStorage(fs.FSStorageConfig{Root: options.fsPath}, l)
		if err != nil {
			return Handle{}, err
		}
		return Handle{
			Storage:  storage,
			Download: storage.ParallelDownloadConfig(),
		}, nil
	default:
		return Handle{}, errors.New("neither s3, nor fs storage is specified")
	}
}
