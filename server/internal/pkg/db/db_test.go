package db

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func nilContextForTest() context.Context {
	return nil
}

func (f fakeRow) Scan(dest ...any) error {
	return f.scan(dest...)
}

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
