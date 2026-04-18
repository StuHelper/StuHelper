package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// Sender 通知发送接口，供其他模块调用
type Sender interface {
	Send(ctx context.Context, params SendParams) error
	SendBatch(ctx context.Context, params []SendParams) error
}

// SendParams 发送通知参数
type SendParams struct {
	UserID       int64
	Type         string
	Title        string
	Body         string
	Content      string
	Payload      json.RawMessage
	SourceModule string
	SourceID     string
	SourceURL    string
	CourseID     int64 // 关联课程 ID，用于前端精准跳转；0 表示无关联
}

// Notification 通知实体
type Notification struct {
	ID           string          `json:"id"`
	UserID       int64           `json:"userId"`
	Type         string          `json:"type"`
	Title        string          `json:"title"`
	Body         string          `json:"body,omitempty"`
	Content      string          `json:"content,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	SourceModule string          `json:"sourceModule"`
	SourceID     string          `json:"sourceId"`
	SourceURL    *string         `json:"sourceUrl,omitempty"`
	CourseID     *int64          `json:"courseID,omitempty"`
	IsRead       bool            `json:"isRead"`
	CreatedAt    string          `json:"createdAt"`
}

// Hub 管理 SSE 连接和 Redis Pub/Sub
type Hub struct {
	rdb         *redis.Client
	mu          sync.RWMutex
	connections map[int64]*userConnections // userID -> ordered connections
	stopCh      chan struct{}
	stopOnce    sync.Once
}

type userConnections struct {
	order []chan SSEEvent
}

// SSEEvent SSE 推送事件
type SSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// maxConnsPerUser 每个用户允许的最大 SSE 并发连接数。
// 超出限制时关闭最早建立的连接，防止资源耗尽。
const maxConnsPerUser = 5

// sseBufferSize 为单连接缓冲少量突发事件。
// 取 32 足以覆盖未读数 + 列表刷新等短时 burst，同时避免慢客户端无限积压。
const sseBufferSize = 32

// NewHub 创建通知 Hub
func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:         rdb,
		connections: make(map[int64]*userConnections),
		stopCh:      make(chan struct{}),
	}
}

// Subscribe 注册 SSE 连接。
// 当同一用户的连接数达到 maxConnsPerUser 时，按订阅顺序驱逐最老的现有连接。
func (h *Hub) Subscribe(userID int64) chan SSEEvent {
	ch := make(chan SSEEvent, sseBufferSize)
	h.mu.Lock()
	if h.connections[userID] == nil {
		h.connections[userID] = &userConnections{}
	}
	uc := h.connections[userID]
	if len(uc.order) >= maxConnsPerUser {
		victim := uc.order[0]
		uc.order = uc.order[1:]
		close(victim)
	}
	uc.order = append(uc.order, ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe 注销 SSE 连接
func (h *Hub) Unsubscribe(userID int64, ch chan SSEEvent) {
	h.mu.Lock()
	if conns, ok := h.connections[userID]; ok {
		removed := conns.remove(ch)
		if len(conns.order) == 0 {
			delete(h.connections, userID)
		}
		if removed {
			close(ch)
		}
	}
	h.mu.Unlock()
}

// Broadcast 向用户的所有连接推送事件
func (h *Hub) Broadcast(userID int64, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns := h.connections[userID]
	if conns == nil {
		return
	}
	for _, ch := range conns.order {
		select {
		case ch <- event:
		default:
			metrics.SSEDroppedEventsTotal.Inc()
			logger.L().Warn("SSE channel full, dropping event",
				zap.Int64("user_id", userID),
				zap.String("event", event.Event),
			)
		}
	}
}

// StartRedisSubscriber 启动 Redis Pub/Sub 订阅
// 监听 notify:* 频道，收到消息后广播给对应用户的 SSE 连接
func (h *Hub) StartRedisSubscriber(ctx context.Context) {
	pubsub := h.rdb.PSubscribe(ctx, "notify:*")

	go func() {
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
			case <-h.stopCh:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var userID int64
				if _, err := fmt.Sscanf(msg.Channel, "notify:%d", &userID); err != nil {
					continue
				}
				h.Broadcast(userID, SSEEvent{
					Event: "notification",
					Data:  msg.Payload,
				})
			}
		}
	}()
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

func (u *userConnections) remove(ch chan SSEEvent) bool {
	for i, existing := range u.order {
		if existing != ch {
			continue
		}
		u.order = append(u.order[:i], u.order[i+1:]...)
		return true
	}
	return false
}
