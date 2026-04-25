package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeReviewAccessReader_FailClosedStub(t *testing.T) {
	reader := normalizeReviewAccessReader(nil)

	_, err := reader.ListReviewAccessSchoolConfigs(context.Background())
	assert.ErrorIs(t, err, errReviewAccessReaderNotConfigured)

	_, err = reader.ListReviewAccessSystemConfigs(context.Background())
	assert.ErrorIs(t, err, errReviewAccessReaderNotConfigured)

	_, err = reader.GetReviewAccessSubject(context.Background(), "user-1")
	assert.ErrorIs(t, err, errReviewAccessReaderNotConfigured)
}
