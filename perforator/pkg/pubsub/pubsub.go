package pubsub

import "sync"

// thread safe
type PubSub[T any] struct {
	mutex sync.Mutex

	clientChans map[uint64]chan<- T
	lastID      uint64
}

func NewPubSub[T any]() *PubSub[T] {
	return &PubSub[T]{
		clientChans: make(map[uint64]chan<- T),
		lastID:      0,
	}
}

// will block until finishes all publications
func (p *PubSub[T]) Publish(val T) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, clientChan := range p.clientChans {
		clientChan <- val
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

func (p *PubSub[T]) Subscribe(chanCapacity uint32) *Subscription[T] {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	newChan := make(chan T, chanCapacity)
	id := p.lastID
	p.lastID++
	p.clientChans[id] = newChan

	return &Subscription[T]{
		subChan: newChan,
		id:      id,
		pubSub:  p,
	}
}
