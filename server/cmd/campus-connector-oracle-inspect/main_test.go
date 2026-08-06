package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

func TestClassifyInspectionErrorDoesNotExposeDriverDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		code string
	}{
		{err: nil, code: "ok"},
		{err: context.DeadlineExceeded, code: "timeout"},
		{err: errors.New("oracle roster listener redirect target is not approved"), code: "listener_redirect_not_approved"},
		{err: errors.New("verify Oracle roster runtime identity: mismatch"), code: "runtime_identity_mismatch"},
		{err: errors.New("inspect approved Oracle roster object type"), code: "source_metadata_invalid"},
		{err: errors.New("configured roster operation was not found"), code: "configuration_invalid"},
		{err: errors.New("dial tcp 192.0.2.1:1521: secret-user failed"), code: "upstream_unavailable"},
	}
	for _, test := range tests {
		require.Equal(t, test.code, classifyInspectionError(test.err))
	}
}

func TestClearSnapshotRecordsRemovesIdentifyingValues(t *testing.T) {
	t.Parallel()

	year := 2026
	marker := true
	now := time.Now()
	snapshot := &externaldata.OracleFullRosterSnapshot{Records: []connectorprotocol.RosterRecord{{
		StudentID: "student", Name: "name", DocumentType: "type", DocumentNumber: "document",
		Phone: "phone", StudentStatus: "status", OnCampusStatus: "campus",
		RegistrationStatus: "registration", EducationLevel: "level", StudentCategory: "category",
		EnrollmentYear: &year, ValidFrom: &now, ValidUntil: &now, CurrentMarker: &marker,
		EligibilityCode: "eligible", SourceUpdatedAt: &now,
	}}}
	records := snapshot.Records

	clearSnapshotRecords(snapshot)
	require.Nil(t, snapshot.Records)
	require.Equal(t, connectorprotocol.RosterRecord{}, records[0])
}
