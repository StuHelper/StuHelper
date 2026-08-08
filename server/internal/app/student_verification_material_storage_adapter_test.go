package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/storage"
	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/pkg/objectstorage"
)

func TestNormalizeStudentVerificationMaterialStorageErrorMapsStorageErrors(t *testing.T) {
	for _, source := range []error{
		storage.ErrMountNotFound,
		storage.ErrMountDisabled,
		storage.ErrDriverNotRegistered,
		storage.ErrInvalidObjectKey,
		storage.ErrStoredObjectMissing,
		storage.ErrInvalidStoredObject,
	} {
		err := normalizeStudentVerificationMaterialStorageError(fmt.Errorf("wrapped: %w", source))

		require.ErrorIs(t, err, studentverification.ErrManualMaterialStoreUnavailable)
		require.False(t, errors.Is(err, source), "storage implementation error must not cross the domain adapter")
	}
}

func TestNormalizeStudentVerificationMaterialStorageErrorMapsObjectStorageErrors(t *testing.T) {
	for _, kind := range []objectstorage.ErrorKind{
		objectstorage.ErrorKindConfig,
		objectstorage.ErrorKindAuthentication,
		objectstorage.ErrorKindPermission,
		objectstorage.ErrorKindNotFound,
		objectstorage.ErrorKindNetwork,
	} {
		source := &objectstorage.StoreError{
			Kind: kind, Op: "student-verification-material",
			Resource: "student-verification/manual/case/material.jpg",
			Err:      context.DeadlineExceeded,
		}
		err := normalizeStudentVerificationMaterialStorageError(fmt.Errorf("wrapped: %w", source))

		require.ErrorIs(t, err, studentverification.ErrManualMaterialStoreUnavailable)
		require.False(t, errors.Is(err, source), "object-storage implementation error must not cross the domain adapter")
	}
}
