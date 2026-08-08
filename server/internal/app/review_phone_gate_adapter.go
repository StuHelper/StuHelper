package app

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
)

type reviewPhoneGateAdapter struct {
	service *studentverification.Service
}

func (adapter reviewPhoneGateAdapter) PhonePublishingRequirementSatisfied(
	ctx context.Context,
	userID int64,
) (bool, error) {
	gate, err := adapter.service.GetPhoneGateEligibility(ctx, userID)
	if err != nil {
		return false, err
	}
	return gate.Eligible, nil
}
