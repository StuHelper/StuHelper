package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sijms/go-ora/v2/network"
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
		{err: network.NewOracleError(1017), code: "authentication_rejected"},
		{err: network.NewOracleError(28000), code: "account_locked"},
		{err: network.NewOracleError(28001), code: "password_expired"},
		{err: network.NewOracleError(28041), code: "authentication_protocol_rejected"},
		{err: network.NewOracleError(12514), code: "listener_unavailable"},
		{err: network.NewOracleError(904), code: "oracle_error_00904"},
		{err: network.ErrConnReset, code: "timeout"},
		{err: errors.New("read tcp: i/o timeout"), code: "timeout"},
		{err: errors.New("invalid username/password"), code: "authentication_rejected"},
		{err: errors.New("StuHelper requires an Oracle 12c or newer PBKDF2 password verifier"), code: "legacy_password_verifier"},
		{err: errors.New("session key should be either 64, 96 bytes long"), code: "authentication_protocol_rejected"},
		{err: errors.New("unsupported server version"), code: "server_version_unsupported"},
		{err: errors.New("connect: connection refused"), code: "transport_refused"},
		{err: errors.New("dial tcp: network is unreachable"), code: "transport_unreachable"},
		{err: errors.New("lookup: no such host"), code: "target_resolution_failed"},
		{err: errors.New("unexpected EOF"), code: "transport_closed"},
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
