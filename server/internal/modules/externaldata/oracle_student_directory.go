package externaldata

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"

	"github.com/StuHelper/StuHelper/server/internal/pkg/circuitbreaker"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
)

const defaultOracleQueryTimeout = 3 * time.Second
const defaultOracleConnectTimeout = 5 * time.Second
const defaultOracleMaxOpenConns = 4
const defaultOracleConnMaxLifetime = 5 * time.Minute
const defaultOracleConnMaxIdleTime = time.Minute
const defaultOracleBreakerFailureThreshold = 5
const defaultOracleBreakerSuccessThreshold = 2
const defaultOracleBreakerOpenTimeout = 30 * time.Second
const oracleTLSModeVerifyFull = "verify-full"
const oracleTLSModeDisable = "disable"
const oracleStudentDirectoryMetricClient = "oracle_student_directory"

var oracleIdentifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,127}$`)
var ErrStudentSourceInvalidRecord = errors.New("student source returned an invalid record")
var ErrStudentSourceAmbiguousRecord = errors.New("student source returned ambiguous records")
var ErrStudentSourceInvalidStudentID = errors.New("student source lookup student id is invalid")
var ErrStudentSourceUnavailable = errors.New("student source is unavailable")

type OracleStudentDirectoryConfig struct {
	SchoolCode              string
	Host                    string
	Port                    int
	ServiceName             string
	Username                string
	Password                string
	TLSMode                 string
	TLSCAFile               string
	Schema                  string
	Table                   string
	StudentIDColumn         string
	StudentNameColumn       string
	ConnectTimeout          time.Duration
	QueryTimeout            time.Duration
	MaxOpenConns            int
	MaxIdleConns            int
	ConnMaxLifetime         time.Duration
	ConnMaxIdleTime         time.Duration
	BreakerFailureThreshold int
	BreakerSuccessThreshold int
	BreakerOpenTimeout      time.Duration
}

type OracleStudentDirectory struct {
	db           *sql.DB
	schoolCode   string
	query        string
	probeQuery   string
	queryTimeout time.Duration
	breaker      *circuitbreaker.CircuitBreaker
}

type OracleStudentDirectoryProbe struct {
	ReadableRecordPresent bool
}

func NewOracleStudentDirectory(cfg OracleStudentDirectoryConfig) (*OracleStudentDirectory, error) {
	normalized, err := normalizeOracleStudentDirectoryConfig(cfg)
	if err != nil {
		return nil, err
	}
	options := map[string]string{
		"SOCKET TIMEOUT": strconvSeconds(normalized.QueryTimeout),
	}
	if normalized.ConnectTimeout > 0 {
		options["CONNECTION TIMEOUT"] = strconvSeconds(normalized.ConnectTimeout)
	}
	if normalized.TLSMode == oracleTLSModeVerifyFull {
		options["SSL"] = "true"
		options["SSL VERIFY"] = "true"
	}
	dsn := go_ora.BuildUrl(
		normalized.Host,
		normalized.Port,
		normalized.ServiceName,
		normalized.Username,
		normalized.Password,
		options,
	)

	var db *sql.DB
	if normalized.TLSMode == oracleTLSModeVerifyFull {
		rootCAs, err := loadOracleTLSRootCAs(normalized.TLSCAFile)
		if err != nil {
			return nil, err
		}
		connector, err := newVerifiedTLSOracleConnector(dsn, &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs,
		})
		if err != nil {
			return nil, err
		}
		db = sql.OpenDB(connector)
	} else {
		db, err = sql.Open("oracle", dsn)
		if err != nil {
			return nil, fmt.Errorf("open oracle student directory: %w", err)
		}
	}
	if normalized.MaxOpenConns > 0 {
		db.SetMaxOpenConns(normalized.MaxOpenConns)
	}
	if normalized.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(normalized.MaxIdleConns)
	}
	db.SetConnMaxLifetime(normalized.ConnMaxLifetime)
	db.SetConnMaxIdleTime(normalized.ConnMaxIdleTime)
	directory := newOracleStudentDirectoryWithDB(normalized, db)
	return directory, nil
}

func newOracleStudentDirectoryWithDB(
	cfg OracleStudentDirectoryConfig,
	db *sql.DB,
) *OracleStudentDirectory {
	timeout := cfg.QueryTimeout
	if timeout <= 0 {
		timeout = defaultOracleQueryTimeout
	}
	return &OracleStudentDirectory{
		db:           db,
		schoolCode:   cfg.SchoolCode,
		query:        buildOracleStudentLookupQuery(cfg),
		probeQuery:   buildOracleStudentProbeQuery(cfg),
		queryTimeout: timeout,
		breaker: circuitbreaker.NewNamed(
			"external_student_source_oracle_"+cfg.SchoolCode,
			circuitbreaker.Config{
				FailureThreshold: cfg.BreakerFailureThreshold,
				SuccessThreshold: cfg.BreakerSuccessThreshold,
				Timeout:          cfg.BreakerOpenTimeout,
			},
		),
	}
}

func (d *OracleStudentDirectory) Probe(ctx context.Context) (probe OracleStudentDirectoryProbe, err error) {
	if d == nil || d.db == nil {
		return OracleStudentDirectoryProbe{}, ErrStudentSourceNotConfigured
	}
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(oracleStudentDirectoryMetricClient, "probe", start, err)
	}()
	if !d.breaker.Allow() {
		return OracleStudentDirectoryProbe{}, fmt.Errorf("%w: circuit breaker open", ErrStudentSourceUnavailable)
	}
	pingCtx, cancelPing := withOptionalTimeout(ctx, d.queryTimeout)
	if err := d.db.PingContext(pingCtx); err != nil {
		cancelPing()
		d.breaker.RecordFailure()
		return OracleStudentDirectoryProbe{}, wrapOracleStudentSourceFailure("ping oracle student directory", err)
	}
	cancelPing()

	queryCtx, cancelQuery := withOptionalTimeout(ctx, d.queryTimeout)
	defer cancelQuery()
	var marker int
	if err := d.db.QueryRowContext(queryCtx, d.probeQuery).Scan(&marker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			d.breaker.RecordSuccess()
			return OracleStudentDirectoryProbe{ReadableRecordPresent: false}, nil
		}
		d.breaker.RecordFailure()
		return OracleStudentDirectoryProbe{}, wrapOracleStudentSourceFailure("probe oracle student directory table", err)
	}
	d.breaker.RecordSuccess()
	return OracleStudentDirectoryProbe{ReadableRecordPresent: true}, nil
}

func (d *OracleStudentDirectory) LookupStudent(
	ctx context.Context,
	studentID string,
) (record *StudentRecord, err error) {
	if d == nil || d.db == nil {
		return nil, ErrStudentSourceNotConfigured
	}
	normalizedID, err := normalizeOracleLookupStudentID(studentID)
	if err != nil {
		return nil, err
	}
	if normalizedID == "" {
		return nil, nil
	}
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(oracleStudentDirectoryMetricClient, "lookup", start, err)
	}()
	if !d.breaker.Allow() {
		return nil, fmt.Errorf("%w: circuit breaker open", ErrStudentSourceUnavailable)
	}
	queryCtx, cancel := withOptionalTimeout(ctx, d.queryTimeout)
	defer cancel()

	rows, err := d.db.QueryContext(queryCtx, d.query, normalizedID)
	if err != nil {
		d.breaker.RecordFailure()
		return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
	}
	rowsClosed := false
	closeRows := func() error {
		if rowsClosed {
			return nil
		}
		rowsClosed = true
		return rows.Close()
	}
	defer func() {
		if closeErr := closeRows(); closeErr != nil && err == nil {
			d.breaker.RecordFailure()
			record = nil
			err = wrapOracleStudentSourceFailure("close oracle student record rows", closeErr)
		}
	}()
	record, err = scanOracleStudentRecords(rows, d.schoolCode, normalizedID)
	if err != nil {
		d.breaker.RecordFailure()
		return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
	}
	if err := closeRows(); err != nil {
		d.breaker.RecordFailure()
		return nil, wrapOracleStudentSourceFailure("close oracle student record rows", err)
	}
	d.breaker.RecordSuccess()
	return record, nil
}

func wrapOracleStudentSourceFailure(operation string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrStudentSourceUnavailable, operation, err)
}

func normalizeOracleLookupStudentID(studentID string) (string, error) {
	normalized := schoolauth.NormalizeStudentID(studentID)
	if normalized == "" {
		return "", nil
	}
	if !schoolauth.IsValidStudentID(normalized) {
		return "", ErrStudentSourceInvalidStudentID
	}
	return normalized, nil
}

func (d *OracleStudentDirectory) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

func normalizeOracleStudentDirectoryConfig(cfg OracleStudentDirectoryConfig) (OracleStudentDirectoryConfig, error) {
	var err error
	cfg.SchoolCode, err = normalizeSchoolCode(cfg.SchoolCode)
	if err != nil {
		return cfg, err
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.ServiceName = strings.TrimSpace(cfg.ServiceName)
	cfg.Username = strings.TrimSpace(cfg.Username)
	if cfg.Host == "" || cfg.ServiceName == "" || cfg.Username == "" || cfg.Password == "" {
		return cfg, fmt.Errorf("oracle host, service name, username and password are required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("oracle port must be between 1 and 65535")
	}
	if cfg.Schema, err = normalizeOracleIdentifier(cfg.Schema, "schema"); err != nil {
		return cfg, err
	}
	if cfg.Table, err = normalizeOracleIdentifier(cfg.Table, "table"); err != nil {
		return cfg, err
	}
	if cfg.StudentIDColumn, err = normalizeOracleIdentifier(cfg.StudentIDColumn, "student id column"); err != nil {
		return cfg, err
	}
	if cfg.StudentNameColumn, err = normalizeOracleIdentifier(cfg.StudentNameColumn, "student name column"); err != nil {
		return cfg, err
	}
	if cfg.QueryTimeout <= 0 {
		cfg.QueryTimeout = defaultOracleQueryTimeout
	}
	if cfg.QueryTimeout < time.Second || cfg.QueryTimeout > time.Minute {
		return cfg, fmt.Errorf("oracle query timeout must be between 1 and 60 seconds")
	}
	if cfg.ConnectTimeout < 0 {
		return cfg, fmt.Errorf("oracle connection timeout must not be negative")
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = defaultOracleConnectTimeout
	}
	if cfg.ConnectTimeout < time.Second || cfg.ConnectTimeout > time.Minute {
		return cfg, fmt.Errorf("oracle connection timeout must be between 1 and 60 seconds")
	}
	if cfg.MaxOpenConns < 0 {
		return cfg, fmt.Errorf("oracle max open connections must not be negative")
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = defaultOracleMaxOpenConns
	}
	if cfg.MaxOpenConns > 100 {
		return cfg, fmt.Errorf("oracle max open connections must not exceed 100")
	}
	if cfg.MaxIdleConns < 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
		return cfg, fmt.Errorf("oracle max idle connections must be between 0 and max open connections")
	}
	cfg.TLSMode = strings.ToLower(strings.TrimSpace(cfg.TLSMode))
	if cfg.TLSMode == "" {
		cfg.TLSMode = oracleTLSModeVerifyFull
	}
	switch cfg.TLSMode {
	case oracleTLSModeVerifyFull:
		cfg.TLSCAFile = strings.TrimSpace(cfg.TLSCAFile)
		if cfg.TLSCAFile == "" {
			return cfg, fmt.Errorf("oracle TLS CA file is required when TLS mode is verify-full")
		}
	case oracleTLSModeDisable:
		cfg.TLSCAFile = ""
	default:
		return cfg, fmt.Errorf("oracle TLS mode must be verify-full or disable")
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = defaultOracleConnMaxLifetime
	}
	if cfg.ConnMaxLifetime < 30*time.Second || cfg.ConnMaxLifetime > time.Hour {
		return cfg, fmt.Errorf("oracle connection max lifetime must be between 30 and 3600 seconds")
	}
	if cfg.ConnMaxIdleTime < 0 {
		return cfg, fmt.Errorf("oracle connection max idle time must not be negative")
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = defaultOracleConnMaxIdleTime
	}
	if cfg.ConnMaxIdleTime < 30*time.Second {
		return cfg, fmt.Errorf("oracle connection max idle time must be at least 30 seconds")
	}
	if cfg.ConnMaxIdleTime > cfg.ConnMaxLifetime {
		return cfg, fmt.Errorf("oracle connection max idle time must not exceed max lifetime")
	}
	if cfg.BreakerFailureThreshold < 0 {
		return cfg, fmt.Errorf("oracle circuit breaker failure threshold must not be negative")
	}
	if cfg.BreakerFailureThreshold == 0 {
		cfg.BreakerFailureThreshold = defaultOracleBreakerFailureThreshold
	}
	if cfg.BreakerFailureThreshold > 100 {
		return cfg, fmt.Errorf("oracle circuit breaker failure threshold must not exceed 100")
	}
	if cfg.BreakerSuccessThreshold < 0 {
		return cfg, fmt.Errorf("oracle circuit breaker success threshold must not be negative")
	}
	if cfg.BreakerSuccessThreshold == 0 {
		cfg.BreakerSuccessThreshold = defaultOracleBreakerSuccessThreshold
	}
	if cfg.BreakerSuccessThreshold > 20 {
		return cfg, fmt.Errorf("oracle circuit breaker success threshold must not exceed 20")
	}
	if cfg.BreakerOpenTimeout < 0 {
		return cfg, fmt.Errorf("oracle circuit breaker open timeout must not be negative")
	}
	if cfg.BreakerOpenTimeout == 0 {
		cfg.BreakerOpenTimeout = defaultOracleBreakerOpenTimeout
	}
	if cfg.BreakerOpenTimeout < time.Second || cfg.BreakerOpenTimeout > 10*time.Minute {
		return cfg, fmt.Errorf("oracle circuit breaker open timeout must be between 1 and 600 seconds")
	}
	return cfg, nil
}

func buildOracleStudentLookupQuery(cfg OracleStudentDirectoryConfig) string {
	return fmt.Sprintf(
		"SELECT %s, %s FROM %s.%s WHERE %s = :1 FETCH FIRST 2 ROWS ONLY",
		cfg.StudentIDColumn,
		cfg.StudentNameColumn,
		cfg.Schema,
		cfg.Table,
		cfg.StudentIDColumn,
	)
}

func buildOracleStudentProbeQuery(cfg OracleStudentDirectoryConfig) string {
	return fmt.Sprintf(
		"SELECT 1 FROM %s.%s WHERE TRIM(%s) IS NOT NULL AND TRIM(%s) IS NOT NULL FETCH FIRST 1 ROWS ONLY",
		cfg.Schema,
		cfg.Table,
		cfg.StudentIDColumn,
		cfg.StudentNameColumn,
	)
}

func normalizeOracleIdentifier(value string, label string) (string, error) {
	identifier := strings.ToUpper(strings.TrimSpace(value))
	if !oracleIdentifierPattern.MatchString(identifier) {
		return "", fmt.Errorf("oracle %s identifier is invalid", label)
	}
	return identifier, nil
}

func strconvSeconds(duration time.Duration) string {
	seconds := int64(duration / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}

type sqlRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanOracleStudentRecords(
	rows sqlRows,
	schoolCode string,
	expectedStudentID string,
) (*StudentRecord, error) {
	var record *StudentRecord
	for rows.Next() {
		var id string
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		id = strings.TrimSpace(id)
		studentName := schoolauth.NormalizeAcademicName(name.String)
		if id == "" || id != expectedStudentID || !schoolauth.IsValidStudentID(id) ||
			!name.Valid || !schoolauth.IsValidAcademicName(studentName) {
			return nil, ErrStudentSourceInvalidRecord
		}
		if record == nil {
			record = &StudentRecord{
				SchoolCode:  schoolCode,
				StudentID:   id,
				StudentName: studentName,
			}
			continue
		}
		if record.StudentID != id || record.StudentName != studentName {
			return nil, ErrStudentSourceAmbiguousRecord
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return record, nil
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

type verifiedTLSOracleConnector struct {
	driver        driver.Driver
	driverContext driver.DriverContext
	dsn           string
	tlsConfig     *tls.Config
}

func newVerifiedTLSOracleConnector(
	dsn string,
	tlsConfig *tls.Config,
) (*verifiedTLSOracleConnector, error) {
	if tlsConfig == nil || tlsConfig.RootCAs == nil || tlsConfig.InsecureSkipVerify {
		return nil, fmt.Errorf("oracle verified TLS configuration is incomplete")
	}
	base := go_ora.NewConnector(dsn)
	baseDriver := base.Driver()
	driverContext, ok := baseDriver.(driver.DriverContext)
	if !ok {
		return nil, fmt.Errorf("oracle driver does not support context-aware connectors")
	}
	return &verifiedTLSOracleConnector{
		driver:        baseDriver,
		driverContext: driverContext,
		dsn:           dsn,
		tlsConfig:     tlsConfig.Clone(),
	}, nil
}

func (c *verifiedTLSOracleConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connector, err := c.driverContext.OpenConnector(c.dsn)
	if err != nil {
		return nil, err
	}
	oracleConnector, ok := connector.(*go_ora.OracleConnector)
	if !ok {
		return nil, fmt.Errorf("oracle connector type does not support verified TLS")
	}
	// go-ora assigns ServerName on the supplied tls.Config. Give every
	// database/sql connection its own clone so concurrent dials never mutate a
	// shared configuration object.
	oracleConnector.WithTLSConfig(c.tlsConfig.Clone())
	return oracleConnector.Connect(ctx)
}

func (c *verifiedTLSOracleConnector) Driver() driver.Driver {
	return c.driver
}

func loadOracleTLSRootCAs(path string) (*x509.CertPool, error) {
	// #nosec G304 -- the operator-configured CA path is intentionally read here;
	// the parser below accepts CA certificates only, and production mounts this
	// path read-only from the dedicated Oracle client trust bundle.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read oracle TLS CA file: %w", err)
	}
	roots := x509.NewCertPool()
	certificateCount := 0
	rest := data
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			if strings.TrimSpace(string(rest)) != "" {
				return nil, fmt.Errorf("oracle TLS CA file contains non-PEM data")
			}
			break
		}
		rest = remaining
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("oracle TLS CA file must contain certificates only")
		}
		certificates, err := x509.ParseCertificates(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse oracle TLS CA certificate: %w", err)
		}
		for _, certificate := range certificates {
			if !certificate.IsCA {
				return nil, fmt.Errorf("oracle TLS CA file contains a non-CA certificate")
			}
			roots.AddCert(certificate)
			certificateCount++
		}
	}
	if certificateCount == 0 {
		return nil, fmt.Errorf("oracle TLS CA file contains no certificates")
	}
	return roots, nil
}
