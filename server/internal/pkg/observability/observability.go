package observability

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

// BuildInfo 构建信息，用于资源属性。
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

// ShutdownFunc 关闭可观测性资源。
type ShutdownFunc func(context.Context) error

// Setup 初始化 OpenTelemetry trace provider。
func Setup(ctx context.Context, cfg config.ObservabilityConfig, appEnv string, build BuildInfo) (ShutdownFunc, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	endpoint, insecure, err := normalizeOTLPEndpoint(cfg.OTLPEndpoint, cfg.OTLPInsecure)
	if err != nil {
		return nil, err
	}

	exporterOptions := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTimeout(10 * time.Second),
	}
	if insecure {
		exporterOptions = append(exporterOptions, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("observability: create OTLP exporter: %w", err)
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceNamespace(cfg.ServiceNamespace),
		semconv.ServiceVersion(build.Version),
		semconv.DeploymentEnvironmentName(appEnv),
	}
	if hostname, hostnameErr := os.Hostname(); hostnameErr == nil && strings.TrimSpace(hostname) != "" {
		attrs = append(attrs, semconv.HostName(hostname))
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: create resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TraceSampleRatio))),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithExportTimeout(10*time.Second),
		),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tracerProvider.Shutdown, nil
}

func normalizeOTLPEndpoint(raw string, insecure bool) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", insecure, fmt.Errorf("observability: OTLP endpoint is empty")
	}

	if !strings.Contains(trimmed, "://") {
		return trimmed, insecure, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", insecure, fmt.Errorf("observability: invalid OTLP endpoint %q: %w", raw, err)
	}
	if parsed.Host == "" {
		return "", insecure, fmt.Errorf("observability: invalid OTLP endpoint %q: host is empty", raw)
	}

	return parsed.Host, parsed.Scheme == "http" || insecure, nil
}
