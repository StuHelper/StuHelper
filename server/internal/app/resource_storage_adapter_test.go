package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/resource"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

func TestNormalizeResourceStorageErrorMapsStorageErrors(t *testing.T) {
	for _, tc := range []struct {
		source error
		want   error
	}{
		{source: storage.ErrMountNotFound, want: resource.ErrResourceStorageMountNotFound},
		{source: storage.ErrMountDisabled, want: resource.ErrResourceStorageMountDisabled},
		{source: storage.ErrDriverNotRegistered, want: resource.ErrResourceStorageDriverMissing},
		{source: storage.ErrStoredObjectMissing, want: resource.ErrResourceStoredObjectMissing},
		{source: storage.ErrInvalidStoredObject, want: resource.ErrResourceStoredObjectInvalid},
	} {
		err := normalizeResourceStorageError(fmt.Errorf("wrapped: %w", tc.source))

		require.ErrorIs(t, err, tc.want)
		require.False(t, errors.Is(err, tc.source), "storage module error must not leak through resource adapter")
	}
}
