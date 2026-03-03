package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

// APIError 统一错误响应结构
//
// 安全警告：Details 字段类型为 any，会直接序列化到 JSON 响应中。
// 调用方必须确保 Details 不包含敏感信息（如内部错误堆栈、SQL 语句、用户隐私数据等）。
// 生产环境中建议仅传入验证错误摘要等脱敏信息，避免信息泄露。
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

// Error 返回错误响应并中止后续 handler 链（AbortWithStatusJSON）
func Error(c *gin.Context, status int, code errs.ErrorCode, message string) {
	c.AbortWithStatusJSON(status, Response{
		Success: false,
		Error: &APIError{
			Code:    string(code),
			Message: message,
		},
	})
}

// ErrorWithDetails 返回带详情的错误响应并中止后续 handler 链。
//
// 安全注意：details 参数会直接序列化到响应 JSON 中，调用方必须确保不传入
// 内部错误信息、堆栈跟踪、SQL 语句等敏感数据。仅应传入面向用户的验证错误摘要。
func ErrorWithDetails(c *gin.Context, status int, code errs.ErrorCode, message string, details any) {
	c.AbortWithStatusJSON(status, Response{
		Success: false,
		Error: &APIError{
			Code:    string(code),
			Message: message,
			Details: details,
		},
	})
}

// BadRequest 返回 400 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrBadRequest
func BadRequest(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrBadRequest
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusBadRequest, ec, message)
}

// ValidationError 返回验证错误
func ValidationError(c *gin.Context, message string, details any) {
	ErrorWithDetails(c, http.StatusBadRequest, errs.ErrValidation, message, details)
}

// Unauthorized 返回 401 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrLoginRequired
func Unauthorized(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrLoginRequired
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusUnauthorized, ec, message)
}

// Forbidden 返回 403 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrForbidden
func Forbidden(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrForbidden
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusForbidden, ec, message)
}

// NotFound 返回 404 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrNotFound
func NotFound(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrNotFound
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusNotFound, ec, message)
}

// Conflict 返回 409 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrConflict
func Conflict(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrConflict
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusConflict, ec, message)
}

// InternalError 返回 500 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrInternal
func InternalError(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrInternal
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusInternalServerError, ec, message)
}

// RateLimitExceeded 返回 429 错误
func RateLimitExceeded(c *gin.Context, message ...string) {
	msg := "rate limit exceeded"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(c, http.StatusTooManyRequests, errs.ErrRateLimited, msg)
}

// ServiceUnavailable 返回 503 错误
// 可选的 code 参数允许传入具体业务错误码，不传则使用通用码 errs.ErrServiceUnavailable
func ServiceUnavailable(c *gin.Context, message string, code ...errs.ErrorCode) {
	ec := errs.ErrServiceUnavailable
	if len(code) > 0 {
		ec = code[0]
	}
	Error(c, http.StatusServiceUnavailable, ec, message)
}

