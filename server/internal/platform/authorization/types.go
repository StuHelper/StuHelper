// Package authorization provides StuHelper's single business authorization entry.
package authorization

import (
	"context"
	"errors"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

type Action string

const (
	ActionCapabilityRequire       Action = "capability.require"
	ActionCapabilityRequireAny    Action = "capability.require_any"
	ActionCapabilityRequireGlobal Action = "capability.require_global"
	ActionPrivilegedMFARequire    Action = "mfa.privileged.require"
	ActionStepUpMFARequire        Action = "mfa.step_up.require"
)

const ResourceCapability = "capability"
const ResourceMFA = "mfa"

const DefaultStepUpWindow = 5 * time.Minute

var (
	ErrInvalidSubject        = errors.New("authorization: invalid subject")
	ErrInvalidResource       = errors.New("authorization: invalid resource")
	ErrUnsupportedAction     = errors.New("authorization: unsupported action")
	ErrMFAEnrollmentRequired = errors.New("authorization: mfa enrollment required")
	ErrStepUpRequired        = errors.New("authorization: step-up required")
)

type AuthorizationService interface {
	Authorize(ctx context.Context, subject Subject, action Action, resource Resource) Decision
	BatchAuthorize(ctx context.Context, subject Subject, checks []Check) []Decision
}

type Subject struct {
	UserID              string
	AppID               string
	Roles               []string
	Capabilities        []string
	CapabilityGrants    []capability.Grant
	GlobalCapabilities  []string
	MFAEnrollmentActive bool
	MFAProofVerifiedAt  time.Time
}

type Resource struct {
	Type                 string
	ID                   string
	RequiredCapabilities []string
	SchoolID             string
	StepUpWindow         time.Duration
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

func PrivilegedMFAResource(window time.Duration) Resource {
	return Resource{Type: ResourceMFA, StepUpWindow: window}
}

func StepUpMFAResource(window time.Duration) Resource {
	return Resource{Type: ResourceMFA, StepUpWindow: window}
}

func allow(reason string) Decision {
	return Decision{Allow: true, Reason: reason}
}

func deny(reason string, err error) Decision {
	return Decision{Allow: false, Reason: reason, Error: err}
}
