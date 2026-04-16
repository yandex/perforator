// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

// Package wait provides constructs for waiting on conditionals within specified constraints.
package wait

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrTimeoutExceeded      = errors.New("wait: timeout exceeded")
	ErrAttemptsExceeded     = errors.New("wait: attempts exceeded")
	ErrConditionUnsatisfied = errors.New("wait: condition unsatisfied")
	ErrNoFunction           = errors.New("wait: no function specified")
)

const (
	defaultTimeout = 3 * time.Second
	defaultGap     = 250 * time.Millisecond
)

// A Constraint is something a test assertion can wait on before marking the
// result to be a failure. A Constraint is used in conjunction with either the
// InitialSuccess or ContinualSuccess option. A call to Run will execute the given
// function, returning nil or error depending on the Constraint configuration and
// the results of the function.
//
// InitialSuccess - retry a function until it returns a positive result. If the
// function never returns a positive result before the Constraint threshold is
// exceeded, an error is returned from Run().
//
// ContinualSuccess - retry a function asserting it returns a positive result until
// the Constraint threshold is exceeded. If at any point the function returns a
// negative result, an error is returned from Run().
//
// A Constraint threshold is configured via either Timeout or Attempts (not both).
//
// Timeout - Constraint is time bound.
//
// Attempts - Constraint is iteration bound.
//
// The use of Gap controls the pace of attempts by setting the amount of time to
// wait in between each attempt.
type Constraint struct {
	continual  bool // (initial || continual) success
	now        time.Time
	deadline   time.Time
	gap        time.Duration
	iterations int
	r          runnable
}

// InitialSuccess creates a new Constraint configured by opts that will wait for a
// positive result upon calling Constraint.Run. If the threshold of the Constraint
// is exceeded before reaching a positive result, an error is returned from the
// call to Constraint.Run.
//
// Timeout is used to set a maximum amount of time to wait for success.
// Attempts is used to set a maximum number of attempts to wait for success.
// Gap is used to control the amount of time to wait between retries.
//
// One of ErrorFunc, BoolFunc, or TestFunc represents the function that will
// be run under the constraint.
func InitialSuccess(opts ...Option) *Constraint {
	c := &Constraint{now: time.Now()}
	c.setup(opts...)
	return c
}

// ContinualSuccess creates a new Constraint configured by opts that will assert
// a positive result upon calling Constraint.Run, repeating the call until the
// Constraint reaches its threshold. If the result is negative, an error is
// returned from the call to Constraint.Run.
//
// Timeout is used to set the amount of time to assert success.
// Attempts is used to set the number of iterations to assert success.
// Gap is used to control the amount of time to wait between iterations.
//
// One of ErrorFunc, BoolFunc, or TestFunc represents the function that will
// be run under the constraint.
func ContinualSuccess(opts ...Option) *Constraint {
	c := &Constraint{now: time.Now(), continual: true}
	c.setup(opts...)
	return c
}

// Timeout sets a time bound on a Constraint.
//
// If set, the Attempts constraint configuration is disabled.
//
// Default 3 seconds.
func Timeout(duration time.Duration) Option {
	return func(c *Constraint) {
		c.deadline = time.Now().Add(duration)
		c.iterations = math.MaxInt
	}
}

// Attempts sets an iteration bound on a Constraint.
//
// If set, the Timeout constraint configuration is disabled.
//
// By default a Timeout constraint is set and the Attempts bound is disabled.
func Attempts(max int) Option {
	return func(c *Constraint) {
		c.iterations = max
		c.deadline = time.Date(9999, 0, 0, 0, 0, 0, 0, time.UTC)
	}
}

// Gap sets the amount of time to wait between attempts.
//
// Default 250 milliseconds.
func Gap(duration time.Duration) Option {
	return func(c *Constraint) {
		c.gap = duration
	}
}

// BoolFunc executes f under the thresholds of a Constraint.
func BoolFunc(f func() bool) Option {
	return func(c *Constraint) {
		if c.continual {
			c.r = boolFuncContinual(f)
		} else {
			c.r = boolFuncInitial(f)
		}
	}
}

// Option is used to configure a Constraint.
//
// Understood Option functions include Timeout, Attempts, Gap, InitialSuccess,
// and ContinualSuccess.
type Option func(*Constraint)

type runnable func(*runner) *result

type runner struct {
	c        *Constraint
	attempts int
}

type result struct {
	Err error
}

func boolFuncContinual(f func() bool) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			if !f() {
				return &result{Err: ErrConditionUnsatisfied}
			}

			// used another attempt
			r.attempts++

			// reached the desired attempts
			if r.attempts >= r.c.iterations {
				return &result{Err: nil}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or time
			select {
			case <-ctx.Done():
				return &result{Err: nil}
			case <-timer.C:
				// continue
			}
		}
	}
}

func boolFuncInitial(f func() bool) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			if f() {
				return &result{Err: nil}
			}

			// used another attempt
			r.attempts++

			// check iterations
			if r.attempts > r.c.iterations {
				return &result{Err: ErrAttemptsExceeded}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or timeout
			select {
			case <-ctx.Done():
				return &result{Err: ErrTimeoutExceeded}
			case <-timer.C:
				// continue
			}
		}
	}
}

// ErrorFunc will retry f while it returns a non-nil error, or until a wait
// constraint threshold is exceeded.
func ErrorFunc(f func() error) Option {
	return func(c *Constraint) {
		if c.continual {
			c.r = errFuncContinual(f)
		} else {
			c.r = errFuncInitial(f)
		}
	}
}

func errFuncContinual(f func() error) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			if err := f(); err != nil {
				return &result{Err: err}
			}

			// used another attempt
			r.attempts++

			// reached the desired attempts
			if r.attempts >= r.c.iterations {
				return &result{Err: nil}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or time
			select {
			case <-ctx.Done():
				return &result{Err: nil}
			case <-timer.C:
				// continue
			}
		}
	}
}

func errFuncInitial(f func() error) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			err := f()
			if err == nil {
				return &result{Err: nil}
			}

			// used another attempt
			r.attempts++

			// check iterations
			if r.attempts > r.c.iterations {
				return &result{
					Err: fmt.Errorf("%s: %w", ErrAttemptsExceeded.Error(), err),
				}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or timeout
			select {
			case <-ctx.Done():
				return &result{
					Err: fmt.Errorf("%s: %w", ErrTimeoutExceeded.Error(), err),
				}
			case <-timer.C:
				// continue
			}
		}
	}
}

// TestFunc will retry f while it returns false, or until a wait constraint
// threshold is exceeded. If f never succeeds, the latest returned error is
// wrapped into the result.
func TestFunc(f func() (bool, error)) Option {
	return func(c *Constraint) {
		if c.continual {
			c.r = testFuncContinual(f)
		} else {
			c.r = testFuncInitial(f)
		}
	}
}

func testFuncContinual(f func() (bool, error)) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			ok, err := f()
			if !ok {
				return &result{Err: fmt.Errorf("%s: %w", ErrConditionUnsatisfied.Error(), err)}
			}

			// used another attempt
			r.attempts++

			// reached the desired attempts
			if r.attempts >= r.c.iterations {
				return &result{Err: nil}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or time
			select {
			case <-ctx.Done():
				return &result{Err: nil}
			case <-timer.C:
				// continue
			}
		}
	}
}

func testFuncInitial(f func() (bool, error)) runnable {
	bg := context.Background()
	return func(r *runner) *result {
		ctx, cancel := context.WithDeadline(bg, r.c.deadline)
		defer cancel()

		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			// make an attempt
			ok, err := f()
			if ok {
				return &result{Err: nil}
			}

			// set default error
			if err == nil {
				err = ErrConditionUnsatisfied
			}

			// used another attempt
			r.attempts++

			// check iterations
			if r.attempts > r.c.iterations {
				return &result{
					Err: fmt.Errorf("%s: %w", ErrAttemptsExceeded.Error(), err),
				}
			}

			// reset timer to gap interval
			timer.Reset(r.c.gap)

			// wait for gap or timeout
			select {
			case <-ctx.Done():
				return &result{
					Err: fmt.Errorf("%s: %w", ErrTimeoutExceeded.Error(), err),
				}
			case <-timer.C:
				// continue
			}
		}
	}
}

func (c *Constraint) setup(opts ...Option) {
	for _, opt := range append([]Option{
		Timeout(defaultTimeout),
		Gap(defaultGap),
	}, opts...) {
		opt(c)
	}
}

// Run the Constraint and produce an error result.
func (c *Constraint) Run() error {
	if c.r == nil {
		return ErrNoFunction
	}
	return c.r(&runner{
		c:        c,
		attempts: 0,
	}).Err
}
