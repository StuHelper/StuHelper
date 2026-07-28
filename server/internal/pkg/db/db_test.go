package db

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

type fakeRow struct {
	scan func(dest ...any) error
}

type fakeRows struct {
	closeCount int
	err        error
}

func nilContextForTest() context.Context {
	return nil
}

func (f fakeRow) Scan(dest ...any) error {
	return f.scan(dest...)
}

func (f *fakeRows) Close()                                       { f.closeCount++ }
func (f *fakeRows) Next() bool                                   { return false }
func (f *fakeRows) Scan(...any) error                            { return nil }
func (f *fakeRows) Err() error                                   { return f.err }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (f *fakeRows) RawValues() [][]byte                          { return nil }
func (f *fakeRows) Conn() *pgx.Conn                              { return nil }

func TestRowWithCancelScan_ReleasesRetainedReferences(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	row := &RowWithCancel{
		row: fakeRow{
			scan: func(dest ...any) error {
				*dest[0].(*string) = "ok"
				return nil
			},
		},
		cancel: cancel,
		db:     &DB{},
		ctx:    ctx,
		sql:    "select 1 where payload = $1",
		args:   []any{make([]byte, 1024)},
		table:  "users",
	}

	var got string
	err := row.Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, "ok", got)

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context to be canceled after Scan")
	}

	assert.Nil(t, row.row)
	assert.Nil(t, row.cancel)
	assert.Nil(t, row.db)
	assert.Nil(t, row.ctx)
	assert.Empty(t, row.sql)
	assert.Nil(t, row.args)
	assert.Nil(t, row.span)
	assert.Empty(t, row.table)
	assert.True(t, row.start.IsZero())
}

func TestTableHint_NormalizesContextValue(t *testing.T) {
	assert.Equal(t, metrics.DBTableUnknown, TableHint(context.Background()))
	assert.Equal(t, metrics.DBTableUnknown, TableHint(WithTableHint(context.Background(), "SELECT * FROM users")))
	assert.Equal(t, "users", TableHint(WithTableHint(context.Background(), " Users ")))
	assert.Equal(t, "courses", TableHint(WithTableHint(nilContextForTest(), "courses")))
}

func TestDBWithTimeout_DefaultsNilContext(t *testing.T) {
	d := &DB{timeout: time.Second}

	ctx, cancel := d.withTimeout(nilContextForTest())
	defer cancel()

	assert.NotNil(t, ctx)
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(time.Second), deadline, 100*time.Millisecond)
}

func TestDBStartSpan_DefaultsNilContext(t *testing.T) {
	d := &DB{}

	ctx, span := d.startSpan(nilContextForTest(), "query", "select 1", "users")
	defer span.End()

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
}

func TestDBCloseIsNilAndZeroValueSafe(t *testing.T) {
	var nilDB *DB
	assert.NotPanics(t, func() {
		nilDB.Close()
	})

	zeroDB := &DB{}
	assert.NotPanics(t, func() {
		zeroDB.Close()
		zeroDB.Close()
	})
}

func TestRowsWithCancelCloseIsNilZeroValueAndRepeatSafe(t *testing.T) {
	var nilRows *RowsWithCancel
	assert.NotPanics(t, func() {
		nilRows.Close()
	})

	zeroRows := &RowsWithCancel{}
	assert.NotPanics(t, func() {
		zeroRows.Close()
		zeroRows.Close()
	})

	rows := &fakeRows{}
	cancelCount := 0
	wrapped := &RowsWithCancel{
		rows: rows,
		cancel: func() {
			cancelCount++
		},
	}
	assert.NotPanics(t, func() {
		wrapped.Close()
		wrapped.Close()
	})
	assert.Equal(t, 1, rows.closeCount)
	assert.Equal(t, 1, cancelCount)
	assert.Nil(t, wrapped.rows)
	assert.Nil(t, wrapped.cancel)
	assert.Nil(t, wrapped.span)
}

func TestRowWithCancelScan_RecordsTableHintMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	row := &RowWithCancel{
		row: fakeRow{
			scan: func(dest ...any) error {
				*dest[0].(*string) = "ok"
				return nil
			},
		},
		cancel: cancel,
		ctx:    ctx,
		table:  "audit_events",
	}

	before := testutil.ToFloat64(metrics.DBQueryTotal.WithLabelValues("query_row", "audit_events", "ok"))

	var got string
	err := row.Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, "ok", got)
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.DBQueryTotal.WithLabelValues("query_row", "audit_events", "ok")))
}

func TestRowWithCancelScan_SkipsRetryWhenContextCanceledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	wantErr := &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset by peer")}

	row := &RowWithCancel{
		row: fakeRow{
			scan: func(dest ...any) error {
				return wantErr
			},
		},
		cancel: func() {},
		db:     &DB{},
		ctx:    ctx,
		sql:    "select 1",
		table:  "users",
	}

	err := row.Scan(new(int))

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, row.db)
}

func TestRowWithCancelScan_NeverRetriesDataChangingStatement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wantErr := &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset by peer")}

	row := &RowWithCancel{
		row: fakeRow{
			scan: func(dest ...any) error {
				return wantErr
			},
		},
		cancel: func() {},
		// A retry would dereference this deliberately incomplete DB and panic.
		db:    &DB{},
		ctx:   ctx,
		sql:   "INSERT INTO audit_events (id) VALUES ($1) RETURNING id",
		table: "audit_events",
	}

	require.NotPanics(t, func() {
		err := row.Scan(new(string))
		require.ErrorIs(t, err, wantErr)
	})
}

func TestIsRetryableReadSQLUsesStrictAllowlist(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{name: "select", sql: " \n SELECT id FROM users", want: true},
		{name: "show", sql: "SHOW transaction_read_only", want: true},
		{name: "table", sql: "TABLE users", want: true},
		{name: "values", sql: "VALUES (1)", want: true},
		{name: "insert returning", sql: "INSERT INTO users (id) VALUES (1) RETURNING id", want: false},
		{name: "update returning", sql: "UPDATE users SET name = 'x' RETURNING id", want: false},
		{name: "delete returning", sql: "DELETE FROM users RETURNING id", want: false},
		{name: "cte", sql: "WITH changed AS (DELETE FROM users RETURNING id) SELECT * FROM changed", want: false},
		{name: "explain analyze", sql: "EXPLAIN ANALYZE DELETE FROM users", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetryableReadSQL(tt.sql))
		})
	}
}
