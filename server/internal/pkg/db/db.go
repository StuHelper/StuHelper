package db

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// txTimeoutMultiplier 事务超时相对于单次查询超时的倍数
const txTimeoutMultiplier = 3

// retryBaseDelay 重试基础延迟
const retryBaseDelay = 100 * time.Millisecond

// jitteredRetryDelay 返回带随机抖动的重试延迟，防止惊群效应
func jitteredRetryDelay() time.Duration {
	jitter := float64(retryBaseDelay) * 0.5
	delta := rand.Float64()*2*jitter - jitter
	return retryBaseDelay + time.Duration(delta)
}

// DB 封装 pgxpool.Pool，提供带超时的查询方法
type DB struct {
	pool      *pgxpool.Pool
	timeout   time.Duration
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewDB 创建带超时的数据库封装
func NewDB(pool *pgxpool.Pool, timeout time.Duration) *DB {
	d := &DB{
		pool:    pool,
		timeout: timeout,
		stopCh:  make(chan struct{}),
	}
	go d.collectPoolMetrics()
	return d
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
// 对瞬时连接错误自动重试一次
func (d *DB) Query(ctx context.Context, sql string, args ...any) (*RowsWithCancel, error) {
	ctx, cancel := d.withTimeout(ctx)
	start := time.Now()
	rows, err := d.pool.Query(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("query", "").Observe(duration)

	// 瞬时连接错误：退避后重试一次（仅在 context 未超时时）
	if err != nil && isConnectionError(err) && ctx.Err() == nil {
		logger.L().Warn("query connection error, retrying once", zap.Error(err))
		time.Sleep(jitteredRetryDelay())
		rows, err = d.pool.Query(ctx, sql, args...)
		duration = time.Since(start).Seconds()
		metrics.DBQueryDuration.WithLabelValues("query", "retry").Observe(duration)
	}

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
// 对瞬时连接错误自动重试一次（与 Query 保持一致）
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) *RowWithCancel {
	ctx, cancel := d.withTimeout(ctx)
	start := time.Now()
	row := d.pool.QueryRow(ctx, sql, args...)
	return &RowWithCancel{row: row, cancel: cancel, start: start, db: d, ctx: ctx, sql: sql, args: args}
}

// RowWithCancel 包装 pgx.Row，确保 Scan 完成后才取消 context
type RowWithCancel struct {
	row    pgx.Row
	cancel context.CancelFunc
	start  time.Time
	db     *DB
	ctx    context.Context
	sql    string
	args   []any
}

// Scan 扫描行数据，完成后自动取消 context
// 对瞬时连接错误自动重试一次
func (r *RowWithCancel) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	duration := time.Since(r.start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("query_row", "").Observe(duration)

	// 瞬时连接错误：退避后重试一次
	if err != nil && isConnectionError(err) && r.ctx.Err() == nil {
		logger.L().Warn("query_row connection error, retrying once", zap.Error(err))
		time.Sleep(jitteredRetryDelay())
		retryRow := r.db.pool.QueryRow(r.ctx, r.sql, r.args...)
		err = retryRow.Scan(dest...)
		duration = time.Since(r.start).Seconds()
		metrics.DBQueryDuration.WithLabelValues("query_row", "retry").Observe(duration)
	}

	r.cancel()
	status := "ok"
	if err != nil {
		status = "error"
	}
	metrics.DBQueryTotal.WithLabelValues("query_row", "", status).Inc()
	return err
}

// 编译时断言：RowsWithCancel 实现 pgx.Rows 接口
var _ pgx.Rows = (*RowsWithCancel)(nil)

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
// 对瞬时连接错误自动重试一次（与 Query 保持一致）
func (d *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := d.withTimeout(ctx)
	defer cancel()
	start := time.Now()
	tag, err := d.pool.Exec(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.DBQueryDuration.WithLabelValues("exec", "").Observe(duration)

	// 瞬时连接错误：退避后重试一次（仅在 context 未超时时）
	if err != nil && isConnectionError(err) && ctx.Err() == nil {
		logger.L().Warn("exec connection error, retrying once", zap.Error(err))
		time.Sleep(jitteredRetryDelay())
		tag, err = d.pool.Exec(ctx, sql, args...)
		duration = time.Since(start).Seconds()
		metrics.DBQueryDuration.WithLabelValues("exec", "retry").Observe(duration)
	}

	status := "ok"
	if err != nil {
		status = "error"
	}
	metrics.DBQueryTotal.WithLabelValues("exec", "", status).Inc()
	return tag, err
}

// pingTimeout Ping 专用超时上限
const pingTimeout = 3 * time.Second

// Ping 测试连接
// 使用 min(查询超时, pingTimeout) 避免健康检查挂起过久
func (d *DB) Ping(ctx context.Context) error {
	timeout := min(d.timeout, pingTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return d.pool.Ping(ctx)
}

// Close 关闭连接池（安全支持多次调用）
func (d *DB) Close() {
	d.closeOnce.Do(func() {
		close(d.stopCh)
		d.pool.Close()
	})
}

// collectPoolMetrics 定期采集连接池指标
func (d *DB) collectPoolMetrics() {
	// 启动时立即采集一次，避免前 15 秒无指标
	stat := d.pool.Stat()
	metrics.DBConnectionsActive.Set(float64(stat.AcquiredConns()))

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			stat := d.pool.Stat()
			metrics.DBConnectionsActive.Set(float64(stat.AcquiredConns()))
		}
	}
}

// Begin 开始事务（事务内的操作由调用者控制超时）
func (d *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctx)
}

// WithTx 事务包装函数，自动处理提交和回滚
// 事务使用 txTimeoutMultiplier 倍查询超时作为上限，防止长事务占用连接
// 如果 fn 返回 error，事务会回滚；否则提交
func (d *DB) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	txTimeout := d.timeout * txTimeoutMultiplier
	ctx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// 使用独立 context 执行 Rollback，原 ctx 可能已超时
			rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer rbCancel()
			// 保护 Rollback 本身不 panic，避免吞掉原始 panic
			func() {
				defer func() {
					if rbPanic := recover(); rbPanic != nil {
						logger.L().Warn("tx rollback panicked during panic recovery",
							zap.Any("rollback_panic", rbPanic))
					}
				}()
				if rbErr := tx.Rollback(rbCtx); rbErr != nil {
					logger.L().Warn("tx rollback failed during panic recovery",
						zap.Error(rbErr))
				}
			}()
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

// isConnectionError 判断是否为瞬时连接错误（网络断开、连接重置等），
// 这类错误值得重试；SQL 语法错误、约束冲突等不应重试。
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// 连接被关闭 / EOF
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// 不重试 context.DeadlineExceeded：超时意味着客户端无法确认服务端是否已完成执行，
	// 重试非幂等写操作（INSERT/UPDATE）会导致重复写入
	// 网络层错误（connection refused / reset / timeout 等）
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	// pgconn 连接级错误：Class 08 = Connection Exception
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
		return true
	}
	return false
}
