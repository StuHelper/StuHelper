package audit

import (
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// EventType 审计事件类型
type EventType string

const (
	// 认证相关
	EventUserLogin       EventType = "user.login"
	EventUserLoginFailed EventType = "user.login_failed"
	EventUserLogout      EventType = "user.logout"
	EventUserLogoutAll   EventType = "user.logout_all"
	EventTokenRefresh    EventType = "token.refresh"
	EventTokenRevoked    EventType = "token.revoked"

	// 通用数据操作
	EventDataAccess EventType = "data.access"
	EventDataCreate EventType = "data.create"
	EventDataUpdate EventType = "data.update"
	EventDataDelete EventType = "data.delete"
	EventDataExport EventType = "data.export"

	// 用户操作
	EventUserReviewPost   EventType = "user.review_post"
	EventUserReviewEdit   EventType = "user.review_edit"
	EventUserReviewDelete EventType = "user.review_delete"
	EventUserVote         EventType = "user.vote"
	EventUserReport       EventType = "user.report"
	EventUserReply        EventType = "user.reply"
	EventUserFavorite     EventType = "user.favorite"

	// 管理员操作
	EventAdminReviewHide    EventType = "admin.review_hide"
	EventAdminReviewDelete  EventType = "admin.review_delete"
	EventAdminReviewRestore EventType = "admin.review_restore"
	EventAdminReportResolve EventType = "admin.report_resolve"
	EventAdminConfigChange  EventType = "admin.config_change"
	EventAdminUserBan       EventType = "admin.user_ban"
	EventAdminBatchOp       EventType = "admin.batch_operation"

	// 系统操作
	EventSystemCronStart    EventType = "system.cron_start"
	EventSystemCronComplete EventType = "system.cron_complete"
	EventSystemCacheRefresh EventType = "system.cache_refresh"
	EventSystemStatsUpdate  EventType = "system.stats_update"
	EventSystemError        EventType = "system.error"
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
	Duration  time.Duration
	Details   map[string]any
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
		zap.String("ip", logger.MaskIP(e.IP)),
		zap.String("user_agent", e.UserAgent),
		zap.String("request_id", e.RequestID),
		zap.String("resource", e.Resource),
		zap.String("action", e.Action),
		zap.String("result", e.Result),
		zap.Time("timestamp", e.Timestamp),
	}

	if e.Duration > 0 {
		fields = append(fields, zap.Duration("duration", e.Duration))
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
		Details:   map[string]any{"reason": reason},
	})
}
