package app

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

func (rt *Runtime) initAdmissionOperatorAccess(userRepo *user.Repository) admission.OperatorAccessGateway {
	client, err := rt.newCasdoorRoleSyncClient()
	if err != nil {
		return admission.NewRoleOperatorAccessGateway(func(context.Context, int64, string) (bool, error) {
			return false, err
		})
	}
	hasRole := platformcasdoor.BuildRoleMembershipFunc(client, userRepo.GetCasdoorSubject)
	return admission.NewRoleOperatorAccessGateway(hasRole)
}
