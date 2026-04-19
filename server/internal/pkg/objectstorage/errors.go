package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/aws/smithy-go"
)

type ErrorKind string

const (
	ErrorKindConfig         ErrorKind = "config"
	ErrorKindAuthentication ErrorKind = "authentication"
	ErrorKindPermission     ErrorKind = "permission"
	ErrorKindNotFound       ErrorKind = "not_found"
	ErrorKindNetwork        ErrorKind = "network"
	ErrorKindUnknown        ErrorKind = "unknown"
)

type StoreError struct {
	Kind     ErrorKind
	Op       string
	Resource string
	Err      error
}

func (e *StoreError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s %s (%s): %v", e.Op, e.Resource, e.Kind, e.Err)
}

func (e *StoreError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsKind(err error, kind ErrorKind) bool {
	var target *StoreError
	return errors.As(err, &target) && target.Kind == kind
}

func wrapError(op, resource string, err error) error {
	if err == nil {
		return nil
	}
	return &StoreError{
		Kind:     classifyErrorKind(err),
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

func classifyErrorKind(err error) ErrorKind {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ErrorKindNetwork
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorKindNetwork
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := strings.ToLower(strings.TrimSpace(apiErr.ErrorCode()))
		switch {
		case strings.Contains(code, "nosuchbucket"),
			strings.Contains(code, "nosuchkey"),
			strings.Contains(code, "notfound"):
			return ErrorKindNotFound
		case strings.Contains(code, "accessdenied"),
			strings.Contains(code, "forbidden"):
			return ErrorKindPermission
		case strings.Contains(code, "signaturedoesnotmatch"),
			strings.Contains(code, "invalidaccesskeyid"),
			strings.Contains(code, "auth"),
			strings.Contains(code, "token"):
			return ErrorKindAuthentication
		}
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "no such host"),
		strings.Contains(message, "connection refused"),
		strings.Contains(message, "timeout"):
		return ErrorKindNetwork
	case strings.Contains(message, "not found"),
		strings.Contains(message, "no such bucket"),
		strings.Contains(message, "no such key"):
		return ErrorKindNotFound
	case strings.Contains(message, "access denied"),
		strings.Contains(message, "accessdenied"),
		strings.Contains(message, "forbidden"):
		return ErrorKindPermission
	case strings.Contains(message, "credentials"),
		strings.Contains(message, "signaturedoesnotmatch"),
		strings.Contains(message, "signature"),
		strings.Contains(message, "access key"):
		return ErrorKindAuthentication
	default:
		return ErrorKindUnknown
	}
}
