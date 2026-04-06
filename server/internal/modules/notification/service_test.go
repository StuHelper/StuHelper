package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHub_SubscribeAndBroadcast(t *testing.T) {
	hub := NewHub(nil)
	defer hub.Stop()

	ch := hub.Subscribe(42)
	defer hub.Unsubscribe(42, ch)

	hub.Broadcast(42, SSEEvent{Event: "notification", Data: "test"})

	event := <-ch
	assert.Equal(t, "notification", event.Event)
	assert.Equal(t, "test", event.Data)
}

func TestHub_UnsubscribeRemovesConnection(t *testing.T) {
	hub := NewHub(nil)
	defer hub.Stop()

	ch := hub.Subscribe(42)
	hub.Unsubscribe(42, ch)

	hub.mu.RLock()
	_, exists := hub.connections[42]
	hub.mu.RUnlock()

	assert.False(t, exists, "user should have no connections after unsubscribe")
}

func TestHub_BroadcastToNonexistentUser(t *testing.T) {
	hub := NewHub(nil)
	defer hub.Stop()

	// should not panic
	hub.Broadcast(999, SSEEvent{Event: "test", Data: "data"})
}

func TestNilIfEmpty(t *testing.T) {
	assert.Nil(t, nilIfEmpty(""))
	result := nilIfEmpty("hello")
	assert.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestHub_StopTwiceDoesNotPanic(t *testing.T) {
	hub := NewHub(nil)
	assert.NotPanics(t, func() {
		hub.Stop()
		hub.Stop()
	})
}
