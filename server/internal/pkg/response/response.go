package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError 统一错误响应结构
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Response 统一响应结构
type Response struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// 预定义错误码 (旧版，已废弃)
// Deprecated: 请使用 errs 包中的新版错误码，旧版将在 v2.0 移除
// 详细说明见 docs/api/error-codes.md
const (
	ErrCodeBadRequest     = "BAD_REQUEST"         // -> errs.ErrBadRequest (A00400)
	ErrCodeUnauthorized   = "UNAUTHORIZED"        // -> errs.ErrLoginRequired (A01010)
	ErrCodeForbidden      = "FORBIDDEN"           // -> errs.ErrForbidden (A01020)
	ErrCodeNotFound       = "NOT_FOUND"           // -> errs.ErrNotFound (A00404)
	ErrCodeConflict       = "CONFLICT"            // -> errs.ErrConflict (A00409)
	ErrCodeInternal       = "INTERNAL_ERROR"      // -> errs.ErrInternal (B00001)
	ErrCodeValidation     = "VALIDATION_ERROR"    // -> errs.ErrValidation (A00422)
	ErrCodeRateLimit      = "RATE_LIMIT_EXCEEDED" // -> errs.ErrRateLimited (A00429)
	ErrCodeServiceUnavail = "SERVICE_UNAVAILABLE" // -> errs.ErrServiceUnavailable (B00004)
)

// Success 返回成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created 返回创建成功响应
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// Error 返回错误响应
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails 返回带详情的错误响应
func ErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest 返回 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, ErrCodeBadRequest, message)
}

// ValidationError 返回验证错误
func ValidationError(c *gin.Context, message string, details any) {
	ErrorWithDetails(c, http.StatusBadRequest, ErrCodeValidation, message, details)
}

// Unauthorized 返回 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// Forbidden 返回 403 错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, ErrCodeForbidden, message)
}

// NotFound 返回 404 错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// Conflict 返回 409 错误
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, ErrCodeConflict, message)
}

// InternalError 返回 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, ErrCodeInternal, message)
}

// RateLimitExceeded 返回 429 错误
func RateLimitExceeded(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, ErrCodeRateLimit, "rate limit exceeded")
}

// ServiceUnavailable 返回 503 错误
func ServiceUnavailable(c *gin.Context, message string) {
	Error(c, http.StatusServiceUnavailable, ErrCodeServiceUnavail, message)
}

// PageMeta 分页元数据
type PageMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// PagedResponse 分页响应结构
type PagedResponse struct {
	Success bool     `json:"success"`
	Data    any      `json:"data"`
	Meta    PageMeta `json:"meta"`
}

// Paginated 返回分页响应
func Paginated(c *gin.Context, data any, total, page, pageSize int) {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, PagedResponse{
		Success: true,
		Data:    data,
		Meta: PageMeta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}
