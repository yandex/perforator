package blob

import (
	"errors"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/fs"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/s3"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewStorage(l xlog.Logger, reg metrics.Registry, opts ...Option) (res models.Storage, err error) {
	reg = reg.WithPrefix("blob")

	options := defaultOpts()
	for _, opt := range opts {
		opt(options)
	}

	switch {
	case options.s3client != nil:
		res, err = s3.NewS3Storage(l, reg, options.s3client, options.s3bucket)
	case options.fsPath != "":
		res, err = fs.NewFSStorage(fs.FSStorageConfig{Root: options.fsPath}, l)
	default:
		return nil, errors.New("neither s3, nor fs storage is specified")
	}

	return
}
