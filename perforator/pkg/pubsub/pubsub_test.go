package pubsub

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestPubSub_Simple(t *testing.T) {
	pubsub := NewPubSub[uint32]()

	val := uint32(42)

	sub := pubsub.Subscribe(1)
	pubsub.Publish(val)

	receivedVal := <-sub.Chan()
	require.Equal(t, val, receivedVal)

	select {
	case receivedVal = <-sub.Chan():
		require.FailNow(t, "channel must not contain any element, got %v", receivedVal)
	default:
	}

	sub.Close()

	_, ok := <-sub.Chan()
	require.False(t, ok)

	pubsub.CloseAll()
}

func TestPubSub_MultipleSubSimple(t *testing.T) {
	pubsub := NewPubSub[uint32]()

	val := uint32(42)

	g, _ := errgroup.WithContext(context.Background())

	waitSubsribe := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		waitSubsribe.Add(1)
		g.Go(func() error {
			sub := pubsub.Subscribe(1)
			waitSubsribe.Done()

			receivedVal := <-sub.Chan()
			require.Equal(t, val, receivedVal)
			sub.Close()
			_, ok := <-sub.Chan()
			require.False(t, ok)
			return nil
		})
	}

	waitSubsribe.Wait()

	pubsub.Publish(val)

	_ = g.Wait()

	pubsub.CloseAll()
}

func TestPubSub_MultiplePublishOrder(t *testing.T) {
	pubsub := NewPubSub[uint32]()

	g, _ := errgroup.WithContext(context.Background())

	items := 20
	sub := pubsub.Subscribe(uint32(items))

	g.Go(func() error {
		receivedCount := 0
		lastReceivedVal := uint32(0)
		for receivedVal := range sub.Chan() {
			require.Equal(t, lastReceivedVal+1, receivedVal)
			lastReceivedVal = receivedVal
			receivedCount++
			if receivedCount == items {
				sub.Close()
			}
		}

		return nil
	})

	g.Go(func() error {
		for i := 0; i < items; i++ {
			pubsub.Publish(uint32(i + 1))
		}
		return nil
	})

	_ = g.Wait()

	pubsub.CloseAll()
}
