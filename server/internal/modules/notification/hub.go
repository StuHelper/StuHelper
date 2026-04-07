package notification

import (
	"context"
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
	SourceModule string
	SourceID     string
	SourceURL    string
	CourseID     int64 // 关联课程 ID，用于前端精准跳转；0 表示无关联
}

// Notification 通知实体
type Notification struct {
	ID           string  `json:"id"`
	UserID       int64   `json:"userId"`
	Type         string  `json:"type"`
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	SourceModule string  `json:"sourceModule"`
	SourceID     string  `json:"sourceId"`
	SourceURL    *string `json:"sourceUrl,omitempty"`
	CourseID     *int64  `json:"courseID,omitempty"`
	IsRead       bool    `json:"isRead"`
	CreatedAt    string  `json:"createdAt"`
}

// Hub 管理 SSE 连接和 Redis Pub/Sub
type Hub struct {
	rdb         *redis.Client
	mu          sync.RWMutex
	connections map[int64]map[chan SSEEvent]struct{} // userID -> set of channels
	stopCh      chan struct{}
	stopOnce    sync.Once
}

// SSEEvent SSE 推送事件
type SSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// maxConnsPerUser 每个用户允许的最大 SSE 并发连接数。
// 超出限制时关闭最早建立的连接，防止资源耗尽。
const maxConnsPerUser = 5

// NewHub 创建通知 Hub
func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:         rdb,
		connections: make(map[int64]map[chan SSEEvent]struct{}),
		stopCh:      make(chan struct{}),
	}
}

// Subscribe 注册 SSE 连接。
// 当同一用户的连接数达到 maxConnsPerUser 时，驱逐一个现有连接以腾出名额。
func (h *Hub) Subscribe(userID int64) chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	h.mu.Lock()
	if h.connections[userID] == nil {
		h.connections[userID] = make(map[chan SSEEvent]struct{})
	}
	if len(h.connections[userID]) >= maxConnsPerUser {
		// 驱逐一个现有连接（map 迭代顺序不确定，等效于随机驱逐）
		for victim := range h.connections[userID] {
			delete(h.connections[userID], victim)
			close(victim)
			break
		}
	}
	h.connections[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 注销 SSE 连接
func (h *Hub) Unsubscribe(userID int64, ch chan SSEEvent) {
	h.mu.Lock()
	if conns, ok := h.connections[userID]; ok {
		delete(conns, ch)
		if len(conns) == 0 {
			delete(h.connections, userID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

// Broadcast 向用户的所有连接推送事件
func (h *Hub) Broadcast(userID int64, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.connections[userID] {
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
		defer pubsub.Close()
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
				// 频道格式: notify:{userID}
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
