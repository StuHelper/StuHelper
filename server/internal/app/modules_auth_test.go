package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
)

func TestAdminMFAMiddlewaresSkippedInDevelopment(t *testing.T) {
	require.Empty(t, adminMFAMiddlewares("development", nil))
}

func TestAdminMFAMiddlewaresEnabledOutsideDevelopment(t *testing.T) {
	require.Len(t, adminMFAMiddlewares("production", fakeMFAContextRepository{}), 2)
}

type fakeMFAContextRepository struct{}

func (fakeMFAContextRepository) GetInternalUserID(context.Context, string) (int64, error) {
	return 1, nil
}

func (fakeMFAContextRepository) GetMFAEnrollment(context.Context, int64) (*user.MFAEnrollment, error) {
	return &user.MFAEnrollment{Active: true, Methods: []string{user.MFAMethodTOTP}}, nil
}
