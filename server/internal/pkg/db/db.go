package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

// txTimeoutMultiplier 事务超时相对于单次查询超时的倍数
const txTimeoutMultiplier = 3

// retryBaseDelay 重试基础延迟
const retryBaseDelay = 100 * time.Millisecond

type tableHintContextKey struct{}

// WithTableHint attaches explicit repository table metadata for DB metrics.
// The value is normalized before storage so downstream instrumentation never
// needs to inspect SQL text or accept high-cardinality fragments.
func WithTableHint(ctx context.Context, table string) context.Context {
	return context.WithValue(ctxutil.Normalize(ctx), tableHintContextKey{}, metrics.NormalizeDBTable(table))
}

// TableHint returns the normalized DB metrics table label stored on ctx.
func TableHint(ctx context.Context) string {
	if ctx == nil {
		return metrics.DBTableUnknown
	}
	if table, ok := ctx.Value(tableHintContextKey{}).(string); ok {
		return metrics.NormalizeDBTable(table)
	}
	return metrics.DBTableUnknown
}

// jitteredRetryDelay 返回带随机抖动的重试延迟，防止惊群效应
func jitteredRetryDelay() time.Duration {
	jitter := float64(retryBaseDelay) * 0.5
	delta := rand.Float64()*2*jitter - jitter // #nosec G404 -- retry jitter is not security-sensitive randomness.
	return retryBaseDelay + time.Duration(delta)
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

// DB 封装 pgxpool.Pool，提供带超时的查询方法
type DB struct {
	pool      *pgxpool.Pool
	timeout   time.Duration
	stopCh    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewDB 创建带超时的数据库封装
func NewDB(pool *pgxpool.Pool, timeout time.Duration) *DB {
	d := &DB{
		pool:    pool,
		timeout: timeout,
		stopCh:  make(chan struct{}),
	}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.collectPoolMetrics()
	}()
	return d
}

// withTimeout 创建带超时的 context
func (d *DB) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctxutil.Normalize(ctx), d.timeout)
}

// withExplicitTimeout creates a caller-cancellable context for a database
// operation whose expected duration differs from the ordinary query budget.
// Non-positive values deliberately fall back to the DB-wide timeout.
func (d *DB) withExplicitTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = d.timeout
	}
	return context.WithTimeout(ctxutil.Normalize(ctx), timeout)
}

// Query 执行带超时的查询
// 返回 RowsWithCancel 包装类型，确保 rows 消费完毕后才取消 context
// 对瞬时连接错误自动重试一次
func (d *DB) Query(ctx context.Context, sql string, args ...any) (*RowsWithCancel, error) {
	table := TableHint(ctx)
	ctx, cancel := d.withTimeout(ctx)
	ctx, span := d.startSpan(ctx, "query", sql, table)
	start := time.Now()
	rows, err := d.pool.Query(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.ObserveDBQueryDuration("query", table, duration)

	// 只读 SQL 遇到瞬时连接错误时退避重试一次。写语句即使通过 Query
	// 执行，也不能透明重试：连接断开时无法确认服务端是否已经完成写入。
	if err != nil && isRetryableReadSQL(sql) && isConnectionError(err) && ctx.Err() == nil {
		logger.L().Warn("query connection error, retrying once", zap.Error(err))
		if waitForRetry(ctx, jitteredRetryDelay()) {
			retryStart := time.Now()
			rows, err = d.pool.Query(ctx, sql, args...)
			duration = time.Since(retryStart).Seconds()
			metrics.ObserveDBQueryDuration("query_retry", table, duration)
		}
	}

	status := "ok"
	if err != nil {
		status = "error"
		recordSpanError(span, err)
		span.End()
		cancel()
		metrics.ObserveDBQueryTotal("query", table, status)
		return nil, err
	}
	metrics.ObserveDBQueryTotal("query", table, status)
	return &RowsWithCancel{rows: rows, cancel: cancel, span: span}, nil
}

// QueryRow 执行带超时的单行查询
// 返回 RowWithCancel 包装类型，确保 Scan 完成后才取消 context
// 明确的只读 SQL遇到瞬时连接错误时自动重试一次（与 Query 保持一致）。
// INSERT/UPDATE/DELETE ... RETURNING 不会重试，避免产生重复写入。
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) *RowWithCancel {
	table := TableHint(ctx)
	ctx, cancel := d.withTimeout(ctx)
	ctx, span := d.startSpan(ctx, "query_row", sql, table)
	start := time.Now()
	row := d.pool.QueryRow(ctx, sql, args...)
	return &RowWithCancel{row: row, cancel: cancel, start: start, db: d, ctx: ctx, sql: sql, args: args, span: span, table: table}
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
	span   trace.Span
	table  string
}

func (r *RowWithCancel) release() {
	r.row = nil
	r.cancel = nil
	r.db = nil
	r.ctx = nil
	r.sql = ""
	r.args = nil
	r.span = nil
	r.table = ""
	r.start = time.Time{}
}

// Scan 扫描行数据，完成后自动取消 context
// 对瞬时连接错误自动重试一次
func (r *RowWithCancel) Scan(dest ...any) error {
	defer r.release()

	err := r.row.Scan(dest...)
	duration := time.Since(r.start).Seconds()
	metrics.ObserveDBQueryDuration("query_row", r.table, duration)

	// 只读 SQL 才允许退避重试；写语句的结果不确定性必须交给上层处理。
	if err != nil && isRetryableReadSQL(r.sql) && isConnectionError(err) && r.ctx.Err() == nil {
		logger.L().Warn("query_row connection error, retrying once", zap.Error(err))
		if waitForRetry(r.ctx, jitteredRetryDelay()) {
			retryStart := time.Now()
			retryRow := r.db.pool.QueryRow(r.ctx, r.sql, r.args...)
			err = retryRow.Scan(dest...)
			duration = time.Since(retryStart).Seconds()
			metrics.ObserveDBQueryDuration("query_row_retry", r.table, duration)
		}
	}

	if r.cancel != nil {
		r.cancel()
	}
	status := "ok"
	if err != nil {
		status = "error"
		recordSpanError(r.span, err)
	}
	if r.span != nil {
		r.span.End()
	}
	metrics.ObserveDBQueryTotal("query_row", r.table, status)
	return err
}

// 编译时断言：RowsWithCancel 实现 pgx.Rows 接口
var _ pgx.Rows = (*RowsWithCancel)(nil)

var tracer = otel.Tracer("github.com/StuHelper/StuHelper/server/internal/pkg/db")

// RowsWithCancel 包装 pgx.Rows，确保 Close 后才取消 context
type RowsWithCancel struct {
	rows   pgx.Rows
	cancel context.CancelFunc
	span   trace.Span
}

func (r *RowsWithCancel) Next() bool                    { return r.rows.Next() }
func (r *RowsWithCancel) Scan(dest ...any) error        { return r.rows.Scan(dest...) }
func (r *RowsWithCancel) Err() error                    { return r.rows.Err() }
func (r *RowsWithCancel) CommandTag() pgconn.CommandTag { return r.rows.CommandTag() }
func (r *RowsWithCancel) FieldDescriptions() []pgconn.FieldDescription {
	return r.rows.FieldDescriptions()
}
func (r *RowsWithCancel) Values() ([]any, error) { return r.rows.Values() }
func (r *RowsWithCancel) RawValues() [][]byte    { return r.rows.RawValues() }
func (r *RowsWithCancel) Conn() *pgx.Conn        { return r.rows.Conn() }
func (r *RowsWithCancel) Close() {
	if r == nil {
		return
	}
	rows := r.rows
	cancel := r.cancel
	span := r.span
	r.rows = nil
	r.cancel = nil
	r.span = nil

	if rows != nil {
		rows.Close()
	}
	if cancel != nil {
		cancel()
	}
	if span != nil {
		if rows != nil {
			if err := rows.Err(); err != nil {
				recordSpanError(span, err)
			}
		}
		span.End()
	}
}

// Exec 执行带超时的命令。
// 与 Query 不同，Exec 可能承载非幂等写操作，因此禁止自动重试，避免重复写入。
func (d *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return d.exec(ctx, d.timeout, sql, args...)
}

// ExecWithTimeout executes a command with an operation-specific timeout.
// The context remains derived from the caller, so shutdown and caller
// cancellation always win over the explicit budget.
func (d *DB) ExecWithTimeout(
	ctx context.Context,
	timeout time.Duration,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return d.exec(ctx, timeout, sql, args...)
}

func (d *DB) exec(
	ctx context.Context,
	timeout time.Duration,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	table := TableHint(ctx)
	ctx, cancel := d.withExplicitTimeout(ctx, timeout)
	defer cancel()
	ctx, span := d.startSpan(ctx, "exec", sql, table)
	defer span.End()
	start := time.Now()
	tag, err := d.pool.Exec(ctx, sql, args...)
	duration := time.Since(start).Seconds()
	metrics.ObserveDBQueryDuration("exec", table, duration)

	status := "ok"
	if err != nil {
		status = "error"
		recordSpanError(span, err)
	}
	metrics.ObserveDBQueryTotal("exec", table, status)
	return tag, err
}

// pingTimeout Ping 专用超时上限
const pingTimeout = 3 * time.Second

// Ping 测试连接
// 使用 min(查询超时, pingTimeout) 避免健康检查挂起过久
func (d *DB) Ping(ctx context.Context) error {
	timeout := min(d.timeout, pingTimeout)
	ctx, cancel := context.WithTimeout(ctxutil.Normalize(ctx), timeout)
	defer cancel()
	return d.pool.Ping(ctx)
}

// Close 关闭连接池（安全支持多次调用）
func (d *DB) Close() {
	if d == nil {
		return
	}
	d.closeOnce.Do(func() {
		if d.stopCh != nil {
			close(d.stopCh)
		}
		d.wg.Wait()
		if d.pool != nil {
			d.pool.Close()
		}
	})
}

// collectPoolMetrics 定期采集连接池指标
func (d *DB) collectPoolMetrics() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "db: collectPoolMetrics goroutine panicked: %v\n", r)
		}
	}()
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

func (d *DB) startSpan(ctx context.Context, operation, sql, table string) (context.Context, trace.Span) {
	return tracer.Start(ctxutil.Normalize(ctx), "postgres."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation.name", extractSQLOperation(sql)),
			attribute.String("db.query.table", metrics.NormalizeDBTable(table)),
		),
	)
}

func extractSQLOperation(sql string) string {
	fields := strings.Fields(strings.TrimSpace(sql))
	if len(fields) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(fields[0])
}

// isRetryableReadSQL deliberately uses a small allowlist. CTE and EXPLAIN may
// contain data-changing statements, while SELECT can be safely retried under
// the repository convention that data mutations use explicit DML.
func isRetryableReadSQL(sql string) bool {
	switch extractSQLOperation(sql) {
	case "SELECT", "SHOW", "TABLE", "VALUES":
		return true
	default:
		return false
	}
}

func recordSpanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Begin 开始事务（事务内的操作由调用者控制超时）
func (d *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctxutil.Normalize(ctx))
}

// WithTx 事务包装函数，自动处理提交和回滚
// 事务使用 txTimeoutMultiplier 倍查询超时作为上限，防止长事务占用连接
// 如果 fn 返回 error，事务会回滚；否则提交
func (d *DB) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	txTimeout := d.timeout * txTimeoutMultiplier
	ctx, cancel := context.WithTimeout(ctxutil.Normalize(ctx), txTimeout)
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
		rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rbCancel()
		if rbErr := tx.Rollback(rbCtx); rbErr != nil {
			logger.L().Warn("tx rollback failed",
				zap.Error(rbErr))
		}
		return err
	}

	// 回调可能忽略了 context 并在事务期限到达后返回 nil。此时继续使用独立
	// context 提交会把已经超时或被调用方取消的请求写入数据库，违背调用方对
	// 事务边界的预期。提交前显式检查，并用独立短 context 完成回滚。
	if err := ctx.Err(); err != nil {
		rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rbCancel()
		if rbErr := tx.Rollback(rbCtx); rbErr != nil {
			logger.L().Warn("tx rollback failed after context ended",
				zap.Error(rbErr))
		}
		return fmt.Errorf("transaction context ended before commit: %w", err)
	}

	commitCtx, commitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer commitCancel()
	return tx.Commit(commitCtx)
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
