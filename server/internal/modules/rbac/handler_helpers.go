package rbac

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// resolveUserID 从路由参数解析内部用户 ID（参数值为内部 BIGINT user ID）
func (h *Handler) resolveUserID(c *gin.Context, paramName string) (int64, error) {
	userID, err := strconv.ParseInt(c.Param(paramName), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user ID")
		return 0, fmt.Errorf("invalid user ID param: %s", c.Param(paramName))
	}
	return userID, nil
}
