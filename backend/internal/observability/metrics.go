// observability/metrics.go — Sprint 36: Enhanced Prometheus metrics.
package observability

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// BusinessMetrics holds all business-level Prometheus metrics.
// Thread-safe for concurrent use from multiple goroutines.
type BusinessMetrics struct {
	enviosTotal     sync.Map // key="cadoc|status", value=*atomic.Int64
	validationTotal sync.Map // key="rule|severity", value=*atomic.Int64
	webhookTotal    sync.Map // key="status", value=*atomic.Int64
	activeTenants   atomic.Int64
}

// NewBusinessMetrics creates a zeroed BusinessMetrics.
func NewBusinessMetrics() *BusinessMetrics { return &BusinessMetrics{} }

// IncEnvios increments the submissions counter for cadoc + status.
func (m *BusinessMetrics) IncEnvios(cadoc, status string) {
	m.getOrCreateCounter(&m.enviosTotal, cadoc+"|"+status).Add(1)
}

// IncValidation increments the validation error counter for rule + severity.
func (m *BusinessMetrics) IncValidation(rule, severity string) {
	m.getOrCreateCounter(&m.validationTotal, rule+"|"+severity).Add(1)
}

// IncWebhook increments the webhook deliveries counter for status.
func (m *BusinessMetrics) IncWebhook(status string) {
	m.getOrCreateCounter(&m.webhookTotal, status).Add(1)
}

// SetActiveTenants sets the current number of active tenants.
func (m *BusinessMetrics) SetActiveTenants(n int64) { m.activeTenants.Store(n) }

func (m *BusinessMetrics) getOrCreateCounter(mp *sync.Map, key string) *atomic.Int64 {
	if v, ok := mp.Load(key); ok {
		return v.(*atomic.Int64)
	}
	c := &atomic.Int64{}
	actual, _ := mp.LoadOrStore(key, c)
	return actual.(*atomic.Int64)
}

// Render returns all metrics in Prometheus text exposition format.
func (m *BusinessMetrics) Render() string {
	var b strings.Builder

	// radiant_envios_total{cadoc, status}
	fmt.Fprintf(&b, "# HELP radiant_envios_total Total submissions by CADOC and status\n")
	fmt.Fprintf(&b, "# TYPE radiant_envios_total counter\n")
	m.renderCounter(&b, &m.enviosTotal, "radiant_envios_total", []string{"cadoc", "status"})

	// radiant_validation_errors_total{rule, severity}
	fmt.Fprintf(&b, "\n# HELP radiant_validation_errors_total Total validation errors by rule and severity\n")
	fmt.Fprintf(&b, "# TYPE radiant_validation_errors_total counter\n")
	m.renderCounter(&b, &m.validationTotal, "radiant_validation_errors_total", []string{"rule", "severity"})

	// radiant_webhook_deliveries_total{status}
	fmt.Fprintf(&b, "\n# HELP radiant_webhook_deliveries_total Total webhook deliveries by status\n")
	fmt.Fprintf(&b, "# TYPE radiant_webhook_deliveries_total counter\n")
	m.renderCounter(&b, &m.webhookTotal, "radiant_webhook_deliveries_total", []string{"status"})

	// radiant_active_tenants (gauge)
	fmt.Fprintf(&b, "\n# HELP radiant_active_tenants Number of active tenants\n")
	fmt.Fprintf(&b, "# TYPE radiant_active_tenants gauge\n")
	fmt.Fprintf(&b, "radiant_active_tenants %d\n", m.activeTenants.Load())

	return b.String()
}

func (m *BusinessMetrics) renderCounter(b *strings.Builder, mp *sync.Map, name string, labels []string) {
	keys := []string{}
	mp.Range(func(k, _ any) bool {
		keys = append(keys, k.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		v, _ := mp.Load(key)
		n := v.(*atomic.Int64).Load()
		parts := strings.SplitN(key, "|", len(labels))
		lvals := make([]string, len(labels))
		for i := range labels {
			if i < len(parts) {
				lvals[i] = parts[i]
			}
		}
		labelStrs := make([]string, len(labels))
		for i, l := range labels {
			labelStrs[i] = fmt.Sprintf(`%s=%q`, l, lvals[i])
		}
		fmt.Fprintf(b, "%s{%s} %d\n", name, strings.Join(labelStrs, ","), n)
	}
}

// GlobalBusinessMetrics is the package-level instance used by observability.Inc*.
var GlobalBusinessMetrics = NewBusinessMetrics()

// IncEnvios is a package-level convenience helper.
func IncEnvios(cadoc, status string) { GlobalBusinessMetrics.IncEnvios(cadoc, status) }
func IncValidation(rule, sev string) { GlobalBusinessMetrics.IncValidation(rule, sev) }
func IncWebhook(status string)       { GlobalBusinessMetrics.IncWebhook(status) }
func SetActiveTenants(n int64)       { GlobalBusinessMetrics.SetActiveTenants(n) }

// QuantileHistogram is an approximate quantile histogram using linear interpolation.
// Tracks request latencies in pre-defined bucket bounds (ms).
type QuantileHistogram struct {
	mu      sync.Mutex
	count   int64
	sum     int64
	buckets map[int64]int64
}

// NewQuantileHistogram creates a histogram with standard latency buckets.
func NewQuantileHistogram() *QuantileHistogram {
	return &QuantileHistogram{
		buckets: map[int64]int64{
			5: 0, 10: 0, 25: 0, 50: 0,
			100: 0, 250: 0, 500: 0, 1000: 0,
			2500: 0, 5000: 0,
		},
	}
}

// Record adds a sample in milliseconds.
func (h *QuantileHistogram) Record(ms int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += ms
	for bound := range h.buckets {
		if ms <= bound {
			h.buckets[bound]++
		}
	}
}

func (h *QuantileHistogram) Count() int64 { h.mu.Lock(); defer h.mu.Unlock(); return h.count }
func (h *QuantileHistogram) Sum() int64   { h.mu.Lock(); defer h.mu.Unlock(); return h.sum }

// Quantile returns the approximate value at the given quantile (0.0–1.0).
func (h *QuantileHistogram) Quantile(phi float64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0
	}
	target := int64(math.Ceil(float64(h.count) * phi))
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
