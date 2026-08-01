package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

type fakeExternalDirectorySource struct {
	record *externaldata.StudentRecord
	err    error
}

func TestExternalStudentDirectoryAdapterMapsSourceFailureToDependencyError(t *testing.T) {
	sourceErr := errors.New("oracle unavailable")
	registry, err := externaldata.NewStudentDirectoryRegistry([]externaldata.StudentSource{{
		Name:       "test",
		SchoolCode: "4111010006",
		Directory:  fakeExternalDirectorySource{err: sourceErr},
	}})
	require.NoError(t, err)

	record, handled, err := newExternalStudentDirectoryAdapter(registry).LookupStudent(
		context.Background(),
		"4111010006",
		"20250001",
	)

	require.Nil(t, record)
	require.True(t, handled)
	require.ErrorIs(t, err, user.ErrAcademicLookupUnavailable)
	require.ErrorIs(t, err, sourceErr)
}

func TestExternalStudentDirectoryAdapterKeepsIntegrityErrorsOnDependencyContract(t *testing.T) {
	registry, err := externaldata.NewStudentDirectoryRegistry([]externaldata.StudentSource{{
		Name:       "test",
		SchoolCode: "4111010006",
		Directory: fakeExternalDirectorySource{
			err: externaldata.ErrStudentSourceInvalidRecord,
		},
	}})
	require.NoError(t, err)

	record, handled, err := newExternalStudentDirectoryAdapter(registry).LookupStudent(
		context.Background(),
		"4111010006",
		"20250001",
	)

	require.Nil(t, record)
	require.True(t, handled)
	require.ErrorIs(t, err, user.ErrAcademicLookupUnavailable)
	require.ErrorIs(t, err, externaldata.ErrStudentSourceInvalidRecord)
}

func (s fakeExternalDirectorySource) LookupStudent(context.Context, string) (*externaldata.StudentRecord, error) {
	return s.record, s.err
}

func TestExternalStudentDirectoryAdapterMapsStudentRecord(t *testing.T) {
	registry, err := externaldata.NewStudentDirectoryRegistry([]externaldata.StudentSource{
		{
			Name:       "test",
			SchoolCode: "4111010006",
			Directory: fakeExternalDirectorySource{
				record: &externaldata.StudentRecord{
					StudentID:   "20250001",
					StudentName: "张三",
				},
			},
		},
	})
	require.NoError(t, err)

	record, handled, err := newExternalStudentDirectoryAdapter(registry).LookupStudent(
		context.Background(),
		"4111010006",
		"20250001",
	)

	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, "4111010006", record.SchoolCode)
	require.Equal(t, "20250001", record.StudentID)
	require.Equal(t, "张三", record.StudentName)
}
