package audit

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const defaultAdminOperationRetentionDays = 90

// EventType 审计事件类型
type EventType string

const (
	// 认证相关
	EventUserLogin       EventType = "iam.auth.login"
	EventUserLoginFailed EventType = "iam.auth.login_failed"
	EventUserLogout      EventType = "iam.auth.logout"
	EventUserLogoutAll   EventType = "iam.auth.logout_all"
	EventTokenRefresh    EventType = "iam.token.refresh" // #nosec G101 -- audit event type, not a secret.
	EventTokenRevoked    EventType = "iam.token.revoked" // #nosec G101 -- audit event type, not a secret.

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

// Event 审计事件统一 envelope。
type Event struct {
	Type          EventType
	Category      string
	ActorType     string
	UserID        string
	Username      string
	IP            string
	UserAgent     string
	RequestID     string
	TraceID       string
	Resource      string
	ResourceType  string
	ResourceID    string
	ScopeSchoolID string
	Action        string
	Result        string
	Reason        string
	Before        any
	After         any
	Duration      time.Duration
	Details       map[string]any
	Timestamp     time.Time
}

func EventFromContext(ctx context.Context, event Event) Event {
	return normalizeEvent(eventWithContextFields(ctx, event))
}

func eventWithContextFields(ctx context.Context, event Event) Event {
	if ctx == nil {
		return event
	}
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() && event.TraceID == "" {
		event.TraceID = sc.TraceID().String()
	}
	if event.RequestID == "" {
		event.RequestID = logger.RequestIDFromContext(ctx)
	}
	return event
}

// Log 记录审计日志，并在配置仓储后持久化到 audit_events。
func Log(e Event) {
	LogContext(context.Background(), e)
}

// LogContext 记录带上下文的审计日志。
//
// 事件字段会从 ctx 补齐 request_id / trace_id；持久化使用 WithoutCancel 派生上下文，
// 保留 trace baggage 和 request-scoped values，同时避免客户端断开取消安全审计写入。
func LogContext(ctx context.Context, e Event) {
	if ctx == nil {
		ctx = context.Background()
	}
	e = EventFromContext(ctx, e)

	fields := []zap.Field{
		zap.String("audit_type", string(e.Type)),
		zap.String("audit_category", e.Category),
		zap.String("actor_type", e.ActorType),
		zap.String("user_id", e.UserID),
		zap.String("username", logger.MaskSensitiveData(e.Username)),
		zap.String("ip", logger.MaskIP(e.IP)),
		zap.String("user_agent", e.UserAgent),
		zap.String("request_id", e.RequestID),
		zap.String("trace_id", e.TraceID),
		zap.String("resource_type", e.ResourceType),
		zap.String("resource_id", e.ResourceID),
		zap.String("action", e.Action),
		zap.String("result", e.Result),
		zap.String("reason", e.Reason),
		zap.Time("timestamp", e.Timestamp),
	}

	if e.ScopeSchoolID != "" {
		fields = append(fields, zap.String("scope_school_id", e.ScopeSchoolID))
	}
	if e.Duration > 0 {
		fields = append(fields, zap.Duration("duration", e.Duration))
	}
	if e.Before != nil {
		fields = append(fields, zap.Any("before", e.Before))
	}
	if e.After != nil {
		fields = append(fields, zap.Any("after", e.After))
	}
	if len(e.Details) > 0 {
		fields = append(fields, zap.Any("details", e.Details))
	}

	logger.L().Info("audit_log", fields...)

	repo := configuredRepository()
	if repo == nil {
		return
	}
	if err := repo.WriteEvent(auditWriteContext(ctx), e); err != nil {
		logger.L().Warn("persist audit event failed", zap.Error(err))
	}
}

func auditWriteContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

// LogSuccess 记录成功的审计事件
func LogSuccess(eventType EventType, userID, username, ip, userAgent, requestID string) {
	LogSuccessContext(context.Background(), eventType, userID, username, ip, userAgent, requestID)
}

// LogSuccessContext 记录带上下文的成功审计事件。
func LogSuccessContext(ctx context.Context, eventType EventType, userID, username, ip, userAgent, requestID string) {
	LogContext(ctx, Event{
		Type:         eventType,
		ActorType:    "user",
		UserID:       userID,
		Username:     username,
		IP:           ip,
		UserAgent:    userAgent,
		RequestID:    requestID,
		ResourceType: defaultResourceType(eventType),
		Action:       defaultAction(eventType),
		Result:       "success",
	})
}

// LogFailure 记录失败的审计事件
func LogFailure(eventType EventType, ip, userAgent, requestID, reason string) {
	LogFailureContext(context.Background(), eventType, ip, userAgent, requestID, reason)
}

// LogFailureContext 记录带上下文的失败审计事件。
func LogFailureContext(ctx context.Context, eventType EventType, ip, userAgent, requestID, reason string) {
	LogContext(ctx, Event{
		Type:         eventType,
		ActorType:    "user",
		IP:           ip,
		UserAgent:    userAgent,
		RequestID:    requestID,
		ResourceType: defaultResourceType(eventType),
		Action:       defaultAction(eventType),
		Result:       "failure",
		Reason:       reason,
		Details:      map[string]any{"reason": reason},
	})
}

func defaultResourceType(eventType EventType) string {
	switch eventType {
	case EventUserLogin, EventUserLoginFailed, EventUserLogout, EventUserLogoutAll:
		return "auth.session"
	case EventTokenRefresh, EventTokenRevoked:
		return "auth.token"
	default:
		return "system"
	}
}

func defaultAction(eventType EventType) string {
	switch eventType {
	case EventUserLogin, EventUserLoginFailed:
		return "login"
	case EventUserLogout:
		return "logout"
	case EventUserLogoutAll:
		return "logout_all"
	case EventTokenRefresh:
		return "refresh"
	case EventTokenRevoked:
		return "revoke"
	default:
		return "unknown"
	}
}
