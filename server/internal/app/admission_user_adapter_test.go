package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
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
