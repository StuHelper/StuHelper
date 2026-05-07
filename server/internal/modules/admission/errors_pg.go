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
