package httputil

import (
	"fmt"
	"math"
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

// ParsePage 解析分页参数。
// 无效或缺失的查询参数会被静默钳位到合法范围：
//   - page: [1, MaxPage]，非法值（非数字/负数/零）默认为 1
//   - pageSize: [1, MaxPageSize]，非法值默认为 DefaultPageSize
//
// 这是有意为之的设计：分页参数来自用户输入，返回错误会降低 API 易用性，
// 静默钳位更符合 "宽容输入、严格输出" 原则。
func ParsePage(c *gin.Context) (page, pageSize int) {
	// strconv.Atoi 错误被有意忽略：解析失败时 page=0，下方钳位逻辑会将其修正为 1
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
	// strconv.Atoi 错误被有意忽略：解析失败时 pageSize=0，下方钳位逻辑会将其修正为 DefaultPageSize
	pageSize, _ = strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// ClampPageSize 将 pageSize 限制在 [1, MaxPageSize] 范围内（纵深防御：即使 handler 层已校验）
func ClampPageSize(pageSize int) int {
	if pageSize < 1 {
		return DefaultPageSize
	}
	if pageSize > MaxPageSize {
		return MaxPageSize
	}
	return pageSize
}

// SafeOffset 计算安全的分页偏移量，确保 page 和 pageSize 在合法范围内。
// 同时检查 page 上界和乘法溢出，防止极端值导致的性能问题或整数溢出。
func SafeOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if page > MaxPage {
		page = MaxPage
	}
	pageSize = ClampPageSize(pageSize)
	// 溢出保护：(page-1) * pageSize 可能在极端值下溢出 int
	offset := page - 1
	if offset > 0 && pageSize > math.MaxInt/offset {
		return MaxPage * MaxPageSize // 安全上界
	}
	return offset * pageSize
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
func HashUserID(userID string) (string, error) {
	return crypto.HMACHash(userID)
}

// EscapeLikePattern 转义 LIKE 查询中的特殊字符
func EscapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// SanitizeCacheKey 清理缓存 key 中的特殊字符，防止缓存 key 注入。
// 失败时返回原始输入的截断值作为降级，避免空字符串导致缓存碰撞。
func SanitizeCacheKey(s string) string {
	hash, err := crypto.HMACHashShort(s, 16)
	if err != nil {
		safe := EscapeLikePattern(s)
		if len(safe) > 64 {
			safe = safe[:64]
		}
		return safe
	}
	return hash
}

// maxUserAgentLen UserAgent 字段最大长度，超出截断
const maxUserAgentLen = 256

// TruncateUserAgent 截断 User-Agent 字符串到安全长度（按 rune 截断，避免破坏 UTF-8）。
// 调用方应通过 c.GetHeader("User-Agent") 获取原始值后传入。
func TruncateUserAgent(ua string) string {
	if len(ua) <= maxUserAgentLen {
		return ua
	}
	runes := []rune(ua)
	if len(runes) > maxUserAgentLen {
		runes = runes[:maxUserAgentLen]
	}
	return string(runes)
}
