package user

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	MFAMethodTOTP     = "totp"
	MFAMethodWebAuthn = "webauthn"
)

var (
	ErrInvalidMFAMethod            = errors.New("invalid mfa method")
	ErrMFAEnrollmentMethodRequired = errors.New("active mfa enrollment requires at least one method")
	ErrMFAEnrollmentNotFound       = errors.New("mfa enrollment not found")
	ErrMFAUserInvalid              = errors.New("mfa user id is invalid")
	ErrMFARecoveryUserInvalid      = ErrMFAUserInvalid
)

type MFAEnrollment struct {
	UserID                int64
	Active                bool
	Methods               []string
	RecoveryCodesIssuedAt *time.Time
	ResetRequired         bool
	LastEnrolledAt        *time.Time
	LastDisabledAt        *time.Time
	LastResetAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MFAEnrollmentUpsert struct {
	UserID                int64
	Active                bool
	Methods               []string
	RecoveryCodesIssuedAt *time.Time
	ResetRequired         bool
}

type MFAEnrollmentChangeAction string

const (
	MFAEnrollmentChangeDisable MFAEnrollmentChangeAction = "disable"
	MFAEnrollmentChangeReset   MFAEnrollmentChangeAction = "reset"
)

type MFAEnrollmentStateChange struct {
	UserID        int64
	Active        bool
	Methods       []string
	ResetRequired bool
	ChangedAt     time.Time
	Action        MFAEnrollmentChangeAction
}

func (r *Repository) UpsertMFAEnrollment(ctx context.Context, params MFAEnrollmentUpsert) error {
	return upsertMFAEnrollment(ctx, mfaEnrollmentUpsertQuery{
		Exec:   r.db.Exec,
		Params: params,
		Op:     "UpsertMFAEnrollment",
	})
}

func (r *Repository) UpsertMFAEnrollmentTx(ctx context.Context, tx pgx.Tx, params MFAEnrollmentUpsert) error {
	return upsertMFAEnrollment(ctx, mfaEnrollmentUpsertQuery{
		Exec:   tx.Exec,
		Params: params,
		Op:     "UpsertMFAEnrollmentTx",
	})
}

func (r *Repository) UpdateMFAEnrollmentStateTx(ctx context.Context, tx pgx.Tx, params MFAEnrollmentStateChange) error {
	if params.UserID <= 0 || !validMFAEnrollmentChange(params.Action) {
		return ErrMFARecoveryUserInvalid
	}
	methods, err := normalizeMFAMethods(params.Methods, params.Active)
	if err != nil {
		return err
	}
	var userID int64
	err = tx.QueryRow(ctx, `
		UPDATE user_mfa_enrollment
		SET active = $2,
		    methods = $3,
		    reset_required = $4,
		    recovery_codes_issued_at = NULL,
		    last_disabled_at = CASE WHEN $5 = 'disable' THEN $6 ELSE last_disabled_at END,
		    last_reset_at = CASE WHEN $5 = 'reset' THEN $6 ELSE last_reset_at END,
		    updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id
	`, params.UserID, params.Active, methods, params.ResetRequired, params.Action, params.ChangedAt).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMFAEnrollmentNotFound
	}
	if err != nil {
		return fmt.Errorf("UpdateMFAEnrollmentStateTx: %w", err)
	}
	return nil
}

func validMFAEnrollmentChange(action MFAEnrollmentChangeAction) bool {
	return action == MFAEnrollmentChangeDisable || action == MFAEnrollmentChangeReset
}

type mfaEnrollmentExec func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

type mfaEnrollmentUpsertQuery struct {
	Exec   mfaEnrollmentExec
	Params MFAEnrollmentUpsert
	Op     string
}

func upsertMFAEnrollment(ctx context.Context, query mfaEnrollmentUpsertQuery) error {
	params := query.Params
	methods, err := normalizeMFAMethods(params.Methods, params.Active)
	if err != nil {
		return err
	}
	_, err = query.Exec(ctx, `
		INSERT INTO user_mfa_enrollment (
			user_id, active, methods, recovery_codes_issued_at, reset_required,
			last_enrolled_at, last_disabled_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5,
			CASE WHEN $2 THEN NOW() ELSE NULL END,
			CASE WHEN $2 THEN NULL ELSE NOW() END,
			NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			active = EXCLUDED.active,
			methods = EXCLUDED.methods,
			recovery_codes_issued_at = EXCLUDED.recovery_codes_issued_at,
			reset_required = EXCLUDED.reset_required,
			last_enrolled_at = CASE WHEN EXCLUDED.active THEN NOW() ELSE user_mfa_enrollment.last_enrolled_at END,
			last_disabled_at = CASE WHEN EXCLUDED.active THEN user_mfa_enrollment.last_disabled_at ELSE NOW() END,
			updated_at = NOW()
	`, params.UserID, params.Active, methods, params.RecoveryCodesIssuedAt, params.ResetRequired)
	if err != nil {
		return fmt.Errorf("%s: %w", query.Op, err)
	}
	return nil
}

func (r *Repository) GetMFAEnrollment(ctx context.Context, userID int64) (*MFAEnrollment, error) {
	var item MFAEnrollment
	err := r.db.QueryRow(ctx, `
		SELECT user_id, active, methods, recovery_codes_issued_at, reset_required,
		       last_enrolled_at, last_disabled_at, last_reset_at, created_at, updated_at
		FROM user_mfa_enrollment
		WHERE user_id = $1
	`, userID).Scan(
		&item.UserID,
		&item.Active,
		&item.Methods,
		&item.RecoveryCodesIssuedAt,
		&item.ResetRequired,
		&item.LastEnrolledAt,
		&item.LastDisabledAt,
		&item.LastResetAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetMFAEnrollment: %w", err)
	}
	return &item, nil
}

func normalizeMFAMethods(methods []string, active bool) ([]string, error) {
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		item := strings.ToLower(strings.TrimSpace(method))
		if item == "" {
			continue
		}
		if !validMFAMethod(item) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMFAMethod, item)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	if active && len(out) == 0 {
		return nil, ErrMFAEnrollmentMethodRequired
	}
	return out, nil
}

func validMFAMethod(method string) bool {
	return method == MFAMethodTOTP || method == MFAMethodWebAuthn
}
