package app

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

type userVerificationStatusAdapter struct {
	service *studentverification.Service
}

func newUserVerificationStatusAdapter(service *studentverification.Service) userVerificationStatusAdapter {
	return userVerificationStatusAdapter{service: service}
}

func (adapter userVerificationStatusAdapter) GetCurrentStudentStatus(
	ctx context.Context,
	userID int64,
) (user.CurrentStudentStatus, error) {
	status, err := adapter.service.GetCurrentStudentStatus(ctx, userID)
	if err != nil {
		return user.CurrentStudentStatus{}, err
	}
	return user.CurrentStudentStatus{Eligible: status.Eligible, SchoolID: status.SchoolID}, nil
}

func (adapter userVerificationStatusAdapter) GetCurrentPhoneStatus(
	ctx context.Context,
	userID int64,
) (user.CurrentPhoneStatus, error) {
	status, err := adapter.service.GetPhoneStatus(ctx, userID)
	if err != nil {
		return user.CurrentPhoneStatus{}, err
	}
	return user.CurrentPhoneStatus{
		MaskedPhone:                    status.MaskedPhone,
		PublishingRequirementSatisfied: status.PublishingRequirementSatisfied,
	}, nil
}
