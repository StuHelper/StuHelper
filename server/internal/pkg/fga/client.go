// Package fga 封装 OpenFGA 关系型授权引擎客户端。
// 提供资源关系写入和权限检查能力。
package fga

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	openfga "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

// Client OpenFGA 授权引擎客户端
type Client struct {
	fga     *client.OpenFgaClient
	storeID string
	modelID string
}

var tracer = otel.Tracer("github.com/StuHelper/StuHelper/server/internal/pkg/fga")

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

// ListObjects lists object IDs of one type for which the user has the relation.
func (c *Client) ListObjects(ctx context.Context, user, relation, objectType string) ([]string, error) {
	if err := validateTupleField(user, "user"); err != nil {
		return nil, err
	}
	if err := validateTupleField(relation, "relation"); err != nil {
		return nil, err
	}
	if err := validateTupleField(objectType, "object type"); err != nil {
		return nil, err
	}
	start := time.Now()
	ctx, span := c.startSpan(ctx, "list_objects", relation, objectType)
	defer span.End()

	body := client.ClientListObjectsRequest{
		User:     user,
		Relation: relation,
		Type:     objectType,
	}
	resp, err := c.fga.ListObjects(ctx).Body(body).Execute()
	metrics.ObserveExternalRequest("openfga", "list_objects", start, err)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("fga: list objects failed for %s#%s@%s: %w", objectType, relation, user, err)
	}

	objects := append([]string(nil), resp.Objects...)
	sort.Strings(objects)
	return objects, nil
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
	return c.deleteTuples(ctx, tuples, false)
}

// DeleteTuplesIgnoringMissing 批量删除授权关系，已不存在的 tuple 视为幂等成功。
func (c *Client) DeleteTuplesIgnoringMissing(ctx context.Context, tuples []Tuple) error {
	return c.deleteTuples(ctx, tuples, true)
}

func (c *Client) deleteTuples(ctx context.Context, tuples []Tuple, ignoreMissing bool) error {
	if len(tuples) == 0 {
		return nil
	}
	start := time.Now()
	ctx, span := c.startSpan(ctx, "delete", tuples[0].Relation, tuples[0].Object)
	defer span.End()
	span.SetAttributes(attribute.Int("fga.tuple_count", len(tuples)))

	deletes := make([]openfga.TupleKeyWithoutCondition, len(tuples))
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
		deletes[i] = openfga.TupleKeyWithoutCondition{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		}
	}

	body := client.ClientWriteRequest{
		Deletes: deletes,
	}
	request := c.fga.Write(ctx).Body(body)
	if ignoreMissing {
		request = request.Options(client.ClientWriteOptions{
			Conflict: client.ClientWriteConflictOptions{
				OnMissingDeletes: client.CLIENT_WRITE_REQUEST_ON_MISSING_DELETES_IGNORE,
			},
		})
	}
	_, err := request.Execute()
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

	ctx, span := c.startSpan(ctx, "read", relation, object)
	defer span.End()

	body := client.ClientReadRequest{
		Object:   openfga.PtrString(object),
		Relation: openfga.PtrString(relation),
	}
	result := make([]Tuple, 0)
	continuationToken := ""
	seenTokens := make(map[string]struct{})
	for {
		options := client.ClientReadOptions{
			PageSize:    openfga.PtrInt32(100),
			Consistency: openfga.CONSISTENCYPREFERENCE_HIGHER_CONSISTENCY.Ptr(),
		}
		if continuationToken != "" {
			options.ContinuationToken = openfga.PtrString(continuationToken)
		}
		requestStart := time.Now()
		resp, err := c.fga.Read(ctx).Body(body).Options(options).Execute()
		metrics.ObserveExternalRequest("openfga", "read_tuples", requestStart, err)
		if err != nil {
			recordSpanError(span, err)
			return nil, fmt.Errorf("fga: read tuples failed for %s#%s: %w", object, relation, err)
		}
		for _, tuple := range resp.Tuples {
			result = append(result, Tuple{
				User:     tuple.Key.GetUser(),
				Relation: tuple.Key.GetRelation(),
				Object:   tuple.Key.GetObject(),
			})
		}
		continuationToken = strings.TrimSpace(resp.GetContinuationToken())
		if continuationToken == "" {
			break
		}
		if _, repeated := seenTokens[continuationToken]; repeated {
			return nil, fmt.Errorf("fga: read tuples returned a repeated continuation token for %s#%s", object, relation)
		}
		seenTokens[continuationToken] = struct{}{}
	}
	return result, nil
}

// TupleExists 精确读取一个 direct tuple 是否存在。
// 该方法不使用 Check，避免把模型中的 computed userset 误判成待删除的 direct tuple。
func (c *Client) TupleExists(ctx context.Context, tuple Tuple) (bool, error) {
	if err := validateTupleField(tuple.User, "user"); err != nil {
		return false, err
	}
	if err := validateTupleField(tuple.Relation, "relation"); err != nil {
		return false, err
	}
	if err := validateTupleField(tuple.Object, "object"); err != nil {
		return false, err
	}

	start := time.Now()
	ctx, span := c.startSpan(ctx, "read_exact", tuple.Relation, tuple.Object)
	defer span.End()

	body := client.ClientReadRequest{
		User:     openfga.PtrString(tuple.User),
		Relation: openfga.PtrString(tuple.Relation),
		Object:   openfga.PtrString(tuple.Object),
	}
	resp, err := c.fga.Read(ctx).
		Body(body).
		Options(client.ClientReadOptions{
			Consistency: openfga.CONSISTENCYPREFERENCE_HIGHER_CONSISTENCY.Ptr(),
		}).
		Execute()
	metrics.ObserveExternalRequest("openfga", "read_tuple", start, err)
	if err != nil {
		recordSpanError(span, err)
		return false, fmt.Errorf(
			"fga: read tuple failed for %s#%s@%s: %w",
			tuple.Object,
			tuple.Relation,
			tuple.User,
			err,
		)
	}

	for _, existing := range resp.Tuples {
		if existing.Key.GetUser() == tuple.User &&
			existing.Key.GetRelation() == tuple.Relation &&
			existing.Key.GetObject() == tuple.Object {
			return true, nil
		}
	}
	return false, nil
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
