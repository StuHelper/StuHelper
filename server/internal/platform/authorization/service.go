package authorization

import (
	"context"
	"fmt"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

type Service struct {
	now             func() time.Time
	disableMFAGates bool
}

func NewService(opts ...ServiceOption) *Service {
	return newServiceWithClock(time.Now, opts...)
}

func newServiceWithClock(now func() time.Time, opts ...ServiceOption) *Service {
	service := &Service{now: now}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(service)
	}
	return service
}

func (s *Service) Authorize(ctx context.Context, subject Subject, action Action, resource Resource) Decision {
	if err := ctx.Err(); err != nil {
		return deny("request context is closed", err)
	}
	if subject.UserID == "" {
		return deny("subject user is missing", ErrInvalidSubject)
	}

	switch action {
	case ActionCapabilityRequire:
		return authorizeCapability(subject, resource)
	case ActionCapabilityRequireAny:
		return authorizeAnyCapability(subject, resource)
	case ActionCapabilityRequireGlobal:
		return authorizeGlobalCapability(subject, resource)
	case ActionPrivilegedMFARequire:
		return s.authorizePrivilegedMFA(subject, resource)
	case ActionStepUpMFARequire:
		return s.authorizeMFA(subject, resource)
	default:
		return deny(fmt.Sprintf("unsupported action %q", action), ErrUnsupportedAction)
	}
}

func (s *Service) BatchAuthorize(ctx context.Context, subject Subject, checks []Check) []Decision {
	decisions := make([]Decision, len(checks))
	for i, check := range checks {
		decisions[i] = s.Authorize(ctx, subject, check.Action, check.Resource)
	}
	return decisions
}

func authorizeCapability(subject Subject, resource Resource) Decision {
	if err := validateCapabilityResource(resource); err != nil {
		return deny("capability resource is invalid", err)
	}
	if capability.Has(subject.Capabilities, resource.ID) {
		return allow("capability granted")
	}
	return deny("capability denied", nil)
}

func authorizeAnyCapability(subject Subject, resource Resource) Decision {
	required := capability.Normalize(resource.RequiredCapabilities)
	if len(required) == 0 {
		return deny("capability resource is invalid", ErrInvalidResource)
	}
	if capability.HasAny(subject.Capabilities, required...) {
		return allow("one required capability granted")
	}
	return deny("capability denied", nil)
}

func authorizeGlobalCapability(subject Subject, resource Resource) Decision {
	if err := validateCapabilityResource(resource); err != nil {
		return deny("capability resource is invalid", err)
	}
	if capability.HasGlobalGrant(subject.CapabilityGrants, resource.ID) {
		return allow("global capability granted")
	}
	return deny("global capability denied", nil)
}

func validateCapabilityResource(resource Resource) error {
	if resource.Type != ResourceCapability || resource.ID == "" {
		return ErrInvalidResource
	}
	return nil
}

func (s *Service) authorizePrivilegedMFA(subject Subject, resource Resource) Decision {
	if s.disableMFAGates {
		return allow("mfa enforcement disabled")
	}
	if resource.Type != ResourceMFA {
		return deny("mfa resource is invalid", ErrInvalidResource)
	}
	if !hasPrivilegedRole(subject.Roles) {
		return allow("privileged mfa not required")
	}
	return s.authorizeMFA(subject, resource)
}

func (s *Service) authorizeMFA(subject Subject, resource Resource) Decision {
	if s.disableMFAGates {
		return allow("mfa enforcement disabled")
	}
	if resource.Type != ResourceMFA {
		return deny("mfa resource is invalid", ErrInvalidResource)
	}
	if !subject.MFAEnrollmentActive {
		return deny("mfa enrollment required", ErrMFAEnrollmentRequired)
	}
	if !mfaProofFresh(s.now(), subject.MFAProofVerifiedAt, stepUpWindow(resource)) {
		return deny("step-up required", ErrStepUpRequired)
	}
	return allow("mfa proof accepted")
}

func hasPrivilegedRole(roles []string) bool {
	for _, role := range roles {
		if role == "super_admin" || role == "school_admin" {
			return true
		}
	}
	return false
}

func stepUpWindow(resource Resource) time.Duration {
	if resource.StepUpWindow > 0 {
		return resource.StepUpWindow
	}
	return DefaultStepUpWindow
}

func mfaProofFresh(now, verifiedAt time.Time, window time.Duration) bool {
	if verifiedAt.IsZero() {
		return false
	}
	if verifiedAt.After(now) {
		return false
	}
	return !verifiedAt.Before(now.Add(-window))
}
