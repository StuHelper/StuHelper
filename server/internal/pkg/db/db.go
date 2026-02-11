package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// DB 封装 pgxpool.Pool，提供带超时的查询方法
type DB struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

// NewDB 创建带超时的数据库封装
func NewDB(pool *pgxpool.Pool, timeout time.Duration) *DB {
	return &DB{
		pool:    pool,
		timeout: timeout,
	}
}

// Pool 返回底层连接池
func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// withTimeout 创建带超时的 context
func (d *DB) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d.timeout)
}

// Query 执行带超时的查询
// 返回 RowsWithCancel 包装类型，确保 rows 消费完毕后才取消 context
func (d *DB) Query(ctx context.Context, sql string, args ...any) (*RowsWithCancel, error) {
	ctx, cancel := d.withTimeout(ctx)
	start := time.Now()
	rows, err := d.pool.Query(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("query", "").Observe(duration)
	status := "ok"
	if err != nil {
		status = "error"
		cancel()
		metrics.DBQueryTotal.WithLabelValues("query", "", status).Inc()
		return nil, err
	}
	metrics.DBQueryTotal.WithLabelValues("query", "", status).Inc()
	return &RowsWithCancel{rows: rows, cancel: cancel}, nil
}

// QueryRow 执行带超时的单行查询
// 返回 RowWithCancel 包装类型，确保 Scan 完成后才取消 context
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) *RowWithCancel {
	ctx, cancel := d.withTimeout(ctx)
	start := time.Now()
	row := d.pool.QueryRow(ctx, sql, args...)
	return &RowWithCancel{row: row, cancel: cancel, start: start}
}

// RowWithCancel 包装 pgx.Row，确保 Scan 完成后才取消 context
type RowWithCancel struct {
	row    pgx.Row
	cancel context.CancelFunc
	start  time.Time
}

// Scan 扫描行数据，完成后自动取消 context
func (r *RowWithCancel) Scan(dest ...any) error {
	defer r.cancel()
	err := r.row.Scan(dest...)
	duration := time.Since(r.start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("query_row", "").Observe(duration)
	status := "ok"
	if err != nil {
		status = "error"
	}
	metrics.DBQueryTotal.WithLabelValues("query_row", "", status).Inc()
	return err
}

// RowsWithCancel 包装 pgx.Rows，确保 Close 后才取消 context
type RowsWithCancel struct {
	rows   pgx.Rows
	cancel context.CancelFunc
}

func (r *RowsWithCancel) Next() bool                              { return r.rows.Next() }
func (r *RowsWithCancel) Scan(dest ...any) error                  { return r.rows.Scan(dest...) }
func (r *RowsWithCancel) Err() error                              { return r.rows.Err() }
func (r *RowsWithCancel) CommandTag() pgconn.CommandTag            { return r.rows.CommandTag() }
func (r *RowsWithCancel) FieldDescriptions() []pgconn.FieldDescription {
	return r.rows.FieldDescriptions()
}
func (r *RowsWithCancel) Values() ([]any, error)  { return r.rows.Values() }
func (r *RowsWithCancel) RawValues() [][]byte      { return r.rows.RawValues() }
func (r *RowsWithCancel) Conn() *pgx.Conn          { return r.rows.Conn() }
func (r *RowsWithCancel) Close() {
	r.rows.Close()
	r.cancel()
}

// Exec 执行带超时的命令
func (d *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := d.withTimeout(ctx)
	defer cancel()
	start := time.Now()
	tag, err := d.pool.Exec(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("exec", "").Observe(duration)
	status := "ok"
	if err != nil {
		status = "error"
	}
	metrics.DBQueryTotal.WithLabelValues("exec", "", status).Inc()
	return tag, err
}

// Ping 测试连接
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := d.withTimeout(ctx)
	defer cancel()
	return d.pool.Ping(ctx)
}

// Close 关闭连接池
func (d *DB) Close() {
	d.pool.Close()
}

// Begin 开始事务（事务内的操作由调用者控制超时）
func (d *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctx)
}

// WithTx 事务包装函数，自动处理提交和回滚
// 事务使用 3 倍查询超时作为上限，防止长事务占用连接
// 如果 fn 返回 error，事务会回滚；否则提交
func (d *DB) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	txTimeout := d.timeout * 3
	ctx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logger.L().Warn("tx rollback failed during panic recovery",
					zap.Error(rbErr))
			}
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			logger.L().Warn("tx rollback failed",
				zap.Error(rbErr))
		}
		return err
	}

	return tx.Commit(ctx)
}
