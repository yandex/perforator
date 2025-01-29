package asyncfilecache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func createLogger() (xlog.Logger, error) {
	lconf := zap.KVConfig(log.DebugLevel)
	lconf.OutputPaths = []string{"stderr"}
	return xlog.TryNew(zap.New(lconf))
}

func initTest(t *testing.T, maxSize uint64) (context.Context, xlog.Logger, *FileCache) {
	reg := metricsmock.NewRegistry(nil)
	l, err := createLogger()
	require.NoError(t, err)

	cache, err := NewFileCache(&Config{
		MaxSize:  fmt.Sprintf("%dB", maxSize),
		MaxItems: 100000,
		RootPath: "./filecache_test",
	}, l, reg)
	require.NoError(t, err)

	cache.EvictReleased()

	return context.Background(), l, cache
}

func TestFileCache_Simple(t *testing.T) {
	ctx, _, cache := initTest(t, 3)

	fileName := "aboba"

	acquiredRef, inserted, err := cache.Acquire(fileName, 1)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, filepath.Join(cache.Dir(), fileName), acquiredRef.Path())
	require.Equal(t, uint64(1), acquiredRef.Size())

	writer, done, err := acquiredRef.Open()
	require.NoError(t, err)

	_, err = writer.WriteAt([]byte{'a'}, 0)
	require.NoError(t, err)
	err = done()
	require.NoError(t, err)

	_, inserted, err = cache.Acquire("toobig", 10)
	require.Error(t, err)
	require.False(t, inserted)

	err = acquiredRef.WaitStored(ctx)
	require.NoError(t, err)

	acquiredRef.Close()

	bigFilename := "big"

	newAcquired, inserted, err := cache.Acquire(bigFilename, 3)
	require.NoError(t, err)
	require.True(t, inserted)

	// check that previous file was evicted
	_, err = os.Stat(filepath.Join(cache.Dir(), fileName))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	writer, done, err = newAcquired.Open()
	require.NoError(t, err)
	_, err = writer.WriteAt([]byte{'a'}, 0)
	require.NoError(t, err)

	err = done()
	// size mismatch error here
	require.Error(t, err)

	_, err = os.Stat(filepath.Join(cache.Dir(), bigFilename+TmpSuffix))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(cache.Dir(), bigFilename))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	newAcquired.Close()

	// file must have been evicted because of unsucccessful write
	_, err = os.Stat(filepath.Join(cache.Dir(), bigFilename))
	require.Error(t, err)

	_, err = os.Stat(filepath.Join(cache.Dir(), bigFilename+TmpSuffix))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	cache.EvictReleased()
}

func TestFileCache_Errors(t *testing.T) {
	_, _, cache := initTest(t, 5)

	entry := ""
	_, _, err := cache.Acquire(entry, 3)
	require.Error(t, err)

	entry = "a"
	ref, inserted, err := cache.Acquire(entry, 2)
	require.NoError(t, err)
	require.True(t, inserted)

	writer, done, err := ref.Open()
	require.NoError(t, err)

	_, err = writer.WriteAt([]byte{'a'}, 2)
	require.Error(t, err)

	_, _, err = cache.Acquire("b", 4)
	// not enough cache capacity
	require.Error(t, err)

	err = done()
	require.Error(t, err)
	ref.Close()

	_, err = os.Stat(filepath.Join(cache.Dir(), entry))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	cache.EvictReleased()
}

func TestFileCache_MultipleAcquiredRef(t *testing.T) {
	_, _, cache := initTest(t, 5)

	key := "a"
	acquiredRef, inserted, err := cache.Acquire(key, 2)
	require.NoError(t, err)
	require.True(t, inserted)

	_, _, err = cache.Acquire(key, 3)
	require.Error(t, err)

	acquiredRef2, inserted2, err := cache.Acquire(key, 2)
	require.NoError(t, err)
	require.False(t, inserted2)

	writer, done, err := acquiredRef.Open()
	require.NoError(t, err)
	_, err = writer.WriteAt([]byte{'a'}, 0)
	require.NoError(t, err)
	_ = done

	writer, done, err = acquiredRef2.Open()
	require.NoError(t, err)
	_, err = writer.WriteAt([]byte{'a'}, 1)
	require.NoError(t, err)

	err = done()
	require.NoError(t, err)

	acquiredRef.Close()
	acquiredRef2.Close()

	_, err = os.Stat(filepath.Join(cache.Dir(), key))
	require.NoError(t, err)

	cache.EvictReleased()
}

func TestFileCache_WaitStored(t *testing.T) {
	ctx, _, cache := initTest(t, 5)

	ref, inserted, err := cache.Acquire("a", 1)
	require.NoError(t, err)
	require.True(t, inserted)

	written := atomic.Bool{}
	go func() {
		time.Sleep(300 * time.Millisecond)
		writer, done, err := ref.Open()
		require.NoError(t, err)
		_, err = writer.WriteAt([]byte{'a'}, 0)
		require.NoError(t, err)
		written.Store(true)
		err = done()
		require.NoError(t, err)
	}()

	err = ref.WaitStored(ctx)
	require.NoError(t, err)
	require.True(t, written.Load())

	err = ref.WaitStored(ctx)
	require.NoError(t, err)

	ref.Close()

	cache.EvictReleased()
}

func TestFileCache_Concurrent(t *testing.T) {
	_, _, cache := initTest(t, 150)

	g, _ := errgroup.WithContext(context.Background())

	storedFiles := sync.Map{}

	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("f%d", i)
		i := i

		numberOfRefs := 4
		acquiredGroup := sync.WaitGroup{}
		acquiredGroup.Add(numberOfRefs)

		for j := 0; j < numberOfRefs; j++ {
			j := j
			g.Go(func() error {
				ref, inserted, err := cache.Acquire(fileName, uint64(i))
				require.NoError(t, err)
				acquiredGroup.Done()
				if inserted {
					_, loaded := storedFiles.LoadOrStore(fileName, true)
					require.False(t, loaded, "file must have been inserted only one time")
				}

				writer, done, err := ref.Open()
				require.NoError(t, err)

				// simulate only one writer for each file
				if j == i%numberOfRefs {
					_, err = writer.WriteAt(make([]byte, i), 0)
					require.NoError(t, err)

					err = done()
					require.NoError(t, err)
				} else {
					acquiredGroup.Wait()
				}

				ref.Close()

				return nil
			})
		}
	}

	_ = g.Wait()

	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("f%d", i)
		_, ok := storedFiles.Load(fileName)
		require.True(t, ok)
		_, err := os.Stat(filepath.Join(cache.Dir(), fileName))
		require.NoError(t, err)
	}

	cache.EvictReleased()
}
