package review

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueConstraintViolation(err error, constraintNames ...string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	if len(constraintNames) == 0 {
		return true
	}
	for _, name := range constraintNames {
		if pgErr.ConstraintName == name {
			return true
		}
	}
	return false
}
