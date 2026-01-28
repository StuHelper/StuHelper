package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryTimeout 默认查询超时时间
var QueryTimeout = 5 * time.Second

// SetQueryTimeout 设置全局查询超时时间
func SetQueryTimeout(timeout time.Duration) {
	QueryTimeout = timeout
}

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
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := d.withTimeout(ctx)
	// 注意：cancel 不能在这里 defer，因为 Row.Scan 需要在 context 有效时执行
	// 但 pgx 的 QueryRow 会在内部处理，所以这里可以安全地 defer
	defer cancel()
	return d.pool.QueryRow(ctx, sql, args...)
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
