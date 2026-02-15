package course

import (
	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

const (
	defaultPageSize = httputil.DefaultPageSize
	maxPageSize     = httputil.MaxPageSize
	maxSearchLength = 100
)

// parsePage 解析分页参数
func parsePage(c *gin.Context) (int, int) {
	return httputil.ParsePage(c)
}

// parseIDParam 解析路径参数中的 ID
func parseIDParam(c *gin.Context, name string) (int64, error) {
	return httputil.ParseIDParam(c, name)
}

// hashUserID 使用 HMAC-SHA256 对用户 ID 进行哈希
func hashUserID(userID string) (string, error) {
	return httputil.HashUserID(userID)
}

// escapeLikePattern 转义 LIKE 查询中的特殊字符
func escapeLikePattern(s string) string {
	return httputil.EscapeLikePattern(s)
}

// sanitizeCacheKey 清理缓存 key 中的特殊字符
// 失败时返回原始输入的截断值作为降级，避免空字符串导致缓存碰撞
func sanitizeCacheKey(s string) string {
	hash, err := crypto.HMACHashShort(s, 16)
	if err != nil {
		// 降级：截断原始输入，仅保留安全字符
		safe := httputil.EscapeLikePattern(s)
		if len(safe) > 64 {
			safe = safe[:64]
		}
		return safe
	}
	return hash
}
