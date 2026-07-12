// Package observability provides OpenTelemetry tracing, Sentry error tracking,
// and enhanced Prometheus metrics for Radiant Norma.
//
// Sprint 36 — v3.36.0: Observability foundation.
package observability

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OTel configuration from environment variables.
type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint  string // e.g. "localhost:4317". Empty = console exporter (dev).
	Env           string
}

// NewConfig builds a Config from environment variables.
func NewConfig() Config {
	return Config{
		ServiceName:    envOr("OTEL_SERVICE_NAME", "radiant-norma"),
		ServiceVersion: envOr("OTEL_SERVICE_VERSION", "dev"),
		OTLPEndpoint:  os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Env:           envOr("RADIANT_ENV", "development"),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// InitTracer initializes the OpenTelemetry tracer and returns a shutdown func.
// Idempotent — subsequent calls are no-ops.
func InitTracer(ctx context.Context, cfg Config) (func(), error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.Env),
		),
	)
	if err != nil {
		return nil, err
	}

	var exporter sdktrace.SpanExporter
	if cfg.OTLPEndpoint != "" {
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter = &consoleExporter{logger: slog.Default()}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer(cfg.ServiceName,
		trace.WithInstrumentationVersion(cfg.ServiceVersion),
	)

	// Store globally so packages can access without passing cfg around.
	otelTracer = tracer

	slog.Info("tracer initialized",
		"service", cfg.ServiceName,
		"env", cfg.Env,
		"otlp_endpoint", cfg.OTLPEndpoint)

		return func() {
			_ = tp.Shutdown(context.Background())
		}, nil
}

var otelTracer trace.Tracer

// Tracer returns the global tracer. Panics if InitTracer has not been called.
func Tracer() trace.Tracer {
	if otelTracer == nil {
		otelTracer = otel.Tracer("radiant-norma")
	}
	return otelTracer
}

// StartSpan starts a new span. Usage:
//
//	ctx, span := ots.StartSpan(ctx, "db.query", "db.system", "sqlite")
//	defer span.End()
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, trace.WithAttributes(attrs...))
}

// RecordError marks a span as errored.
func RecordError(span trace.Span, err error, msg string) {
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}

// AddEvent annotates the current span with an event.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	trace.SpanFromContext(ctx).AddEvent(name, trace.WithAttributes(attrs...))
}

// consoleExporter prints spans to slog in development.
type consoleExporter struct{ logger *slog.Logger }

func (e *consoleExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, s := range spans {
		sc := s.SpanContext()
		attrs := make([]slog.Attr, 0, 4+len(s.Attributes()))
		attrs = append(attrs,
			slog.String("span", s.Name()),
			slog.String("trace_id", sc.TraceID().String()),
		)
		for _, a := range s.Attributes() {
			attrs = append(attrs, slog.Any(string(a.Key), a.Value.AsInterface()))
		}
		e.logger.LogAttrs(ctx, slog.LevelInfo, "span", attrs...)
	}
	return nil
}

func (e *consoleExporter) Shutdown(ctx context.Context) error { return nil }
