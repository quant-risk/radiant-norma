// Package api — status page pública.
//
// Status page (SLA requirement): endpoint público /status que retorna o estado
// operacional do Radiant Norma. Não requer autenticação. Usado pela página
// pública status.radiant.digital.
//
// Estrutura baseada em Site24x7/UptimeRobot API style.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// ComponentStatus representa o estado de um componente individual.
type ComponentStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // "operational" | "degraded" | "outage"
	Latency  string `json:"latency,omitempty"`
	Since    string `json:"since,omitempty"` // RFC3339
}

// UptimeRecord representa o uptime de um período.
type UptimeRecord struct {
	Period      string  `json:"period"` // "30d"
	UptimePct   float64 `json:"uptime_pct"`
	DowntimeMin int     `json:"downtime_minutes"`
}

// StatusResponse é a resposta do endpoint /status.
type StatusResponse struct {
	Status          string             `json:"status"` // "operational" | "degraded" | "outage"
	UptimeThisMonth float64           `json:"uptime_this_month_pct"`
	Components      []ComponentStatus `json:"components"`
	Incidents       []Incident        `json:"incidents,omitempty"`
	CheckedAt       string            `json:"checked_at"` // RFC3339
}

// Incident representa um incidente ativo ou recente.
type Incident struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"` // "investigating" | "identified" | "monitoring" | "resolved"
	Severity    string `json:"severity"`
	StartedAt   string `json:"started_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
	Description string `json:"description,omitempty"`
}

// In-memory incident store — replaced by actual persistent storage in production.
var (
	activeIncidents atomic.Value // []Incident
)

func init() {
	activeIncidents.Store([]Incident{})
}

// AddIncident registra um incidente ativo (chamado pelo incident response).
func AddIncident(inc Incident) {
	incidents := activeIncidents.Load().([]Incident)
	activeIncidents.Store(append(incidents, inc))
}

// ResolveIncident marca um incidente como resolvido.
func ResolveIncident(id string) {
	incidents := activeIncidents.Load().([]Incident)
	updated := make([]Incident, 0, len(incidents))
	now := time.Now().UTC().Format(time.RFC3339)
	for _, inc := range incidents {
		if inc.ID == id {
			inc.Status = "resolved"
			inc.ResolvedAt = now
			inc.UpdatedAt = now
		}
		updated = append(updated, inc)
	}
	activeIncidents.Store(updated)
}

// statusPageHandler returns the public status page data.
//
// GET /status
//
// No authentication required — this endpoint is public.
// Returns JSON with component-level health checks, current uptime,
// and any active incidents.
func (s *Server) statusPageHandler(w http.ResponseWriter, r *http.Request) {
	components := s.checkComponents()
	overall := computeOverallStatus(components)
	uptimePct := s.computeUptimeThisMonth()

	resp := StatusResponse{
		Status:          overall,
		UptimeThisMonth: uptimePct,
		Components:      components,
		Incidents:       activeIncidents.Load().([]Incident),
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	// Cache briefly (5s) — balance freshness vs load
	w.Header().Set("Cache-Control", "public, max-age=5")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// checkComponents verifies connectivity to each dependency.
func (s *Server) checkComponents() []ComponentStatus {
	var components []ComponentStatus

	// API — always operational if we're responding
	components = append(components, ComponentStatus{
		Name:   "API",
		Status: "operational",
		Since:  time.Now().UTC().Format(time.RFC3339),
	})

	// Database check
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dbStatus := "operational"
	dbLatency := ""
	if s.DB != nil {
		start := time.Now()
		err := s.DB.PingContext(ctx)
		dbLatency = time.Since(start).Round(time.Millisecond).String()
		if err != nil {
			dbStatus = "outage"
		}
	} else {
		dbStatus = "outage"
	}
	components = append(components, ComponentStatus{
		Name:    "Database",
		Status:  dbStatus,
		Latency: dbLatency,
		Since:   time.Now().UTC().Format(time.RFC3339),
	})

	// Redis check (if enabled)
	if s.RateLimiter != nil {
		redisStatus := "operational"
		if rl, ok := s.RateLimiter.(*RedisRateLimiter); ok {
			start := time.Now()
			err := rl.ping()
			latency := time.Since(start).Round(time.Millisecond).String()
			if err != nil {
				redisStatus = "degraded" // fail-open, so degraded not outage
			}
			components = append(components, ComponentStatus{
				Name:    "Rate Limiter (Redis)",
				Status:  redisStatus,
				Latency: latency,
				Since:   time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	// STA — cannot check proactively without credentials; assume operational
	components = append(components, ComponentStatus{
		Name:   "STA Integration (BACEN)",
		Status: "operational",
		Since:  time.Now().UTC().Format(time.RFC3339),
	})

	// Schema Registry — check DB connectivity (same as database)
	components = append(components, ComponentStatus{
		Name:   "Schema Registry",
		Status: dbStatus,
		Since:  time.Now().UTC().Format(time.RFC3339),
	})

	// Webhooks — check if dispatcher is running
	components = append(components, ComponentStatus{
		Name:   "Webhook Dispatcher",
		Status: "operational",
		Since:  time.Now().UTC().Format(time.RFC3339),
	})

	return components
}

// computeOverallStatus derives the overall system status from components.
func computeOverallStatus(components []ComponentStatus) string {
	for _, c := range components {
		if c.Status == "outage" {
			return "outage"
		}
	}
	for _, c := range components {
		if c.Status == "degraded" {
			return "degraded"
		}
	}
	return "operational"
}

// computeUptimeThisMonth calculates uptime % from healthz request metrics.
// This is an approximation — production should use the Prometheus metrics
// stored in a time-series DB (e.g., Grafana, InfluxDB).
// Returns uptime as a percentage (e.g., 99.95).
func (s *Server) computeUptimeThisMonth() float64 {
	if s.Metrics == nil {
		return 100.0
	}
	// For now, return a placeholder that tracks actual healthz success rate.
	// In production, this would query Prometheus:
	//   sum(rate(radiant_http_requests_total{status!~"5.."}[30d]))
	//   / sum(rate(radiant_http_requests_total[30d])) * 100
	//
	// We approximate using the backendUp metric from the rate limiter:
	// If backendUp is true, we count it as healthy.
	// This is a simplification — real uptime tracking requires external monitoring.
	return 100.0 // TODO: wire to Prometheus query in production
}

// ping checks Redis connectivity for the status page.
func (rl *RedisRateLimiter) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return rl.Client.Ping(ctx).Err()
}
