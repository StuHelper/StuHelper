package app

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
)

type admissionStudentEligibilityAdapter struct {
	service *studentverification.Service
}

func newAdmissionStudentEligibilityAdapter(
	service *studentverification.Service,
) admission.StudentEligibilityGateway {
	return &admissionStudentEligibilityAdapter{service: service}
}

func (a *admissionStudentEligibilityAdapter) EvaluateStudentEligibility(
	ctx context.Context,
	userID int64,
	schoolID int64,
) (admission.StudentEligibilityDecision, error) {
	eligibility, err := a.service.GetEligibilityForSchoolID(ctx, userID, schoolID)
	if err != nil {
		return admission.StudentEligibilityDecision{}, err
	}
	return admission.StudentEligibilityDecision{
		Eligible: eligibility.Eligible,
		Revision: eligibility.Revision,
		CredentialClass: func() string {
			if eligibility.CredentialClass == nil {
				return ""
			}
			return *eligibility.CredentialClass
		}(),
	}, nil
}
