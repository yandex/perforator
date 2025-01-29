package bundle

import (
	"context"
	"errors"
	"fmt"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	tasks "github.com/yandex/perforator/perforator/internal/asynctask/compound"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
	binarycompound "github.com/yandex/perforator/perforator/pkg/storage/binary/compound"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/pg"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	profilecompound "github.com/yandex/perforator/perforator/pkg/storage/profile/compound"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrPostgresClusterNotSpecified = errors.New("postgres cluster is not specified")
	ErrMetaStorageIsNotSpecified   = errors.New("no meta storage is specified")
	ErrS3StorageIsNotSpecified     = errors.New("s3 storage is not specified")
	ErrTasksStorageIsNotSpecified  = errors.New("no t tasks storage is specified")
)

type StorageBundle struct {
	conf *Config

	DBs *databases.Databases

	ProfileStorage    profilestorage.Storage
	BinaryStorage     binarystorage.StorageSelector
	MicroscopeStorage microscope.Storage
	TaskStorage       asynctask.TaskService
}

func NewStorageBundleFromConfig(ctx context.Context, l xlog.Logger, reg metrics.Registry, configPath string) (*StorageBundle, error) {
	conf, err := ParseConfig(configPath, false /* strict */)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return NewStorageBundle(ctx, l, reg, conf)
}

func NewStorageBundle(ctx context.Context, l xlog.Logger, reg metrics.Registry, c *Config) (*StorageBundle, error) {
	res := &StorageBundle{
		conf: c,
	}
	var err error

	res.DBs, err = databases.NewDatabases(ctx, l, &c.DBs)
	if err != nil {
		return nil, fmt.Errorf("failed to init dbs: %w", err)
	}

	if c.ProfileStorage != nil {
		if res.DBs.S3Client == nil {
			return nil, ErrS3StorageIsNotSpecified
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
			c.BinaryStorage.GSYMS3Bucket,
		))

		res.BinaryStorage, err = binarycompound.NewStorage(l, reg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to init binary storage: %w", err)
		}
	}

	if c.MicroscopeStorage != nil {
		if res.DBs.PostgresCluster == nil {
			return nil, ErrPostgresClusterNotSpecified
		}

		res.MicroscopeStorage = pg.NewPostgresMicroscopeStorage(l, res.DBs.PostgresCluster)
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
