// Package api — métricas Prometheus para rate limiter + endpoint /metrics.
//
// Sprint 17 — v3.7.0 [S17.5]: observability.
// Sem métricas, operador não sabe se rate limit está bloqueando tráfego
// legítimo, se Redis está fail-open, ou qual bucket está mais pressionado.
//
// Implementação hand-rolled (não usa github.com/prometheus/client_golang)
// porque:
//   - Precisamos só de counters + 1 gauge — overhead da lib é overkill
//   - Text exposition format é ~30 linhas de código (ver Render())
//   - Zero dependência adicional = binary size não cresce
//
// Trade-off: se no futuro precisarmos de histogram/quantile/summary,
// migrar para prometheus/client_golang. Por enquanto counters bastam.
//
// Format: Prometheus text exposition format v0.0.4 (compat com scrapers
// modernos). Cada métrica tem HELP + TYPE + linhas com labels.

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
}

// NewMetrics cria instância zerada.
func NewMetrics() *Metrics {
	m := &Metrics{}
	m.backendUp.Store(true)
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
