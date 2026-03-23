package gsymcompound

import (
	"errors"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/blob"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	gsympg "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta/pg"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	errUnspecifiedMetaStorage = errors.New("unspecified meta storage")
)

func NewStorage(logger xlog.Logger, reg metrics.Registry, opts ...Option) (gsymstorage.Storage, error) {
	options := defaultOpts()
	for _, applyOpt := range opts {
		applyOpt(options)
	}

	if options.s3client == nil {
		return nil, errors.New("no blob storage is specified")
	}

	blobStorage, err := blob.NewStorage(logger, reg.WithPrefix("gsym_storage"), blob.WithS3(options.s3client, options.s3bucket))
	if err != nil {
		return nil, err
	}

	switch {
	case options.postgresCluster != nil:
		gsymMetaStorage := gsympg.NewPostgresGSYMStorage(logger, reg, options.postgresCluster, gsympg.Options{})
		return gsymstorage.NewStorage(gsymMetaStorage, blobStorage, logger), nil
	default:
		return nil, errUnspecifiedMetaStorage
	}
}
