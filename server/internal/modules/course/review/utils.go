package review

import (
	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

const (
	defaultPageSize = httputil.DefaultPageSize
	maxPageSize     = httputil.MaxPageSize
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
func hashUserID(userID string) string {
	return httputil.HashUserID(userID)
}
