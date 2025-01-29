package asyncfilecache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/pubsub"
	"github.com/yandex/perforator/perforator/pkg/weightedlru"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrAlreadyReleased           = errors.New("acquired file reference already released")
	ErrWriteFailed               = errors.New("failed to write file")
	ErrFSStateSubscriptionClosed = errors.New("fs state subscription channel was closed")
	ErrDifferentSizePerKey       = errors.New("multiple acquire were called on same key with different size")
	ErrEmptyEntry                = errors.New("entry name is empty")
)

const (
	TmpSuffix        = ".tmp"
	DefaultCacheSize = 10000

	subFileStateChannelCapacity = 4
)

type Config struct {
	MaxSize  string `yaml:"max_size"`
	MaxItems uint64 `yaml:"max_items"`
	RootPath string `yaml:"root_path"`
}

type FileCache struct {
	l xlog.Logger
	r metrics.Registry
	c *Config

	cacheDir string

	cache *weightedlru.WeightedLRUCache
}

type FileState int

const (
	Absent FileState = iota
	Opened
	Stored
	WriteFailed
)

// item stored in weighted lru cache
type cacheEntry struct {
	size      uint64
	finalPath string

	closeWriter sync.Once
	closeError  error

	openWriter sync.Once
	openError  error
	writer     *WriterAt

	mutex sync.RWMutex
	state FileState // guarded by mutex

	pubsub *pubsub.PubSub[FileState]
}

func (e *cacheEntry) tryOpen() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.state == Absent {
		e.state = Opened
		e.pubsub.Publish(e.state)
		return true
	}

	return false
}

func (e *cacheEntry) setFSState(state FileState) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.state != state {
		e.state = state
		e.pubsub.Publish(e.state)
	}
}

func (e *cacheEntry) getFSState() FileState {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.state
}

func NewFileCache(
	conf *Config,
	l xlog.Logger,
	r metrics.Registry,
) (*FileCache, error) {
	err := os.Mkdir(conf.RootPath, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create path %s: %w", conf.RootPath, err)
	}

	maxSize, err := humanize.ParseBytes(conf.MaxSize)
	if err != nil {
		return nil, err
	}

	cache, err := weightedlru.NewWeightedLRUCache(
		maxSize,
		int(conf.MaxItems),
		evictLRUCallback(l.Logger()),
	)
	if err != nil {
		return nil, err
	}

	fileCache := &FileCache{
		l:        l,
		r:        r,
		c:        conf,
		cacheDir: conf.RootPath,
		cache:    cache,
	}

	err = fileCache.initCacheDir()
	if err != nil {
		return nil, err
	}

	return fileCache, nil
}

func (c *FileCache) EvictReleased() {
	c.cache.PurgeReleased()
}

func (c *FileCache) initCacheDir() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(c.cacheDir, entry.Name())

		if strings.HasSuffix(entry.Name(), TmpSuffix) {
			err := os.Remove(fullPath)
			if err != nil {
				c.l.Logger().Error("Failed to delete tmp file on init",
					log.String("entry", entry.Name()),
					log.Error(err),
				)
			}

			continue
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		_, err = c.cache.Add(
			fullPath,
			uint64(info.Size()),
			func() interface{} {
				return &cacheEntry{
					finalPath: fullPath,
					size:      uint64(info.Size()),
					state:     Stored,
					pubsub:    pubsub.NewPubSub[FileState](),
				}
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type AcquiredFileReference struct {
	// common between multiple AcquiredFileReferences
	entry    *cacheEntry
	sub      *pubsub.Subscription[FileState]
	cache    *FileCache
	released bool
}

func (f *AcquiredFileReference) Open() (io.WriterAt, func() error, error) {
	if f.released {
		return nil, nil, ErrAlreadyReleased
	}

	f.entry.openWriter.Do(func() {
		f.entry.writer, f.entry.openError = newWriter(f.entry)
	})

	return f.entry.writer, f.entry.writer.Close, f.entry.openError
}

func (f *AcquiredFileReference) Path() string {
	return f.entry.finalPath
}

func (f *AcquiredFileReference) Size() uint64 {
	return f.entry.size
}

func (f *AcquiredFileReference) State() FileState {
	return f.entry.getFSState()
}

func (f *AcquiredFileReference) WaitStored(ctx context.Context) error {
	if f.entry.getFSState() == Stored {
		return nil
	}

	for {
		select {
		case state, ok := <-f.sub.Chan():
			if !ok {
				return ErrFSStateSubscriptionClosed
			}

			switch state {
			case Stored:
				return nil
			case WriteFailed:
				return ErrWriteFailed
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (f *AcquiredFileReference) Close() {
	f.released = true
	f.sub.Close()

	// order Release for cache entry,
	//   we do not want to accidentally release and purge if status changes to Dumped shortly after getting it
	f.entry.mutex.Lock()
	defer f.entry.mutex.Unlock()

	if f.entry.state != Stored {
		f.cache.cache.ReleaseTryPurge(f.entry.finalPath)
		return
	}

	f.cache.cache.Release(f.entry.finalPath)
}

////////////////////////////////////////////////////////////////////////////////

func (c *FileCache) Acquire(entryName string, size uint64) (acquiredRef *AcquiredFileReference, inserted bool, err error) {
	if entryName == "" {
		return nil, false, ErrEmptyEntry
	}

	fullPath := filepath.Join(c.cacheDir, entryName)

	var lockedItem *weightedlru.LockedItem
	lockedItem, err = c.cache.Acquire(fullPath, size, func() interface{} {
		return &cacheEntry{
			size:      size,
			finalPath: fullPath,
			state:     Absent,
			pubsub:    pubsub.NewPubSub[FileState](),
		}
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to acquire path %s from weighted lru cache: %w", fullPath, err)
	}
	inserted = lockedItem.Inserted

	defer func() {
		if err != nil {
			c.cache.ReleaseTryPurge(fullPath)
		}
	}()

	entry := lockedItem.Value.(*cacheEntry)
	if entry.size != size {
		err = ErrDifferentSizePerKey
		return
	}

	acquiredRef = &AcquiredFileReference{
		entry: entry,
		sub:   entry.pubsub.Subscribe(subFileStateChannelCapacity),
		cache: c,
	}
	return
}

func (c *FileCache) Dir() string {
	return c.cacheDir
}
