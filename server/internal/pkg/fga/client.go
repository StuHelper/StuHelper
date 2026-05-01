// Package fga 封装 OpenFGA 关系型授权引擎客户端。
// 提供资源关系写入和权限检查能力。
package fga

import (
	"context"
	"fmt"
	"strings"
	"time"

	openfga "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// Client OpenFGA 授权引擎客户端
type Client struct {
	fga     *client.OpenFgaClient
	storeID string
	modelID string
}

var tracer = otel.Tracer("git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga")

// DefaultWriteTimeout 是 FGA 写入/同步操作的默认超时，防止请求或后台协程无限阻塞。
const DefaultWriteTimeout = 10 * time.Second

// NewClient 创建 OpenFGA 客户端。
// OpenFGA 是运行时必需依赖，缺少关键配置时直接返回错误。
func NewClient(cfg config.OpenFGAConfig) (*Client, error) {
	if cfg.StoreID == "" {
		return nil, fmt.Errorf("fga: StoreID is required")
	}
	if cfg.APIUrl == "" {
		return nil, fmt.Errorf("fga: APIUrl is required")
	}
	if cfg.AuthorizationModelID == "" {
		return nil, fmt.Errorf("fga: AuthorizationModelID is required")
	}

	fgaClient, err := client.NewSdkClient(&client.ClientConfiguration{
		ApiUrl:               cfg.APIUrl,
		StoreId:              cfg.StoreID,
		AuthorizationModelId: cfg.AuthorizationModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("fga: failed to create client: %w", err)
	}

	return &Client{
		fga:     fgaClient,
		storeID: cfg.StoreID,
		modelID: cfg.AuthorizationModelID,
	}, nil
}

// Tuple 表示一条授权关系
type Tuple struct {
	User     string // 如 "user:123"，ID 部分使用 StuHelper users.id
	Relation string // 如 "author", "can_delete"
	Object   string // 如 "review:100"
}

// validateTupleField 验证 FGA tuple 字段格式（type:id）
func validateTupleField(field, name string) error {
	if field == "" {
		return fmt.Errorf("fga: %s must not be empty", name)
	}
	if strings.ContainsAny(field, "\x00\n\r") {
		return fmt.Errorf("fga: %s contains invalid characters", name)
	}
	return nil
}

// Check 检查用户对资源是否具有指定关系（权限）
func (c *Client) Check(ctx context.Context, user, relation, object string) (bool, error) {
	if err := validateTupleField(user, "user"); err != nil {
		return false, err
	}
	if err := validateTupleField(relation, "relation"); err != nil {
		return false, err
	}
	if err := validateTupleField(object, "object"); err != nil {
		return false, err
	}
	start := time.Now()
	ctx, span := c.startSpan(ctx, "check", relation, object)
	defer span.End()
	body := client.ClientCheckRequest{
		User:     user,
		Relation: relation,
		Object:   object,
	}
	resp, err := c.fga.Check(ctx).Body(body).Execute()
	metrics.ObserveExternalRequest("openfga", "check", start, err)
	if err != nil {
		recordSpanError(span, err)
		return false, fmt.Errorf("fga: check failed for %s#%s@%s: %w", object, relation, user, err)
	}
	if resp.Allowed == nil {
		return false, nil
	}
	return *resp.Allowed, nil
}

// WriteTuples 批量写入授权关系
func (c *Client) WriteTuples(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	start := time.Now()
	ctx, span := c.startSpan(ctx, "write", tuples[0].Relation, tuples[0].Object)
	defer span.End()
	span.SetAttributes(attribute.Int("fga.tuple_count", len(tuples)))

	writes := make([]openfga.TupleKey, len(tuples))
	for i, t := range tuples {
		if err := validateTupleField(t.User, "user"); err != nil {
			return err
		}
		if err := validateTupleField(t.Relation, "relation"); err != nil {
			return err
		}
		if err := validateTupleField(t.Object, "object"); err != nil {
			return err
		}
		writes[i] = openfga.TupleKey{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		}
	}

	body := client.ClientWriteRequest{
		Writes: writes,
	}
	_, err := c.fga.Write(ctx).Body(body).Execute()
	metrics.ObserveExternalRequest("openfga", "write_tuples", start, err)
	if err != nil {
		recordSpanError(span, err)
		return fmt.Errorf("fga: write tuples failed: %w", err)
	}
	return nil
}

// DeleteTuples 批量删除授权关系
func (c *Client) DeleteTuples(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	start := time.Now()
	ctx, span := c.startSpan(ctx, "delete", tuples[0].Relation, tuples[0].Object)
	defer span.End()
	span.SetAttributes(attribute.Int("fga.tuple_count", len(tuples)))

	deletes := make([]openfga.TupleKeyWithoutCondition, len(tuples))
	for i, t := range tuples {
		deletes[i] = openfga.TupleKeyWithoutCondition{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		}
	}

	body := client.ClientWriteRequest{
		Deletes: deletes,
	}
	_, err := c.fga.Write(ctx).Body(body).Execute()
	metrics.ObserveExternalRequest("openfga", "delete_tuples", start, err)
	if err != nil {
		recordSpanError(span, err)
		return fmt.Errorf("fga: delete tuples failed: %w", err)
	}
	return nil
}

// ReadTuples 读取指定对象/关系的现有 tuples，用于幂等重建投影。
func (c *Client) ReadTuples(ctx context.Context, object, relation string) ([]Tuple, error) {
	if err := validateTupleField(object, "object"); err != nil {
		return nil, err
	}
	if err := validateTupleField(relation, "relation"); err != nil {
		return nil, err
	}

	start := time.Now()
	ctx, span := c.startSpan(ctx, "read", relation, object)
	defer span.End()

	body := client.ClientReadRequest{
		Object:   openfga.PtrString(object),
		Relation: openfga.PtrString(relation),
	}
	resp, err := c.fga.Read(ctx).Body(body).Execute()
	metrics.ObserveExternalRequest("openfga", "read_tuples", start, err)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("fga: read tuples failed for %s#%s: %w", object, relation, err)
	}

	result := make([]Tuple, 0, len(resp.Tuples))
	for _, tuple := range resp.Tuples {
		result = append(result, Tuple{
			User:     tuple.Key.GetUser(),
			Relation: tuple.Key.GetRelation(),
			Object:   tuple.Key.GetObject(),
		})
	}
	return result, nil
}

func (c *Client) startSpan(ctx context.Context, operation, relation, object string) (context.Context, trace.Span) {
	objectType := object
	if idx := strings.IndexByte(object, ':'); idx > 0 {
		objectType = object[:idx]
	}
	return tracer.Start(ctx, "openfga."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("authorization.system", "openfga"),
			attribute.String("fga.store_id", c.storeID),
			attribute.String("fga.model_id", c.modelID),
			attribute.String("fga.relation", relation),
			attribute.String("fga.object_type", objectType),
		),
	)
}

func recordSpanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
