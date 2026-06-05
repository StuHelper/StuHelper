package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

func TestNormalizeAdmissionMaterialStorageErrorMapsStorageErrors(t *testing.T) {
	for _, source := range []error{
		storage.ErrMountNotFound,
		storage.ErrMountDisabled,
		storage.ErrDriverNotRegistered,
		storage.ErrStoredObjectMissing,
		storage.ErrInvalidStoredObject,
	} {
		err := normalizeAdmissionMaterialStorageError(fmt.Errorf("wrapped: %w", source))

		require.ErrorIs(t, err, admission.ErrAdmissionMaterialStoreUnavailable)
		require.False(t, errors.Is(err, source), "storage module error must not leak through admission material adapter")
	}
}

func TestNormalizeAdmissionMaterialStorageErrorMapsObjectStorageErrors(t *testing.T) {
	for _, kind := range []objectstorage.ErrorKind{
		objectstorage.ErrorKindConfig,
		objectstorage.ErrorKindAuthentication,
		objectstorage.ErrorKindPermission,
		objectstorage.ErrorKindNotFound,
		objectstorage.ErrorKindNetwork,
	} {
		source := &objectstorage.StoreError{
			Kind:     kind,
			Op:       "admission-material",
			Resource: "admission/freshman/application/material.png",
			Err:      context.DeadlineExceeded,
		}
		err := normalizeAdmissionMaterialStorageError(fmt.Errorf("wrapped: %w", source))

		require.ErrorIs(t, err, admission.ErrAdmissionMaterialStoreUnavailable)
		require.False(t, errors.Is(err, source), "object storage error must not leak through admission material adapter")
	}
}
