package authorization

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
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
