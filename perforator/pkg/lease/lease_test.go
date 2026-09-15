package lease

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type stubLeaseStorage struct {
	acquire func(context.Context) (bool, error)
	renew   func(context.Context) (bool, error)
	release func(context.Context) error
}

func (s *stubLeaseStorage) Acquire(ctx context.Context, _, _ string, _ time.Duration) (bool, error) {
	if s.acquire != nil {
		return s.acquire(ctx)
	}
	return true, nil
}

func (s *stubLeaseStorage) Renew(ctx context.Context, _, _ string, _ time.Duration) (bool, error) {
	if s.renew != nil {
		return s.renew(ctx)
	}
	return true, nil
}

func (s *stubLeaseStorage) Release(ctx context.Context, _, _ string) error {
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
				// Deliberately return a successful response after cancellation.
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
			runErr = LockAndRun(ctx, xlog.NewNop(), s, "lease", "holder", func(ctx context.Context) {
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
		// The request deadline and lease cancellation race at the same instant.
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
		require.NoError(t, runErr) // LockAndRun's action has no error return.
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
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		require.NoError(t, h.hold(t.Context()))
		defer h.close()
		// First renewal starts at 10s and returns at 18s. Its deadline is
		// therefore 40s, not 48s. The next request remains blocked.
		time.Sleep(31 * time.Second)
		synctest.Wait()
		require.NoError(t, h.context().Err(), "successful renewal must extend the lease")
		require.EqualValues(t, 2, calls.Load(), "the second renewal must still be blocked before expiry")
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.context()), ErrLeaseLost)
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
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		require.NoError(t, h.hold(t.Context()))
		defer h.close()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		require.EqualValues(t, 1, calls.Load())
		require.NoError(t, h.context().Err())
		time.Sleep(80 * time.Second)
		synctest.Wait()
		require.Greater(t, calls.Load(), int32(2))
		require.NoError(t, h.context().Err())
	})
}

func TestLeaseExpiresAfterRenewalErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &stubLeaseStorage{renew: func(context.Context) (bool, error) {
			return false, errors.New("unavailable")
		}}
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		require.NoError(t, h.hold(t.Context()))
		defer h.close()
		time.Sleep(31 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.context()), ErrLeaseLost)
	})
}

func TestLeaseStopsWhenRenewalReportsLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &stubLeaseStorage{renew: func(context.Context) (bool, error) { return false, nil }}
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		require.NoError(t, h.hold(t.Context()))
		defer h.close()
		time.Sleep(11 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.context()), ErrLeaseLost)
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
		err := LockAndRun(ctx, xlog.NewNop(), s, "lease", "holder", func(context.Context) {
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
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		require.NoError(t, h.hold(ctx))
		time.Sleep(11 * time.Second)
		synctest.Wait()
		cancel()
		require.NoError(t, h.close())
		require.True(t, released)
		require.ErrorIs(t, context.Cause(h.context()), context.Canceled)
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
				err := LockAndRun(ctx, xlog.NewNop(), s, "lease", "holder", func(context.Context) {
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
				err := LockAndRun(t.Context(), xlog.NewNop(), s, "lease", "holder", func(context.Context) {}, WithTTL(30*time.Second))
				if lateAcquire {
					require.ErrorIs(t, err, ErrLeaseLost)
				} else {
					require.NoError(t, err) // Release remains best-effort.
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
			err := LockAndRun(t.Context(), xlog.NewNop(), s, "lease", "holder", nil, WithTTL(ttl))
			require.ErrorContains(t, err, "must be positive")
			require.False(t, called)
		})
	}
	h := newLeaseHolder(xlog.NewNop(), &stubLeaseStorage{}, "lease", "holder", WithRenewInterval(-time.Second))
	require.ErrorContains(t, h.hold(t.Context()), "must be positive")
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
				h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
				startedAt := time.Now()
				require.NoError(t, h.hold(t.Context()))
				defer h.close()
				// Renewal starts at 10s; the original lease expires at 30s.
				time.Sleep(32 * time.Second)
				synctest.Wait()
				h.mu.Lock()
				expiresAt := h.leaseExpiresAt
				h.mu.Unlock()
				if latency < 20*time.Second {
					require.NoError(t, h.context().Err())
					require.Equal(t, startedAt.Add(40*time.Second), expiresAt)
				} else {
					require.ErrorIs(t, context.Cause(h.context()), ErrLeaseLost)
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
		h := newLeaseHolder(xlog.NewNop(), s, "lease", "holder", WithTTL(30*time.Second))
		startedAt := time.Now()
		h.leaseExpiresAt = startedAt.Add(30 * time.Second)
		h.leaseCtx, h.cancel = context.WithCancelCause(t.Context())
		// Run renewal alone to model the expiry watcher not being scheduled yet.
		h.workers.Add(1)
		go h.runRenewal()
		defer h.close()
		time.Sleep(32 * time.Second)
		synctest.Wait()
		require.NoError(t, h.context().Err())
		h.mu.Lock()
		expiresAt := h.leaseExpiresAt
		h.mu.Unlock()
		require.Equal(t, startedAt.Add(40*time.Second), expiresAt)
	})
}

func TestLeaseExpiryRechecksDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newLeaseHolder(xlog.NewNop(), &stubLeaseStorage{}, "lease", "holder",
			WithTTL(30*time.Second), WithRenewInterval(time.Hour))
		require.NoError(t, h.hold(t.Context()))
		defer h.close()
		time.Sleep(10 * time.Second)
		// Extend the deadline while the timer still targets the old one.
		h.mu.Lock()
		h.leaseExpiresAt = time.Now().Add(30 * time.Second)
		h.mu.Unlock()
		time.Sleep(21 * time.Second)
		synctest.Wait()
		require.NoError(t, h.context().Err(), "the old timer must recheck the extended deadline")
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(h.context()), ErrLeaseLost)
	})
}
