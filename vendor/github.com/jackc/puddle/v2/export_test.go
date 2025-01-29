package puddle

import "context"

func (p *Pool[T]) AcquireRaw(ctx context.Context) (*Resource[T], error) {
	return p.acquire(ctx)
}

var AcquireSemAll = acquireSemAll
