// Command external-student-source-smoke verifies external student data sources
// without printing credentials or raw student records.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/StuHelper/StuHelper/server/internal/modules/externaldata"
)

const defaultSmokeTimeout = 15 * time.Second

type smokeConfig struct {
	SourceName            string
	Provider              string
	RequireReadableRecord bool
	RequireSample         bool
	SampleStudentID       string
	SampleExpectedName    string
	Timeout               time.Duration
	Oracle                externaldata.OracleStudentDirectoryConfig
}

type smokeEvidence struct {
	CheckedAt             string               `json:"checkedAt"`
	Provider              string               `json:"provider"`
	ReadableRecordPresent bool                 `json:"readableRecordPresent"`
	SchoolCode            string               `json:"schoolCode"`
	SourceName            string               `json:"sourceName"`
	TimeoutSeconds        int64                `json:"timeoutSeconds"`
	Oracle                oracleEvidence       `json:"oracle"`
	Sample                sampleLookupEvidence `json:"sample"`
}

type oracleEvidence struct {
	ApprovedObjectGrantCount     int    `json:"approvedObjectGrantCount"`
	ApprovedSystemGrantCount     int    `json:"approvedSystemGrantCount"`
	BreakerFailureThreshold      int    `json:"breakerFailureThreshold"`
	BreakerOpenSeconds           int64  `json:"breakerOpenSeconds"`
	BreakerSuccessThreshold      int    `json:"breakerSuccessThreshold"`
	ConnMaxIdleTimeSeconds       int64  `json:"connMaxIdleTimeSeconds"`
	ConnMaxLifetimeSeconds       int64  `json:"connMaxLifetimeSeconds"`
	HostConfigured               bool   `json:"hostConfigured"`
	ExpectedIdentityConfigured   bool   `json:"expectedIdentityConfigured"`
	RuntimeIdentityHashPrefix    string `json:"runtimeIdentityHashPrefix"`
	RuntimeIdentityMatched       bool   `json:"runtimeIdentityMatched"`
	LeastPrivilegeGrantsVerified bool   `json:"leastPrivilegeGrantsVerified"`
	RoleGrantCount               int    `json:"roleGrantCount"`
	SystemGrantCount             int    `json:"systemGrantCount"`
	ObjectGrantCount             int    `json:"objectGrantCount"`
	ColumnGrantCount             int    `json:"columnGrantCount"`
	MaxIdleConns                 int    `json:"maxIdleConns"`
	MaxOpenConns                 int    `json:"maxOpenConns"`
	Port                         int    `json:"port"`
	Schema                       string `json:"schema"`
	ServiceName                  string `json:"serviceName"`
	StudentIDColumn              string `json:"studentIDColumn"`
	StudentNameColumn            string `json:"studentNameColumn"`
	Table                        string `json:"table"`
	TLSMode                      string `json:"tlsMode"`
	TLSVerified                  bool   `json:"tlsVerified"`
	UsernameConfigured           bool   `json:"usernameConfigured"`
}

type sampleLookupEvidence struct {
	Enabled              bool   `json:"enabled"`
	ExpectedNameProvided bool   `json:"expectedNameProvided"`
	Found                bool   `json:"found"`
	NameMatched          *bool  `json:"nameMatched,omitempty"`
	NamePresent          bool   `json:"namePresent"`
	StudentIDHashPrefix  string `json:"studentIDHashPrefix,omitempty"`
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "external-student-source-smoke: warning: failed to load .env: %v\n", err)
	}

	evidence, err := run(context.Background(), os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "external-student-source-smoke: %s\n", redactSensitiveText(err.Error(), os.Getenv))
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		fmt.Fprintf(os.Stderr, "external-student-source-smoke: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string) (evidence smokeEvidence, runErr error) {
	cfg, err := smokeConfigFromEnv(getenv)
	if err != nil {
		return smokeEvidence{}, err
	}

	directory, err := externaldata.NewOracleStudentDirectory(cfg.Oracle)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("initialize oracle student source: %w", err)
	}
	defer func() {
		if err := directory.Close(); err != nil && runErr == nil {
			runErr = fmt.Errorf("close oracle student source: %w", err)
		}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	securityProbe, err := directory.ProbeRuntimeSecurity(timeoutCtx)
	if err != nil {
		return smokeEvidence{}, err
	}
	probe, err := directory.Probe(timeoutCtx)
	if err != nil {
		return smokeEvidence{}, err
	}
	if cfg.RequireReadableRecord && !probe.ReadableRecordPresent {
		return smokeEvidence{}, fmt.Errorf("oracle student source has no readable record with configured student id/name columns")
	}

	evidence = smokeEvidence{
		CheckedAt:             time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		Provider:              cfg.Provider,
		ReadableRecordPresent: probe.ReadableRecordPresent,
		SchoolCode:            cfg.Oracle.SchoolCode,
		SourceName:            cfg.SourceName,
		TimeoutSeconds:        int64(cfg.Timeout / time.Second),
		Oracle: oracleEvidence{
			ApprovedObjectGrantCount:     securityProbe.ApprovedObjectGrantCount,
			ApprovedSystemGrantCount:     securityProbe.ApprovedSystemGrantCount,
			BreakerFailureThreshold:      cfg.Oracle.BreakerFailureThreshold,
			BreakerOpenSeconds:           int64(cfg.Oracle.BreakerOpenTimeout / time.Second),
			BreakerSuccessThreshold:      cfg.Oracle.BreakerSuccessThreshold,
			ConnMaxIdleTimeSeconds:       int64(cfg.Oracle.ConnMaxIdleTime / time.Second),
			ConnMaxLifetimeSeconds:       int64(cfg.Oracle.ConnMaxLifetime / time.Second),
			HostConfigured:               strings.TrimSpace(cfg.Oracle.Host) != "",
			ExpectedIdentityConfigured:   strings.TrimSpace(cfg.Oracle.ExpectedUsername) != "",
			RuntimeIdentityHashPrefix:    hashPrefix(securityProbe.SessionUsername),
			RuntimeIdentityMatched:       securityProbe.IdentityMatched,
			LeastPrivilegeGrantsVerified: securityProbe.LeastPrivilegeVerified,
			RoleGrantCount:               securityProbe.RoleGrantCount,
			SystemGrantCount:             securityProbe.SystemGrantCount,
			ObjectGrantCount:             securityProbe.ObjectGrantCount,
			ColumnGrantCount:             securityProbe.ColumnGrantCount,
			MaxIdleConns:                 cfg.Oracle.MaxIdleConns,
			MaxOpenConns:                 cfg.Oracle.MaxOpenConns,
			Port:                         cfg.Oracle.Port,
			Schema:                       strings.ToUpper(strings.TrimSpace(cfg.Oracle.Schema)),
			ServiceName:                  cfg.Oracle.ServiceName,
			StudentIDColumn:              strings.ToUpper(strings.TrimSpace(cfg.Oracle.StudentIDColumn)),
			StudentNameColumn:            strings.ToUpper(strings.TrimSpace(cfg.Oracle.StudentNameColumn)),
			Table:                        strings.ToUpper(strings.TrimSpace(cfg.Oracle.Table)),
			TLSMode:                      strings.ToLower(strings.TrimSpace(cfg.Oracle.TLSMode)),
			TLSVerified:                  strings.EqualFold(strings.TrimSpace(cfg.Oracle.TLSMode), "verify-full"),
			UsernameConfigured:           strings.TrimSpace(cfg.Oracle.Username) != "",
		},
		Sample: sampleLookupEvidence{
			Enabled:              cfg.SampleStudentID != "",
			ExpectedNameProvided: cfg.SampleExpectedName != "",
			StudentIDHashPrefix:  hashPrefix(cfg.SampleStudentID),
		},
	}

	if cfg.SampleStudentID != "" {
		record, lookupErr := directory.LookupStudent(timeoutCtx, cfg.SampleStudentID)
		if lookupErr != nil {
			return smokeEvidence{}, lookupErr
		}
		evidence.Sample.Found = record != nil
		if record != nil {
			evidence.Sample.NamePresent = strings.TrimSpace(record.StudentName) != ""
			if cfg.SampleExpectedName != "" {
				matched := normalizePersonName(record.StudentName) == normalizePersonName(cfg.SampleExpectedName)
				evidence.Sample.NameMatched = &matched
				if !matched {
					return evidence, fmt.Errorf("oracle student source sample lookup returned a different name")
				}
			}
		}
		if !evidence.Sample.Found {
			return evidence, fmt.Errorf("oracle student source sample lookup did not find the configured student id")
		}
	}

	return evidence, nil
}

func smokeConfigFromEnv(getenv func(string) string) (smokeConfig, error) {
	enabled := strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ENABLED"))
	if !truthy(enabled) {
		return smokeConfig{}, fmt.Errorf("EXTERNAL_STUDENT_SOURCE_ENABLED must be true for this smoke")
	}
	provider := strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_PROVIDER"))
	if provider == "" {
		provider = "oracle"
	}
	if provider != "oracle" {
		return smokeConfig{}, fmt.Errorf("EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle for this smoke")
	}

	timeoutSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_SMOKE_TIMEOUT_SECONDS", int(defaultSmokeTimeout/time.Second))
	if err != nil {
		return smokeConfig{}, err
	}
	if timeoutSeconds < 1 || timeoutSeconds > 120 {
		return smokeConfig{}, fmt.Errorf("EXTERNAL_STUDENT_SOURCE_SMOKE_TIMEOUT_SECONDS must be between 1 and 120")
	}

	requireReadableRecord, err := envBool(getenv, "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_READABLE_RECORD", true)
	if err != nil {
		return smokeConfig{}, err
	}
	requireSample, err := envBool(getenv, "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE", false)
	if err != nil {
		return smokeConfig{}, err
	}
	sampleStudentID := strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID"))
	if requireSample && sampleStudentID == "" {
		return smokeConfig{}, fmt.Errorf("EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID is required when EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE=true")
	}

	port, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_PORT", 2484)
	if err != nil {
		return smokeConfig{}, err
	}
	connectTimeoutSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS", 5)
	if err != nil {
		return smokeConfig{}, err
	}
	queryTimeoutSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS", 3)
	if err != nil {
		return smokeConfig{}, err
	}
	maxOpenConns, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS", 4)
	if err != nil {
		return smokeConfig{}, err
	}
	maxIdleConns, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS", 1)
	if err != nil {
		return smokeConfig{}, err
	}
	connMaxLifetimeSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS", 300)
	if err != nil {
		return smokeConfig{}, err
	}
	connMaxIdleTimeSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS", 60)
	if err != nil {
		return smokeConfig{}, err
	}
	breakerFailureThreshold, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD", 5)
	if err != nil {
		return smokeConfig{}, err
	}
	breakerSuccessThreshold, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD", 2)
	if err != nil {
		return smokeConfig{}, err
	}
	breakerOpenSeconds, err := envInt(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS", 30)
	if err != nil {
		return smokeConfig{}, err
	}

	return smokeConfig{
		SourceName:            envOrDefault(getenv, "EXTERNAL_STUDENT_SOURCE_NAME", "buaa-academic-oracle"),
		Provider:              provider,
		RequireReadableRecord: requireReadableRecord,
		RequireSample:         requireSample,
		SampleStudentID:       sampleStudentID,
		SampleExpectedName:    strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME")),
		Timeout:               time.Duration(timeoutSeconds) * time.Second,
		Oracle: externaldata.OracleStudentDirectoryConfig{
			SchoolCode:              strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE")),
			Host:                    strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_HOST")),
			Port:                    port,
			ServiceName:             strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME")),
			Username:                strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME")),
			ExpectedUsername:        strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME")),
			Password:                getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD"),
			TLSMode:                 envOrDefault(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE", "verify-full"),
			TLSCAFile:               envOrDefault(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE", "/external-student-source-tls/ca.crt"),
			Schema:                  strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA")),
			Table:                   strings.TrimSpace(getenv("EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE")),
			StudentIDColumn:         envOrDefault(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN", "XH"),
			StudentNameColumn:       envOrDefault(getenv, "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN", "XM"),
			ConnectTimeout:          time.Duration(connectTimeoutSeconds) * time.Second,
			QueryTimeout:            time.Duration(queryTimeoutSeconds) * time.Second,
			MaxOpenConns:            maxOpenConns,
			MaxIdleConns:            maxIdleConns,
			ConnMaxLifetime:         time.Duration(connMaxLifetimeSeconds) * time.Second,
			ConnMaxIdleTime:         time.Duration(connMaxIdleTimeSeconds) * time.Second,
			BreakerFailureThreshold: breakerFailureThreshold,
			BreakerSuccessThreshold: breakerSuccessThreshold,
			BreakerOpenTimeout:      time.Duration(breakerOpenSeconds) * time.Second,
		},
	}, nil
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envBool(getenv func(string) string, key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", key)
	}
}

func envInt(getenv func(string) string, key string, fallback int) (int, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func envOrDefault(getenv func(string) string, key string, fallback string) string {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func hashPrefix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func normalizePersonName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), "")
}

func redactSensitiveText(text string, getenv func(string) string) string {
	keys := []string{
		"EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID",
		"EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME",
	}
	for _, key := range keys {
		value := strings.TrimSpace(getenv(key))
		if value != "" {
			text = strings.ReplaceAll(text, value, "<redacted>")
		}
	}
	return text
}
