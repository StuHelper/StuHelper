package externaldata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

const defaultOracleQueryTimeout = 3 * time.Second

var oracleIdentifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,127}$`)

type OracleStudentDirectoryConfig struct {
	SchoolCode        string
	Host              string
	Port              int
	ServiceName       string
	Username          string
	Password          string
	Schema            string
	Table             string
	StudentIDColumn   string
	StudentNameColumn string
	ConnectTimeout    time.Duration
	QueryTimeout      time.Duration
	MaxOpenConns      int
	MaxIdleConns      int
}

type OracleStudentDirectory struct {
	db           *sql.DB
	schoolCode   string
	query        string
	probeQuery   string
	queryTimeout time.Duration
}

type OracleStudentDirectoryProbe struct {
	ReadableRecordPresent bool
}

func NewOracleStudentDirectory(cfg OracleStudentDirectoryConfig) (*OracleStudentDirectory, error) {
	normalized, err := normalizeOracleStudentDirectoryConfig(cfg)
	if err != nil {
		return nil, err
	}
	options := map[string]string{}
	if normalized.ConnectTimeout > 0 {
		options["CONNECTION TIMEOUT"] = strconvSeconds(normalized.ConnectTimeout)
	}
	dsn := go_ora.BuildUrl(
		normalized.Host,
		normalized.Port,
		normalized.ServiceName,
		normalized.Username,
		normalized.Password,
		options,
	)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("open oracle student directory: %w", err)
	}
	if normalized.MaxOpenConns > 0 {
		db.SetMaxOpenConns(normalized.MaxOpenConns)
	}
	if normalized.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(normalized.MaxIdleConns)
	}
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
	}
}

func (d *OracleStudentDirectory) Probe(ctx context.Context) (OracleStudentDirectoryProbe, error) {
	if d == nil || d.db == nil {
		return OracleStudentDirectoryProbe{}, ErrStudentSourceNotConfigured
	}
	queryCtx := ctx
	cancel := func() {}
	if d.queryTimeout > 0 {
		queryCtx, cancel = context.WithTimeout(ctx, d.queryTimeout)
	}
	defer cancel()

	if err := d.db.PingContext(queryCtx); err != nil {
		return OracleStudentDirectoryProbe{}, fmt.Errorf("ping oracle student directory: %w", err)
	}

	var marker int
	if err := d.db.QueryRowContext(queryCtx, d.probeQuery).Scan(&marker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OracleStudentDirectoryProbe{ReadableRecordPresent: false}, nil
		}
		return OracleStudentDirectoryProbe{}, fmt.Errorf("probe oracle student directory table: %w", err)
	}
	return OracleStudentDirectoryProbe{ReadableRecordPresent: true}, nil
}

func (d *OracleStudentDirectory) LookupStudent(ctx context.Context, studentID string) (*StudentRecord, error) {
	if d == nil || d.db == nil {
		return nil, ErrStudentSourceNotConfigured
	}
	normalizedID := strings.TrimSpace(studentID)
	if normalizedID == "" {
		return nil, nil
	}
	queryCtx := ctx
	cancel := func() {}
	if d.queryTimeout > 0 {
		queryCtx, cancel = context.WithTimeout(ctx, d.queryTimeout)
	}
	defer cancel()

	var id string
	var name sql.NullString
	if err := d.db.QueryRowContext(queryCtx, d.query, normalizedID).Scan(&id, &name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup oracle student record: %w", err)
	}
	return &StudentRecord{
		SchoolCode:  d.schoolCode,
		StudentID:   strings.TrimSpace(id),
		StudentName: strings.TrimSpace(name.String),
	}, nil
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
	cfg.Password = strings.TrimSpace(cfg.Password)
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
	return cfg, nil
}

func buildOracleStudentLookupQuery(cfg OracleStudentDirectoryConfig) string {
	return fmt.Sprintf(
		"SELECT %s, %s FROM %s.%s WHERE %s = :1 FETCH FIRST 1 ROWS ONLY",
		cfg.StudentIDColumn,
		cfg.StudentNameColumn,
		cfg.Schema,
		cfg.Table,
		cfg.StudentIDColumn,
	)
}

func buildOracleStudentProbeQuery(cfg OracleStudentDirectoryConfig) string {
	return fmt.Sprintf(
		"SELECT 1 FROM %s.%s WHERE %s IS NOT NULL AND %s IS NOT NULL FETCH FIRST 1 ROWS ONLY",
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
