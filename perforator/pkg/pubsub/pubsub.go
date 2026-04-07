package pubsub

import "sync"

type pubSubOptions struct {
	nonBlocking bool
}

type PubSubOption func(*pubSubOptions)

func WithNonBlockingPublish() PubSubOption {
	return func(o *pubSubOptions) {
		o.nonBlocking = true
	}
}

// thread safe
type PubSub[T any] struct {
	mutex sync.Mutex

	clientChans map[uint64]chan<- T
	lastID      uint64
	nonBlocking bool
}

func NewPubSub[T any](opts ...PubSubOption) *PubSub[T] {
	options := &pubSubOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return &PubSub[T]{
		clientChans: make(map[uint64]chan<- T),
		lastID:      0,
		nonBlocking: options.nonBlocking,
	}
}

// will block until finishes all publications if nonBlocking is false
func (p *PubSub[T]) Publish(val T) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, clientChan := range p.clientChans {
		if p.nonBlocking {
			select {
			case clientChan <- val:
			default:
			}
		} else {
			clientChan <- val
		}
	}
}

func (p *PubSub[T]) closeImpl(id uint64) {
	clientChan := p.clientChans[id]
	if clientChan == nil {
		return
	}

	close(p.clientChans[id])
	delete(p.clientChans, id)
}

func (p *PubSub[T]) close(id uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.closeImpl(id)
}

func (p *PubSub[T]) CloseAll() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for id := range p.clientChans {
		p.closeImpl(id)
	}
}

type Subscription[T any] struct {
	subChan <-chan T
	id      uint64
	pubSub  *PubSub[T]
}

func (s *Subscription[T]) Chan() <-chan T {
	return s.subChan
}

func (s *Subscription[T]) Close() {
	s.pubSub.close(s.id)
}

type subscriptionOptions struct {
	chanCapacity uint32
}

func defaultSubscriptionOptions() *subscriptionOptions {
	return &subscriptionOptions{
		chanCapacity: 0,
	}
}

type SubscriptionOption func(o *subscriptionOptions)

func WithChanCapacity(chanCapacity uint32) SubscriptionOption {
	return func(o *subscriptionOptions) {
		o.chanCapacity = chanCapacity
	}
}

func (p *PubSub[T]) Subscribe(opts ...SubscriptionOption) *Subscription[T] {
	options := defaultSubscriptionOptions()
	for _, opt := range opts {
		opt(options)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	newChan := make(chan T, options.chanCapacity)
	id := p.lastID
	p.lastID++
	p.clientChans[id] = newChan

	return &Subscription[T]{
		subChan: newChan,
		id:      id,
		pubSub:  p,
	}
}
