package unwindtable

type freelist[T any] struct {
	buf chan T
}

func newFreelist[T any](size int) *freelist[T] {
	return &freelist[T]{
		buf: make(chan T, size),
	}
}

func (f *freelist[T]) put(value T) {
	select {
	case f.buf <- value:
	default:
		panic("overflow")
	}
}

func (f *freelist[T]) get() (T, bool) {
	var res T
	select {
	case res = <-f.buf:
		return res, true
	default:
		return res, false
	}
}

func (f *freelist[T]) TotalItems() int {
	return cap(f.buf)
}

func (f *freelist[T]) FreeItems() int {
	return len(f.buf)
}
