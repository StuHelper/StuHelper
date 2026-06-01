package admission

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgUniqueViolation = "23505"

func isMemberBlacklistActiveUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgUniqueViolation {
		return false
	}
	switch pgErr.ConstraintName {
	case "member_blacklist_global_active_key", "member_blacklist_guild_active_key":
		return true
	default:
		return false
	}
}

func isFreshmanMaterialApplicationUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgUniqueViolation &&
		pgErr.ConstraintName == "freshman_verification_materials_application_id_key"
}

func isFreshmanApplicationPendingUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgUniqueViolation &&
		pgErr.ConstraintName == "freshman_verification_applications_pending_user_school_idx"
}

func isFreshmanCameraHandoffActiveUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgUniqueViolation &&
		pgErr.ConstraintName == "freshman_camera_handoffs_active_application_idx"
}
