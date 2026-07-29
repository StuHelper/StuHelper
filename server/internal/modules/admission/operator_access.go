package admission

import (
	"context"
	"fmt"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

type OperatorRoleMembershipFunc func(ctx context.Context, userID int64, role string) (bool, error)

type RoleOperatorAccessGateway struct {
	hasRole         OperatorRoleMembershipFunc
	capabilityRoles map[string][]string
}

func NewRoleOperatorAccessGateway(hasRole OperatorRoleMembershipFunc) *RoleOperatorAccessGateway {
	return &RoleOperatorAccessGateway{
		hasRole:         hasRole,
		capabilityRoles: buildCapabilityRoleIndex(),
	}
}

func (g *RoleOperatorAccessGateway) UserHasCapability(
	ctx context.Context,
	userID int64,
	capabilityName string,
) (bool, error) {
	if g == nil || g.hasRole == nil {
		return false, ErrAdmissionOperatorAccessUnavailable
	}
	for _, role := range g.capabilityRoles[capabilityName] {
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

func buildCapabilityRoleIndex() map[string][]string {
	roleCaps := capability.GetRoleCapabilities()
	index := map[string][]string{}
	for role, caps := range roleCaps {
		for _, capName := range caps {
			index[capName] = append(index[capName], role)
		}
	}
	return index
}
