package academics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcademicReadOperationsRejectInvalidInputsBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	for _, offeringID := range []int64{0, -1} {
		offering, err := svc.GetOffering(ctx, offeringID)
		require.ErrorIs(t, err, ErrInvalidOfferingID)
		assert.Nil(t, offering)
	}

	for _, externalUserID := range []string{"", "   "} {
		courses, err := svc.ListMyCourses(ctx, externalUserID, "2026-SPRING")
		require.ErrorIs(t, err, ErrAcademicUserRequired)
		assert.Nil(t, courses)

		schedule, err := svc.ListMySchedule(ctx, externalUserID, "2026-SPRING")
		require.ErrorIs(t, err, ErrAcademicUserRequired)
		assert.Nil(t, schedule)
	}
}
