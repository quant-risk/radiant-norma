// Package api — métricas Prometheus para rate limiter + endpoint /metrics.
//
// Sprint 17 — v3.7.0 [S17.5]: observability.
// Sprint 36 — v3.36.0: enhanced business metrics.
//
// Hand-rolled (não usa github.com/prometheus/client_golang).
// Trade-off: histogram t-digest simplificado (basta para P50/P95/P99aprox).
// Se precisarmos de percentis exatos, migrar para prometheus/client_golang.

package api

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Metrics centraliza contadores Prometheus-friendly.
//
// Uso:
//
//	m := NewMetrics()
//	m.IncDropped("heavy", "redis")
//	m.IncAllowed("heavy", "redis")
//	m.IncFailOpen("redis")
//	// ... em /v1/metrics handler:
//	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
//	io.WriteString(w, m.Render())
type Metrics struct {
	// Counters: chave = bucket + backend, valor = count
	// sync.Map para acesso concorrente sem contenção global.
	dropped sync.Map // map[string]*atomic.Int64 — key = "bucket|backend"
	allowed sync.Map

	// Counters sem labels (singleton)
	failOpen     atomic.Int64
	allowedTotal atomic.Int64
	droppedTotal atomic.Int64
	backendUp    atomic.Bool // true se backend está respondendo

	// Sprint 36 — Enhanced: request duration histogram per endpoint + method.
	reqDurations sync.Map // key="endpoint|method", value=*histogram

	// Sprint 36 — request total counter (endpoint, method, status_code).
	requestsTotal sync.Map // key="endpoint|method|status", value=*atomic.Int64

	// Sprint 36 — Enhanced: business counters.
	enviosTotal     sync.Map // key="cadoc|status"
	validationTotal sync.Map // key="rule|severity"
	webhookTotal    sync.Map // key="status"
	activeTenants   atomic.Int64

	// Stable bucket key list for histogram rendering.
	seenDurBuckets    map[string]bool
	seenDurBucketsMu  sync.Mutex
}

// NewMetrics cria instância zerada.
func NewMetrics() *Metrics {
	m := &Metrics{}
	m.backendUp.Store(true)
	m.seenDurBuckets = make(map[string]bool)
	return m
}

// counterKey gera chave canônica para map.
func counterKey(bucket, backend string) string {
	return bucket + "|" + backend
}

// getOrCreate busca ou cria counter atomic na map.
func (m *Metrics) getOrCreate(mp *sync.Map, key string) *atomic.Int64 {
	if v, ok := mp.Load(key); ok {
		return v.(*atomic.Int64)
	}
	counter := &atomic.Int64{}
	actual, _ := mp.LoadOrStore(key, counter)
	return actual.(*atomic.Int64)
}

// IncDropped incrementa contador de denials para bucket + backend.
func (m *Metrics) IncDropped(bucket, backend string) {
	m.getOrCreate(&m.dropped, counterKey(bucket, backend)).Add(1)
	m.droppedTotal.Add(1)
}

// IncAllowed incrementa contador de allows para bucket + backend.
func (m *Metrics) IncAllowed(bucket, backend string) {
	m.getOrCreate(&m.allowed, counterKey(bucket, backend)).Add(1)
	m.allowedTotal.Add(1)
}

// IncFailOpen incrementa contador quando Redis backend falhou e
// retornou allow (fail-open path).
func (m *Metrics) IncFailOpen(backend string) {
	m.failOpen.Add(1)
}

// SetBackendUp marca backend como up/down. Redis: chama após cada
// Allow (true se OK, false se fail-open). Útil pra alerting
// `radiant_rate_limit_backend_up == 0`.
func (m *Metrics) SetBackendUp(up bool) {
	m.backendUp.Store(up)
}

// Render serializa métricas no Prometheus text exposition format.
//
// Order: HELP + TYPE vêm antes das linhas. Buckets ordenados
// alfabeticamente pra output estável (diff-friendly).
func (m *Metrics) Render() string {
	var b strings.Builder

	// radiant_rate_limit_allowed_total{bucket, backend}
	m.writeCounter(&b,
		"radiant_rate_limit_allowed_total",
		"Total number of requests allowed by rate limiter",
		&m.allowed,
		[]string{"bucket", "backend"})

	// radiant_rate_limit_dropped_total{bucket, backend}
	m.writeCounter(&b,
		"radiant_rate_limit_dropped_total",
		"Total number of requests denied by rate limiter (429)",
		&m.dropped,
		[]string{"bucket", "backend"})

	// radiant_rate_limit_fail_open_total (sem labels)
	fmt.Fprintf(&b, "# HELP radiant_rate_limit_fail_open_total Total rate limit failures that fell back to allow (Redis down)\n")
	fmt.Fprintf(&b, "# TYPE radiant_rate_limit_fail_open_total counter\n")
	fmt.Fprintf(&b, "radiant_rate_limit_fail_open_total %d\n", m.failOpen.Load())

	// radiant_rate_limit_backend_up (gauge 0/1)
	fmt.Fprintf(&b, "# HELP radiant_rate_limit_backend_up Whether the rate limit backend is responding (1) or in fail-open (0)\n")
	fmt.Fprintf(&b, "# TYPE radiant_rate_limit_backend_up gauge\n")
	if m.backendUp.Load() {
		fmt.Fprintf(&b, "radiant_rate_limit_backend_up 1\n")
	} else {
		fmt.Fprintf(&b, "radiant_rate_limit_backend_up 0\n")
	}

	// Sprint 36 — Enhanced business metrics.
	m.renderBusinessMetrics(&b)

	return b.String()
}

// writeCounter renderiza 1 counter com labels ordenados alfabeticamente.
func (m *Metrics) writeCounter(b *strings.Builder, name, help string, mp *sync.Map, labelNames []string) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)

	// Coleta keys e ordena
	keys := []string{}
	mp.Range(func(k, _ any) bool {
		keys = append(keys, k.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		parts := strings.SplitN(key, "|", len(labelNames))
		v, _ := mp.Load(key)
		count := v.(*atomic.Int64).Load()

		// Render labels
		labels := make([]string, len(labelNames))
		for i, name := range labelNames {
			val := ""
			if i < len(parts) {
				val = parts[i]
			}
			labels[i] = fmt.Sprintf(`%s="%s"`, name, escapeLabelValue(val))
		}
		fmt.Fprintf(b, "%s{%s} %d\n", name, strings.Join(labels, ","), count)
	}
}

// escapeLabelValue escapa caracteres especiais em Prometheus label values.
// Spec: https://prometheus.io/docs/instrumenting/exposition_formats/
func escapeLabelValue(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// WriteTo renderiza métricas direto num io.Writer. Helper para handler.
func (m *Metrics) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, m.Render())
	return int64(n), err
}

// ============================================================================
// Sprint 36 — Enhanced business metrics
// ============================================================================

// ObserveRequest records a request's latency in milliseconds.
// Also increments the request_total counter with the given status code.
func (m *Metrics) ObserveRequest(endpoint, method string, statusCode int, durationMs int64) {
	key := endpoint + "|" + method
	h := m.getOrCreateHistogram(&m.reqDurations, key)
	h.Record(durationMs)
	m.seenDurBucketsMu.Lock()
	m.seenDurBuckets[key] = true
	m.seenDurBucketsMu.Unlock()

	// Sprint 36 — request_total counter.
	reqKey := fmt.Sprintf("%s|%s|%d", endpoint, method, statusCode)
	m.getOrCreate(&m.requestsTotal, reqKey).Add(1)
}

// IncRequest increments the request counter for endpoint + method + status.
// Call this from HTTP middleware after the response is written.
func (m *Metrics) IncRequest(endpoint, method string, statusCode int) {
	reqKey := fmt.Sprintf("%s|%s|%d", endpoint, method, statusCode)
	m.getOrCreate(&m.requestsTotal, reqKey).Add(1)
}

func (m *Metrics) getOrCreateHistogram(mp *sync.Map, key string) *histogram {
	if v, ok := mp.Load(key); ok {
		return v.(*histogram)
	}
	h := newHistogram()
	actual, _ := mp.LoadOrStore(key, h)
	return actual.(*histogram)
}

// IncEnvios increments the submissions counter for cadoc + status.
func (m *Metrics) IncEnvios(cadoc, status string) {
	key := cadoc + "|" + status
	m.getOrCreate(&m.enviosTotal, key).Add(1)
}

// IncValidation increments the validation error counter for rule + severity.
func (m *Metrics) IncValidation(rule, severity string) {
	key := rule + "|" + severity
	m.getOrCreate(&m.validationTotal, key).Add(1)
}

// IncWebhook increments the webhook deliveries counter for status.
func (m *Metrics) IncWebhook(status string) {
	m.getOrCreate(&m.webhookTotal, status).Add(1)
}

// SetActiveTenants sets the current number of active tenants (gauge).
func (m *Metrics) SetActiveTenants(n int64) {
	m.activeTenants.Store(n)
}

// renderBusinessMetrics appends enhanced business metrics to the builder.
func (m *Metrics) renderBusinessMetrics(b *strings.Builder) {
	// Request duration histograms.
	m.seenDurBucketsMu.Lock()
	keys := make([]string, 0, len(m.seenDurBuckets))
	for k := range m.seenDurBuckets {
		keys = append(keys, k)
	}
	m.seenDurBucketsMu.Unlock()
	sort.Strings(keys)

	for _, key := range keys {
		v, _ := m.reqDurations.Load(key)
		h := v.(*histogram)
		parts := strings.SplitN(key, "|", 2)
		endpoint, method := parts[0], parts[1]

		fmt.Fprintf(b, "# HELP radiant_http_request_duration_ms HTTP request latency in milliseconds\n")
		fmt.Fprintf(b, "# TYPE radiant_http_request_duration_ms histogram\n")

		for _, q := range []struct {
			q   string
			phi float64
		}{
			{"0.5", 0.5}, {"0.9", 0.9}, {"0.95", 0.95}, {"0.99", 0.99},
		} {
			val := h.Quantile(q.phi)
			fmt.Fprintf(b, "radiant_http_request_duration_ms{endpoint=%q,method=%q,quantile=%q} %.0f\n",
				endpoint, method, q.q, float64(val))
		}
		fmt.Fprintf(b, "radiant_http_request_duration_ms_sum{endpoint=%q,method=%q} %.0f\n",
			endpoint, method, float64(h.Sum()))
		fmt.Fprintf(b, "radiant_http_request_duration_ms_count{endpoint=%q,method=%q} %d\n",
			endpoint, method, h.Count())
	}

	// Sprint 36 — radiant_http_requests_total{endpoint, method, status_code}.
	fmt.Fprintf(b, "\n# HELP radiant_http_requests_total Total HTTP requests by endpoint, method and status code\n")
	fmt.Fprintf(b, "# TYPE radiant_http_requests_total counter\n")
	m.writeCounter(b, "radiant_http_requests_total", "", &m.requestsTotal, []string{"endpoint", "method", "status"})

	// Envios.
	fmt.Fprintf(b, "\n# HELP radiant_envios_total Total submissions by CADOC and status\n")
	fmt.Fprintf(b, "# TYPE radiant_envios_total counter\n")
	m.writeCounter(b, "radiant_envios_total", "", &m.enviosTotal, []string{"cadoc", "status"})

	// Validation errors.
	fmt.Fprintf(b, "\n# HELP radiant_validation_errors_total Total validation errors by rule and severity\n")
	fmt.Fprintf(b, "# TYPE radiant_validation_errors_total counter\n")
	m.writeCounter(b, "radiant_validation_errors_total", "", &m.validationTotal, []string{"rule", "severity"})

	// Webhook deliveries.
	fmt.Fprintf(b, "\n# HELP radiant_webhook_deliveries_total Total webhook deliveries by status\n")
	fmt.Fprintf(b, "# TYPE radiant_webhook_deliveries_total counter\n")
	m.writeCounter(b, "radiant_webhook_deliveries_total", "", &m.webhookTotal, []string{"status"})

	// Active tenants gauge.
	fmt.Fprintf(b, "\n# HELP radiant_active_tenants Number of active tenants\n")
	fmt.Fprintf(b, "# TYPE radiant_active_tenants gauge\n")
	fmt.Fprintf(b, "radiant_active_tenants %d\n", m.activeTenants.Load())
}

// histogram is an approximate quantile histogram using linear interpolation.
// Buckets have fixed upper bounds (5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000 ms).
type histogram struct {
	mu      sync.Mutex
	count   int64
	sum     int64
	buckets map[int64]int64 // upper bound → count
}

func newHistogram() *histogram {
	return &histogram{
		buckets: map[int64]int64{
			5: 0, 10: 0, 25: 0, 50: 0,
			100: 0, 250: 0, 500: 0, 1000: 0,
			2500: 0, 5000: 0,
		},
	}
}

func (h *histogram) Record(v int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += v
	for bound := range h.buckets {
		if v <= bound {
			h.buckets[bound]++
		}
	}
}

func (h *histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

func (h *histogram) Sum() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

// Quantile returns the approximate quantile (0.0–1.0) value.
func (h *histogram) Quantile(phi float64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0
	}
	target := int64(float64(h.count) * phi)
	var cumulative int64
	bounds := make([]int64, 0, len(h.buckets))
	for b := range h.buckets {
		bounds = append(bounds, b)
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i] < bounds[j] })
	for _, bound := range bounds {
		cumulative += h.buckets[bound]
		if cumulative >= target {
			return bound
		}
	}
	return bounds[len(bounds)-1]
}

