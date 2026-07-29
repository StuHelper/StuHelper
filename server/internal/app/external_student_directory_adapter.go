package app

import (
	"context"
	"fmt"

	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
)

type externalStudentDirectoryAdapter struct {
	registry *externaldata.StudentDirectoryRegistry
}

func newExternalStudentDirectoryAdapter(registry *externaldata.StudentDirectoryRegistry) externalStudentDirectoryAdapter {
	return externalStudentDirectoryAdapter{registry: registry}
}

func (a externalStudentDirectoryAdapter) LookupStudent(
	ctx context.Context,
	schoolCode string,
	studentID string,
) (*user.ExternalStudentRecord, bool, error) {
	record, handled, err := a.registry.LookupStudent(ctx, schoolCode, studentID)
	if err != nil {
		return nil, handled, fmt.Errorf("%w: %w", user.ErrAcademicLookupUnavailable, err)
	}
	if record == nil {
		return nil, handled, nil
	}
	return &user.ExternalStudentRecord{
		SchoolCode:  record.SchoolCode,
		StudentID:   record.StudentID,
		StudentName: record.StudentName,
	}, handled, nil
}
