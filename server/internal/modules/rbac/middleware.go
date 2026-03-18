package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appcapability "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Gin context 缓存键。RequireAnyPermission 会填充这些值，
// 使下游 RequirePermission 跳过重复 DB 查询。
const (
	ctxKeyInternalUserID = "rbac.internal_user_id"
	ctxKeyEffectivePerms = "rbac.effective_perms"
)

const maxPermissionBodyRead = 1 << 20 // 1MB，读取请求体最多 1MB 用于学校上下文提取

// PermissionService 定义权限中间件和其他需要授权检查的调用方（如评课访问
// 事实解析）所需的 RBAC 方法。
type PermissionService interface {
	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
	GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error)
	// CheckPermission 加载 effective permissions，查找命名权限并验证 scope。
	// 在没有缓存 effective permissions 时使用（如 admin 中间件链外部）。
	CheckPermission(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error)
	// CheckPermissionScope 验证已匹配的 effective permission 的 scope 约束。
	// 在 effective permissions 已缓存时使用（如 admin 中间件链内部）。
	CheckPermissionScope(ctx context.Context, ep EffectivePermission, userID int64, schoolID *string) (bool, error)
}

// ---------------------------------------------------------------------------
// 共享辅助函数 — 解析 + 缓存到 Gin context
// ---------------------------------------------------------------------------

// resolveInternalUserID 返回缓存的内部用户 ID，或解析后缓存并返回。
func resolveInternalUserID(c *gin.Context, service PermissionService) (int64, bool) {
	if v, ok := c.Get(ctxKeyInternalUserID); ok {
		return v.(int64), true //nolint:errcheck // type set by our middleware
	}

	externalID := middleware.GetUserID(c)
	if externalID == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		c.Abort()
		return 0, false
	}

	userID, err := service.GetInternalUserID(c.Request.Context(), externalID)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve user ID for permission check",
			zap.String("external_id", externalID),
			zap.Error(err),
		)
		response.InternalError(c, "permission check failed")
		c.Abort()
		return 0, false
	}

	c.Set(ctxKeyInternalUserID, userID)
	return userID, true
}

// resolveEffectivePermissions 返回缓存的生效权限列表，或加载后缓存并返回。
func resolveEffectivePermissions(c *gin.Context, service PermissionService, userID int64) ([]EffectivePermission, bool) {
	if v, ok := c.Get(ctxKeyEffectivePerms); ok {
		return v.([]EffectivePermission), true //nolint:errcheck // type set by our middleware
	}

	perms, err := service.GetEffectivePermissions(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to load effective permissions",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "permission check failed")
		c.Abort()
		return nil, false
	}

	c.Set(ctxKeyEffectivePerms, perms)
	return perms, true
}

// ---------------------------------------------------------------------------
// 中间件
// ---------------------------------------------------------------------------

// RequirePermission 检查当前用户是否持有指定权限，包含 scope 验证（school, role）。
func RequirePermission(service PermissionService, permName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := resolveInternalUserID(c, service)
		if !ok {
			return
		}

		perms, ok := resolveEffectivePermissions(c, service, userID)
		if !ok {
			return
		}

		// 在生效权限集中查找目标权限
		var found *EffectivePermission
		for i := range perms {
			if perms[i].Name == permName {
				found = &perms[i]
				break
			}
		}
		if found == nil || !found.Granted {
			response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
			c.Abort()
			return
		}

		// 验证 scope 约束（school 白名单、role 要求）
		schoolID, err := resolveSchoolID(c)
		if err != nil {
			logger.FromGin(c).Error("failed to resolve school context for permission check",
				zap.Int64("user_id", userID),
				zap.String("permission", permName),
				zap.Error(err),
			)
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}

		allowed, err := service.CheckPermissionScope(c.Request.Context(), *found, userID, schoolID)
		if err != nil {
			logger.FromGin(c).Error("permission scope check failed",
				zap.Int64("user_id", userID),
				zap.String("permission", permName),
				zap.Error(err),
			)
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}
		if !allowed {
			response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 路由组级纵深防御中间件，要求用户至少持有一个给定权限。
// 它会将解析出的内部用户 ID 和生效权限缓存到 Gin context，
// 使下游 RequirePermission 跳过重复 DB 查询。
func RequireAnyPermission(service PermissionService, permNames ...string) gin.HandlerFunc {
	normalized := appcapability.Normalize(permNames)

	return func(c *gin.Context) {
		if len(normalized) == 0 {
			logger.FromGin(c).Error("RequireAnyPermission called without permissions")
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}

		userID, ok := resolveInternalUserID(c, service)
		if !ok {
			return
		}

		perms, ok := resolveEffectivePermissions(c, service, userID)
		if !ok {
			return
		}

		if !hasAnyGrantedPermission(perms, normalized) {
			response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasAnyGrantedPermission 检查 names 中是否有至少一个名称在生效权限列表中被授予。
func hasAnyGrantedPermission(perms []EffectivePermission, names []string) bool {
	for i := range perms {
		if !perms[i].Granted {
			continue
		}
		for _, name := range names {
			if perms[i].Name == name {
				return true
			}
		}
	}
	return false
}

func resolveSchoolID(c *gin.Context) (*string, error) {
	if sid := resolveSchoolIDFromPath(c); sid != nil {
		return sid, nil
	}
	if sid := resolveSchoolIDFromQuery(c); sid != nil {
		return sid, nil
	}
	return resolveSchoolIDFromBody(c)
}

func resolveSchoolIDFromPath(c *gin.Context) *string {
	for _, param := range c.Params {
		if !isSchoolIDKey(param.Key) {
			continue
		}
		if value, ok := normalizeSchoolIDValue(param.Value); ok {
			return &value
		}
	}
	return nil
}

func resolveSchoolIDFromQuery(c *gin.Context) *string {
	keys := []string{"schoolID", "schoolId", "school_id"}
	for _, key := range keys {
		if value, ok := normalizeSchoolIDValue(c.Query(key)); ok {
			return &value
		}
	}
	return nil
}

func resolveSchoolIDFromBody(c *gin.Context) (*string, error) {
	if c.Request == nil || c.Request.Body == nil {
		return nil, nil
	}
	if !strings.Contains(strings.ToLower(c.ContentType()), "application/json") {
		return nil, nil
	}
	if c.Request.ContentLength < 0 || c.Request.ContentLength > maxPermissionBodyRead {
		return nil, nil
	}

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		// 请求体解析失败交给业务 handler；这里只做 best-effort 的上下文提取。
		return nil, nil
	}

	if value, ok := findTopLevelSchoolIDValue(payload); ok {
		return &value, nil
	}
	return nil, nil
}

func findTopLevelSchoolIDValue(payload map[string]any) (string, bool) {
	for _, key := range []string{"schoolID", "schoolId", "school_id"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		if normalized, ok := normalizeSchoolIDAny(raw); ok {
			return normalized, true
		}
	}
	return "", false
}

func isSchoolIDKey(key string) bool {
	switch key {
	case "schoolID", "schoolId", "school_id":
		return true
	default:
		return false
	}
}

func normalizeSchoolIDAny(v any) (string, bool) {
	switch value := v.(type) {
	case string:
		return normalizeSchoolIDValue(value)
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return "", false
		}
		if value == math.Trunc(value) {
			return strconv.FormatInt(int64(value), 10), true
		}
		return strconv.FormatFloat(value, 'f', -1, 64), true
	default:
		return "", false
	}
}

func normalizeSchoolIDValue(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}
