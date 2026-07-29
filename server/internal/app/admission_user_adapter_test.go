package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

func TestNormalizeAdmissionQQBindingErrorMapsUserConflicts(t *testing.T) {
	for _, sourceErr := range []error{
		user.ErrQQBindingUserConflict,
		user.ErrQQBindingQQAlreadyBound,
	} {
		err := normalizeAdmissionQQBindingError(fmt.Errorf("wrapped: %w", sourceErr))

		require.ErrorIs(t, err, admission.ErrAdmissionQQMismatch)
		require.False(t, errors.Is(err, sourceErr), "user module error must not leak through admission adapter")
	}
}

func TestNormalizeAdmissionAcademicLookupErrorMapsDependencyFailure(t *testing.T) {
	sourceErr := fmt.Errorf("wrapped: %w", user.ErrAcademicLookupUnavailable)

	err := normalizeAdmissionAcademicLookupError(sourceErr)

	require.ErrorIs(t, err, admission.ErrAdmissionAcademicLookupUnavailable)
	require.False(t, errors.Is(err, user.ErrAcademicLookupUnavailable))
}
