package compound

import (
	"errors"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/blob"
	storage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewStorage(
	logger xlog.Logger,
	reg metrics.Registry,
	opts ...Option,
) (storage.Storage, error) {
	options := defaultOpts()
	for _, opt := range opts {
		opt(options)
	}

	if options.clickhouseConf == nil || options.clickhouseConn == nil {
		return nil, errors.New("no meta storage is specified")
	}

	if options.s3client == nil {
		return nil, errors.New("no blob storage is specified")
	}

	blobStorage, err := blob.NewStorage(logger, reg.WithPrefix("profile_storage"), blob.WithS3(options.s3client, options.s3bucket))
	if err != nil {
		return nil, err
	}

	metaStorage, err := clickhouse.NewStorage(logger, reg, options.clickhouseConn, options.clickhouseConf)
	if err != nil {
		return nil, err
	}

	return storage.NewStorage(logger, metaStorage, blobStorage, options.blobDownloadConcurrency), nil
}
