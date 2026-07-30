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
	"io"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/circuitbreaker"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

func TestNormalizeOracleStudentDirectoryConfig(t *testing.T) {
	cfg, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "stuhelper_academic_ro",
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

func TestNormalizeOracleStudentDirectoryConfigRejectsAdministrativeAccounts(t *testing.T) {
	for _, username := range []string{"SYS", "system", "SYSBACKUP", "SYSDG", "SYSKM", "SYSRAC"} {
		t.Run(username, func(t *testing.T) {
			_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
				SchoolCode:        "4111010006",
				Host:              "oracle.example.test",
				Port:              2484,
				ServiceName:       "ORCLPDB1",
				Username:          username,
				Password:          "secret",
				TLSMode:           "disable",
				Schema:            "USR_JWBIZ",
				Table:             "T_XS_JBXX",
				StudentIDColumn:   "XH",
				StudentNameColumn: "XM",
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "dedicated non-administrative account")
		})
	}
}

func TestNormalizeOracleStudentDirectoryConfigRejectsSourceSchemaOwner(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              2484,
		ServiceName:       "ORCLPDB1",
		Username:          "usr_jwbiz",
		Password:          "secret",
		TLSMode:           "disable",
		Schema:            "USR_JWBIZ",
		Table:             "T_XS_JBXX",
		StudentIDColumn:   "XH",
		StudentNameColumn: "XM",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not own the source schema")
}

func TestNormalizeOracleStudentDirectoryConfigRejectsUnsafeIdentifiers(t *testing.T) {
	_, err := normalizeOracleStudentDirectoryConfig(OracleStudentDirectoryConfig{
		SchoolCode:        "4111010006",
		Host:              "oracle.example.test",
		Port:              1521,
		ServiceName:       "ORCLPDB1",
		Username:          "stuhelper_academic_ro",
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

func TestScanOracleStudentRecordsRejectsIdentityMismatch(t *testing.T) {
	rows := &fakeOracleRows{records: []fakeOracleRecord{{
		id:   "20250002",
		name: sql.NullString{String: "张三", Valid: true},
	}}}

	_, err := scanOracleStudentRecords(rows, "4111010006", "20250001")
	require.ErrorIs(t, err, ErrStudentSourceRecordIdentityMismatch)
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

func TestClassifyOracleStudentSourceError(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineCtx, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()

	tests := []struct {
		name          string
		ctx           context.Context
		err           error
		wantOutcome   oracleStudentSourceErrorOutcome
		wantIntegrity string
	}{
		{
			name:        "caller cancellation",
			ctx:         canceledCtx,
			err:         context.Canceled,
			wantOutcome: oracleStudentSourceErrorNeutral,
		},
		{
			name:        "caller deadline",
			ctx:         deadlineCtx,
			err:         context.DeadlineExceeded,
			wantOutcome: oracleStudentSourceErrorNeutral,
		},
		{
			name:        "directory timeout",
			ctx:         context.Background(),
			err:         context.DeadlineExceeded,
			wantOutcome: oracleStudentSourceErrorFailure,
		},
		{
			name:          "invalid record",
			ctx:           context.Background(),
			err:           ErrStudentSourceInvalidRecord,
			wantOutcome:   oracleStudentSourceErrorNeutral,
			wantIntegrity: oracleStudentSourceIntegrityInvalidRecord,
		},
		{
			name:          "ambiguous record",
			ctx:           context.Background(),
			err:           ErrStudentSourceAmbiguousRecord,
			wantOutcome:   oracleStudentSourceErrorNeutral,
			wantIntegrity: oracleStudentSourceIntegrityAmbiguousRecord,
		},
		{
			name:          "identity mismatch",
			ctx:           context.Background(),
			err:           ErrStudentSourceRecordIdentityMismatch,
			wantOutcome:   oracleStudentSourceErrorFailure,
			wantIntegrity: oracleStudentSourceIntegrityIdentityMismatch,
		},
		{
			name:        "backend failure",
			ctx:         context.Background(),
			err:         errors.New("connection reset"),
			wantOutcome: oracleStudentSourceErrorFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome, integrity := classifyOracleStudentSourceError(tt.ctx, tt.err)
			assert.Equal(t, tt.wantOutcome, outcome)
			assert.Equal(t, tt.wantIntegrity, integrity)
		})
	}
}

func TestOracleStudentDirectoryCallerCancellationIsNeutralAndReleasesHalfOpenProbe(t *testing.T) {
	queryStarted := make(chan struct{})
	var queryStartedOnce sync.Once
	connector := &scriptedOracleConnector{
		query: func(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
			queryStartedOnce.Do(func() { close(queryStarted) })
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	directory := newScriptedOracleTestDirectory(
		t,
		connector,
		time.Second,
		1,
		time.Nanosecond,
	)
	require.True(t, directory.breaker.Allow())
	directory.breaker.RecordFailure()
	require.Eventually(t, func() bool {
		return directory.breaker.State() == circuitbreaker.StateHalfOpen
	}, time.Second, time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := directory.LookupStudent(ctx, "20250001")
		errCh <- err
	}()
	select {
	case <-queryStarted:
	case <-time.After(time.Second):
		t.Fatal("Oracle lookup did not reach the scripted query")
	}
	cancel()

	var err error
	select {
	case err = <-errCh:
	case <-time.After(time.Second):
		t.Fatal("caller cancellation did not release the Oracle lookup")
	}
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrStudentSourceUnavailable)
	breakerMetrics := directory.breaker.Metrics()
	assert.Equal(t, circuitbreaker.StateHalfOpen.String(), breakerMetrics["state"])
	assert.Equal(t, 0, breakerMetrics["failures"])
	assert.Equal(t, false, breakerMetrics["probe_in_flight"])
	require.True(t, directory.breaker.Allow(), "neutral cancellation must release the recovery probe")
	directory.breaker.RecordNeutral()
}

func TestOracleStudentDirectoryInternalQueryTimeoutIsFailure(t *testing.T) {
	connector := &scriptedOracleConnector{
		query: func(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	directory := newScriptedOracleTestDirectory(
		t,
		connector,
		10*time.Millisecond,
		1,
		time.Hour,
	)

	_, err := directory.LookupStudent(context.Background(), "20250001")
	require.ErrorIs(t, err, ErrStudentSourceUnavailable)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, circuitbreaker.StateOpen, directory.breaker.State())
	assert.Equal(t, 1, directory.breaker.Metrics()["failures"])
}

func TestOracleStudentDirectoryProbeCallerCancellationIsNeutral(t *testing.T) {
	pingStarted := make(chan struct{})
	var pingStartedOnce sync.Once
	connector := &scriptedOracleConnector{
		ping: func(ctx context.Context) error {
			pingStartedOnce.Do(func() { close(pingStarted) })
			<-ctx.Done()
			return ctx.Err()
		},
	}
	directory := newScriptedOracleTestDirectory(
		t,
		connector,
		time.Second,
		1,
		time.Hour,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := directory.Probe(ctx)
		errCh <- err
	}()
	select {
	case <-pingStarted:
	case <-time.After(time.Second):
		t.Fatal("Oracle probe did not reach the scripted ping")
	}
	cancel()

	var err error
	select {
	case err = <-errCh:
	case <-time.After(time.Second):
		t.Fatal("caller cancellation did not release the Oracle probe")
	}
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrStudentSourceUnavailable)
	assert.Equal(t, circuitbreaker.StateClosed, directory.breaker.State())
	assert.Equal(t, 0, directory.breaker.Metrics()["failures"])
}

func TestOracleStudentDirectoryRecordIntegrityErrorsAreNeutral(t *testing.T) {
	tests := []struct {
		name       string
		rows       [][]driver.Value
		wantErr    error
		wantReason string
	}{
		{
			name:       "invalid record",
			rows:       [][]driver.Value{{"20250001", nil}},
			wantErr:    ErrStudentSourceInvalidRecord,
			wantReason: oracleStudentSourceIntegrityInvalidRecord,
		},
		{
			name: "ambiguous record",
			rows: [][]driver.Value{
				{"20250001", "张三"},
				{"20250001", "李四"},
			},
			wantErr:    ErrStudentSourceAmbiguousRecord,
			wantReason: oracleStudentSourceIntegrityAmbiguousRecord,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &scriptedOracleConnector{
				query: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
					return &scriptedOracleRows{values: tt.rows}, nil
				},
			}
			directory := newScriptedOracleTestDirectory(
				t,
				connector,
				time.Second,
				1,
				time.Hour,
			)
			counter := metrics.ExternalDataIntegrityErrorsTotal.WithLabelValues(
				oracleStudentDirectoryMetricClient,
				tt.wantReason,
			)
			before := promtest.ToFloat64(counter)

			_, err := directory.LookupStudent(context.Background(), "20250001")
			require.ErrorIs(t, err, tt.wantErr)
			require.NotErrorIs(t, err, ErrStudentSourceUnavailable)
			assert.Equal(t, circuitbreaker.StateClosed, directory.breaker.State())
			assert.Equal(t, 0, directory.breaker.Metrics()["failures"])
			assert.Equal(t, before+1, promtest.ToFloat64(counter))
		})
	}
}

func TestOracleStudentDirectoryIntegrityErrorReleasesHalfOpenProbe(t *testing.T) {
	connector := &scriptedOracleConnector{
		query: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedOracleRows{values: [][]driver.Value{{"20250001", nil}}}, nil
		},
	}
	directory := newScriptedOracleTestDirectory(
		t,
		connector,
		time.Second,
		1,
		time.Nanosecond,
	)
	require.True(t, directory.breaker.Allow())
	directory.breaker.RecordFailure()
	require.Eventually(t, func() bool {
		return directory.breaker.State() == circuitbreaker.StateHalfOpen
	}, time.Second, time.Millisecond)

	_, err := directory.LookupStudent(context.Background(), "20250001")
	require.ErrorIs(t, err, ErrStudentSourceInvalidRecord)
	require.NotErrorIs(t, err, ErrStudentSourceUnavailable)
	breakerMetrics := directory.breaker.Metrics()
	assert.Equal(t, circuitbreaker.StateHalfOpen.String(), breakerMetrics["state"])
	assert.Equal(t, false, breakerMetrics["probe_in_flight"])
	require.True(t, directory.breaker.Allow(), "neutral integrity error must release the recovery probe")
	directory.breaker.RecordNeutral()
}

func TestOracleStudentDirectoryIdentityMismatchIsSourceFailure(t *testing.T) {
	connector := &scriptedOracleConnector{
		query: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedOracleRows{values: [][]driver.Value{{"20250002", "张三"}}}, nil
		},
	}
	directory := newScriptedOracleTestDirectory(
		t,
		connector,
		time.Second,
		1,
		time.Hour,
	)
	counter := metrics.ExternalDataIntegrityErrorsTotal.WithLabelValues(
		oracleStudentDirectoryMetricClient,
		oracleStudentSourceIntegrityIdentityMismatch,
	)
	before := promtest.ToFloat64(counter)

	_, err := directory.LookupStudent(context.Background(), "20250001")
	require.ErrorIs(t, err, ErrStudentSourceUnavailable)
	require.ErrorIs(t, err, ErrStudentSourceRecordIdentityMismatch)
	assert.Equal(t, circuitbreaker.StateOpen, directory.breaker.State())
	assert.Equal(t, 1, directory.breaker.Metrics()["failures"])
	assert.Equal(t, before+1, promtest.ToFloat64(counter))
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

type scriptedOracleConnector struct {
	query func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
	ping  func(context.Context) error
}

type scriptedOracleConn struct {
	connector *scriptedOracleConnector
}

type scriptedOracleRows struct {
	values  [][]driver.Value
	columns []string
	index   int
}

type scriptedOracleDriver struct{}

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

func newScriptedOracleTestDirectory(
	t *testing.T,
	connector *scriptedOracleConnector,
	queryTimeout time.Duration,
	failureThreshold int,
	openTimeout time.Duration,
) *OracleStudentDirectory {
	t.Helper()
	db := sql.OpenDB(connector)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return newOracleStudentDirectoryWithDB(OracleStudentDirectoryConfig{
		SchoolCode:              "4111010006",
		Schema:                  "USR_JWBIZ",
		Table:                   "T_XS_JBXX",
		StudentIDColumn:         "XH",
		StudentNameColumn:       "XM",
		QueryTimeout:            queryTimeout,
		BreakerFailureThreshold: failureThreshold,
		BreakerSuccessThreshold: 1,
		BreakerOpenTimeout:      openTimeout,
	}, db)
}

func (c *scriptedOracleConnector) Connect(context.Context) (driver.Conn, error) {
	return &scriptedOracleConn{connector: c}, nil
}

func (*scriptedOracleConnector) Driver() driver.Driver {
	return &scriptedOracleDriver{}
}

func (*scriptedOracleDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("scripted Oracle driver must be opened through its connector")
}

func (*scriptedOracleConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*scriptedOracleConn) Close() error {
	return nil
}

func (*scriptedOracleConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *scriptedOracleConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	if c.connector.query == nil {
		return &scriptedOracleRows{}, nil
	}
	return c.connector.query(ctx, query, args)
}

func (c *scriptedOracleConn) Ping(ctx context.Context) error {
	if c.connector.ping == nil {
		return nil
	}
	return c.connector.ping(ctx)
}

func (r *scriptedOracleRows) Columns() []string {
	if len(r.columns) > 0 {
		return r.columns
	}
	return []string{"XH", "XM"}
}

func (*scriptedOracleRows) Close() error {
	return nil
}

func (r *scriptedOracleRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
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
