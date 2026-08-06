// Command campus-connector-oracle-inspect performs the fixed, read-only Oracle
// identity and data-dictionary inspection used before approving a roster
// operation. It never accepts SQL. An explicit --full-snapshot-check runs the
// same fixed full SELECT as the connector and emits aggregate evidence only.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/campusconnector/node"
	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
)

type inspectionEvidence struct {
	CheckedAt               string                                  `json:"checkedAt"`
	ObjectType              string                                  `json:"objectType"`
	Columns                 []externaldata.OracleRosterSourceColumn `json:"columns"`
	RuntimeIdentityMatched  bool                                    `json:"runtimeIdentityMatched"`
	PreferredPrivilegeShape bool                                    `json:"preferredPrivilegeShape"`
	FullSnapshotCheck       *fullSnapshotCheckEvidence              `json:"fullSnapshotCheck,omitempty"`
}

type fullSnapshotCheckEvidence struct {
	SourceStartedAt       string `json:"sourceStartedAt"`
	SourceCutoffAt        string `json:"sourceCutoffAt"`
	RowsRead              int64  `json:"rowsRead"`
	RecordsEmitted        int64  `json:"recordsEmitted"`
	MissingDocumentNumber int64  `json:"missingDocumentNumber"`
	InvalidDocumentNumber int64  `json:"invalidDocumentNumber"`
	MissingPhone          int64  `json:"missingPhone"`
	InvalidPhone          int64  `json:"invalidPhone"`
	MissingEnrollmentYear int64  `json:"missingEnrollmentYear"`
	InvalidEnrollmentYear int64  `json:"invalidEnrollmentYear"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "campus connector Oracle inspection failed: %s\n", classifyInspectionError(err))
		os.Exit(1)
	}
}

// classifyInspectionError keeps operational output actionable without ever
// returning a driver error that could embed an endpoint, username, service
// name, DSN, or other connection detail.
func classifyInspectionError(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	value := strings.ToLower(err.Error())
	switch {
	case strings.Contains(value, "listener redirect target is not approved"):
		return "listener_redirect_not_approved"
	case strings.Contains(value, "runtime identity"):
		return "runtime_identity_mismatch"
	case strings.Contains(value, "approved oracle roster object"),
		strings.Contains(value, "approved oracle roster column"),
		strings.Contains(value, "source exposes no readable column"):
		return "source_metadata_invalid"
	case strings.Contains(value, "configuration"), strings.Contains(value, "configured roster operation"):
		return "configuration_invalid"
	default:
		return "upstream_unavailable"
	}
}

func run() error {
	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0}); err != nil {
		return err
	}
	defaultConfig := strings.TrimSpace(os.Getenv("CAMPUS_CONNECTOR_NODE_CONFIG_FILE"))
	configPath := flag.String("config", defaultConfig, "path to the non-secret connector operation configuration")
	operationKey := flag.String("operation", "", "exact roster operation key")
	fullSnapshotCheck := flag.Bool(
		"full-snapshot-check", false,
		"run the configured fixed full SELECT and emit aggregate evidence without uploading rows",
	)
	flag.Parse()
	cfg, err := node.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	operation, ok := cfg.Operation(strings.TrimSpace(*operationKey))
	if !ok {
		return fmt.Errorf("configured roster operation was not found")
	}
	snapshotConfig, err := node.BuildOracleRosterSnapshotConfig(operation)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(operation.TimeoutMilliseconds)*time.Millisecond)
	defer cancel()
	evidence := inspectionEvidence{CheckedAt: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)}
	if *fullSnapshotCheck {
		snapshot, err := externaldata.ReadOracleFullRosterSnapshot(ctx, snapshotConfig)
		if err != nil {
			return err
		}
		defer clearSnapshotRecords(snapshot)
		evidence.ObjectType = snapshot.SourceInspection.ObjectType
		evidence.Columns = snapshot.SourceInspection.Columns
		evidence.RuntimeIdentityMatched = snapshot.SourceInspection.RuntimeIdentityMatched
		evidence.PreferredPrivilegeShape = snapshot.SourceInspection.LeastPrivilegeVerified
		evidence.FullSnapshotCheck = &fullSnapshotCheckEvidence{
			SourceStartedAt: snapshot.SourceStartedAt.UTC().Format(time.RFC3339Nano),
			SourceCutoffAt:  snapshot.SourceCutoffAt.UTC().Format(time.RFC3339Nano),
			RowsRead:        snapshot.Quality.RowsRead, RecordsEmitted: snapshot.Quality.RecordsEmitted,
			MissingDocumentNumber: snapshot.Quality.MissingDocumentNumber,
			InvalidDocumentNumber: snapshot.Quality.InvalidDocumentNumber,
			MissingPhone:          snapshot.Quality.MissingPhone, InvalidPhone: snapshot.Quality.InvalidPhone,
			MissingEnrollmentYear: snapshot.Quality.MissingEnrollmentYear,
			InvalidEnrollmentYear: snapshot.Quality.InvalidEnrollmentYear,
		}
	} else {
		inspection, err := externaldata.InspectOracleRosterSource(ctx, snapshotConfig)
		if err != nil {
			return err
		}
		evidence.ObjectType = inspection.ObjectType
		evidence.Columns = inspection.Columns
		evidence.RuntimeIdentityMatched = inspection.RuntimeIdentityMatched
		evidence.PreferredPrivilegeShape = inspection.LeastPrivilegeVerified
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(evidence)
}

func clearSnapshotRecords(snapshot *externaldata.OracleFullRosterSnapshot) {
	if snapshot == nil {
		return
	}
	for index := range snapshot.Records {
		record := &snapshot.Records[index]
		record.StudentID = ""
		record.Name = ""
		record.DocumentType = ""
		record.DocumentNumber = ""
		record.Phone = ""
		record.StudentStatus = ""
		record.OnCampusStatus = ""
		record.RegistrationStatus = ""
		record.EducationLevel = ""
		record.StudentCategory = ""
		record.EligibilityCode = ""
		record.EnrollmentYear = nil
		record.ValidFrom = nil
		record.ValidUntil = nil
		record.CurrentMarker = nil
		record.SourceUpdatedAt = nil
	}
	snapshot.Records = nil
}
