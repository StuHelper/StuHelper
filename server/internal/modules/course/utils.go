package course

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxSearchLength = 100
)

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	idStr := c.Param(name)
	return strconv.ParseInt(idStr, 10, 64)
}

// hashUserID 使用 HMAC-SHA256 对用户 ID 进行哈希，防止枚举和关联攻击
func hashUserID(userID string) string {
	return crypto.HMACHash(userID)
}

// escapeLikePattern 转义 LIKE 查询中的特殊字符
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// sanitizeCacheKey 清理缓存 key 中的特殊字符，防止缓存 key 注入
func sanitizeCacheKey(s string) string {
	return crypto.HMACHashShort(s, 16)
}
