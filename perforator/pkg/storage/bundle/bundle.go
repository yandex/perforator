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
	binarycompound "github.com/yandex/perforator/perforator/pkg/storage/binary/compound"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	clustertop_factory "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/factory"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	cpo_factory "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation/factory"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	gsymstorage "github.com/yandex/perforator/perforator/pkg/storage/gsym"
	gsymcompound "github.com/yandex/perforator/perforator/pkg/storage/gsym/compound"
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
	conf *Config

	DBs *databases.Databases

	ProfileStorage                  profilestorage.Storage
	BinaryStorage                   binarystorage.Storage
	GSYMStorage                     gsymstorage.Storage
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
	res := &StorageBundle{
		conf: c,
	}
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to init profile storage: %w", err)
		}
	}

	if c.BinaryStorage != nil {
		opts, err := res.createOptsFromMetaStorageType(c.BinaryStorage.MetaStorage, binaries)
		if err != nil {
			return nil, fmt.Errorf("failed to create binary storage options: %w", err)
		}
		opts = append(opts, binarycompound.WithS3(
			res.DBs.S3Client,
			c.BinaryStorage.S3Bucket,
		))

		res.BinaryStorage, err = binarycompound.NewStorage(l, reg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to init binary storage: %w", err)
		}
	}

	if c.BinaryStorage != nil && c.BinaryStorage.GSYMS3Bucket != "" {
		opts, err := res.createGSYMOptsFromMetaStorageType(c.BinaryStorage.MetaStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to create gsym storage options: %w", err)
		}
		opts = append(opts, gsymcompound.WithS3(
			res.DBs.S3Client,
			c.BinaryStorage.GSYMS3Bucket,
		))

		res.GSYMStorage, err = gsymcompound.NewStorage(l, reg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to init gsym storage: %w", err)
		}
	}

	if c.MicroscopeStorage != nil {
		if res.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}

		res.MicroscopeStorage = postgres_microscope.NewPostgresMicroscopeStorage(l, res.DBs.PostgresCluster)
	}

	if c.CustomProfilingOperationStorage != nil {
		opts, err := res.createCPOStorageOpts(*c.CustomProfilingOperationStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom profiling operation storage options: %w", err)
		}

		res.CustomProfilingOperationStorage, err = cpo_factory.NewStorage(l, *c.CustomProfilingOperationStorage, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom profiling operation storage: %w", err)
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
		opts, err := res.createOptsFromTasksStorageType(c.TaskStorage.StorageType)
		if err != nil {
			return nil, fmt.Errorf("failed to create tasks storage options: %w", err)
		}

		res.TaskStorage, err = tasks.NewTasksService(l, reg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to init tasks service: %w", err)
		}
	}

	return res, nil
}

type storageContent int

const (
	binaries storageContent = iota
)

func (b *StorageBundle) createOptsFromMetaStorageType(metaStorageType binarystorage.MetaStorageType, content storageContent) ([]binarycompound.Option, error) {
	opts := []binarycompound.Option{}
	switch metaStorageType {
	case binarystorage.PostgresMetaStorage:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		opts = append(opts, binarycompound.WithPostgresMetaStorage(b.DBs.PostgresCluster))
	default:
		return nil, ErrMetaStorageIsNotSpecified
	}

	return opts, nil
}

func (b *StorageBundle) createGSYMOptsFromMetaStorageType(metaStorageType binarystorage.MetaStorageType) ([]gsymcompound.Option, error) {
	opts := []gsymcompound.Option{}
	switch metaStorageType {
	case binarystorage.PostgresMetaStorage:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		opts = append(opts, gsymcompound.WithPostgresMetaStorage(b.DBs.PostgresCluster))
	default:
		return nil, ErrMetaStorageIsNotSpecified
	}

	return opts, nil
}

func (b *StorageBundle) createOptsFromTasksStorageType(tasksStorageType tasks.TasksStorageType) ([]tasks.Option, error) {
	opts := []tasks.Option{}
	switch tasksStorageType {
	case tasks.Postgres:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}
		opts = append(opts, tasks.WithPostgresTasksStorage(b.conf.TaskStorage, b.DBs.PostgresCluster))
	case tasks.InMemory:
		opts = append(opts, tasks.WithInMemoryTasksStorage(b.conf.TaskStorage))
	default:
		return nil, ErrTasksStorageIsNotSpecified
	}

	return opts, nil
}

func (b *StorageBundle) createCPOStorageOpts(storageType custom_profiling_operation.CustomProfilingOperationStorageType) ([]cpo_factory.Option, error) {
	opts := []cpo_factory.Option{}
	switch storageType {
	case custom_profiling_operation.Postgres:
		if b.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}

		opts = append(opts, cpo_factory.WithPostgresCluster(b.DBs.PostgresCluster))
	}

	return opts, nil
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

		opts = append(opts, clustertop_factory.WithPostgresCluster(b.DBs.PostgresCluster), clustertop_factory.WithClickhouseConnection(b.DBs.ClickhouseConn))
	}

	return opts, nil
}
