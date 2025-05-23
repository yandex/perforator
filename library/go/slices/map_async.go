package slices

import (
	"context"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type mapAsyncOption func(*mapAsyncParams)

type mapAsyncParams struct {
	limit      int
	panicCatch func(recover any) error
}

func WithLimit(limit int) mapAsyncOption {
	return func(o *mapAsyncParams) {
		o.limit = limit
	}
}

func WithPanicsCatch(fn func(recover any) error) mapAsyncOption {
	return func(o *mapAsyncParams) {
		o.panicCatch = fn
	}
}

// MapA is like MapE, but runs mapping of each element in a dedicated goroutine
func MapA[S ~[]T, T, M any](ctx context.Context, s S, fn func(context.Context, T) (M, error), options ...mapAsyncOption) ([]M, error) {
	if s == nil {
		return []M(nil), nil
	}
	if len(s) == 0 {
		return make([]M, 0), nil
	}

	g, ctx := errgroup.WithContext(ctx)
	params := &mapAsyncParams{
		panicCatch: func(recover any) error {
			return xerrors.Errorf("coroutine panicked with message: %w\nStack trace: %s", recover, string(debug.Stack()))
		},
	}
	for _, opt := range options {
		opt(params)
	}
	if params.limit != 0 {
		g.SetLimit(params.limit)
	}

	res := make([]M, len(s))
	for i, v := range s {
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = params.panicCatch(r)
				}
			}()
			transformed, err := fn(ctx, v)
			if err != nil {
				return err
			}
			res[i] = transformed
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return res, nil
}
