package pubsub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPubSub_Blocking(t *testing.T) {
	ps := NewPubSub[int]()
	sub := ps.Subscribe(WithChanCapacity(1))

	// Publish first value - should succeed immediately (capacity 1)
	ps.Publish(1)

	// Publish second value - should block. We use a goroutine to detect blocking.
	done := make(chan struct{})
	go func() {
		ps.Publish(2)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Publish should have blocked")
	case <-time.After(50 * time.Millisecond):
		// Expected to block
	}

	// Read the first value to unblock the second publish
	val := <-sub.Chan()
	assert.Equal(t, 1, val)

	select {
	case <-done:
		// Successfully unblocked
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Publish should have unblocked")
	}

	val = <-sub.Chan()
	assert.Equal(t, 2, val)
}

func TestPubSub_NonBlocking(t *testing.T) {
	ps := NewPubSub[int](WithNonBlockingPublish())
	sub := ps.Subscribe(WithChanCapacity(1))

	// Publish first value - should succeed
	ps.Publish(1)

	// Publish second value - should NOT block, but value will be dropped
	done := make(chan struct{})
	go func() {
		ps.Publish(2)
		close(done)
	}()

	select {
	case <-done:
		// Expected to not block
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Publish should not have blocked")
	}

	// Read the first value
	val := <-sub.Chan()
	assert.Equal(t, 1, val)

	// Channel should now be empty (2 was dropped)
	select {
	case val = <-sub.Chan():
		t.Fatalf("Expected channel to be empty, but got %d", val)
	default:
		// Expected
	}
}
