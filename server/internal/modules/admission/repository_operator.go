package admission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserIDByQQID(ctx context.Context, qqID string) (*int64, error) {
	ctx = withDBTable(ctx, "user_qq_bindings")
	var userID int64
	err := r.db.QueryRow(ctx, `
		SELECT user_id
		FROM user_qq_bindings
		WHERE qq_id = $1
	`, qqID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserIDByQQID: %w", err)
	}
	return &userID, nil
}

func (r *Repository) GetFreshmanApplicationForReviewPolicy(
	ctx context.Context,
	applicationID string,
) (*FreshmanApplication, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	app, err := scanFreshmanApplication(r.db.QueryRow(ctx, `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		       department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
		FROM freshman_verification_applications
		WHERE id = $1
	`, applicationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdmissionApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetFreshmanApplicationForReviewPolicy: %w", err)
	}
	return app, nil
}

func (r *Repository) GetFreshmanApplicationByIDForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	applicationID string,
) (*FreshmanApplication, error) {
	app, err := scanFreshmanApplication(tx.QueryRow(ctx, `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		       department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
		FROM freshman_verification_applications
		WHERE id = $1
		FOR UPDATE
	`, applicationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdmissionApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetFreshmanApplicationByIDForUpdate: %w", err)
	}
	return app, nil
}

func (r *Repository) ReviewFreshmanApplicationTx(
	ctx context.Context,
	tx pgx.Tx,
	input freshmanApplicationReviewUpdate,
) (*FreshmanApplication, error) {
	app, err := scanFreshmanApplication(tx.QueryRow(ctx, `
		UPDATE freshman_verification_applications
		SET status = $2, review_reason = $3, reviewed_by_user_id = $4,
		    reviewed_by_operator_qq_id = $5, provisional_expires_at = $6,
		    reviewed_at = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		          department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
	`, input.ApplicationID, input.Status, input.Reason, input.ReviewerUserID,
		input.OperatorQQID, input.ProvisionalExpiresAt, input.ReviewedAt))
	if err != nil {
		return nil, fmt.Errorf("ReviewFreshmanApplicationTx: %w", err)
	}
	return app, nil
}

type freshmanApplicationReviewUpdate struct {
	ApplicationID        string
	Status               FreshmanApplicationStatus
	Reason               *string
	ReviewerUserID       *int64
	OperatorQQID         *string
	ProvisionalExpiresAt *time.Time
	ReviewedAt           time.Time
}
