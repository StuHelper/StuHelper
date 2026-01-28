package review

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	cacheTTL        = 5 * time.Minute
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
