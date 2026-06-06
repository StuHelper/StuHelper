package notification

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type recordedRealtimeEvent struct {
	userID int64
	event  SSEEvent
}

type recordingRealtimePublisher struct {
	mu     sync.Mutex
	events []recordedRealtimeEvent
	err    error
}

func (p *recordingRealtimePublisher) Publish(_ context.Context, userID int64, event SSEEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, recordedRealtimeEvent{
		userID: userID,
		event:  event,
	})
	return p.err
}

func (p *recordingRealtimePublisher) snapshot() []recordedRealtimeEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	events := make([]recordedRealtimeEvent, len(p.events))
	copy(events, p.events)
	return events
}

func TestBuildRealtimeNotificationPayloadIncludesRealtimeFields(t *testing.T) {
	t.Parallel()

	params := SendParams{
		UserID:       42,
		Type:         TypeReply,
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

func TestDecodeRealtimeEnvelopeReturnsStructuredObject(t *testing.T) {
	t.Parallel()

	raw := `{"userId":42,"event":{"event":"notification","data":{"id":"notif-1","userId":42,"type":"reply","title":"新的回复","body":"body","content":"body","payload":{"replyId":"r-1"},"sourceModule":"course.review","sourceId":"reply-1","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}}}`

	envelope, err := decodeRealtimeEnvelope(raw)
	require.NoError(t, err)

	assert.Equal(t, int64(42), envelope.UserID)
	assert.Equal(t, "notification", envelope.Event.Event)
	data, ok := envelope.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "notif-1", data["id"])
	assert.Equal(t, float64(42), data["userId"])
	assert.Equal(t, false, data["isRead"])
	assert.Equal(t, "2026-04-23T09:00:00Z", data["createdAt"])
}

func TestDecodeRealtimePayloadAcceptsLegacyNotificationPayload(t *testing.T) {
	t.Parallel()

	raw := `{"id":"notif-1","userId":42,"type":"reply","title":"新的回复","body":"body","content":"body","payload":{"replyId":"r-1"},"sourceModule":"course.review","sourceId":"reply-1","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}`

	envelope, err := decodeRealtimePayload(raw, 42, realtimePayloadLegacy)
	require.NoError(t, err)

	assert.Equal(t, int64(42), envelope.UserID)
	assert.Equal(t, "notification", envelope.Event.Event)
	data, ok := envelope.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "notif-1", data["id"])
	assert.Equal(t, TypeReply, data["type"])
	assert.Equal(t, float64(42), data["userId"])
	assert.Equal(t, false, data["isRead"])
	assert.Equal(t, "2026-04-23T09:00:00Z", data["createdAt"])
}

func TestNewRealtimePanicsOnNilDependencies(t *testing.T) {
	fixture := redisfixture.Start(t)
	assert.PanicsWithValue(t, "notification.NewRealtime: redis client must not be nil", func() {
		NewRealtime(nil, NewHub())
	})
	assert.PanicsWithValue(t, "notification.NewRealtime: hub must not be nil", func() {
		NewRealtime(fixture.Client, nil)
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

func TestRealtimeStartSubscriberSkipsMissingRedis(t *testing.T) {
	called := false
	start := func(string, func(context.Context)) {
		called = true
	}

	var nilRealtime *Realtime
	assert.NotPanics(t, func() {
		nilRealtime.StartSubscriber(context.Background(), start)
	})
	assert.False(t, called)

	zeroRealtime := &Realtime{}
	assert.NotPanics(t, func() {
		zeroRealtime.StartSubscriber(nilContextForNotificationTest(), start)
	})
	assert.False(t, called)
}

func TestRealtimeStartSubscriberRequiresStarter(t *testing.T) {
	fixture := redisfixture.Start(t)
	realtime := NewRealtime(fixture.Client, NewHub())

	assert.PanicsWithValue(t, "notification.Realtime.StartSubscriber: starter is required", func() {
		realtime.StartSubscriber(context.Background(), nil)
	})
}

func nilContextForNotificationTest() context.Context {
	return nil
}

func TestRealtimePublishNotificationUsesLegacyPayload(t *testing.T) {
	t.Parallel()

	fixture := redisfixture.Start(t)
	realtime := NewRealtime(fixture.Client, NewHub())
	userID := int64(42)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	legacySub := fixture.Client.Subscribe(ctx, notificationRealtimeChannel(userID))
	defer legacySub.Close()
	_, err := legacySub.Receive(ctx)
	require.NoError(t, err)

	require.NoError(t, realtime.Publish(ctx, userID, SSEEvent{
		Event: "notification",
		Data: Notification{
			ID:           "notif-1",
			UserID:       userID,
			Type:         TypeReply,
			Title:        "新的回复",
			Body:         "body",
			Content:      "body",
			SourceModule: "course.review",
			SourceID:     "reply-1",
			IsRead:       false,
			CreatedAt:    "2026-04-23T09:00:00Z",
		},
	}))

	msg, err := legacySub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, notificationRealtimeChannel(userID), msg.Channel)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(msg.Payload), &payload))
	assert.Equal(t, "notif-1", payload["id"])
	assert.Equal(t, TypeReply, payload["type"])
	assert.Equal(t, float64(userID), payload["userId"])
	assert.NotContains(t, payload, "event")
}

func TestRealtimePublishNonNotificationUsesV2Only(t *testing.T) {
	t.Parallel()

	fixture := redisfixture.Start(t)
	realtime := NewRealtime(fixture.Client, NewHub())
	userID := int64(42)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	legacySub := fixture.Client.PSubscribe(ctx, notificationRealtimeChannelPattern)
	defer legacySub.Close()
	_, err := legacySub.Receive(ctx)
	require.NoError(t, err)

	v2Sub := fixture.Client.Subscribe(ctx, notificationRealtimeV2Channel(userID))
	defer v2Sub.Close()
	_, err = v2Sub.Receive(ctx)
	require.NoError(t, err)

	require.NoError(t, realtime.Publish(ctx, userID, SSEEvent{
		Event: "unread_count",
		Data:  map[string]int{"count": 3},
	}))

	msg, err := v2Sub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, notificationRealtimeV2Channel(userID), msg.Channel)

	envelope, err := decodeRealtimeEnvelope(msg.Payload)
	require.NoError(t, err)
	assert.Equal(t, userID, envelope.UserID)
	assert.Equal(t, "unread_count", envelope.Event.Event)
	data, ok := envelope.Event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(3), data["count"])

	legacyCtx, legacyCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer legacyCancel()
	legacyMsg, err := legacySub.ReceiveMessage(legacyCtx)
	if err == nil {
		t.Fatalf("expected no legacy notify:* message, got channel=%s payload=%s", legacyMsg.Channel, legacyMsg.Payload)
	}
}

func TestRealtimePublishesV2EnvelopeAndSubscriberBroadcastsToHub(t *testing.T) {
	t.Parallel()

	fixture := redisfixture.Start(t)
	hub := NewHub()
	realtime := NewRealtime(fixture.Client, hub)
	userID := int64(42)

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	realtime.StartSubscriber(ctx, func(name string, run func(context.Context)) {
		assert.Equal(t, notificationRealtimeSubscriberName, name)
		close(started)
		go run(ctx)
	})
	defer realtime.Stop()

	<-started
	require.Eventually(t, func() bool {
		return fixture.Server.PubSubNumPat() >= 2
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, realtime.Publish(ctx, userID, SSEEvent{
		Event: "unread_count",
		Data:  map[string]int{"count": 0},
	}))

	select {
	case event := <-ch:
		assert.Equal(t, "unread_count", event.Event)
		data, ok := event.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(0), data["count"])
	case <-time.After(2 * time.Second):
		t.Fatal("expected realtime event")
	}
}

func TestRealtimeSubscriberBroadcastsLegacyNotificationPayload(t *testing.T) {
	t.Parallel()

	fixture := redisfixture.Start(t)
	hub := NewHub()
	realtime := NewRealtime(fixture.Client, hub)
	userID := int64(42)

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	realtime.StartSubscriber(ctx, func(name string, run func(context.Context)) {
		assert.Equal(t, notificationRealtimeSubscriberName, name)
		close(started)
		go run(ctx)
	})
	defer realtime.Stop()

	<-started
	require.Eventually(t, func() bool {
		return fixture.Server.PubSubNumPat() >= 2
	}, time.Second, 10*time.Millisecond)

	raw := `{"id":"notif-1","userId":42,"type":"reply","title":"新的回复","body":"body","content":"body","sourceModule":"course.review","sourceId":"reply-1","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}`
	require.NoError(t, fixture.Client.Publish(ctx, notificationRealtimeChannel(userID), raw).Err())

	select {
	case event := <-ch:
		assert.Equal(t, "notification", event.Event)
		data, ok := event.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "notif-1", data["id"])
		assert.Equal(t, TypeReply, data["type"])
		assert.Equal(t, float64(42), data["userId"])
	case <-time.After(2 * time.Second):
		t.Fatal("expected legacy realtime event")
	}
}

func TestRealtimeSubscriberDropsEnvelopeOnLegacyChannel(t *testing.T) {
	t.Parallel()

	fixture := redisfixture.Start(t)
	hub := NewHub()
	realtime := NewRealtime(fixture.Client, hub)
	channelUserID := int64(42)

	ch := hub.Subscribe(channelUserID)
	defer hub.Unsubscribe(channelUserID, ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	realtime.StartSubscriber(ctx, func(name string, run func(context.Context)) {
		assert.Equal(t, notificationRealtimeSubscriberName, name)
		close(started)
		go run(ctx)
	})
	defer realtime.Stop()

	<-started
	require.Eventually(t, func() bool {
		return fixture.Server.PubSubNumPat() >= 2
	}, time.Second, 10*time.Millisecond)

	mismatch := `{"userId":43,"event":{"event":"notification","data":{"id":"notif-mismatch","userId":43,"type":"reply","title":"新的回复","sourceModule":"course.review","sourceId":"reply-1","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}}}`
	require.NoError(t, fixture.Client.Publish(ctx, notificationRealtimeChannel(channelUserID), mismatch).Err())
	valid := `{"id":"notif-valid","userId":42,"type":"reply","title":"新的回复","body":"body","content":"body","sourceModule":"course.review","sourceId":"reply-2","isRead":false,"createdAt":"2026-04-23T09:00:00Z"}`
	require.NoError(t, fixture.Client.Publish(ctx, notificationRealtimeChannel(channelUserID), valid).Err())

	select {
	case event := <-ch:
		assert.Equal(t, "notification", event.Event)
		data, ok := event.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "notif-valid", data["id"])
		assert.Equal(t, float64(42), data["userId"])
	case <-time.After(2 * time.Second):
		t.Fatal("expected valid realtime event after mismatched envelope")
	}
}

func TestServiceMarkReadBroadcastsNotificationRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	realtime := &recordingRealtimePublisher{}
	service := NewService(repo, realtime)

	userID := seedNotificationUser(t, fixture, "notif-read-user")
	notifID, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         TypeReply,
		Title:        "新的回复",
		Body:         "reply body",
		SourceModule: "course.review",
		SourceID:     "reply-1",
	})
	require.NoError(t, err)

	require.NoError(t, service.MarkRead(ctx, notifID, userID))

	events := realtime.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, userID, events[0].userID)
	assert.Equal(t, "notification_read", events[0].event.Event)
	data, ok := events[0].event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, notifID, data["id"])
	assert.Equal(t, userID, data["userId"])
	assert.Equal(t, true, data["isRead"])
}

func TestServiceMarkReadMissingNotificationIsNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	realtime := &recordingRealtimePublisher{}
	service := NewService(repo, realtime)

	userID := seedNotificationUser(t, fixture, "notif-read-missing-user")

	require.NoError(t, service.MarkRead(ctx, "550e8400-e29b-41d4-a716-446655440999", userID))
	assert.Empty(t, realtime.snapshot())
}

func TestServiceMarkAllReadBroadcastsReadAllAndUnreadCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	realtime := &recordingRealtimePublisher{}
	service := NewService(repo, realtime)

	userID := seedNotificationUser(t, fixture, "notif-read-all-user")
	_, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         TypeReply,
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

	require.NoError(t, service.MarkAllRead(ctx, userID))

	var sawReadAll bool
	var sawUnreadCount bool
	events := realtime.snapshot()
	require.Len(t, events, 2)
	for _, recorded := range events {
		assert.Equal(t, userID, recorded.userID)
		switch recorded.event.Event {
		case "notification_read_all":
			sawReadAll = true
			data, ok := recorded.event.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, userID, data["userId"])
			assert.Equal(t, true, data["isRead"])
		case "unread_count":
			sawUnreadCount = true
			data, ok := recorded.event.Data.(map[string]int)
			require.True(t, ok)
			assert.Equal(t, 0, data["count"])
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
