// Package authorization provides StuHelper's single business authorization entry.
package authorization

import (
	"context"
	"errors"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

type Action string

const (
	ActionCapabilityRequire       Action = "capability.require"
	ActionCapabilityRequireAny    Action = "capability.require_any"
	ActionCapabilityRequireGlobal Action = "capability.require_global"
)

const ResourceCapability = "capability"

var (
	ErrInvalidSubject    = errors.New("authorization: invalid subject")
	ErrInvalidResource   = errors.New("authorization: invalid resource")
	ErrUnsupportedAction = errors.New("authorization: unsupported action")
)

type AuthorizationService interface {
	Authorize(ctx context.Context, subject Subject, action Action, resource Resource) Decision
	BatchAuthorize(ctx context.Context, subject Subject, checks []Check) []Decision
}

type Subject struct {
	UserID             string
	AppID              string
	Roles              []string
	Capabilities       []string
	CapabilityGrants   []capability.Grant
	GlobalCapabilities []string
}

type Resource struct {
	Type                 string
	ID                   string
	RequiredCapabilities []string
	SchoolID             string
}

type Check struct {
	Action   Action
	Resource Resource
}

type Decision struct {
	Allow  bool
	Reason string
	Error  error
}

func CapabilityResource(capabilityName string) Resource {
	return Resource{Type: ResourceCapability, ID: capabilityName}
}

func AnyCapabilityResource(capabilities ...string) Resource {
	copied := append([]string(nil), capabilities...)
	return Resource{Type: ResourceCapability, RequiredCapabilities: copied}
}

func GlobalCapabilityResource(capabilityName string) Resource {
	return Resource{Type: ResourceCapability, ID: capabilityName}
}

func allow(reason string) Decision {
	return Decision{Allow: true, Reason: reason}
}

func deny(reason string, err error) Decision {
	return Decision{Allow: false, Reason: reason, Error: err}
}
