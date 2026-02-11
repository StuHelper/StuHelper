package httputil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

// 分页默认值
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxPage         = 1000 // 防止过大的 OFFSET 导致性能问题
)

// ParsePage 解析分页参数
func ParsePage(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	if page > MaxPage {
		page = MaxPage
	}
	pageSizeStr := c.Query("page_size")
	if pageSizeStr == "" {
		pageSizeStr = c.Query("pageSize")
	}
	pageSize, _ = strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// ParseIDParam 解析路径参数中的 ID
func ParseIDParam(c *gin.Context, name string) (int64, error) {
	idStr := c.Param(name)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, strconv.ErrRange
	}
	return id, nil
}

// ParseUUIDParam 解析路径参数中的 UUID
func ParseUUIDParam(c *gin.Context, name string) (string, error) {
	raw := c.Param(name)
	if _, err := uuid.Parse(raw); err != nil {
		return "", fmt.Errorf("invalid UUID parameter %q: %w", name, err)
	}
	return raw, nil
}

// HashUserID 使用 HMAC-SHA256 对用户 ID 进行哈希，防止枚举和关联攻击
func HashUserID(userID string) string {
	return crypto.HMACHash(userID)
}

// EscapeLikePattern 转义 LIKE 查询中的特殊字符
func EscapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// SanitizeCacheKey 清理缓存 key 中的特殊字符，防止缓存 key 注入
func SanitizeCacheKey(s string) string {
	return crypto.HMACHashShort(s, 16)
}
