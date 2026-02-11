package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
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

// 旧版错误码别名 (向后兼容，将在 v2.0 移除)
// Deprecated: 请直接使用 errs 包中的错误码
const (
	ErrCodeBadRequest     = errs.ErrBadRequest       // A00400
	ErrCodeUnauthorized   = errs.ErrLoginRequired    // A01010
	ErrCodeForbidden      = errs.ErrForbidden        // A01020
	ErrCodeNotFound       = errs.ErrNotFound         // A00404
	ErrCodeConflict       = errs.ErrConflict         // A00409
	ErrCodeInternal       = errs.ErrInternal         // B00001
	ErrCodeValidation     = errs.ErrValidation       // A00422
	ErrCodeRateLimit      = errs.ErrRateLimited      // A00429
	ErrCodeServiceUnavail = errs.ErrServiceUnavailable // B00004
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

