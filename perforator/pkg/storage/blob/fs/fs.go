package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/atomicfs"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var _ models.Storage = (*FSStorage)(nil)

type FSStorageConfig struct {
	Root string `yaml:"root"`
}

type FSStorage struct {
	c FSStorageConfig
	l xlog.Logger
}

func NewFSStorage(c FSStorageConfig, l xlog.Logger) (*FSStorage, error) {
	return &FSStorage{c, l.WithName("fsstorage")}, nil
}

// Delete implements Storage
func (s *FSStorage) Delete(ctx context.Context, key string) error {
	s.keylog(key).Debug(ctx, "Removing value")
	err := os.Remove(s.makepath(key))
	if err != nil {
		if os.IsNotExist(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		}
		s.keylog(key).Error(ctx, "Failed to remove value", log.Error(err))
		return fmt.Errorf("failed to remove value: %w", err)
	}
	return nil
}

// DeleteObjects implements Storage
func (s *FSStorage) DeleteObjects(ctx context.Context, keys []string) error {
	return errors.New("delete objects is unsupported in fs storage")
}

// Get implements Storage
func (s *FSStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.makepath(key))
	if err != nil {
		if os.IsNotExist(err) {
			err = &models.ErrNoExist{Err: err, Key: key}
		}
		return nil, fmt.Errorf("failed to locate key %s: %w", key, err)
	}
	return f, nil
}

// Size implements Storage
func (s *FSStorage) Size(ctx context.Context, key string) (uint64, error) {
	path := s.makepath(key)
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed os.Stat on %s: %w", path, err)
	}

	return uint64(info.Size()), nil
}

// Put implements Storage
func (s *FSStorage) Put(ctx context.Context, key string, src io.Reader) (err error) {
	path := s.makepath(key)

	l := s.keylog(key)
	l.Debug(ctx, "Generated key", log.String("path", path))

	defer func() {
		if err != nil {
			l.Error(ctx, "Failed to put value", log.Error(err))
		}
	}()

	if err = ensurepath(path); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := atomicfs.Create(path, atomicfs.WithMode(0644))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Discard()

	if _, err = io.Copy(file, src); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return file.Close()
}

func (s *FSStorage) List(ctx context.Context, pagination *models.Pagination, shards *storage.ShardParams) ([]string, error) {
	return nil, fmt.Errorf("operation not supported")
}

func (s *FSStorage) makepath(key string) string {
	return filepath.Join(s.c.Root, "store", key)
}

func ensurepath(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func (s *FSStorage) keylog(key string) xlog.Logger {
	return s.l.With(log.String("key", key))
}
