package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/externaldata"
)

type fakeExternalDirectorySource struct {
	record *externaldata.StudentRecord
	err    error
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
