package externaldata

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"database/sql/driver"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"sync/atomic"
	"testing"
	"time"

	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

func TestNormalizeOracleStudentDirectoryConfig(t *testing.T) {
	cfg, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "SYSTEM",
		Password:          "secret",
		TLSMode:           "disable",
		Schema:            "usr_jwbiz",
		Table:             "t_xs_jbxx",
		StudentIDColumn:   "xh",
		StudentNameColumn: "xm",
	})
	require.NoError(t, err)

	assert.Equal(t, "USR_JWBIZ", cfg.Schema)
	assert.Equal(t, "T_XS_JBXX", cfg.Table)
	assert.Equal(t, "XH", cfg.StudentIDColumn)
	assert.Equal(t, "XM", cfg.StudentNameColumn)
	assert.Equal(t, defaultOracleQueryTimeout, cfg.QueryTimeout)
	assert.Equal(t, defaultOracleConnMaxLifetime, cfg.ConnMaxLifetime)
	assert.Equal(t, defaultOracleConnMaxIdleTime, cfg.ConnMaxIdleTime)
	assert.Equal(t, defaultOracleBreakerFailureThreshold, cfg.BreakerFailureThreshold)
	assert.Equal(t, defaultOracleBreakerSuccessThreshold, cfg.BreakerSuccessThreshold)
	assert.Equal(t, defaultOracleBreakerOpenTimeout, cfg.BreakerOpenTimeout)
}

func TestNormalizeOracleStudentDirectoryConfigRejectsUnsafeIdentifiers(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "SYSTEM",
		Password:          "secret",
		TLSMode:           "disable",
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX;DROP TABLE USERS",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
		QueryTimeout:      time.Second,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oracle table identifier is invalid")
}

func TestNormalizeOracleStudentDirectoryConfigRejectsNegativeBreakerConfig(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:              "4111010006",
		Host:                    "oracle.example.test",
		Port:                    2484,
		ServiceName:             "ORCLPDB1",
		Username:                "stuhelper_ro",
		Password:                "secret",
		TLSMode:                 "disable",
		Schema:                  "USR_JWBIZ",
		Table:                   "T_XS_JBXX",
		StudentIDColumn:         "XH",
		StudentNameColumn:       "XM",
		BreakerFailureThreshold: -1,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failure threshold must not be negative")
}

func TestBuildOracleStudentLookupQuery(t *testing.T) {
	query := buildOracleStudentLookupQuery(OracleStudentDirectoryConfig{
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})

	assert.Equal(
		t,
		"SELECT XH, XM FROM USR_JWBIZ.T_XS_JBXX WHERE XH = :1 FETCH FIRST 2 ROWS ONLY",
		query,
	)
}

func TestBuildOracleStudentProbeQuery(t *testing.T) {
	query := buildOracleStudentProbeQuery(OracleStudentDirectoryConfig{
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})

	assert.Equal(
		t,
		"SELECT 1 FROM USR_JWBIZ.T_XS_JBXX WHERE TRIM(XH) IS NOT NULL AND TRIM(XM) IS NOT NULL FETCH FIRST 1 ROWS ONLY",
		query,
	)
}

func TestNormalizeOracleLookupStudentID(t *testing.T) {
	id, err := normalizeOracleLookupStudentID(" 20250001 ")
	require.NoError(t, err)
	assert.Equal(t, "20250001", id)

	_, err = normalizeOracleLookupStudentID("student/id")
	require.ErrorIs(t, err, ErrStudentSourceInvalidStudentID)

	id, err = normalizeOracleLookupStudentID("   ")
	require.NoError(t, err)
	assert.Empty(t, id)
}

func TestNormalizeOracleStudentDirectoryConfigRequiresVerifiedTLSCA(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              2484,
		ServiceName:       "ORCLPDB1",
		Username:          "stuhelper_ro",
		Password:          "secret",
		TLSMode:           "verify-full",
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS CA file is required")
}

func TestScanOracleStudentRecords(t *testing.T) {
	rows := &fakeOracleRows{records: []fakeOracleRecord{
		{id: "20250001 ", name: sql.NullString{String: " 张三 ", Valid: true}},
		{id: "20250001", name: sql.NullString{String: "张三", Valid: true}},
	}}

	record, err := scanOracleStudentRecords(rows, "4111010006", "20250001")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "20250001", record.StudentID)
	assert.Equal(t, "张三", record.StudentName)
}

func TestScanOracleStudentRecordsRejectsAmbiguousNames(t *testing.T) {
	rows := &fakeOracleRows{records: []fakeOracleRecord{
		{id: "20250001", name: sql.NullString{String: "张三", Valid: true}},
		{id: "20250001", name: sql.NullString{String: "李四", Valid: true}},
	}}

	_, err := scanOracleStudentRecords(rows, "4111010006", "20250001")
	require.ErrorIs(t, err, ErrStudentSourceAmbiguousRecord)
}

func TestScanOracleStudentRecordsRejectsInvalidRecord(t *testing.T) {
	rows := &fakeOracleRows{records: []fakeOracleRecord{{
		id:   "20250002",
		name: sql.NullString{String: "张三", Valid: true},
	}}}

	_, err := scanOracleStudentRecords(rows, "4111010006", "20250001")
	require.ErrorIs(t, err, ErrStudentSourceInvalidRecord)
}

func TestScanOracleStudentRecordsRejectsUnsafeName(t *testing.T) {
	rows := &fakeOracleRows{records: []fakeOracleRecord{{
		id:   "20250001",
		name: sql.NullString{String: "张\u200b三", Valid: true},
	}}}

	_, err := scanOracleStudentRecords(rows, "4111010006", "20250001")
	require.ErrorIs(t, err, ErrStudentSourceInvalidRecord)
}

func TestOracleStudentDirectoryCircuitBreakerFailsFastAndEmitsMetrics(t *testing.T) {
	connector := &failingOracleConnector{}
	db := sql.OpenDB(connector)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	directory := newOracleStudentDirectoryWithDB(OracleStudentDirectoryConfig{
		SchoolCode:              "4111010006",
		Schema:                  "USR_JWBIZ",
		Table:                   "T_XS_JBXX",
		StudentIDColumn:         "XH",
		StudentNameColumn:       "XM",
		QueryTimeout:            time.Second,
		BreakerFailureThreshold: 1,
		BreakerSuccessThreshold: 1,
		BreakerOpenTimeout:      time.Hour,
	}, db)
	before := promtest.ToFloat64(
		metrics.ExternalRequestsTotal.WithLabelValues(oracleStudentDirectoryMetricClient, "lookup", "error"),
	)

	_, err := directory.LookupStudent(context.Background(), "20250001")
	require.ErrorIs(t, err, ErrStudentSourceUnavailable)
	assert.Equal(t, int32(1), connector.connects.Load())

	_, err = directory.LookupStudent(context.Background(), "20250001")
	require.ErrorIs(t, err, ErrStudentSourceUnavailable)
	assert.ErrorContains(t, err, "circuit breaker open")
	assert.Equal(t, int32(1), connector.connects.Load(), "open breaker must not dial Oracle")
	after := promtest.ToFloat64(
		metrics.ExternalRequestsTotal.WithLabelValues(oracleStudentDirectoryMetricClient, "lookup", "error"),
	)
	assert.Equal(t, before+2, after)
}

func TestLoadOracleTLSRootCAs(t *testing.T) {
	caPath := writeOracleTestCA(t, true)
	roots, err := loadOracleTLSRootCAs(caPath)
	require.NoError(t, err)
	require.NotNil(t, roots)
}

func TestLoadOracleTLSRootCAsRejectsNonCAAndPrivateKey(t *testing.T) {
	_, err := loadOracleTLSRootCAs(writeOracleTestCA(t, false))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-CA certificate")

	privateKeyPath := t.TempDir() + "/private-key.pem"
	require.NoError(t, os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte("not-a-real-key"),
	}), 0o600))
	_, err = loadOracleTLSRootCAs(privateKeyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificates only")
}

type fakeOracleRecord struct {
	id   string
	name sql.NullString
}

type fakeOracleRows struct {
	records []fakeOracleRecord
	index   int
	err     error
}

type failingOracleConnector struct {
	connects atomic.Int32
}

func (c *failingOracleConnector) Connect(context.Context) (driver.Conn, error) {
	c.connects.Add(1)
	return nil, errors.New("dial failed")
}

func (c *failingOracleConnector) Driver() driver.Driver {
	return failingOracleDriver{}
}

type failingOracleDriver struct{}

func (failingOracleDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("dial failed")
}

func (r *fakeOracleRows) Next() bool {
	if r.index >= len(r.records) {
		return false
	}
	r.index++
	return true
}

func (r *fakeOracleRows) Scan(dest ...any) error {
	record := r.records[r.index-1]
	*dest[0].(*string) = record.id
	*dest[1].(*sql.NullString) = record.name
	return nil
}

func (r *fakeOracleRows) Err() error {
	return r.err
}

func writeOracleTestCA(t *testing.T, isCA bool) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "StuHelper Oracle test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	path := t.TempDir() + "/ca.crt"
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}), 0o644))
	return path
}
