package app

import (
	"context"
	"strconv"

	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	authorizationmodule "github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

type admissionAuthorizationGateway struct {
	authorization admissionAccessSnapshotResolver
}

type admissionAccessSnapshotResolver interface {
	ResolveAccessSnapshotByUserID(
		ctx context.Context,
		userID int64,
	) (authorizationmodule.AccessSnapshot, error)
}

func (g admissionAuthorizationGateway) UserHasCapabilityInSchool(
	ctx context.Context,
	userID int64,
	capabilityName string,
	schoolID int64,
) (bool, error) {
	snapshot, err := g.authorization.ResolveAccessSnapshotByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	access := capability.BuildUserAccessSnapshot(
		capability.ExpandRoleGrants(snapshot.Roles, snapshot.RoleScopes),
	)
	return capability.HasGrantInSchool(
		access.CapabilityGrants,
		capabilityName,
		strconv.FormatInt(schoolID, 10),
	), nil
}

func (rt *Runtime) initAdmissionOperatorAccess(
	authorization admissionAccessSnapshotResolver,
) admission.OperatorAccessGateway {
	return admissionAuthorizationGateway{authorization: authorization}
}
