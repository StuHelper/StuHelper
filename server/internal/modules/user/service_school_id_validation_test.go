package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchoolScopedServicesRejectInvalidSchoolIDBeforeDependencies(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.GetAcademicInfo(ctx, 0, "20240001")
	require.ErrorIs(t, err, ErrSchoolNotFound)

	_, err = svc.GetAcademicInfo(ctx, -1, "20240001")
	require.ErrorIs(t, err, ErrSchoolNotFound)

	_, _, err = svc.ListProfiles(ctx, "", int64Ptr(-1), 1, 20)
	require.ErrorIs(t, err, ErrSchoolNotFound)

	_, _, err = svc.ListProfiles(ctx, "", int64Ptr(0), 1, 20)
	require.ErrorIs(t, err, ErrSchoolNotFound)

	err = svc.UpdateSchoolConfig(ctx, 0, UpdateSchoolConfigInput{})
	require.ErrorIs(t, err, ErrSchoolNotFound)

	err = svc.UpdateSchoolConfig(ctx, -1, UpdateSchoolConfigInput{})
	require.ErrorIs(t, err, ErrSchoolNotFound)
}

func int64Ptr(value int64) *int64 {
	return &value
}
