package notification

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeSSEEventName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "notification.updated", sanitizeSSEEventName("notification.updated"))
	assert.Equal(t, "message", sanitizeSSEEventName(""))
	assert.Equal(t, "message", sanitizeSSEEventName(" \n\t "))
	assert.Equal(t, "notification_drop_table", sanitizeSSEEventName("notification\n:drop table"))
	assert.Equal(t, "unread_count", sanitizeSSEEventName(" unread count "))
}

func TestWriteSSE_SanitizesEventName(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := writeSSE(&buf, "unread\ncount", map[string]int{"count": 2})
	require.NoError(t, err)
	assert.Equal(t, "event: unread_count\ndata: {\"count\":2}\n\n", buf.String())
}

func TestWriteSSE_WritesStructuredNotificationObject(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := writeSSE(&buf, "notification", map[string]any{
		"id":        "notif-1",
		"isRead":    false,
		"createdAt": "2026-04-23T09:00:00Z",
	})
	require.NoError(t, err)
	assert.Equal(t, "event: notification\ndata: {\"createdAt\":\"2026-04-23T09:00:00Z\",\"id\":\"notif-1\",\"isRead\":false}\n\n", buf.String())
}
