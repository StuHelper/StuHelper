package app

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/externaldata"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
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
	if err != nil || record == nil {
		return nil, handled, err
	}
	return &user.ExternalStudentRecord{
		SchoolCode:  record.SchoolCode,
		StudentID:   record.StudentID,
		StudentName: record.StudentName,
	}, handled, nil
}
