package app

import (
	"context"
	"errors"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type admissionUserGateway struct {
	service *user.Service
}

func newAdmissionUserGateway(service *user.Service) admissionUserGateway {
	return admissionUserGateway{service: service}
}

func (g admissionUserGateway) EnsureQQBindingForUserTx(
	ctx context.Context,
	tx db.Tx,
	userID int64,
	qqID string,
) error {
	_, err := g.service.EnsureQQBindingForUserTx(ctx, tx, userID, qqID)
	return normalizeAdmissionQQBindingError(err)
}

func normalizeAdmissionQQBindingError(err error) error {
	if errors.Is(err, user.ErrQQBindingUserConflict) ||
		errors.Is(err, user.ErrQQBindingQQAlreadyBound) {
		return admission.ErrAdmissionQQMismatch
	}
	return err
}

func (g admissionUserGateway) GetAcademicInfo(
	ctx context.Context,
	schoolID int64,
	studentID string,
) (*admission.AcademicStudent, error) {
	student, err := g.service.GetAcademicInfo(ctx, schoolID, studentID)
	if err != nil {
		return nil, normalizeAdmissionAcademicLookupError(err)
	}
	if student == nil {
		return nil, nil
	}
	return &admission.AcademicStudent{
		StudentID: student.XH,
		Name:      student.XM,
	}, nil
}

func normalizeAdmissionAcademicLookupError(err error) error {
	if errors.Is(err, user.ErrAcademicLookupUnavailable) {
		return admission.ErrAdmissionAcademicLookupUnavailable
	}
	return err
}

func (g admissionUserGateway) EnqueueFreshmanProvisionalRoleSyncTx(
	ctx context.Context,
	tx db.Tx,
	userID int64,
	approved bool,
) error {
	return g.service.EnqueueFreshmanProvisionalRoleSyncTx(ctx, tx, userID, approved)
}
