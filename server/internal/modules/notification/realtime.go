package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	notificationRealtimeChannelPrefix  = "notify:"
	notificationRealtimeChannelPattern = notificationRealtimeChannelPrefix + "*"
	notificationRealtimeSubscriberName = "notification redis subscriber"
)

// RealtimePublisher publishes notification realtime events after durable writes.
type RealtimePublisher interface {
	Publish(ctx context.Context, userID int64, event SSEEvent) error
}

// Realtime owns notification Redis Pub/Sub and bridges cross-process events into
// the local SSE hub.
type Realtime struct {
	rdb      *redis.Client
	hub      *Hub
	stopCh   chan struct{}
	stopOnce sync.Once
}

type realtimeEnvelope struct {
	UserID int64    `json:"userId"`
	Event  SSEEvent `json:"event"`
}

// NewRealtime 创建通知实时组件。
func NewRealtime(rdb *redis.Client, hub *Hub) *Realtime {
	if rdb == nil {
		panic("notification.NewRealtime: redis client must not be nil")
	}
	if hub == nil {
		panic("notification.NewRealtime: hub must not be nil")
	}
	return &Realtime{
		rdb:    rdb,
		hub:    hub,
		stopCh: make(chan struct{}),
	}
}

// Publish publishes a realtime event through Redis Pub/Sub.
func (r *Realtime) Publish(ctx context.Context, userID int64, event SSEEvent) error {
	if r == nil || r.rdb == nil {
		return fmt.Errorf("notification realtime publish: redis client is not configured")
	}
	data, err := json.Marshal(realtimeEnvelope{
		UserID: userID,
		Event:  event,
	})
	if err != nil {
		return fmt.Errorf("notification realtime marshal: %w", err)
	}
	if err := r.rdb.Publish(ctx, notificationRealtimeChannel(userID), data).Err(); err != nil {
		return fmt.Errorf("notification realtime publish: %w", err)
	}
	return nil
}

// StartSubscriber 启动 Redis Pub/Sub 订阅。
// 调用方必须传入 start，以统一托管 goroutine 生命周期。
func (r *Realtime) StartSubscriber(ctx context.Context, start func(string, func(context.Context))) {
	if r == nil || r.rdb == nil {
		logger.L().Warn("notification redis subscriber not started: redis client is not configured")
		return
	}
	if start == nil {
		panic("notification.Realtime.StartSubscriber: starter is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pubsub := r.rdb.PSubscribe(ctx, notificationRealtimeChannelPattern)

	run := func(ctx context.Context) {
		defer func() {
			if err := pubsub.Close(); err != nil {
				logger.L().Warn("failed to close notification pubsub", zap.Error(err))
			}
		}()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.stopCh:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				userID, ok := notificationRealtimeUserIDFromChannel(msg.Channel)
				if !ok {
					continue
				}
				envelope, err := decodeRealtimePayload(msg.Payload, userID)
				if err != nil {
					logger.L().Warn("failed to decode notification realtime payload",
						zap.Int64("user_id", userID),
						zap.Error(err),
					)
					continue
				}
				if envelope.UserID != userID {
					logger.L().Warn("notification realtime payload user mismatch",
						zap.Int64("channel_user_id", userID),
						zap.Int64("payload_user_id", envelope.UserID),
					)
					continue
				}
				r.hub.Broadcast(userID, envelope.Event)
			}
		}
	}

	start(notificationRealtimeSubscriberName, run)
}

// Stop 停止实时订阅。
func (r *Realtime) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		if r.stopCh != nil {
			close(r.stopCh)
		}
	})
}

func notificationRealtimeChannel(userID int64) string {
	return notificationRealtimeChannelPrefix + strconv.FormatInt(userID, 10)
}

func notificationRealtimeUserIDFromChannel(channel string) (int64, bool) {
	raw, ok := strings.CutPrefix(channel, notificationRealtimeChannelPrefix)
	if !ok {
		return 0, false
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}

func decodeRealtimeEnvelope(raw string) (realtimeEnvelope, error) {
	var envelope realtimeEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return realtimeEnvelope{}, err
	}
	if envelope.Event.Event == "" {
		return realtimeEnvelope{}, fmt.Errorf("missing realtime event name")
	}
	return envelope, nil
}

func decodeRealtimePayload(raw string, channelUserID int64) (realtimeEnvelope, error) {
	envelope, err := decodeRealtimeEnvelope(raw)
	if err == nil {
		return envelope, nil
	}

	var legacyPayload map[string]any
	if legacyErr := json.Unmarshal([]byte(raw), &legacyPayload); legacyErr != nil {
		return realtimeEnvelope{}, err
	}
	if legacyPayload["event"] != nil {
		return realtimeEnvelope{}, err
	}
	if legacyPayload["id"] == nil || legacyPayload["type"] == nil {
		return realtimeEnvelope{}, err
	}
	return realtimeEnvelope{
		UserID: channelUserID,
		Event: SSEEvent{
			Event: "notification",
			Data:  legacyPayload,
		},
	}, nil
}
