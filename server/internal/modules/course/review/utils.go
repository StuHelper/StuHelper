package review

import (
	"time"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

const (
	defaultPageSize = httputil.DefaultPageSize
	maxPageSize     = httputil.MaxPageSize
	cacheTTL        = 5 * time.Minute
)

// parsePage 解析分页参数（使用 httputil 包）
func parsePage(c *gin.Context) (int, int) {
	return httputil.ParsePage(c)
}

// parseIDParam 解析路径参数中的 ID（使用 httputil 包）
func parseIDParam(c *gin.Context, name string) (int64, error) {
	return httputil.ParseIDParam(c, name)
}

// hashUserID 使用 HMAC-SHA256 对用户 ID 进行哈希（使用 httputil 包）
func hashUserID(userID string) string {
	return httputil.HashUserID(userID)
}
