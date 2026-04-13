package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHub_EvictsOldestConnectionAndUnsubscribeIsIdempotent(t *testing.T) {
	hub := NewHub(nil)
	defer hub.Stop()

	const userID int64 = 42

	conns := make([]chan SSEEvent, 0, maxConnsPerUser+1)
	for i := 0; i < maxConnsPerUser; i++ {
		conns = append(conns, hub.Subscribe(userID))
	}

	evicted := conns[0]
	conns = append(conns, hub.Subscribe(userID))

	assertClosedChannel(t, evicted)
	assertOpenChannel(t, conns[1])

	hub.mu.RLock()
	require.Len(t, hub.connections[userID].order, maxConnsPerUser)
	assert.Equal(t, conns[1], hub.connections[userID].order[0], "oldest survivor should remain at the front")
	hub.mu.RUnlock()

	assert.NotPanics(t, func() {
		for _, ch := range conns {
			hub.Unsubscribe(userID, ch)
		}
	})

	hub.mu.RLock()
	_, exists := hub.connections[userID]
	hub.mu.RUnlock()
	assert.False(t, exists)
}

func assertClosedChannel(t *testing.T, ch chan SSEEvent) {
	t.Helper()

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "expected channel to be closed")
	default:
		t.Fatal("expected channel to be closed")
	}
}

func assertOpenChannel(t *testing.T, ch chan SSEEvent) {
	t.Helper()

	select {
	case <-ch:
		t.Fatal("expected channel to remain open and empty")
	default:
	}
}
