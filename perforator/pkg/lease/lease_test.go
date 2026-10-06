package lease

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type stubLeaseStorage struct {
	acquire func(context.Context) (bool, error)
	renew   func(context.Context) (bool, error)
	release func(context.Context) error
}

func (s *stubLeaseStorage) Acquire(ctx context.Context, _ string, _ time.Duration) (bool, error) {
	if s.acquire != nil {
		return s.acquire(ctx)
	}
	return true, nil
}

func (s *stubLeaseStorage) Renew(ctx context.Context, _ string, _ time.Duration) (bool, error) {
	if s.renew != nil {
		return s.renew(ctx)
	}
	return true, nil
}

func (s *stubLeaseStorage) Release(ctx context.Context, _ string) error {
	if s.release != nil {
		return s.release(ctx)
	}
	return nil
}

func TestLeaseExpiresWhileRenewIsBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		finishRenew, finishAction := make(chan struct{}), make(chan struct{})
		defer func() {
			select {
			case <-finishRenew:
			default:
				close(finishRenew)
			}
			select {
			case <-finishAction:
			default:
				close(finishAction)
			}
		}()
		var actionCtx, renewCtx context.Context
		var actionReturned, renewReturned, released, returned bool
		var runErr error
		calls := 0
		s := &stubLeaseStorage{
			renew: func(ctx context.Context) (bool, error) {
				calls++
				renewCtx = ctx
				// Return success after cancellation.
				<-finishRenew
				renewReturned = true
				return true, nil
			},
			release: func(ctx context.Context) error {
				assert.NoError(t, ctx.Err(), "release must remain usable after lease expiry")
				assert.True(t, actionReturned, "release must wait for the action")
				assert.True(t, renewReturned, "release must wait for renewal")
				released = true
				return nil
			},
		}
		startedAt := time.Now()
		go func() {
			runErr = LockAndRun(ctx, xlog.NewNop(), s, func(ctx context.Context, _ string) {
				actionCtx = ctx
				<-ctx.Done()
				<-finishAction
				actionReturned = true
			}, WithTTL(30*time.Second))
			returned = true
		}()
		synctest.Wait()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		require.NotNil(t, renewCtx)
		deadline, ok := renewCtx.Deadline()
		require.True(t, ok)
		require.Equal(t, startedAt.Add(30*time.Second), deadline)

		time.Sleep(20 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(actionCtx), ErrLeaseLost)
		// Request timeout races with lease cancellation.
		require.Error(t, renewCtx.Err(), "renewal must be canceled when the lease expires")
		require.Equal(t, 1, calls, "renewals must not overlap")
		require.False(t, released)
		require.False(t, returned)

		close(finishAction)
		synctest.Wait()
		require.False(t, released, "renewal has not returned yet")
		close(finishRenew)
		synctest.Wait()
		require.True(t, released)
		require.True(t, returned)
		require.NoError(t, runErr) // Lease loss is reported through the action context.
		require.ErrorIs(t, context.Cause(actionCtx), ErrLeaseLost, "a late success cannot revive the lease")
	})
}

func TestLeaseRenewalIncludesRequestLatency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		s := &stubLeaseStorage{renew: func(ctx context.Context) (bool, error) {
			if calls.Add(1) == 1 {
				time.Sleep(8 * time.Second)
				return true, nil
			}
			<-ctx.Done()
			return false, ctx.Err()
		}}
		h := startTestLease(t, s, WithTTL(30*time.Second), WithRenewInterval(10*time.Second))
		defer h.stop()
		// Renewal runs from 10s to 18s; expiry is 40s, not 48s. Next request blocks.
		time.Sleep(31 * time.Second)
		synctest.Wait()
		require.NoError(t, h.leaseCtx.Err(), "successful renewal must extend the lease")
		require.EqualValues(t, 2, calls.Load(), "the second renewal must still be blocked before expiry")
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.leaseCtx), ErrLeaseLost)
	})
}

func TestLeaseRetriesRenewalErrorsBeforeExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		s := &stubLeaseStorage{renew: func(context.Context) (bool, error) {
			if calls.Add(1) == 1 {
				return false, errors.New("temporary failure")
			}
			return true, nil
		}}
		h := startTestLease(t, s, WithTTL(30*time.Second), WithRenewInterval(10*time.Second))
		defer h.stop()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		require.EqualValues(t, 1, calls.Load())
		require.NoError(t, h.leaseCtx.Err())
		time.Sleep(80 * time.Second)
		synctest.Wait()
		require.Greater(t, calls.Load(), int32(2))
		require.NoError(t, h.leaseCtx.Err())
	})
}

func TestLeaseExpiresAfterRenewalErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &stubLeaseStorage{renew: func(context.Context) (bool, error) {
			return false, errors.New("unavailable")
		}}
		h := startTestLease(t, s, WithTTL(30*time.Second), WithRenewInterval(10*time.Second))
		defer h.stop()
		time.Sleep(31 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.leaseCtx), ErrLeaseLost)
	})
}

func TestLeaseStopsWhenRenewalReportsLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &stubLeaseStorage{renew: func(context.Context) (bool, error) { return false, nil }}
		h := startTestLease(t, s, WithTTL(30*time.Second), WithRenewInterval(10*time.Second))
		defer h.stop()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.leaseCtx), ErrLeaseLost)
	})
}

func TestLeaseReturnsParentCancellationCause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)
		cause := errors.New("shutdown")
		called, released := false, false
		s := &stubLeaseStorage{
			acquire: func(context.Context) (bool, error) {
				cancel(cause)
				return true, nil
			},
			release: func(ctx context.Context) error {
				assert.NoError(t, ctx.Err())
				released = true
				return nil
			},
		}
		err := LockAndRun(ctx, xlog.NewNop(), s, func(context.Context, string) {
			called = true
		})
		require.ErrorIs(t, err, cause)
		require.False(t, called)
		require.True(t, released)
	})
}

func TestLeaseCancellationWaitsForRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var renewReturned, released bool
		s := &stubLeaseStorage{
			renew: func(ctx context.Context) (bool, error) {
				<-ctx.Done()
				renewReturned = true
				return false, ctx.Err()
			},
			release: func(ctx context.Context) error {
				assert.True(t, renewReturned)
				assert.NoError(t, ctx.Err(), "release must use a context detached from parent cancellation")
				released = true
				return nil
			},
		}

		var actionCtx context.Context
		done := make(chan error, 1)
		go func() {
			done <- LockAndRun(ctx, xlog.NewNop(), s, func(ctx context.Context, _ string) {
				actionCtx = ctx
				<-ctx.Done()
			}, WithTTL(30*time.Second))
		}()
		synctest.Wait()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		cancel()
		require.NoError(t, <-done)
		require.True(t, released)
		require.ErrorIs(t, context.Cause(actionCtx), context.Canceled)
	})
}

func TestLeaseRejectsLateAcquisition(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		name := "active_parent"
		if cancelParent {
			name = "canceled_parent"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				held, called, releases := false, false, 0
				s := &stubLeaseStorage{
					acquire: func(context.Context) (bool, error) {
						time.Sleep(31 * time.Second)
						held = true
						if cancelParent {
							cancel()
						}
						return true, nil
					},
					release: func(ctx context.Context) error {
						releases++
						if err := ctx.Err(); err != nil {
							return err
						}
						held = false
						return nil
					},
				}
				err := LockAndRun(ctx, xlog.NewNop(), s, func(context.Context, string) {
					called = true
				}, WithTTL(30*time.Second))
				require.ErrorIs(t, err, ErrLeaseLost)
				require.False(t, called)
				require.Equal(t, 1, releases)
				require.False(t, held, "late acquisition must release the storage lease")
			})
		})
	}
}

func TestLeaseReleaseTimeout(t *testing.T) {
	for _, lateAcquire := range []bool{false, true} {
		name := "after_action"
		if lateAcquire {
			name = "late_acquisition"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var releaseErr error
				var elapsed time.Duration
				s := &stubLeaseStorage{
					acquire: func(context.Context) (bool, error) {
						if lateAcquire {
							time.Sleep(31 * time.Second)
						}
						return true, nil
					},
					release: func(ctx context.Context) error {
						assert.NoError(t, ctx.Err())
						startedAt := time.Now()
						<-ctx.Done()
						elapsed = time.Since(startedAt)
						releaseErr = ctx.Err()
						return releaseErr
					},
				}
				err := LockAndRun(t.Context(), xlog.NewNop(), s, func(context.Context, string) {}, WithTTL(30*time.Second))
				if lateAcquire {
					require.ErrorIs(t, err, ErrLeaseLost)
				} else {
					require.NoError(t, err) // Release is best-effort.
				}
				require.ErrorIs(t, releaseErr, context.DeadlineExceeded)
				require.Equal(t, leaseReleaseTimeout, elapsed)
			})
		})
	}
}

func TestLeaseRejectsInvalidIntervals(t *testing.T) {
	for _, ttl := range []time.Duration{-time.Second, 0, time.Nanosecond, 2 * time.Nanosecond} {
		t.Run(ttl.String(), func(t *testing.T) {
			called := false
			s := &stubLeaseStorage{acquire: func(context.Context) (bool, error) {
				called = true
				return true, nil
			}}
			err := LockAndRun(t.Context(), xlog.NewNop(), s, nil, WithTTL(ttl))
			require.ErrorContains(t, err, "must be positive")
			require.False(t, called)
		})
	}
	err := LockAndRun(t.Context(), xlog.NewNop(), &stubLeaseStorage{},
		nil, WithRenewInterval(-time.Second))
	require.ErrorContains(t, err, "must be positive")
}

func TestLeaseRenewalAtExpiryBoundary(t *testing.T) {
	for _, latency := range []time.Duration{19 * time.Second, 21 * time.Second} {
		t.Run(latency.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				s := &stubLeaseStorage{renew: func(ctx context.Context) (bool, error) {
					calls++
					if calls == 1 {
						time.Sleep(latency)
						return true, nil
					}
					<-ctx.Done()
					return false, ctx.Err()
				}}
				startedAt := time.Now()
				h := startTestLease(t, s, WithTTL(30*time.Second), WithRenewInterval(10*time.Second))
				defer h.stop()
				// Renewal starts at 10s; lease expires at 30s.
				time.Sleep(32 * time.Second)
				synctest.Wait()
				h.mu.Lock()
				expiresAt := h.leaseExpiresAt
				h.mu.Unlock()
				if latency < 20*time.Second {
					require.NoError(t, h.leaseCtx.Err())
					require.Equal(t, startedAt.Add(40*time.Second), expiresAt)
				} else {
					require.ErrorIs(t, context.Cause(h.leaseCtx), ErrLeaseLost)
					require.Equal(t, startedAt.Add(30*time.Second), expiresAt)
				}
			})
		})
	}
}

func TestLeaseAcceptsLateRenewalBeforeCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		s := &stubLeaseStorage{renew: func(ctx context.Context) (bool, error) {
			calls++
			if calls == 1 {
				time.Sleep(21 * time.Second)
				return true, nil
			}
			<-ctx.Done()
			return false, ctx.Err()
		}}
		startedAt := time.Now()
		h := &leaseHolder{
			logger: xlog.NewNop(), target: s, token: "holder",
			options:        leaseOptions{ttl: 30 * time.Second, renewInterval: 10 * time.Second},
			leaseExpiresAt: startedAt.Add(30 * time.Second),
		}
		h.leaseCtx, h.cancel = context.WithCancelCause(t.Context())
		// Delay the expiry watcher while renewal runs.
		h.workers.Add(1)
		go h.runRenewal()
		defer h.stop()
		time.Sleep(32 * time.Second)
		synctest.Wait()
		require.NoError(t, h.leaseCtx.Err())
		h.mu.Lock()
		expiresAt := h.leaseExpiresAt
		h.mu.Unlock()
		require.Equal(t, startedAt.Add(40*time.Second), expiresAt)
	})
}

func TestLeaseExpiryRechecksDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := startTestLease(t, &stubLeaseStorage{}, WithTTL(30*time.Second), WithRenewInterval(time.Hour))
		defer h.stop()
		time.Sleep(10 * time.Second)
		// Extend expiry before the old timer fires.
		h.mu.Lock()
		h.leaseExpiresAt = time.Now().Add(30 * time.Second)
		h.mu.Unlock()
		time.Sleep(21 * time.Second)
		synctest.Wait()
		require.NoError(t, h.leaseCtx.Err(), "the old timer must recheck the extended deadline")
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.leaseCtx), ErrLeaseLost)
	})
}

func TestLeaseCustomReleaseTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var elapsed time.Duration
		storage := &stubLeaseStorage{release: func(ctx context.Context) error {
			start := time.Now()
			<-ctx.Done()
			elapsed = time.Since(start)
			return ctx.Err()
		}}
		err := LockAndRun(t.Context(), xlog.NewNop(), storage,
			func(context.Context, string) {}, WithReleaseTimeout(2*time.Second))
		require.NoError(t, err)
		require.Equal(t, 2*time.Second, elapsed)
	})
}

type leaseTestCounter struct {
	metrics.Counter
	count atomic.Int64
}

func (c *leaseTestCounter) Inc()         { c.count.Add(1) }
func (c *leaseTestCounter) Value() int64 { return c.count.Load() }

type leaseTestGauge struct {
	metrics.Gauge
	bits atomic.Uint64
}

func (g *leaseTestGauge) Set(value float64) { g.bits.Store(math.Float64bits(value)) }
func (g *leaseTestGauge) Value() float64    { return math.Float64frombits(g.bits.Load()) }

func TestLeaseMetrics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		counter := &leaseTestCounter{}
		gauge := &leaseTestGauge{}
		calls := 0
		held := &stubLeaseStorage{renew: func(context.Context) (bool, error) {
			calls++
			if calls == 1 {
				return false, errors.New("storage unavailable")
			}
			return false, nil
		}}
		run := &leaseHolder{target: held, token: "holder", logger: xlog.NewNop(),
			options:        leaseOptions{ttl: time.Minute, renewInterval: time.Second, renewalErrors: counter},
			leaseExpiresAt: time.Now().Add(time.Minute), leaseHeld: gauge,
		}
		require.NoError(t, run.start(t.Context()))
		defer run.stop()
		synctest.Wait()
		require.Equal(t, 1.0, gauge.Value())

		time.Sleep(time.Second)
		synctest.Wait()
		require.EqualValues(t, 1, counter.Value())
		require.NoError(t, run.leaseCtx.Err())

		time.Sleep(time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(run.leaseCtx), ErrLeaseLost)
		require.EqualValues(t, 1, counter.Value(), "ownership loss is not a storage error")
		require.Zero(t, gauge.Value())
	})
}

func TestLockAndRunBusyDoesNotInvokeOrRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		target := &stubLeaseStorage{
			acquire: func(context.Context) (bool, error) { return false, nil },
			release: func(context.Context) error { t.Error("released someone else's lease"); return nil },
		}
		started := time.Now()
		err := LockAndRun(t.Context(), xlog.NewNop(), target,
			func(context.Context, string) { t.Error("ran without acquiring") },
			WithWaitForLease(false))
		require.ErrorIs(t, err, ErrLeaseBusy)
		require.Equal(t, started, time.Now(), "busy acquisition must not wait")
	})
}

func TestLockAndRunReleasesAfterAction(t *testing.T) {
	called, released := false, false
	target := &stubLeaseStorage{release: func(context.Context) error { released = true; return nil }}
	err := LockAndRun(t.Context(), xlog.NewNop(), target,
		func(context.Context, string) { called = true })
	require.NoError(t, err)
	require.True(t, called)
	require.True(t, released)
}

func TestLockAndRunAcquireRetryAttempts(t *testing.T) {
	for _, tc := range []struct {
		name        string
		retries     uint32
		useDefaults bool
		succeed     bool
		wantCalls   int
	}{
		{name: "no retries", wantCalls: 1},
		{name: "one retry", retries: 1, wantCalls: 2},
		{name: "success on last retry", retries: 2, succeed: true, wantCalls: 3},
		{name: "default attempts unchanged", useDefaults: true, wantCalls: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				failure := errors.New("database unavailable")
				calls, ran := 0, false
				target := &stubLeaseStorage{acquire: func(context.Context) (bool, error) {
					calls++
					if tc.succeed && calls == tc.wantCalls {
						return true, nil
					}
					return false, failure
				}}
				opts := []LeaseOption{WithTTL(3 * time.Second)}
				if !tc.useDefaults {
					opts = append(opts, WithMaxAcquireRetryAttempts(tc.retries))
				}
				started := time.Now()
				err := LockAndRun(t.Context(), xlog.NewNop(), target,
					func(context.Context, string) { ran = true }, opts...)
				if tc.succeed {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, failure)
				}
				require.Equal(t, tc.succeed, ran)
				require.Equal(t, tc.wantCalls, calls)
				require.Equal(t, time.Duration(tc.wantCalls-1)*time.Second, time.Since(started))
			})
		})
	}
}

func TestLockAndRunBoundsAcquisitionRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		target := &stubLeaseStorage{acquire: func(ctx context.Context) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		}}
		started := time.Now()
		err := LockAndRun(t.Context(), xlog.NewNop(), target,
			func(context.Context, string) { t.Error("ran after acquisition error") },
			WithOperationTimeout(time.Second), WithMaxAcquireRetryAttempts(0))
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, time.Second, time.Since(started))
	})
}

func TestLockAndRunBoundsRenewalRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var requests atomic.Int32
		target := &stubLeaseStorage{renew: func(ctx context.Context) (bool, error) {
			requests.Add(1)
			<-ctx.Done()
			return false, ctx.Err()
		}}
		err := LockAndRun(t.Context(), xlog.NewNop(), target,
			func(ctx context.Context, _ string) {
				<-ctx.Done()
				require.ErrorIs(t, context.Cause(ctx), ErrLeaseLost)
			}, WithTTL(10*time.Second), WithRenewInterval(2*time.Second), WithOperationTimeout(time.Second))
		require.NoError(t, err)
		require.GreaterOrEqual(t, requests.Load(), int32(4), "blocked requests must time out before expiry")
	})
}

func TestLockAndRunUsesFreshToken(t *testing.T) {
	target := &tokenTestLease{}
	var previous string
	for range 2 {
		err := LockAndRun(t.Context(), xlog.NewNop(), target,
			func(_ context.Context, token string) {
				require.NotEmpty(t, token)
				require.Equal(t, target.token, token)
				require.NotEqual(t, previous, token)
				previous = token
			})
		require.NoError(t, err)
		require.Equal(t, previous, target.releasedToken)
	}
}

type tokenTestLease struct{ token, releasedToken string }

func (l *tokenTestLease) Acquire(_ context.Context, token string, _ time.Duration) (bool, error) {
	l.token = token
	return true, nil
}
func (l *tokenTestLease) Renew(_ context.Context, token string, _ time.Duration) (bool, error) {
	return token == l.token, nil
}
func (l *tokenTestLease) Release(_ context.Context, token string) error {
	l.releasedToken = token
	return nil
}

// Start heartbeat with inspectable expiry.
func startTestLease(t *testing.T, target Lease, opts ...LeaseOption) *leaseHolder {
	t.Helper()
	options := defaultLeaseOptions()
	for _, opt := range opts {
		opt(&options)
	}
	run := &leaseHolder{target: target, token: "holder", options: options, logger: xlog.NewNop(),
		leaseExpiresAt: time.Now().Add(options.ttl)}
	require.NoError(t, run.start(t.Context()))
	return run
}
