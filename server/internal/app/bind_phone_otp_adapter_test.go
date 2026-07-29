package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/auth"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

func TestNormalizeBindPhoneOTPErrorMapsAuthErrors(t *testing.T) {
	for _, tc := range []struct {
		source error
		want   error
	}{
		{source: auth.ErrOTPPhoneRateLimited, want: user.ErrBindPhoneOTPPhoneRateLimited},
		{source: auth.ErrOTPCooldown, want: user.ErrBindPhoneOTPCooldown},
		{source: auth.ErrOTPExpired, want: user.ErrBindPhoneOTPExpired},
		{source: auth.ErrOTPMaxAttempts, want: user.ErrBindPhoneOTPMaxAttempts},
		{source: auth.ErrOTPInvalidCode, want: user.ErrBindPhoneOTPInvalidCode},
	} {
		err := normalizeBindPhoneOTPError(fmt.Errorf("wrapped: %w", tc.source))

		require.ErrorIs(t, err, tc.want)
		require.False(t, errors.Is(err, tc.source), "auth module error must not leak through user OTP adapter")
	}
}
