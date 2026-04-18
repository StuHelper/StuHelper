package review

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueConstraintViolation(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", ConstraintName: "idx_reviews_user_course"}

	assert.True(t, isUniqueConstraintViolation(err))
	assert.True(t, isUniqueConstraintViolation(err, "idx_reviews_user_course"))
	assert.False(t, isUniqueConstraintViolation(err, "uq_review_reports_user"))
	assert.False(t, isUniqueConstraintViolation(fmt.Errorf("wrapped: %w", err), "uq_review_reports_user"))
	assert.True(t, isUniqueConstraintViolation(fmt.Errorf("wrapped: %w", err), "idx_reviews_user_course"))
	assert.False(t, isUniqueConstraintViolation(assert.AnError, "idx_reviews_user_course"))
}
