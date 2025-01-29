package puddle_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/puddle/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

type Counter struct {
	mutex sync.Mutex
	n     int
}

// Next increments the counter and returns the value
func (c *Counter) Next() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.n += 1
	return c.n
}

// Value returns the counter
func (c *Counter) Value() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.n
}

func createConstructor() (puddle.Constructor[int], *Counter) {
	var c Counter
	f := func(ctx context.Context) (int, error) {
		return c.Next(), nil
	}
	return f, &c
}

func stubDestructor(int) {}

func TestNewPoolRequiresMaxSizeGreaterThan0(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: -1})
	assert.Nil(t, pool)
	assert.Error(t, err)

	pool, err = puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 0})
	assert.Nil(t, pool)
	assert.Error(t, err)
}

func TestPoolAcquireCreatesResourceWhenNoneIdle(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())
	assert.WithinDuration(t, time.Now(), res.CreationTime(), time.Second)
	res.Release()
}

func TestPoolAcquireCallsConstructorWithAcquireContextValuesButNotDeadline(t *testing.T) {
	constructor := func(ctx context.Context) (int, error) {
		if ctx.Value("test") != "from Acquire" {
			return 0, errors.New("did not get value from Acquire")
		}
		if _, ok := ctx.Deadline(); ok {
			return 0, errors.New("should not have gotten deadline from Acquire")
		}

		return 1, nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.WithValue(context.Background(), "test", "from Acquire")
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	res, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())
	assert.WithinDuration(t, time.Now(), res.CreationTime(), time.Second)
	res.Release()
}

func TestPoolAcquireCalledConstructorIsNotCanceledByAcquireCancellation(t *testing.T) {
	constructor := func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 1, nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	res, err := pool.Acquire(ctx)
	assert.Nil(t, res)
	assert.Equal(t, context.DeadlineExceeded, err)

	time.Sleep(200 * time.Millisecond)

	assert.EqualValues(t, 1, pool.Stat().TotalResources())
	assert.EqualValues(t, 1, pool.Stat().CanceledAcquireCount())
}

func TestPoolAcquireDoesNotCreatesResourceWhenItWouldExceedMaxSize(t *testing.T) {
	constructor, createCounter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)

	wg := &sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 100; j++ {
				res, err := pool.Acquire(context.Background())
				assert.NoError(t, err)
				assert.Equal(t, 1, res.Value())
				res.Release()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 1, createCounter.Value())
	assert.EqualValues(t, 1, pool.Stat().TotalResources())
}

func TestPoolAcquireWithCancellableContext(t *testing.T) {
	constructor, createCounter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)

	wg := &sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 100; j++ {
				ctx, cancel := context.WithCancel(context.Background())
				res, err := pool.Acquire(ctx)
				assert.NoError(t, err)
				assert.Equal(t, 1, res.Value())
				res.Release()
				cancel()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 1, createCounter.Value())
	assert.EqualValues(t, 1, pool.Stat().TotalResources())
}

func TestPoolAcquireReturnsErrorFromFailedResourceCreate(t *testing.T) {
	errCreateFailed := errors.New("create failed")
	constructor := func(ctx context.Context) (int, error) {
		return 0, errCreateFailed
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	assert.Equal(t, errCreateFailed, err)
	assert.Nil(t, res)
}

func TestPoolAcquireCreatesResourceRespectingContext(t *testing.T) {
	var cancel func()
	constructor := func(ctx context.Context) (int, error) {
		cancel()
		// sleep to give a chance for the acquire to recognize it's cancelled
		time.Sleep(10 * time.Millisecond)
		return 1, nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)
	defer pool.Close()

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err = pool.Acquire(ctx)
	assert.ErrorIs(t, err, context.Canceled)

	// wait for the constructor to sleep and then for the resource to be added back
	// to the idle pool
	time.Sleep(100 * time.Millisecond)

	stat := pool.Stat()
	assert.EqualValues(t, 1, stat.IdleResources())
	assert.EqualValues(t, 1, stat.TotalResources())
}

func TestPoolAcquireReusesResources(t *testing.T) {
	constructor, createCounter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())

	res.Release()

	res, err = pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())

	res.Release()

	assert.Equal(t, 1, createCounter.Value())
}

func TestPoolTryAcquire(t *testing.T) {
	constructor, createCounter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)

	// Pool is initially empty so TryAcquire fails but starts construction of resource in the background.
	res, err := pool.TryAcquire(context.Background())
	require.EqualError(t, err, puddle.ErrNotAvailable.Error())
	assert.Nil(t, res)

	// Wait for background creation to complete.
	time.Sleep(100 * time.Millisecond)

	res, err = pool.TryAcquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())
	defer res.Release()

	res, err = pool.TryAcquire(context.Background())
	require.EqualError(t, err, puddle.ErrNotAvailable.Error())
	assert.Nil(t, res)

	assert.Equal(t, 1, createCounter.Value())
}

func TestPoolTryAcquireReturnsErrorWhenPoolIsClosed(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	pool.Close()

	res, err := pool.TryAcquire(context.Background())
	assert.Equal(t, puddle.ErrClosedPool, err)
	assert.Nil(t, res)
}

func TestPoolTryAcquireWithFailedResourceCreate(t *testing.T) {
	errCreateFailed := errors.New("create failed")
	constructor := func(ctx context.Context) (int, error) {
		return 0, errCreateFailed
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.TryAcquire(context.Background())
	require.EqualError(t, err, puddle.ErrNotAvailable.Error())
	assert.Nil(t, res)
}

func TestPoolAcquireNilContextDoesNotLeavePoolLocked(t *testing.T) {
	constructor, createCounter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	assert.Panics(t, func() { pool.Acquire(nil) })

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())
	res.Release()

	assert.Equal(t, 1, createCounter.Value())
}

func TestPoolAcquireContextAlreadyCanceled(t *testing.T) {
	constructor := func(ctx context.Context) (int, error) {
		panic("should never be called")
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := pool.Acquire(ctx)
	assert.Equal(t, context.Canceled, err)
	assert.Nil(t, res)
}

func TestPoolAcquireContextCanceledDuringCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	timeoutChan := time.After(1 * time.Second)

	var constructorCalls Counter
	constructor := func(ctx context.Context) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timeoutChan:
		}
		return constructorCalls.Next(), nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(ctx)
	assert.Equal(t, context.Canceled, err)
	assert.Nil(t, res)
}

func TestPoolAcquireAllIdle(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	resources := make([]*puddle.Resource[int], 4)

	resources[0], err = pool.Acquire(context.Background())
	require.NoError(t, err)
	resources[1], err = pool.Acquire(context.Background())
	require.NoError(t, err)
	resources[2], err = pool.Acquire(context.Background())
	require.NoError(t, err)
	resources[3], err = pool.Acquire(context.Background())
	require.NoError(t, err)

	assert.Len(t, pool.AcquireAllIdle(), 0)

	resources[0].Release()
	resources[3].Release()

	assert.ElementsMatch(t, []*puddle.Resource[int]{resources[0], resources[3]}, pool.AcquireAllIdle())

	resources[0].Release()
	resources[3].Release()
	resources[1].Release()
	resources[2].Release()

	assert.ElementsMatch(t, resources, pool.AcquireAllIdle())

	resources[0].Release()
	resources[1].Release()
	resources[2].Release()
	resources[3].Release()
}

func TestPoolAcquireAllIdleWhenClosedIsNil(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	pool.Close()
	assert.Nil(t, pool.AcquireAllIdle())
}

func TestPoolCreateResource(t *testing.T) {
	constructor, counter := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	err = pool.CreateResource(context.Background())
	require.NoError(t, err)

	stats := pool.Stat()
	assert.EqualValues(t, 1, stats.IdleResources())

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, counter.Value(), res.Value())
	assert.True(t, res.LastUsedNanotime() > 0, "should set LastUsedNanotime so that idle calculations can still work")
	assert.Equal(t, 1, res.Value())
	assert.WithinDuration(t, time.Now(), res.CreationTime(), time.Second)
	res.Release()

	assert.EqualValues(t, 0, pool.Stat().EmptyAcquireCount(), "should have been a warm resource")
}

func TestPoolCreateResourceReturnsErrorFromFailedResourceCreate(t *testing.T) {
	errCreateFailed := errors.New("create failed")
	constructor := func(ctx context.Context) (int, error) {
		return 0, errCreateFailed
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	err = pool.CreateResource(context.Background())
	assert.Equal(t, errCreateFailed, err)
}

func TestPoolCreateResourceReturnsErrorWhenAlreadyClosed(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	pool.Close()
	err = pool.CreateResource(context.Background())
	assert.Equal(t, puddle.ErrClosedPool, err)
}

func TestPoolCreateResourceReturnsErrorWhenClosedWhileCreatingResource(t *testing.T) {
	// There is no way to guarantee the correct order of the pool being closed while the resource is being constructed.
	// But these sleeps should make it extremely likely. (Ah, the lengths we go for 100% test coverage...)
	constructor := func(ctx context.Context) (int, error) {
		time.Sleep(500 * time.Millisecond)
		return 123, nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	acquireErrChan := make(chan error)
	go func() {
		err := pool.CreateResource(context.Background())
		acquireErrChan <- err
	}()

	time.Sleep(250 * time.Millisecond)
	pool.Close()

	err = <-acquireErrChan
	assert.Equal(t, puddle.ErrClosedPool, err)
}

func TestPoolCreateResourceReturnsErrorWhenPoolFull(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 2})
	require.NoError(t, err)
	defer pool.Close()

	err = pool.CreateResource(context.Background())
	require.NoError(t, err)

	stats := pool.Stat()
	assert.EqualValues(t, 1, stats.IdleResources())

	err = pool.CreateResource(context.Background())
	require.NoError(t, err)

	stats = pool.Stat()
	assert.EqualValues(t, 2, stats.IdleResources())

	err = pool.CreateResource(context.Background())
	require.Error(t, err)
}

func TestPoolCloseClosesAllIdleResources(t *testing.T) {
	constructor, _ := createConstructor()

	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	p, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	resources := make([]*puddle.Resource[int], 4)
	for i := range resources {
		var err error
		resources[i], err = p.Acquire(context.Background())
		require.Nil(t, err)
	}

	for _, res := range resources {
		res.Release()
	}

	p.Close()

	assert.Equal(t, len(resources), destructorCalls.Value())
}

func TestPoolCloseBlocksUntilAllResourcesReleasedAndClosed(t *testing.T) {
	constructor, _ := createConstructor()
	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	p, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	resources := make([]*puddle.Resource[int], 4)
	for i := range resources {
		var err error
		resources[i], err = p.Acquire(context.Background())
		require.Nil(t, err)
	}

	for _, res := range resources {
		go func(res *puddle.Resource[int]) {
			time.Sleep(100 * time.Millisecond)
			res.Release()
		}(res)
	}

	p.Close()
	assert.Equal(t, len(resources), destructorCalls.Value())
}

func TestPoolCloseIsSafeToCallMultipleTimes(t *testing.T) {
	constructor, _ := createConstructor()

	p, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	p.Close()
	p.Close()
}

func TestPoolResetDestroysAllIdleResources(t *testing.T) {
	constructor, _ := createConstructor()

	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	p, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	resources := make([]*puddle.Resource[int], 4)
	for i := range resources {
		var err error
		resources[i], err = p.Acquire(context.Background())
		require.Nil(t, err)
	}

	for _, res := range resources {
		res.Release()
	}

	require.EqualValues(t, 4, p.Stat().TotalResources())
	p.Reset()
	require.EqualValues(t, 0, p.Stat().TotalResources())

	// Destructors are called in the background. No way to know when they are all finished.
	for i := 0; i < 100; i++ {
		if destructorCalls.Value() == len(resources) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Equal(t, len(resources), destructorCalls.Value())

	p.Close()
}

func TestPoolResetDestroysCheckedOutResourcesOnReturn(t *testing.T) {
	constructor, _ := createConstructor()

	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	p, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	resources := make([]*puddle.Resource[int], 4)
	for i := range resources {
		var err error
		resources[i], err = p.Acquire(context.Background())
		require.Nil(t, err)
	}

	require.EqualValues(t, 4, p.Stat().TotalResources())
	p.Reset()
	require.EqualValues(t, 4, p.Stat().TotalResources())

	for _, res := range resources {
		res.Release()
	}

	require.EqualValues(t, 0, p.Stat().TotalResources())

	// Destructors are called in the background. No way to know when they are all finished.
	for i := 0; i < 100; i++ {
		if destructorCalls.Value() == len(resources) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Equal(t, len(resources), destructorCalls.Value())

	p.Close()
}

func TestPoolStatResources(t *testing.T) {
	startWaitChan := make(chan struct{})
	waitingChan := make(chan struct{})
	endWaitChan := make(chan struct{})

	var constructorCalls Counter
	constructor := func(ctx context.Context) (int, error) {
		select {
		case <-startWaitChan:
			close(waitingChan)
			<-endWaitChan
		default:
		}

		return constructorCalls.Next(), nil
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	defer pool.Close()

	resAcquired, err := pool.Acquire(context.Background())
	require.Nil(t, err)

	close(startWaitChan)
	go func() {
		res, err := pool.Acquire(context.Background())
		require.Nil(t, err)
		res.Release()
	}()
	<-waitingChan
	stat := pool.Stat()

	assert.EqualValues(t, 2, stat.TotalResources())
	assert.EqualValues(t, 1, stat.ConstructingResources())
	assert.EqualValues(t, 1, stat.AcquiredResources())
	assert.EqualValues(t, 0, stat.IdleResources())
	assert.EqualValues(t, 10, stat.MaxResources())

	resAcquired.Release()

	stat = pool.Stat()
	assert.EqualValues(t, 2, stat.TotalResources())
	assert.EqualValues(t, 1, stat.ConstructingResources())
	assert.EqualValues(t, 0, stat.AcquiredResources())
	assert.EqualValues(t, 1, stat.IdleResources())
	assert.EqualValues(t, 10, stat.MaxResources())

	close(endWaitChan)
}

func TestPoolStatSuccessfulAcquireCounters(t *testing.T) {
	constructor, _ := createConstructor()
	sleepConstructor := func(ctx context.Context) (int, error) {
		// sleep to make sure we don't fail the AcquireDuration test
		time.Sleep(time.Nanosecond)
		return constructor(ctx)
	}
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: sleepConstructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)
	defer pool.Close()

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	res.Release()

	stat := pool.Stat()
	assert.Equal(t, int64(1), stat.AcquireCount())
	assert.Equal(t, int64(1), stat.EmptyAcquireCount())
	assert.Positive(t, stat.AcquireDuration(), "expected stat.AcquireDuration() > 0 but %v", stat.AcquireDuration())
	assert.Equal(t, stat.EmptyAcquireWaitTime(), stat.AcquireDuration())
	lastAcquireDuration := stat.AcquireDuration()

	res, err = pool.Acquire(context.Background())
	require.NoError(t, err)
	res.Release()

	stat = pool.Stat()
	assert.Equal(t, int64(2), stat.AcquireCount())
	assert.Equal(t, int64(1), stat.EmptyAcquireCount())
	assert.Greater(t, stat.AcquireDuration(), lastAcquireDuration)
	assert.Less(t, stat.EmptyAcquireWaitTime(), stat.AcquireDuration())
	lastAcquireDuration = stat.AcquireDuration()

	wg := &sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			res, err = pool.Acquire(context.Background())
			require.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			res.Release()
			wg.Done()
		}()
	}

	wg.Wait()

	stat = pool.Stat()
	assert.Equal(t, int64(4), stat.AcquireCount())
	assert.Equal(t, int64(2), stat.EmptyAcquireCount())
	assert.Greater(t, stat.AcquireDuration(), lastAcquireDuration)
	assert.Less(t, stat.EmptyAcquireWaitTime(), stat.AcquireDuration())
	lastAcquireDuration = stat.AcquireDuration()
}

func TestPoolStatCanceledAcquireBeforeStart(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = pool.Acquire(ctx)
	require.Equal(t, context.Canceled, err)

	stat := pool.Stat()
	assert.Equal(t, int64(0), stat.AcquireCount())
	assert.Equal(t, int64(1), stat.CanceledAcquireCount())
}

func TestPoolStatCanceledAcquireDuringCreate(t *testing.T) {
	constructor := func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	}

	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	_, err = pool.Acquire(ctx)
	require.Equal(t, context.Canceled, err)

	// sleep to give the constructor goroutine time to mark cancelled
	time.Sleep(10 * time.Millisecond)

	stat := pool.Stat()
	assert.Equal(t, int64(0), stat.AcquireCount())
	assert.Equal(t, int64(1), stat.CanceledAcquireCount())
}

func TestPoolStatCanceledAcquireDuringWait(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)
	defer pool.Close()

	res, err := pool.Acquire(context.Background())
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	_, err = pool.Acquire(ctx)
	require.Equal(t, context.Canceled, err)

	res.Release()

	stat := pool.Stat()
	assert.Equal(t, int64(1), stat.AcquireCount())
	assert.Equal(t, int64(1), stat.CanceledAcquireCount())
}

func TestResourceHijackRemovesResourceFromPoolButDoesNotDestroy(t *testing.T) {
	constructor, _ := createConstructor()
	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())

	res.Hijack()

	assert.EqualValues(t, 0, pool.Stat().TotalResources())
	assert.EqualValues(t, 0, destructorCalls.Value())

	// Can still call Value, CreationTime and IdleDuration
	res.Value()
	res.CreationTime()
	res.IdleDuration()
}

func TestResourceDestroyRemovesResourceFromPool(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Value())

	assert.EqualValues(t, 1, pool.Stat().TotalResources())
	res.Destroy()
	for i := 0; i < 1000; i++ {
		if pool.Stat().TotalResources() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	assert.EqualValues(t, 0, pool.Stat().TotalResources())
}

func TestResourceLastUsageTimeTracking(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 1})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	t1 := res.LastUsedNanotime()
	res.Release()

	// Greater than zero after initial usage
	res, err = pool.Acquire(context.Background())
	require.NoError(t, err)
	t2 := res.LastUsedNanotime()
	d2 := res.IdleDuration()
	assert.True(t, t2 > t1)
	res.ReleaseUnused()

	// ReleaseUnused does not update usage tracking
	res, err = pool.Acquire(context.Background())
	require.NoError(t, err)
	t3 := res.LastUsedNanotime()
	d3 := res.IdleDuration()
	assert.EqualValues(t, t2, t3)
	assert.True(t, d3 > d2)
	res.Release()

	// Release does update usage tracking
	res, err = pool.Acquire(context.Background())
	require.NoError(t, err)
	t4 := res.LastUsedNanotime()
	assert.True(t, t4 > t3)
	res.Release()
}

func TestResourcePanicsOnUsageWhenNotAcquired(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)

	res, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	res.Release()

	assert.PanicsWithValue(t, "tried to release resource that is not acquired", res.Release)
	assert.PanicsWithValue(t, "tried to release resource that is not acquired", res.ReleaseUnused)
	assert.PanicsWithValue(t, "tried to destroy resource that is not acquired", res.Destroy)
	assert.PanicsWithValue(t, "tried to hijack resource that is not acquired", res.Hijack)
	assert.PanicsWithValue(t, "tried to access resource that is not acquired or hijacked", func() { res.Value() })
	assert.PanicsWithValue(t, "tried to access resource that is not acquired or hijacked", func() { res.CreationTime() })
	assert.PanicsWithValue(t, "tried to access resource that is not acquired or hijacked", func() { res.LastUsedNanotime() })
	assert.PanicsWithValue(t, "tried to access resource that is not acquired or hijacked", func() { res.IdleDuration() })
}

func TestPoolAcquireReturnsErrorWhenPoolIsClosed(t *testing.T) {
	constructor, _ := createConstructor()
	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: 10})
	require.NoError(t, err)
	pool.Close()

	res, err := pool.Acquire(context.Background())
	assert.Equal(t, puddle.ErrClosedPool, err)
	assert.Nil(t, res)
}

func TestSignalIsSentWhenResourceFailedToCreate(t *testing.T) {
	var c Counter
	constructor := func(context.Context) (a any, err error) {
		if c.Next() == 2 {
			return nil, errors.New("outage")
		}
		return 1, nil
	}
	destructor := func(value any) {}

	pool, err := puddle.NewPool(&puddle.Config[any]{Constructor: constructor, Destructor: destructor, MaxSize: 10})
	require.NoError(t, err)

	res1, err := pool.Acquire(context.Background())
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, _ = pool.Acquire(context.Background())
		}(strconv.Itoa(i))
	}

	// ensure that both goroutines above are waiting for condition variable signal
	time.Sleep(500 * time.Millisecond)
	res1.Destroy()
	wg.Wait()
}

func stressTestDur(t testing.TB) time.Duration {
	s := os.Getenv("STRESS_TEST_DURATION")
	if s == "" {
		s = "1s"
	}

	dur, err := time.ParseDuration(s)
	require.Nil(t, err)
	return dur
}

func TestStress(t *testing.T) {
	constructor, _ := createConstructor()
	var destructorCalls Counter
	destructor := func(int) {
		destructorCalls.Next()
	}

	poolSize := runtime.NumCPU()
	if poolSize < 4 {
		poolSize = 4
	}

	pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: destructor, MaxSize: int32(poolSize)})
	require.NoError(t, err)

	finishChan := make(chan struct{})
	wg := &sync.WaitGroup{}

	releaseOrDestroyOrHijack := func(res *puddle.Resource[int]) {
		n := rand.Intn(100)
		if n < 5 {
			res.Hijack()
			destructor(res.Value())
		} else if n < 10 {
			res.Destroy()
		} else {
			res.Release()
		}
	}

	actions := []func(){
		// Acquire
		func() {
			res, err := pool.Acquire(context.Background())
			if err != nil {
				if err != puddle.ErrClosedPool {
					assert.Failf(t, "stress acquire", "pool.Acquire returned unexpected err: %v", err)
				}
				return
			}

			time.Sleep(time.Duration(rand.Int63n(100)) * time.Millisecond)
			releaseOrDestroyOrHijack(res)
		},
		// Acquire possibly canceled by context
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rand.Int63n(2000))*time.Nanosecond)
			defer cancel()
			res, err := pool.Acquire(ctx)
			if err != nil {
				if err != puddle.ErrClosedPool && err != context.Canceled && err != context.DeadlineExceeded {
					assert.Failf(t, "stress acquire possibly canceled by context", "pool.Acquire returned unexpected err: %v", err)
				}
				return
			}

			time.Sleep(time.Duration(rand.Int63n(2000)) * time.Nanosecond)
			releaseOrDestroyOrHijack(res)
		},
		// TryAcquire
		func() {
			res, err := pool.TryAcquire(context.Background())
			if err != nil {
				if err != puddle.ErrClosedPool && err != puddle.ErrNotAvailable {
					assert.Failf(t, "stress TryAcquire", "pool.TryAcquire returned unexpected err: %v", err)
				}
				return
			}

			time.Sleep(time.Duration(rand.Int63n(100)) * time.Millisecond)
			releaseOrDestroyOrHijack(res)
		},
		// AcquireAllIdle (though under heavy load this will almost certainly always get an empty slice)
		func() {
			resources := pool.AcquireAllIdle()
			for _, res := range resources {
				res.Release()
			}
		},
		// Stat
		func() {
			stat := pool.Stat()
			assert.NotNil(t, stat)
		},
		// CreateResource
		func() {
			err := pool.CreateResource(context.Background())
			if err != nil && !errors.Is(err, puddle.ErrClosedPool) && !errors.Is(err, puddle.ErrNotAvailable) {
				t.Error(err)
			}
		},
	}

	workerCount := int(poolSize) * 2

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-finishChan:
					return
				default:
				}

				actions[rand.Intn(len(actions))]()
			}
		}()
	}

	time.AfterFunc(stressTestDur(t), func() { close(finishChan) })
	wg.Wait()
	pool.Close()
}

func TestStress_AcquireAllIdle_TryAcquire(t *testing.T) {
	r := require.New(t)

	pool := testPool[int32](t)

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
			}

			idleRes := pool.AcquireAllIdle()
			r.Less(len(idleRes), 2)
			for _, res := range idleRes {
				res.Release()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
			}

			res, err := pool.TryAcquire(context.Background())
			if err != nil {
				r.Equal(puddle.ErrNotAvailable, err)
			} else {
				r.NotNil(res)
				res.Release()
			}
		}
	}()

	time.AfterFunc(stressTestDur(t), func() { close(done) })
	wg.Wait()
}

func TestStress_AcquireAllIdle_Acquire(t *testing.T) {
	r := require.New(t)

	pool := testPool[int32](t)

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
			}

			idleRes := pool.AcquireAllIdle()
			r.Less(len(idleRes), 2)
			for _, res := range idleRes {
				r.NotNil(res)
				res.Release()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
			}

			res, err := pool.Acquire(context.Background())
			if err != nil {
				r.Equal(puddle.ErrNotAvailable, err)
			} else {
				r.NotNil(res)
				res.Release()
			}
		}
	}()

	time.AfterFunc(stressTestDur(t), func() { close(done) })
	wg.Wait()
}

func startAcceptOnceDummyServer(laddr string) {
	ln, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatalln("Listen:", err)
	}

	// Listen one time
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln("Accept:", err)
		}

		for {
			buf := make([]byte, 1)
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

}

func ExamplePool() {
	// Dummy server
	laddr := "127.0.0.1:8080"
	startAcceptOnceDummyServer(laddr)

	// Pool creation
	constructor := func(context.Context) (any, error) {
		return net.Dial("tcp", laddr)
	}
	destructor := func(value any) {
		value.(net.Conn).Close()
	}
	maxPoolSize := int32(10)

	pool, err := puddle.NewPool(&puddle.Config[any]{Constructor: constructor, Destructor: destructor, MaxSize: int32(maxPoolSize)})
	if err != nil {
		log.Fatalln("NewPool", err)
	}

	// Use pool multiple times
	for i := 0; i < 10; i++ {
		// Acquire resource
		res, err := pool.Acquire(context.Background())
		if err != nil {
			log.Fatalln("Acquire", err)
		}

		// Type-assert value and use
		_, err = res.Value().(net.Conn).Write([]byte{1})
		if err != nil {
			log.Fatalln("Write", err)
		}

		// Release when done.
		res.Release()
	}

	stats := pool.Stat()
	pool.Close()

	fmt.Println("Connections:", stats.TotalResources())
	fmt.Println("Acquires:", stats.AcquireCount())
	// Output:
	// Connections: 1
	// Acquires: 10
}

func BenchmarkPoolAcquireAndRelease(b *testing.B) {
	benchmarks := []struct {
		poolSize    int32
		clientCount int
		cancellable bool
	}{
		{8, 1, false},
		{8, 2, false},
		{8, 8, false},
		{8, 32, false},
		{8, 128, false},
		{8, 512, false},
		{8, 2048, false},
		{8, 8192, false},

		{64, 2, false},
		{64, 8, false},
		{64, 32, false},
		{64, 128, false},
		{64, 512, false},
		{64, 2048, false},
		{64, 8192, false},

		{512, 2, false},
		{512, 8, false},
		{512, 32, false},
		{512, 128, false},
		{512, 512, false},
		{512, 2048, false},
		{512, 8192, false},

		{8, 2, true},
		{8, 8, true},
		{8, 32, true},
		{8, 128, true},
		{8, 512, true},
		{8, 2048, true},
		{8, 8192, true},

		{64, 2, true},
		{64, 8, true},
		{64, 32, true},
		{64, 128, true},
		{64, 512, true},
		{64, 2048, true},
		{64, 8192, true},

		{512, 2, true},
		{512, 8, true},
		{512, 32, true},
		{512, 128, true},
		{512, 512, true},
		{512, 2048, true},
		{512, 8192, true},
	}

	for _, bm := range benchmarks {
		name := fmt.Sprintf("PoolSize=%d/ClientCount=%d/Cancellable=%v", bm.poolSize, bm.clientCount, bm.cancellable)

		b.Run(name, func(b *testing.B) {
			ctx := context.Background()
			cancel := func() {}
			if bm.cancellable {
				ctx, cancel = context.WithCancel(ctx)
			}

			wg := &sync.WaitGroup{}

			constructor, _ := createConstructor()
			pool, err := puddle.NewPool(&puddle.Config[int]{Constructor: constructor, Destructor: stubDestructor, MaxSize: bm.poolSize})
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < bm.clientCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for j := 0; j < b.N; j++ {
						res, err := pool.Acquire(ctx)
						if err != nil {
							b.Fatal(err)
						}
						res.Release()
					}
				}()
			}

			wg.Wait()
			cancel()
		})
	}
}

func TestAcquireAllSem(t *testing.T) {
	r := require.New(t)

	sem := semaphore.NewWeighted(5)
	r.Equal(4, puddle.AcquireSemAll(sem, 4))
	sem.Release(4)

	r.Equal(5, puddle.AcquireSemAll(sem, 5))
	sem.Release(5)

	r.Equal(5, puddle.AcquireSemAll(sem, 6))
	sem.Release(5)
}

func testPool[T any](t testing.TB) *puddle.Pool[T] {
	cfg := puddle.Config[T]{
		MaxSize: 1,
		Constructor: func(ctx context.Context) (T, error) {
			var zero T
			return zero, nil
		},
		Destructor: func(T) {},
	}

	pool, err := puddle.NewPool(&cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func releaser[T any](t testing.TB) chan<- *puddle.Resource[T] {
	startChan := make(chan struct{})
	workChan := make(chan *puddle.Resource[T], 1)

	go func() {
		close(startChan)

		for r := range workChan {
			r.Release()
		}
	}()
	t.Cleanup(func() { close(workChan) })

	// Wait for goroutine start.
	<-startChan
	return workChan
}

func TestReleaseAfterAcquire(t *testing.T) {
	const cnt = 100000

	r := require.New(t)
	ctx := context.Background()
	pool := testPool[int32](t)
	releaseChan := releaser[int32](t)

	res, err := pool.Acquire(ctx)
	r.NoError(err)
	// We need to release the last connection. Otherwise the pool.Close()
	// method will block and this function will never return.
	defer func() { res.Release() }()

	for i := 0; i < cnt; i++ {
		releaseChan <- res
		res, err = pool.Acquire(ctx)
		r.NoError(err)
	}
}

func BenchmarkAcquire_ReleaseAfterAcquire(b *testing.B) {
	r := require.New(b)
	ctx := context.Background()
	pool := testPool[int32](b)
	releaseChan := releaser[int32](b)

	res, err := pool.Acquire(ctx)
	r.NoError(err)
	// We need to release the last connection. Otherwise the pool.Close()
	// method will block and this function will never return.
	defer func() { res.Release() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		releaseChan <- res
		res, err = pool.Acquire(ctx)
		r.NoError(err)
	}
}

func withCPULoad() {
	// Multiply by 2 to similate overload of the system.
	numGoroutines := runtime.NumCPU() * 2

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			wg.Done()

			// Similate computationally intensive task.
			for j := 0; true; j++ {
			}
		}()
	}

	wg.Wait()
}

func BenchmarkAcquire_ReleaseAfterAcquireWithCPULoad(b *testing.B) {
	r := require.New(b)
	ctx := context.Background()
	pool := testPool[int32](b)
	releaseChan := releaser[int32](b)

	withCPULoad()

	res, err := pool.Acquire(ctx)
	r.NoError(err)
	// We need to release the last connection. Otherwise the pool.Close()
	// method will block and this function will never return.
	defer func() { res.Release() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		releaseChan <- res
		res, err = pool.Acquire(ctx)
		r.NoError(err)
	}
}

func BenchmarkAcquire_MultipleCancelled(b *testing.B) {
	const cancelCnt = 64

	r := require.New(b)
	ctx := context.Background()
	pool := testPool[int32](b)
	releaseChan := releaser[int32](b)

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	res, err := pool.Acquire(ctx)
	r.NoError(err)
	// We need to release the last connection. Otherwise the pool.Close()
	// method will block and this function will never return.
	defer func() { res.Release() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < cancelCnt; j++ {
			_, err = pool.AcquireRaw(cancelCtx)
			r.Equal(context.Canceled, err)
		}

		releaseChan <- res
		res, err = pool.Acquire(ctx)
		r.NoError(err)
	}
}

func BenchmarkAcquire_MultipleCancelledWithCPULoad(b *testing.B) {
	const cancelCnt = 3

	r := require.New(b)
	ctx := context.Background()
	pool := testPool[int32](b)
	releaseChan := releaser[int32](b)

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	withCPULoad()

	res, err := pool.Acquire(ctx)
	r.NoError(err)
	// We need to release the last connection. Otherwise the pool.Close()
	// method will block and this function will never return.
	defer func() { res.Release() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < cancelCnt; j++ {
			_, err = pool.AcquireRaw(cancelCtx)
			r.Equal(context.Canceled, err)
		}

		releaseChan <- res
		res, err = pool.Acquire(ctx)
		r.NoError(err)
	}
}
