package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

func TestNormalizeIdentityPhotoStorageErrorMapsStorageErrors(t *testing.T) {
	for _, tc := range []struct {
		source error
		want   error
	}{
		{source: storage.ErrMountNotFound, want: user.ErrIdentityPhotoStorageUnavailable},
		{source: storage.ErrMountDisabled, want: user.ErrIdentityPhotoStorageUnavailable},
		{source: storage.ErrDriverNotRegistered, want: user.ErrIdentityPhotoStorageUnavailable},
		{source: storage.ErrInvalidObjectKey, want: user.ErrIdentityPhotoStorageUnavailable},
		{source: storage.ErrStoredObjectMissing, want: user.ErrIdentityPhotoStorageUnavailable},
		{source: storage.ErrInvalidStoredObject, want: user.ErrIdentityPhotoStorageUnavailable},
	} {
		err := normalizeIdentityPhotoStorageError(fmt.Errorf("wrapped: %w", tc.source))

		require.ErrorIs(t, err, tc.want)
		require.False(t, errors.Is(err, tc.source), "storage module error must not leak through identity photo adapter")
	}
}

func TestNormalizeIdentityPhotoStorageErrorMapsObjectStorageErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source error
		want   error
	}{
		{
			name: "network",
			source: &objectstorage.StoreError{
				Kind:     objectstorage.ErrorKindNetwork,
				Op:       "upload",
				Resource: "identities/42/front.png",
				Err:      context.DeadlineExceeded,
			},
			want: user.ErrIdentityPhotoStorageTemporaryUnavailable,
		},
		{
			name: "config",
			source: &objectstorage.StoreError{
				Kind:     objectstorage.ErrorKindConfig,
				Op:       "upload",
				Resource: "identities/42/front.png",
				Err:      errors.New("missing bucket"),
			},
			want: user.ErrIdentityPhotoStorageUnavailable,
		},
		{
			name: "auth",
			source: &objectstorage.StoreError{
				Kind:     objectstorage.ErrorKindAuthentication,
				Op:       "upload",
				Resource: "identities/42/front.png",
				Err:      errors.New("invalid credentials"),
			},
			want: user.ErrIdentityPhotoStorageUnavailable,
		},
		{
			name: "permission",
			source: &objectstorage.StoreError{
				Kind:     objectstorage.ErrorKindPermission,
				Op:       "upload",
				Resource: "identities/42/front.png",
				Err:      errors.New("access denied"),
			},
			want: user.ErrIdentityPhotoStorageUnavailable,
		},
		{
			name: "not found",
			source: &objectstorage.StoreError{
				Kind:     objectstorage.ErrorKindNotFound,
				Op:       "presign",
				Resource: "identities/42/front.png",
				Err:      errors.New("object not found"),
			},
			want: user.ErrIdentityPhotoStorageUnavailable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := normalizeIdentityPhotoStorageError(fmt.Errorf("wrapped: %w", tc.source))

			require.ErrorIs(t, err, tc.want)
			require.False(t, errors.Is(err, tc.source), "object storage error must not leak through identity photo adapter")
		})
	}
}
