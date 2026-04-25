package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

type ErrorMapping struct {
	Match   func(error) bool
	Status  int
	Code    errs.ErrorCode
	Message string
}

func MatchError(target error, status int, message string, code ...errs.ErrorCode) ErrorMapping {
	mappedCode := defaultErrorCodeForStatus(status)
	if len(code) > 0 {
		mappedCode = code[0]
	}
	return ErrorMapping{
		Match: func(err error) bool {
			return errors.Is(err, target)
		},
		Status:  status,
		Code:    mappedCode,
		Message: message,
	}
}

func RespondMappedError(c *gin.Context, err error, mappings ...ErrorMapping) bool {
	for _, mapping := range mappings {
		if mapping.Match != nil && mapping.Match(err) {
			Error(c, mapping.Status, mapping.Code, mapping.Message)
			return true
		}
	}
	return false
}

func RespondMappedErrorGroups(c *gin.Context, err error, groups ...[]ErrorMapping) bool {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	if total == 0 {
		return false
	}

	mappings := make([]ErrorMapping, 0, total)
	for _, group := range groups {
		mappings = append(mappings, group...)
	}
	return RespondMappedError(c, err, mappings...)
}

func defaultErrorCodeForStatus(status int) errs.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return errs.ErrBadRequest
	case http.StatusUnauthorized:
		return errs.ErrLoginRequired
	case http.StatusForbidden:
		return errs.ErrForbidden
	case http.StatusNotFound:
		return errs.ErrNotFound
	case http.StatusConflict:
		return errs.ErrConflict
	case http.StatusTooManyRequests:
		return errs.ErrRateLimited
	case http.StatusServiceUnavailable:
		return errs.ErrServiceUnavailable
	default:
		return errs.ErrInternal
	}
}
