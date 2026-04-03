package binarycompound

import (
	"errors"

	"github.com/yandex/perforator/library/go/core/metrics"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/binary/meta/pg"
	"github.com/yandex/perforator/perforator/pkg/storage/blob"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrUnspecifiedMetaStorage = errors.New("unspecified meta storage")
	ErrUnspecifiedBlobStorage = errors.New("unspecified blob storage")
)

func NewStorage(logger xlog.Logger, reg metrics.Registry, opts ...Option) (binarystorage.Storage, error) {
	options := defaultOpts()
	for _, applyOpt := range opts {
		applyOpt(options)
	}

	if options.s3client == nil {
		return nil, errors.New("no blob storage is specified")
	}

	blobStorage, err := blob.NewStorage(logger, reg.WithPrefix("binary_storage"), blob.WithS3(options.s3client, options.s3bucket))
	if err != nil {
		return nil, err
	}

	var metaStorage binarymeta.Storage
	switch {
	case options.postgresCluster != nil:
		metaStorage = pg.NewPostgresBinaryStorage(logger, reg, options.postgresCluster, pg.Options{})
	default:
		return nil, ErrUnspecifiedMetaStorage
	}

	return binarystorage.NewStorage(metaStorage, blobStorage, logger, reg), nil
}
