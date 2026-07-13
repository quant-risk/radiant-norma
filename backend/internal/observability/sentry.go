// observability/sentry.go — Sprint 36: Sentry error tracking.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// SentryConfig holds Sentry configuration from environment variables.
type SentryConfig struct {
	DSN         string
	Environment string
	Release     string
	SampleRate  float64
}

// NewSentryConfig builds a SentryConfig from environment variables.
func NewSentryConfig() SentryConfig {
	return SentryConfig{
		DSN:         os.Getenv("SENTRY_DSN"),
		Environment: envOr("RADIANT_ENV", "development"),
		Release:     os.Getenv("RADIANT_VERSION"),
		SampleRate:  1.0,
	}
}

// InitSentry initializes the Sentry client. Returns a shutdown func.
func InitSentry(cfg SentryConfig) (func(), error) {
	if cfg.DSN == "" {
		slog.Info("sentry disabled (SENTRY_DSN not set)")
		return func() {}, nil
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		AttachStacktrace: true,
		SendDefaultPII:   false,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			scrubEvent(event)
			return event
		},
	}); err != nil {
		return nil, err
	}

	slog.Info("sentry initialized", "environment", cfg.Environment, "release", cfg.Release)
	return func() { sentry.Flush(5 * time.Second) }, nil
}

// scrubEvent removes or redacts sensitive keys from event fields.
func scrubEvent(e *sentry.Event) {
	sensitive := map[string]bool{
		"authorization":    true,
		"x-if-id":          true,
		"cookie":           true,
		"x-api-key":        true,
		"x-bacen-senha":    true,
		"x-sisbacen-token": true,
	}

	isSensitive := func(k string) bool {
		return sensitive[strings.ToLower(k)]
	}

	// Tags: map[string]string
	for k := range e.Tags {
		if isSensitive(k) {
			e.Tags[k] = "[redacted]"
		}
	}

	// Breadcrumbs.
	for _, b := range e.Breadcrumbs {
		for k := range b.Data {
			if isSensitive(k) {
				b.Data[k] = "[redacted]"
			}
		}
	}
}

// CaptureException reports an error to Sentry with trace correlation.
func CaptureException(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetContext("trace", map[string]any{
			"trace_id": sc.TraceID().String(),
			"span_id":  sc.SpanID().String(),
		})
		hub.CaptureException(err)
	})
}

// FlushSentry flushes Sentry's queue. Call before shutdown.
func FlushSentry() { sentry.Flush(5 * time.Second) }

// AddBreadcrumb records a Sentry breadcrumb for an audit event.
// This links mutation audit events (envios, schema changes, wizard steps, etc.)
// to the current Sentry transaction so they appear in error reports.
//
// Sprint 36 — v3.36.0: breadcrumbs on all mutation audit events.
func AddBreadcrumb(ctx context.Context, category, message string, data map[string]any) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category:  category,
		Message:   message,
		Data:     data,
		Timestamp: time.Now(),
	}, nil)
}

// SentryMiddleware returns an HTTP middleware that creates a span per request,
// propagates W3C trace context, records panics, and reports errors to Sentry.
func SentryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(
				r.Context(),
				propagation.HeaderCarrier(r.Header),
			)
			spanName := r.Method + " " + r.URL.Path
			ctx, span := Tracer().Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.route", r.URL.Path),
				),
			)
			defer span.End()

			// Propagate W3C trace context downstream.
			h := make(http.Header)
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(h))
			for k, v := range h {
				r.Header[k] = v
			}

			sw := &statusRecorder{ResponseWriter: w, code: 200}
			r = r.WithContext(ctx)

			defer func() {
				if p := recover(); p != nil {
					span.AddEvent("panic", trace.WithAttributes(
						attribute.String("panic_value", fmt.Sprintf("%v", p)),
					))
					span.SetStatus(codes.Error, "panic")
					sentry.CurrentHub().CaptureException(fmt.Errorf("panic recovered: %v", p))
					panic(p) // re-panic so upstream Recoverer middleware handles it
				}
			}()

			next.ServeHTTP(sw, r)

			span.SetAttributes(attribute.Int("http.status_code", sw.code))
			if sw.code >= 400 {
				span.SetStatus(codes.Error, http.StatusText(sw.code))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.code = code
	sr.ResponseWriter.WriteHeader(code)
}
