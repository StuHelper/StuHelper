package httputil

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

// 分页默认值
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// ParsePage 解析分页参数
func ParsePage(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.Query("page_size"))
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
