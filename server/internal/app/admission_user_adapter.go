package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
)

type admissionUserGateway struct {
	service *user.Service
}

func newAdmissionUserGateway(service *user.Service) admissionUserGateway {
	return admissionUserGateway{service: service}
}

func (g admissionUserGateway) EnsureQQBindingForUserTx(
	ctx context.Context,
	tx pgx.Tx,
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
	if err != nil || student == nil {
		return nil, err
	}
	return &admission.AcademicStudent{
		StudentID: student.XH,
		Name:      student.XM,
	}, nil
}

func (g admissionUserGateway) EnqueueFreshmanProvisionalRoleSyncTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	approved bool,
) error {
	return g.service.EnqueueFreshmanProvisionalRoleSyncTx(ctx, tx, userID, approved)
}
