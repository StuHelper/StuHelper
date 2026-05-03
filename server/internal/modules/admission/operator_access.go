package admission

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

type OperatorRoleMembershipFunc func(ctx context.Context, userID int64, role string) (bool, error)

type RoleOperatorAccessGateway struct {
	hasRole OperatorRoleMembershipFunc
}

func NewRoleOperatorAccessGateway(hasRole OperatorRoleMembershipFunc) *RoleOperatorAccessGateway {
	return &RoleOperatorAccessGateway{hasRole: hasRole}
}

func (g *RoleOperatorAccessGateway) UserHasCapability(
	ctx context.Context,
	userID int64,
	capabilityName string,
) (bool, error) {
	if g == nil || g.hasRole == nil {
		return false, ErrAdmissionOperatorAccessUnavailable
	}
	for _, role := range rolesForCapability(capabilityName) {
		allowed, err := g.hasRole(ctx, userID, role)
		if err != nil {
			return false, fmt.Errorf("operator role membership: %w", err)
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func rolesForCapability(capabilityName string) []string {
	roleCaps := capability.GetRoleCapabilities()
	roles := make([]string, 0, len(roleCaps))
	for role, caps := range roleCaps {
		if capability.Has(caps, capabilityName) {
			roles = append(roles, role)
		}
	}
	return roles
}
