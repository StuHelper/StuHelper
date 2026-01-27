package audit

import (
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// EventType 审计事件类型
type EventType string

const (
	EventUserLogin       EventType = "user.login"
	EventUserLoginFailed EventType = "user.login_failed"
	EventUserLogout      EventType = "user.logout"
	EventUserLogoutAll   EventType = "user.logout_all"
	EventTokenRefresh    EventType = "token.refresh"
	EventTokenRevoked    EventType = "token.revoked"
	EventDataAccess      EventType = "data.access"
	EventDataCreate      EventType = "data.create"
	EventDataUpdate      EventType = "data.update"
	EventDataDelete      EventType = "data.delete"
)

// Event 审计事件
type Event struct {
	Type      EventType
	UserID    string
	Username  string
	IP        string
	UserAgent string
	RequestID string
	Resource  string
	Action    string
	Result    string
	Details   map[string]interface{}
	Timestamp time.Time
}

// Log 记录审计日志
func Log(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	fields := []zap.Field{
		zap.String("audit_type", string(e.Type)),
		zap.String("user_id", e.UserID),
		zap.String("username", logger.MaskSensitiveData(e.Username)),
		zap.String("ip", e.IP),
		zap.String("user_agent", e.UserAgent),
		zap.String("request_id", e.RequestID),
		zap.String("resource", e.Resource),
		zap.String("action", e.Action),
		zap.String("result", e.Result),
		zap.Time("timestamp", e.Timestamp),
	}

	if len(e.Details) > 0 {
		fields = append(fields, zap.Any("details", e.Details))
	}

	logger.L().Info("audit_log", fields...)
}

// LogSuccess 记录成功的审计事件
func LogSuccess(eventType EventType, userID, username, ip, userAgent, requestID string) {
	Log(Event{
		Type:      eventType,
		UserID:    userID,
		Username:  username,
		IP:        ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Result:    "success",
	})
}

// LogFailure 记录失败的审计事件
func LogFailure(eventType EventType, ip, userAgent, requestID, reason string) {
	Log(Event{
		Type:      eventType,
		IP:        ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Result:    "failure",
		Details:   map[string]interface{}{"reason": reason},
	})
}
