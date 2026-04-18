package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRow struct {
	scan func(dest ...any) error
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
	assert.True(t, row.start.IsZero())
}
