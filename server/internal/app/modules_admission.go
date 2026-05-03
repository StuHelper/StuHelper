package app

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
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
