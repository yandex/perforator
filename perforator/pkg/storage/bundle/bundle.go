package bundle

import (
	"context"
	"errors"
	"fmt"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	tasks "github.com/yandex/perforator/perforator/internal/asynctask/compound"
	"github.com/yandex/perforator/perforator/pkg/lease"
	postgres_lease "github.com/yandex/perforator/perforator/pkg/lease/postgres"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	binarypg "github.com/yandex/perforator/perforator/pkg/storage/binary/meta/pg"
	"github.com/yandex/perforator/perforator/pkg/storage/blob"
	blobmodels "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	clustertop_factory "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/factory"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	cpo_postgres "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation/postgres"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	gsympg "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta/pg"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	postgres_microscope "github.com/yandex/perforator/perforator/pkg/storage/microscope/pg"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	profilecompound "github.com/yandex/perforator/perforator/pkg/storage/profile/compound"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrClickhouseConnNotSpecified  = errors.New("clickhouse conn is not specified")
	ErrPostgresClusterNotSpecified = errors.New("postgres cluster is not specified")
	ErrMetaStorageIsNotSpecified   = errors.New("no meta storage is specified")
	ErrS3StorageIsNotSpecified     = errors.New("s3 storage is not specified")
	ErrTasksStorageIsNotSpecified  = errors.New("no tasks storage is specified")
)

type StorageBundle struct {
	DBs *databases.Databases

	ProfileStorage                  profilestorage.Storage
	BinaryStorage                   *binarystorage.BinaryStorage
	GSYMStorage                     *gsymstorage.GSYMStorage
	MicroscopeStorage               microscope.Storage
	TaskStorage                     asynctask.TaskService
	CustomProfilingOperationStorage custom_profiling_operation.Storage
	ClusterTopGenerationsStorage    clustertop.Storage
	LeaseStorage                    lease.Storage
}

// bgCtx should be valid for as long as databases are used
func NewStorageBundleFromConfig(ctx context.Context, bgCtx context.Context, l xlog.Logger, app string, reg metrics.Registry, configPath string) (*StorageBundle, error) {
	conf, err := ParseConfig(configPath, false /* strict */)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return NewStorageBundle(ctx, bgCtx, l, app, reg, conf)
}

// bgCtx should be valid for as long as databases are used
func NewStorageBundle(ctx context.Context, bgCtx context.Context, l xlog.Logger, app string, reg metrics.Registry, c *Config) (*StorageBundle, error) {
	res := &StorageBundle{}
	var err error

	res.DBs, err = databases.NewDatabases(ctx, bgCtx, l, &c.DBs, app, reg)
	if err != nil {
		return nil, fmt.Errorf("failed to init dbs: %w", err)
	}

	if c.ProfileStorage != nil {
		if res.DBs.S3Client == nil {
			return nil, ErrS3StorageIsNotSpecified
		}
		if res.DBs.ClickhouseConn == nil {
			return nil, ErrClickhouseConnNotSpecified
		}

		res.ProfileStorage, err = profilecompound.NewStorage(
			l,
			reg,
			profilecompound.WithClickhouseMetaStorage(res.DBs.ClickhouseConn, &c.ProfileStorage.MetaStorage),
			profilecompound.WithS3(res.DBs.S3Client, c.ProfileStorage.S3Bucket),
			profilecompound.WithBlobDownloadConcurrency(c.ProfileStorage.BlobDownloadConcurrency),
			profilecompound.WithContainerFormat(c.ProfileStorage.WriteInContainerFormat),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to init profile storage: %w", err)
		}
	}

	if c.BinaryStorage != nil {
		metaStorage, err := res.newBinaryMetaStorage(l, reg, c.BinaryStorage.MetaStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to init binary meta storage: %w", err)
		}

		blobStorage, err := res.newS3BlobStorage(l, reg.WithPrefix("binary_storage"), c.BinaryStorage.S3Bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to init binary blob storage: %w", err)
		}

		res.BinaryStorage = binarystorage.NewStorage(metaStorage, blobStorage, l, reg)
	}

	if c.BinaryStorage != nil && c.BinaryStorage.GSYMS3Bucket != "" {
		if c.BinaryStorage.MetaStorage != binarystorage.PostgresMetaStorage {
			return nil, ErrMetaStorageIsNotSpecified
		}
		if res.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		gsymMetaStorage := gsympg.NewPostgresGSYMStorage(l, reg, res.DBs.PostgresCluster, gsympg.Options{})

		blobStorage, err := res.newS3BlobStorage(l, reg.WithPrefix("gsym_storage"), c.BinaryStorage.GSYMS3Bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to init gsym blob storage: %w", err)
		}

		res.GSYMStorage = gsymstorage.NewStorage(gsymMetaStorage, blobStorage, l)
	}

	if c.MicroscopeStorage != nil {
		if res.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}

		res.MicroscopeStorage = postgres_microscope.NewPostgresMicroscopeStorage(l, res.DBs.PostgresCluster)
	}

	if c.CustomProfilingOperationStorage != nil {
		switch *c.CustomProfilingOperationStorage {
		case custom_profiling_operation.Postgres:
			if res.DBs.PostgresCluster == nil {
				return nil, ErrPostgresClusterNotSpecified
			}
			res.CustomProfilingOperationStorage = cpo_postgres.NewStorage(l, res.DBs.PostgresCluster)
		default:
			return nil, fmt.Errorf("unknown custom profiling operation storage type: %s", *c.CustomProfilingOperationStorage)
		}
	}

	if c.ClusterTopStorage != nil {
		opts, err := res.createOptsFromClusterTopGenerationsStorageType(*c.ClusterTopStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster top storage options: %w", err)
		}

		res.ClusterTopGenerationsStorage, err = clustertop_factory.NewStorage(l, opts...)

		if err != nil {
			return nil, fmt.Errorf("failed to init cluster top storage: %w", err)
		}
	}

	if c.LeaseStorage != nil {
		switch *c.LeaseStorage {
		case lease.Postgres:
			if res.DBs.PostgresCluster == nil {
				return nil, ErrPostgresClusterNotSpecified
			}
			res.LeaseStorage = postgres_lease.NewStorage(l, res.DBs.PostgresCluster)
		default:
			return nil, fmt.Errorf("unknown lease storage type: %s", *c.LeaseStorage)
		}
	}

	if c.TaskStorage != nil {
		switch c.TaskStorage.StorageType {
		case tasks.Postgres:
			if res.DBs.PostgresCluster == nil {
				return nil, ErrPostgresClusterNotSpecified
			}
			res.TaskStorage, err = tasks.NewTasksService(l, reg, tasks.WithPostgresTasksStorage(c.TaskStorage, res.DBs.PostgresCluster))
		case tasks.InMemory:
			res.TaskStorage, err = tasks.NewTasksService(l, reg, tasks.WithInMemoryTasksStorage(c.TaskStorage))
		default:
			return nil, ErrTasksStorageIsNotSpecified
		}
		if err != nil {
			return nil, fmt.Errorf("failed to init tasks service: %w", err)
		}
	}

	return res, nil
}

func (b *StorageBundle) newBinaryMetaStorage(l xlog.Logger, reg metrics.Registry, storageType binarystorage.MetaStorageType) (binarymeta.Storage, error) {
	switch storageType {
	case binarystorage.PostgresMetaStorage:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		return binarypg.NewPostgresBinaryStorage(l, reg, b.DBs.PostgresCluster, binarypg.Options{}), nil
	default:
		return nil, ErrMetaStorageIsNotSpecified
	}
}

func (b *StorageBundle) createOptsFromClusterTopGenerationsStorageType(config clustertop.Config) ([]clustertop_factory.Option, error) {
	opts := []clustertop_factory.Option{}
	switch config.GenerationsStorage {
	case clustertop.Postgres:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		if b.DBs.ClickhouseConn == nil {
			return nil, ErrClickhouseConnNotSpecified
		}

		opts = append(
			opts,
			clustertop_factory.WithPostgresCluster(b.DBs.PostgresCluster),
			clustertop_factory.WithClickhouseConnection(b.DBs.ClickhouseConn),
			clustertop_factory.WithAsyncInsertConfig(config.AsyncInsert),
		)
	}

	return opts, nil
}

func (b *StorageBundle) newS3BlobStorage(l xlog.Logger, reg metrics.Registry, bucket string) (blobmodels.Storage, error) {
	if b.DBs.S3Client == nil {
		return nil, ErrS3StorageIsNotSpecified
	}
	return blob.NewStorage(l, reg, blob.WithS3(b.DBs.S3Client, bucket))
}
