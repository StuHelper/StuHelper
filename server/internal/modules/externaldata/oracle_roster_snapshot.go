package externaldata

import (
	"context"
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
	go_ora "github.com/StuHelper/StuHelper/server/internal/thirdparty/goora"
)

type OracleRosterSnapshotConfig struct {
	Host                  string
	Port                  int
	TransportMode         string
	TLSServerName         string
	ServiceName           string
	Username              string
	Password              string
	ExpectedUsername      string
	CAFile                string
	Schema                string
	Table                 string
	Columns               OracleRosterSnapshotColumns
	DefaultDocumentType   string
	AllowedDialTargets    []string
	ActiveFilterColumn    string
	ActiveFilterValue     string
	ActiveEligibilityCode string
	MaximumRows           int
	ConnectTimeout        time.Duration
	QueryTimeout          time.Duration
}

type OracleRosterSnapshotColumns struct {
	StudentID          string
	Name               string
	DocumentType       string
	DocumentNumber     string
	Phone              string
	StudentStatus      string
	OnCampusStatus     string
	RegistrationStatus string
	EducationLevel     string
	StudentCategory    string
	EnrollmentYear     string
	ValidFrom          string
	ValidUntil         string
	CurrentMarker      string
	EligibilityCode    string
	SourceUpdatedAt    string
}

type OracleFullRosterSnapshot struct {
	SourceStartedAt  time.Time
	SourceCutoffAt   time.Time
	SourceInspection OracleRosterSourceInspection
	Records          []connectorprotocol.RosterRecord
	Quality          OracleRosterQualitySummary
}

// OracleRosterQualitySummary contains aggregate, non-identifying evidence
// about values deliberately dropped while mapping the approved source object.
// It must never contain source values or row identifiers.
type OracleRosterQualitySummary struct {
	LeastPrivilegeVerified bool
	RowsRead               int64
	RecordsEmitted         int64
	MissingDocumentNumber  int64
	InvalidDocumentNumber  int64
	MissingPhone           int64
	InvalidPhone           int64
	MissingEnrollmentYear  int64
	InvalidEnrollmentYear  int64
}

// OracleRosterSourceInspection contains source metadata only. It deliberately
// excludes credentials, endpoints, row values, object DDL, and student data.
type OracleRosterSourceInspection struct {
	ObjectType             string
	Columns                []OracleRosterSourceColumn
	RuntimeIdentityMatched bool
	LeastPrivilegeVerified bool
}

type OracleRosterSourceColumn struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Nullable bool   `json:"nullable"`
}

type oracleRosterRowQuality struct {
	MissingDocumentNumber int64
	InvalidDocumentNumber int64
	MissingPhone          int64
	InvalidPhone          int64
	MissingEnrollmentYear int64
	InvalidEnrollmentYear int64
}

// ReadOracleFullRosterSnapshot performs one statement-consistent, read-only
// full scan. The query shape is constructed exclusively from validated local
// identifier configuration; callers cannot supply SQL or predicates.
func ReadOracleFullRosterSnapshot(
	ctx context.Context,
	cfg OracleRosterSnapshotConfig,
) (snapshot *OracleFullRosterSnapshot, returnErr error) {
	if err := validateOracleRosterSnapshotConfig(cfg); err != nil {
		return nil, err
	}
	database, err := openOracleRosterDatabase(cfg)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			if snapshot != nil {
				clearProtocolRecords(snapshot.Records)
				snapshot = nil
			}
			returnErr = errors.Join(returnErr, fmt.Errorf("close oracle roster connection: %w", closeErr))
		}
	}()

	queryCtx, cancelQuery := context.WithTimeout(ctx, cfg.QueryTimeout)
	defer cancelQuery()
	connection, err := database.Conn(queryCtx)
	if err != nil {
		return nil, fmt.Errorf("reserve Oracle roster connection: %w", err)
	}
	defer func() {
		if closeErr := connection.Close(); closeErr != nil {
			if snapshot != nil {
				clearProtocolRecords(snapshot.Records)
				snapshot = nil
			}
			returnErr = errors.Join(returnErr, fmt.Errorf("release oracle roster connection: %w", closeErr))
		}
	}()
	connectCtx, cancelConnect := context.WithTimeout(queryCtx, cfg.ConnectTimeout)
	err = connection.PingContext(connectCtx)
	cancelConnect()
	if err != nil {
		return nil, fmt.Errorf("connect to Oracle roster source: %w", err)
	}

	securityProbe, err := probeOracleRuntimeSecurity(
		queryCtx, connection, cfg.ExpectedUsername, cfg.Schema, cfg.Table,
	)
	if err != nil {
		return nil, fmt.Errorf("verify Oracle roster runtime identity: %w", err)
	}
	metadata, err := inspectOracleRosterSourceMetadata(queryCtx, connection, cfg.Schema, cfg.Table)
	if err != nil {
		return nil, err
	}
	if err := validateConfiguredOracleRosterColumns(metadata, cfg); err != nil {
		return nil, err
	}

	startedAt := time.Now().UTC()
	rows, err := connection.QueryContext(
		queryCtx, buildOracleFullRosterQuery(cfg), cfg.ActiveFilterValue,
	)
	if err != nil {
		return nil, fmt.Errorf("read Oracle full roster: %w", err)
	}
	closeRows := func(cause error) error {
		if closeErr := rows.Close(); closeErr != nil {
			return errors.Join(cause, fmt.Errorf("close oracle roster rows: %w", closeErr))
		}
		return cause
	}
	records := make([]connectorprotocol.RosterRecord, 0, min(cfg.MaximumRows, 65536))
	quality := OracleRosterQualitySummary{
		LeastPrivilegeVerified: securityProbe.LeastPrivilegeVerified,
	}
	for rows.Next() {
		if len(records) >= cfg.MaximumRows {
			clearProtocolRecords(records)
			return nil, closeRows(errors.New("oracle roster exceeded the approved maximum row count"))
		}
		values := make([]sql.NullString, 16)
		destinations := make([]any, len(values))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			clearProtocolRecords(records)
			return nil, closeRows(fmt.Errorf("scan Oracle roster row: %w", err))
		}
		quality.RowsRead++
		record, rowQuality, err := mapOracleRosterRowWithQuality(
			values, cfg.DefaultDocumentType, cfg.ActiveEligibilityCode,
		)
		if err != nil {
			clearProtocolRecords(records)
			return nil, closeRows(err)
		}
		quality.MissingDocumentNumber += rowQuality.MissingDocumentNumber
		quality.InvalidDocumentNumber += rowQuality.InvalidDocumentNumber
		quality.MissingPhone += rowQuality.MissingPhone
		quality.InvalidPhone += rowQuality.InvalidPhone
		quality.MissingEnrollmentYear += rowQuality.MissingEnrollmentYear
		quality.InvalidEnrollmentYear += rowQuality.InvalidEnrollmentYear
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		clearProtocolRecords(records)
		return nil, closeRows(fmt.Errorf("iterate Oracle roster rows: %w", err))
	}
	if err := closeRows(nil); err != nil {
		clearProtocolRecords(records)
		return nil, err
	}
	quality.RecordsEmitted = int64(len(records))
	return &OracleFullRosterSnapshot{
		SourceStartedAt: startedAt,
		// Oracle guarantees statement-level read consistency. Without an
		// approved SCN/flashback contract the exact defensible cutoff is the
		// full SELECT start, not the wall clock after the cursor is drained.
		SourceCutoffAt: startedAt,
		SourceInspection: OracleRosterSourceInspection{
			ObjectType: metadata.ObjectType, Columns: metadata.Columns,
			RuntimeIdentityMatched: securityProbe.IdentityMatched,
			LeastPrivilegeVerified: securityProbe.LeastPrivilegeVerified,
		},
		Records: records,
		Quality: quality,
	}, nil
}

// InspectOracleRosterSource logs in and executes only the fixed runtime-identity
// and read-only data-dictionary SELECTs used by the snapshot reader. It does not
// read roster rows.
func InspectOracleRosterSource(
	ctx context.Context,
	cfg OracleRosterSnapshotConfig,
) (inspection OracleRosterSourceInspection, returnErr error) {
	if err := validateOracleRosterSnapshotConfig(cfg); err != nil {
		return inspection, err
	}
	database, err := openOracleRosterDatabase(cfg)
	if err != nil {
		return inspection, err
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close Oracle roster inspection connection: %w", closeErr))
		}
	}()
	queryCtx, cancelQuery := context.WithTimeout(ctx, cfg.QueryTimeout)
	defer cancelQuery()
	connection, err := database.Conn(queryCtx)
	if err != nil {
		return inspection, fmt.Errorf("reserve Oracle roster inspection connection: %w", err)
	}
	defer func() {
		if closeErr := connection.Close(); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("release Oracle roster inspection connection: %w", closeErr))
		}
	}()
	connectCtx, cancelConnect := context.WithTimeout(queryCtx, cfg.ConnectTimeout)
	err = connection.PingContext(connectCtx)
	cancelConnect()
	if err != nil {
		return inspection, fmt.Errorf("connect to Oracle roster source: %w", err)
	}
	securityProbe, err := probeOracleRuntimeSecurity(
		queryCtx, connection, cfg.ExpectedUsername, cfg.Schema, cfg.Table,
	)
	if err != nil {
		return inspection, fmt.Errorf("verify Oracle roster runtime identity: %w", err)
	}
	metadata, err := inspectOracleRosterSourceMetadata(queryCtx, connection, cfg.Schema, cfg.Table)
	if err != nil {
		return inspection, err
	}
	if err := validateConfiguredOracleRosterColumns(metadata, cfg); err != nil {
		return inspection, err
	}
	inspection = OracleRosterSourceInspection{
		ObjectType: metadata.ObjectType, Columns: metadata.Columns,
		RuntimeIdentityMatched: securityProbe.IdentityMatched,
		LeastPrivilegeVerified: securityProbe.LeastPrivilegeVerified,
	}
	return inspection, nil
}

type oracleRosterSourceMetadata struct {
	ObjectType string
	Columns    []OracleRosterSourceColumn
}

type oracleRosterMetadataQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func inspectOracleRosterSourceMetadata(
	ctx context.Context,
	queryer oracleRosterMetadataQueryer,
	schema string,
	table string,
) (oracleRosterSourceMetadata, error) {
	owner := strings.ToUpper(strings.TrimSpace(schema))
	objectName := strings.ToUpper(strings.TrimSpace(table))
	objectRows, err := queryer.QueryContext(ctx, `
		SELECT OBJECT_TYPE
		FROM ALL_OBJECTS
		WHERE OWNER = :1 AND OBJECT_NAME = :2 AND OBJECT_TYPE IN ('TABLE', 'VIEW')
		ORDER BY OBJECT_TYPE
	`, owner, objectName)
	if err != nil {
		return oracleRosterSourceMetadata{}, fmt.Errorf("inspect approved Oracle roster object type: %w", err)
	}
	closeObjectRows := func(cause error) error {
		if closeErr := objectRows.Close(); closeErr != nil {
			return errors.Join(cause, fmt.Errorf("close approved Oracle roster object type rows: %w", closeErr))
		}
		return cause
	}
	objectTypes := make([]string, 0, 2)
	for objectRows.Next() {
		var objectType string
		if err := objectRows.Scan(&objectType); err != nil {
			return oracleRosterSourceMetadata{}, closeObjectRows(
				fmt.Errorf("scan approved Oracle roster object type: %w", err),
			)
		}
		objectTypes = append(objectTypes, strings.ToUpper(strings.TrimSpace(objectType)))
	}
	if err := objectRows.Err(); err != nil {
		return oracleRosterSourceMetadata{}, closeObjectRows(
			fmt.Errorf("iterate approved Oracle roster object type: %w", err),
		)
	}
	if err := closeObjectRows(nil); err != nil {
		return oracleRosterSourceMetadata{}, err
	}
	if len(objectTypes) != 1 || (objectTypes[0] != "TABLE" && objectTypes[0] != "VIEW") {
		return oracleRosterSourceMetadata{}, errors.New("oracle roster source must resolve to exactly one approved table or view")
	}

	columnRows, err := queryer.QueryContext(ctx, `
		SELECT COLUMN_NAME, DATA_TYPE, NULLABLE
		FROM ALL_TAB_COLUMNS
		WHERE OWNER = :1 AND TABLE_NAME = :2
		ORDER BY COLUMN_ID
	`, owner, objectName)
	if err != nil {
		return oracleRosterSourceMetadata{}, fmt.Errorf("inspect approved Oracle roster columns: %w", err)
	}
	closeColumnRows := func(cause error) error {
		if closeErr := columnRows.Close(); closeErr != nil {
			return errors.Join(cause, fmt.Errorf("close approved Oracle roster column rows: %w", closeErr))
		}
		return cause
	}
	columns := make([]OracleRosterSourceColumn, 0, 64)
	for columnRows.Next() {
		if len(columns) >= 512 {
			return oracleRosterSourceMetadata{}, closeColumnRows(
				errors.New("oracle roster source exposes too many columns"),
			)
		}
		var name, dataType, nullable string
		if err := columnRows.Scan(&name, &dataType, &nullable); err != nil {
			return oracleRosterSourceMetadata{}, closeColumnRows(
				fmt.Errorf("scan approved Oracle roster column: %w", err),
			)
		}
		columns = append(columns, OracleRosterSourceColumn{
			Name: strings.ToUpper(strings.TrimSpace(name)), DataType: strings.ToUpper(strings.TrimSpace(dataType)),
			Nullable: strings.EqualFold(strings.TrimSpace(nullable), "Y"),
		})
	}
	if err := columnRows.Err(); err != nil {
		return oracleRosterSourceMetadata{}, closeColumnRows(
			fmt.Errorf("iterate approved Oracle roster columns: %w", err),
		)
	}
	if err := closeColumnRows(nil); err != nil {
		return oracleRosterSourceMetadata{}, err
	}
	if len(columns) == 0 {
		return oracleRosterSourceMetadata{}, errors.New("oracle roster source exposes no readable column metadata")
	}
	return oracleRosterSourceMetadata{ObjectType: objectTypes[0], Columns: columns}, nil
}

func validateConfiguredOracleRosterColumns(
	metadata oracleRosterSourceMetadata,
	cfg OracleRosterSnapshotConfig,
) error {
	available := make(map[string]struct{}, len(metadata.Columns))
	for _, column := range metadata.Columns {
		available[strings.ToUpper(strings.TrimSpace(column.Name))] = struct{}{}
	}
	configured := []string{
		cfg.Columns.StudentID, cfg.Columns.Name, cfg.Columns.DocumentType, cfg.Columns.DocumentNumber,
		cfg.Columns.Phone, cfg.Columns.StudentStatus, cfg.Columns.OnCampusStatus,
		cfg.Columns.RegistrationStatus, cfg.Columns.EducationLevel, cfg.Columns.StudentCategory,
		cfg.Columns.EnrollmentYear, cfg.Columns.ValidFrom, cfg.Columns.ValidUntil,
		cfg.Columns.CurrentMarker, cfg.Columns.EligibilityCode, cfg.Columns.SourceUpdatedAt,
		cfg.ActiveFilterColumn,
	}
	for _, name := range configured {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, ok := available[strings.ToUpper(strings.TrimSpace(name))]; !ok {
			return errors.New("configured Oracle roster column is absent from the approved source object")
		}
	}
	return nil
}

func validateOracleRosterSnapshotConfig(cfg OracleRosterSnapshotConfig) error {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port < 1 || cfg.Port > 65535 ||
		strings.TrimSpace(cfg.ServiceName) == "" ||
		strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" ||
		strings.TrimSpace(cfg.ExpectedUsername) == "" ||
		!oracleIdentifierPattern.MatchString(cfg.Schema) || !oracleIdentifierPattern.MatchString(cfg.Table) ||
		!oracleIdentifierPattern.MatchString(cfg.ActiveFilterColumn) ||
		strings.TrimSpace(cfg.ActiveFilterValue) == "" || len(cfg.ActiveFilterValue) > 128 ||
		strings.TrimSpace(cfg.ActiveEligibilityCode) == "" || len(cfg.ActiveEligibilityCode) > 128 ||
		cfg.MaximumRows < 1 || cfg.ConnectTimeout <= 0 || cfg.QueryTimeout <= 0 {
		return errors.New("oracle roster snapshot configuration is incomplete")
	}
	if isDisallowedOracleRuntimeUsername(cfg.Username) ||
		!strings.EqualFold(strings.TrimSpace(cfg.Username), strings.TrimSpace(cfg.ExpectedUsername)) {
		return errors.New("oracle roster runtime identity must match the configured existing account")
	}
	switch cfg.TransportMode {
	case "oracle_tls":
		if strings.TrimSpace(cfg.TLSServerName) == "" ||
			net.ParseIP(strings.TrimSpace(cfg.TLSServerName)) != nil || strings.TrimSpace(cfg.CAFile) == "" {
			return errors.New("oracle TLS roster transport requires a non-IP certificate name and CA file")
		}
	case "oracle_ssh_tunnel":
		hostIP := net.ParseIP(strings.TrimSpace(cfg.Host))
		if hostIP == nil || !hostIP.IsLoopback() || cfg.Port < 1024 ||
			strings.TrimSpace(cfg.TLSServerName) != "" || strings.TrimSpace(cfg.CAFile) != "" {
			return errors.New("oracle SSH tunnel roster transport requires a loopback high port and no Oracle TLS fields")
		}
	default:
		return errors.New("oracle roster transport mode is not approved")
	}
	if strings.ContainsFunc(cfg.ActiveFilterValue, unicode.IsControl) {
		return errors.New("oracle roster active filter value is invalid")
	}
	required := []string{
		cfg.Columns.StudentID, cfg.Columns.Name,
		cfg.Columns.CurrentMarker, cfg.Columns.EligibilityCode,
	}
	for _, identifier := range required {
		if !oracleIdentifierPattern.MatchString(identifier) {
			return errors.New("oracle roster snapshot required column is invalid")
		}
	}
	optional := []string{
		cfg.Columns.DocumentType, cfg.Columns.DocumentNumber, cfg.Columns.Phone,
		cfg.Columns.StudentStatus, cfg.Columns.OnCampusStatus,
		cfg.Columns.RegistrationStatus, cfg.Columns.EducationLevel,
		cfg.Columns.StudentCategory, cfg.Columns.EnrollmentYear,
		cfg.Columns.ValidFrom, cfg.Columns.ValidUntil, cfg.Columns.SourceUpdatedAt,
	}
	for _, identifier := range optional {
		if identifier != "" && !oracleIdentifierPattern.MatchString(identifier) {
			return errors.New("oracle roster snapshot optional column is invalid")
		}
	}
	filterColumns := []string{
		cfg.Columns.OnCampusStatus,
		cfg.Columns.CurrentMarker,
		cfg.Columns.EligibilityCode,
	}
	filterApproved := false
	for _, identifier := range filterColumns {
		if identifier != "" && strings.EqualFold(identifier, cfg.ActiveFilterColumn) {
			filterApproved = true
			break
		}
	}
	if !filterApproved {
		return errors.New("oracle roster active filter must use an approved status column")
	}
	if _, err := newOracleAllowlistedDialer(cfg); err != nil {
		return err
	}
	return nil
}

func openOracleRosterDatabase(cfg OracleRosterSnapshotConfig) (*sql.DB, error) {
	dialer, err := newOracleAllowlistedDialer(cfg)
	if err != nil {
		return nil, err
	}
	options := map[string]string{
		"CONNECTION TIMEOUT": strconvSeconds(cfg.ConnectTimeout),
		"SOCKET TIMEOUT":     strconvSeconds(cfg.QueryTimeout),
	}
	var connector driver.Connector
	switch cfg.TransportMode {
	case "oracle_tls":
		rootCAs, err := loadOracleTLSRootCAs(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		options["SSL"] = "true"
		options["SSL VERIFY"] = "true"
		dsn := go_ora.BuildUrl(cfg.Host, cfg.Port, cfg.ServiceName, cfg.Username, cfg.Password, options)
		connector, err = newVerifiedTLSOracleConnectorWithDialer(dsn, &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.TLSServerName,
			RootCAs:    rootCAs,
		}, dialer)
		if err != nil {
			return nil, err
		}
	case "oracle_ssh_tunnel":
		dsn := go_ora.BuildUrl(cfg.Host, cfg.Port, cfg.ServiceName, cfg.Username, cfg.Password, options)
		connector, err = newAllowlistedOracleConnectorWithDialer(dsn, dialer)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("oracle roster transport mode is not approved")
	}
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(0)
	database.SetConnMaxLifetime(cfg.QueryTimeout)
	return database, nil
}

func buildOracleFullRosterQuery(cfg OracleRosterSnapshotConfig) string {
	columns := []string{
		oracleColumn(cfg.Columns.StudentID),
		oracleColumn(cfg.Columns.Name),
		oracleOptionalColumn(cfg.Columns.DocumentType),
		oracleOptionalColumn(cfg.Columns.DocumentNumber),
		oracleOptionalColumn(cfg.Columns.Phone),
		oracleOptionalColumn(cfg.Columns.StudentStatus),
		oracleOptionalColumn(cfg.Columns.OnCampusStatus),
		oracleOptionalColumn(cfg.Columns.RegistrationStatus),
		oracleOptionalColumn(cfg.Columns.EducationLevel),
		oracleOptionalColumn(cfg.Columns.StudentCategory),
		oracleOptionalColumn(cfg.Columns.EnrollmentYear),
		oracleOptionalColumn(cfg.Columns.ValidFrom),
		oracleOptionalColumn(cfg.Columns.ValidUntil),
		oracleColumn(cfg.Columns.CurrentMarker),
		oracleColumn(cfg.Columns.EligibilityCode),
		oracleOptionalColumn(cfg.Columns.SourceUpdatedAt),
	}
	return fmt.Sprintf(
		"SELECT %s FROM %s.%s WHERE %s = :1 ORDER BY %s",
		strings.Join(columns, ", "), cfg.Schema, cfg.Table,
		cfg.ActiveFilterColumn, cfg.Columns.StudentID,
	)
}

func oracleColumn(identifier string) string { return identifier }

func oracleOptionalColumn(identifier string) string {
	if identifier == "" {
		return "CAST(NULL AS VARCHAR2(1))"
	}
	return identifier
}

func mapOracleRosterRow(values []sql.NullString, defaultDocumentType string) (connectorprotocol.RosterRecord, error) {
	record, _, err := mapOracleRosterRowWithQuality(values, defaultDocumentType, "")
	return record, err
}

func mapOracleRosterRowWithQuality(
	values []sql.NullString,
	defaultDocumentType string,
	activeEligibilityCode string,
) (connectorprotocol.RosterRecord, oracleRosterRowQuality, error) {
	quality := oracleRosterRowQuality{}
	if len(values) != 16 || !values[0].Valid || strings.TrimSpace(values[0].String) == "" ||
		!values[1].Valid || strings.TrimSpace(values[1].String) == "" ||
		!values[13].Valid || !values[14].Valid || strings.TrimSpace(values[14].String) == "" {
		return connectorprotocol.RosterRecord{}, oracleRosterRowQuality{}, errors.New("oracle roster row is missing a required field")
	}
	var enrollmentYear *int
	if !values[10].Valid || strings.TrimSpace(values[10].String) == "" {
		quality.MissingEnrollmentYear++
	} else if parsed, err := strconv.Atoi(strings.TrimSpace(values[10].String)); err != nil || parsed < 1900 || parsed > 3000 {
		quality.InvalidEnrollmentYear++
	} else {
		enrollmentYear = &parsed
	}
	currentMarker, err := parseOracleBoolean(values[13].String)
	if err != nil {
		return connectorprotocol.RosterRecord{}, oracleRosterRowQuality{}, err
	}
	validFrom, err := parseOracleOptionalTime(values[11])
	if err != nil {
		return connectorprotocol.RosterRecord{}, oracleRosterRowQuality{}, errors.New("oracle roster valid-from value is invalid")
	}
	validUntil, err := parseOracleOptionalTime(values[12])
	if err != nil {
		return connectorprotocol.RosterRecord{}, oracleRosterRowQuality{}, errors.New("oracle roster valid-until value is invalid")
	}
	sourceUpdatedAt, err := parseOracleOptionalTime(values[15])
	if err != nil {
		return connectorprotocol.RosterRecord{}, oracleRosterRowQuality{}, errors.New("oracle roster source-updated value is invalid")
	}
	documentType := nullString(values[2])
	if documentType == "" {
		documentType = defaultDocumentType
	}
	documentNumber := nullString(values[3])
	if documentNumber == "" {
		quality.MissingDocumentNumber++
		documentType = ""
	} else if strings.EqualFold(strings.TrimSpace(documentType), "mainland_resident_id") {
		normalized, valid := schoolauth.NormalizeMainlandDocumentNumber(documentNumber)
		if !valid {
			quality.InvalidDocumentNumber++
			documentType = ""
			documentNumber = ""
		} else {
			documentNumber = normalized
		}
	}
	phone := nullString(values[4])
	if phone == "" {
		quality.MissingPhone++
	} else if normalized, valid := normalizeOracleMainlandPhone(phone); valid {
		phone = normalized
	} else {
		quality.InvalidPhone++
		phone = ""
	}
	eligibilityCode := strings.TrimSpace(values[14].String)
	if strings.TrimSpace(activeEligibilityCode) != "" {
		eligibilityCode = strings.TrimSpace(activeEligibilityCode)
	}
	return connectorprotocol.RosterRecord{
		StudentID:          strings.TrimSpace(values[0].String),
		Name:               strings.TrimSpace(values[1].String),
		DocumentType:       strings.TrimSpace(documentType),
		DocumentNumber:     documentNumber,
		Phone:              phone,
		StudentStatus:      nullString(values[5]),
		OnCampusStatus:     nullString(values[6]),
		RegistrationStatus: nullString(values[7]),
		EducationLevel:     nullString(values[8]),
		StudentCategory:    nullString(values[9]),
		EnrollmentYear:     enrollmentYear,
		ValidFrom:          validFrom, ValidUntil: validUntil,
		CurrentMarker:   &currentMarker,
		EligibilityCode: eligibilityCode,
		SourceUpdatedAt: sourceUpdatedAt,
	}, quality, nil
}

func normalizeOracleMainlandPhone(value string) (string, bool) {
	normalized := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(normalized, "+86"):
		normalized = strings.TrimPrefix(normalized, "+86")
	case strings.HasPrefix(normalized, "86") && len(normalized) == 13:
		normalized = strings.TrimPrefix(normalized, "86")
	}
	return normalized, phoneutil.IsValidMainlandPhone(normalized)
}

func parseOracleBoolean(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "y", "yes", "true", "是":
		return true, nil
	case "0", "n", "no", "false", "否":
		return false, nil
	default:
		return false, errors.New("oracle roster current-marker value is invalid")
	}
}

func parseOracleOptionalTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(value.String)
	formats := []string{
		time.RFC3339Nano, "2006-01-02 15:04:05 -0700 MST", "2006-01-02 15:04:05",
		"2006-01-02", "20060102",
	}
	for _, format := range formats {
		if parsed, err := time.Parse(format, raw); err == nil {
			value := parsed.UTC()
			return &value, nil
		}
	}
	return nil, errors.New("unsupported oracle date value")
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func clearProtocolRecords(records []connectorprotocol.RosterRecord) {
	for index := range records {
		records[index].StudentID = ""
		records[index].Name = ""
		records[index].DocumentType = ""
		records[index].DocumentNumber = ""
		records[index].Phone = ""
		records[index].EligibilityCode = ""
	}
}

type oracleAllowlistedDialer struct {
	targets       map[string]struct{}
	allowLoopback bool
	dialer        net.Dialer
}

func newOracleAllowlistedDialer(cfg OracleRosterSnapshotConfig) (*oracleAllowlistedDialer, error) {
	if len(cfg.AllowedDialTargets) == 0 || len(cfg.AllowedDialTargets) > 64 {
		return nil, errors.New("oracle roster dial target allowlist must contain between 1 and 64 entries")
	}
	targets := make(map[string]struct{}, len(cfg.AllowedDialTargets))
	for _, raw := range cfg.AllowedDialTargets {
		target, err := normalizeOracleDialTarget(raw, cfg.TransportMode == "oracle_ssh_tunnel")
		if err != nil {
			return nil, err
		}
		targets[target] = struct{}{}
	}
	initial, err := normalizeOracleDialTarget(
		net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), cfg.TransportMode == "oracle_ssh_tunnel",
	)
	if err != nil {
		return nil, fmt.Errorf("normalize initial Oracle roster target: %w", err)
	}
	if _, ok := targets[initial]; !ok {
		return nil, errors.New("oracle roster dial target allowlist must include the configured initial endpoint")
	}
	if cfg.TransportMode == "oracle_ssh_tunnel" && (len(targets) != 1 || len(cfg.AllowedDialTargets) != 1) {
		return nil, errors.New("oracle SSH tunnel must allow exactly one loopback endpoint and no listener redirects")
	}
	return &oracleAllowlistedDialer{
		targets:       targets,
		allowLoopback: cfg.TransportMode == "oracle_ssh_tunnel",
		dialer:        net.Dialer{Timeout: cfg.ConnectTimeout, KeepAlive: 30 * time.Second},
	}, nil
}

func (d *oracleAllowlistedDialer) DialContext(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error) {
	if d == nil || network != "tcp" {
		return nil, errors.New("oracle roster dialer only permits approved TCP targets")
	}
	target, err := normalizeOracleDialTarget(address, d.allowLoopback)
	if err != nil {
		return nil, err
	}
	if _, ok := d.targets[target]; !ok {
		return nil, errors.New("oracle roster listener redirect target is not approved")
	}
	return d.dialer.DialContext(ctx, network, target)
}

func normalizeOracleDialTarget(raw string, allowLoopback bool) (string, error) {
	host, rawPort, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("oracle roster dial target must be an exact host:port endpoint")
	}
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 || host == "" || len(host) > 253 ||
		host == "localhost" || strings.ContainsAny(host, "/?#@%\\") ||
		strings.ContainsFunc(host, unicode.IsSpace) {
		return "", errors.New("oracle roster dial target is invalid")
	}
	if parsed := net.ParseIP(host); parsed != nil &&
		((parsed.IsLoopback() && !allowLoopback) || parsed.IsUnspecified() || parsed.IsMulticast()) {
		return "", errors.New("oracle roster dial target cannot be loopback, unspecified, or multicast")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}
