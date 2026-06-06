package externaldata

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStudentDirectory struct {
	record *StudentRecord
}

type fakeClosingStudentDirectory struct {
	fakeStudentDirectory
	closeCount int
	closeErr   error
}

func (d fakeStudentDirectory) LookupStudent(context.Context, string) (*StudentRecord, error) {
	return d.record, nil
}

func (d *fakeClosingStudentDirectory) Close() error {
	d.closeCount++
	return d.closeErr
}

func TestStudentDirectoryRegistryRoutesBySchoolCode(t *testing.T) {
	registry, err := NewStudentDirectoryRegistry([]StudentSource{{
		Name:       "buaa",
		SchoolCode: "4111010006",
		Directory: fakeStudentDirectory{record: &StudentRecord{
			StudentID:   "20250001",
			StudentName: "张三",
		}},
	}})
	require.NoError(t, err)

	record, handled, err := registry.LookupStudent(context.Background(), "4111010006", "20250001")
	require.NoError(t, err)
	require.True(t, handled)
	require.NotNil(t, record)
	assert.Equal(t, "4111010006", record.SchoolCode)
	assert.Equal(t, "20250001", record.StudentID)

	record, handled, err = registry.LookupStudent(context.Background(), "4111010001", "20250001")
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Nil(t, record)
}

func TestStudentDirectoryRegistryRejectsDuplicateSchools(t *testing.T) {
	_, err := NewStudentDirectoryRegistry([]StudentSource{
		{Name: "first", SchoolCode: "4111010006", Directory: fakeStudentDirectory{}},
		{Name: "second", SchoolCode: "4111010006", Directory: fakeStudentDirectory{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate student source")
}

func TestStudentDirectoryRegistryCloseIsIdempotent(t *testing.T) {
	source := &fakeClosingStudentDirectory{}
	registry, err := NewStudentDirectoryRegistry([]StudentSource{{
		Name:       "buaa",
		SchoolCode: "4111010006",
		Directory:  source,
	}})
	require.NoError(t, err)

	require.NoError(t, registry.Close())
	require.NoError(t, registry.Close())
	assert.Equal(t, 1, source.closeCount)
}

func TestStudentDirectoryRegistryClosesOwnedSourcesOnBuildFailure(t *testing.T) {
	source := &fakeClosingStudentDirectory{}

	_, err := NewStudentDirectoryRegistry([]StudentSource{
		{Name: "first", SchoolCode: "4111010006", Directory: source},
		{Name: "duplicate", SchoolCode: "4111010006", Directory: fakeStudentDirectory{}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate student source")
	assert.Equal(t, 1, source.closeCount)
}

func TestStudentDirectoryRegistryBuildFailureIncludesCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	source := &fakeClosingStudentDirectory{closeErr: closeErr}

	_, err := NewStudentDirectoryRegistry([]StudentSource{
		{Name: "first", SchoolCode: "4111010006", Directory: source},
		{Name: "duplicate", SchoolCode: "4111010006", Directory: fakeStudentDirectory{}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, closeErr)
	assert.Contains(t, err.Error(), "duplicate student source")
	assert.Equal(t, 1, source.closeCount)
}
