package node

import (
	"errors"
	"os"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
)

// BuildOracleRosterSnapshotConfig converts one already validated local
// operation into the fixed Oracle reader configuration. Credentials are read
// only from the operation's environment references and are never serialized.
func BuildOracleRosterSnapshotConfig(operation OperationConfig) (externaldata.OracleRosterSnapshotConfig, error) {
	if err := operation.Validate(); err != nil {
		return externaldata.OracleRosterSnapshotConfig{}, err
	}
	if operation.Type != "roster_snapshot_upload" || operation.OracleRoster == nil {
		return externaldata.OracleRosterSnapshotConfig{}, errors.New("operation is not an Oracle roster snapshot")
	}
	cfg := operation.OracleRoster
	username := os.Getenv(cfg.UsernameEnv)
	password := os.Getenv(cfg.PasswordEnv)
	if username == "" || password == "" {
		return externaldata.OracleRosterSnapshotConfig{}, errors.New("oracle roster secret reference is unavailable")
	}
	timeout := time.Duration(operation.TimeoutMilliseconds) * time.Millisecond
	return externaldata.OracleRosterSnapshotConfig{
		Host: operation.TargetHost, Port: operation.TargetPort, TransportMode: operation.UpstreamProtocol,
		TLSServerName: operation.TLSServerName, ServiceName: cfg.ServiceName,
		Username: username, Password: password, ExpectedUsername: cfg.ExpectedUsername,
		CAFile: cfg.CAFile, Schema: cfg.Schema, Table: cfg.Table,
		AllowedDialTargets: append([]string(nil), cfg.AllowedDialTargets...),
		ActiveFilterColumn: cfg.ActiveFilterColumn, ActiveFilterValue: cfg.ActiveFilterValue,
		ActiveEligibilityCode: cfg.ActiveEligibilityCode,
		Columns: externaldata.OracleRosterSnapshotColumns{
			StudentID: cfg.Columns.StudentID, Name: cfg.Columns.Name,
			DocumentType: cfg.Columns.DocumentType, DocumentNumber: cfg.Columns.DocumentNumber,
			Phone: cfg.Columns.Phone, StudentStatus: cfg.Columns.StudentStatus,
			OnCampusStatus: cfg.Columns.OnCampusStatus, RegistrationStatus: cfg.Columns.RegistrationStatus,
			EducationLevel: cfg.Columns.EducationLevel, StudentCategory: cfg.Columns.StudentCategory,
			EnrollmentYear: cfg.Columns.EnrollmentYear, ValidFrom: cfg.Columns.ValidFrom,
			ValidUntil: cfg.Columns.ValidUntil, CurrentMarker: cfg.Columns.CurrentMarker,
			EligibilityCode: cfg.Columns.EligibilityCode, SourceUpdatedAt: cfg.Columns.SourceUpdatedAt,
		},
		DefaultDocumentType: cfg.DefaultDocumentType, MaximumRows: cfg.MaximumRows,
		ConnectTimeout: min(timeout, 10*time.Second), QueryTimeout: timeout,
	}, nil
}
