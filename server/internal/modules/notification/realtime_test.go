package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestBuildRealtimeNotificationPayloadIncludesRealtimeFields(t *testing.T) {
	t.Parallel()

	params := SendParams{
		UserID:       42,
		Type:         "review_reply",
		Title:        "新的回复",
		Body:         "你收到了一条新的回复",
		Payload:      json.RawMessage(`{"replyId":"r-1"}`),
		SourceModule: "course.review",
		SourceID:     "reply-1",
		SourceURL:    "/reviews/1",
		CourseID:     42,
	}

	payload := buildRealtimeNotificationPayload("notif-1", params, time.Date(2026, 4, 23, 9, 0, 0, 0, time.UTC))

	assert.Equal(t, "notif-1", payload.ID)
	assert.Equal(t, int64(42), payload.UserID)
	assert.False(t, payload.IsRead)
	assert.Equal(t, "2026-04-23T09:00:00Z", payload.CreatedAt)
	assert.Equal(t, json.RawMessage(`{"replyId":"r-1"}`), payload.Payload)
	require.NotNil(t, payload.SourceURL)
	assert.Equal(t, "/reviews/1", *payload.SourceURL)
	require.NotNil(t, payload.CourseID)
	assert.Equal(t, int64(42), *payload.CourseID)
}

func TestDecodeNotificationPubSubPayloadReturnsStructuredObject(t *testing.T) {
	t.Parallel()

	raw := `{"id":"notif-1","userId":42,"type":"review_reply","title":"新的回复","body":"body","content":"body","payload":{"replyId":"r-1"},"sourceModule":"course.review","sourceId":"reply-1","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}`

	payload, err := decodeNotificationPubSubPayload(raw)
	require.NoError(t, err)

	data, ok := payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "notif-1", data["id"])
	assert.Equal(t, float64(42), data["userId"])
	assert.Equal(t, false, data["isRead"])
	assert.Equal(t, "2026-04-23T09:00:00Z", data["createdAt"])
}

func TestNewHubPanicsWhenRedisNil(t *testing.T) {
	assert.PanicsWithValue(t, "notification.NewHub: redis client must not be nil", func() {
		NewHub(nil)
	})
}

func TestHubStopIsNilAndZeroValueSafe(t *testing.T) {
	var nilHub *Hub
	assert.NotPanics(t, func() {
		nilHub.Stop()
	})

	zeroHub := &Hub{}
	assert.NotPanics(t, func() {
		zeroHub.Stop()
		zeroHub.Stop()
	})
}

func TestHubStartRedisSubscriberSkipsMissingRedis(t *testing.T) {
	called := false
	start := func(string, func(context.Context)) {
		called = true
	}

	var nilHub *Hub
	assert.NotPanics(t, func() {
		nilHub.StartRedisSubscriber(context.Background(), start)
	})
	assert.False(t, called)

	zeroHub := &Hub{}
	assert.NotPanics(t, func() {
		zeroHub.StartRedisSubscriber(nilContextForNotificationTest(), start)
	})
	assert.False(t, called)
}

func TestHubStartRedisSubscriberRequiresStarter(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = client.Close() })
	hub := NewHub(client)

	assert.PanicsWithValue(t, "notification.Hub.StartRedisSubscriber: starter is required", func() {
		hub.StartRedisSubscriber(context.Background(), nil)
	})
}

func nilContextForNotificationTest() context.Context {
	return nil
}

func TestServiceMarkReadBroadcastsNotificationRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	hub := NewHub(redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))
	service := NewService(repo, hub, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))

	userID := seedNotificationUser(t, fixture, "notif-read-user")
	notifID, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         "review_reply",
		Title:        "新的回复",
		Body:         "reply body",
		SourceModule: "course.review",
		SourceID:     "reply-1",
	})
	require.NoError(t, err)

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	require.NoError(t, service.MarkRead(ctx, notifID, userID))

	select {
	case event := <-ch:
		assert.Equal(t, "notification_read", event.Event)
		data, ok := event.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, notifID, data["id"])
		assert.Equal(t, userID, data["userId"])
		assert.Equal(t, true, data["isRead"])
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification_read event")
	}
}

func TestServiceMarkReadMissingNotificationIsNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	hub := NewHub(redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))
	service := NewService(repo, hub, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))

	userID := seedNotificationUser(t, fixture, "notif-read-missing-user")
	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	require.NoError(t, service.MarkRead(ctx, "550e8400-e29b-41d4-a716-446655440999", userID))

	select {
	case event := <-ch:
		t.Fatalf("unexpected event for noop mark read: %+v", event)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestServiceMarkAllReadBroadcastsReadAllAndUnreadCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	hub := NewHub(redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))
	service := NewService(repo, hub, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}))

	userID := seedNotificationUser(t, fixture, "notif-read-all-user")
	_, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         "review_reply",
		Title:        "新的回复",
		Body:         "reply body",
		SourceModule: "course.review",
		SourceID:     "reply-1",
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         "system",
		Title:        "系统通知",
		Body:         "system body",
		SourceModule: "system",
		SourceID:     "system-1",
	})
	require.NoError(t, err)

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	require.NoError(t, service.MarkAllRead(ctx, userID))

	var sawReadAll bool
	var sawUnreadCount bool
	for i := 0; i < 2; i++ {
		select {
		case event := <-ch:
			switch event.Event {
			case "notification_read_all":
				sawReadAll = true
				data, ok := event.Data.(map[string]any)
				require.True(t, ok)
				assert.Equal(t, userID, data["userId"])
				assert.Equal(t, true, data["isRead"])
			case "unread_count":
				sawUnreadCount = true
				data, ok := event.Data.(map[string]int)
				require.True(t, ok)
				assert.Equal(t, 0, data["count"])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("expected notification_read_all and unread_count events")
		}
	}

	assert.True(t, sawReadAll)
	assert.True(t, sawUnreadCount)
}

func seedNotificationUser(t *testing.T, fixture *postgresfixture.Fixture, casdoorSubject string) int64 {
	t.Helper()

	var userID int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO users (casdoor_subject, username, email, user_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, casdoorSubject, casdoorSubject, casdoorSubject+"@example.com", "hash-"+casdoorSubject).Scan(&userID)
	require.NoError(t, err)
	return userID
}
