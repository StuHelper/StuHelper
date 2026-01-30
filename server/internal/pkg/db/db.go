package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (d *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := d.withTimeout(ctx)
	defer cancel()
	return d.pool.Query(ctx, sql, args...)
}

// QueryRow 执行带超时的单行查询
// 返回 RowWithCancel 包装类型，确保 Scan 完成后才取消 context
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) *RowWithCancel {
	ctx, cancel := d.withTimeout(ctx)
	row := d.pool.QueryRow(ctx, sql, args...)
	return &RowWithCancel{row: row, cancel: cancel}
}

// RowWithCancel 包装 pgx.Row，确保 Scan 完成后才取消 context
type RowWithCancel struct {
	row    pgx.Row
	cancel context.CancelFunc
}

// Scan 扫描行数据，完成后自动取消 context
func (r *RowWithCancel) Scan(dest ...any) error {
	defer r.cancel()
	return r.row.Scan(dest...)
}

// Exec 执行带超时的命令
func (d *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := d.withTimeout(ctx)
	defer cancel()
	return d.pool.Exec(ctx, sql, args...)
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
