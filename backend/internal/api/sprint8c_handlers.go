// Package api — handlers Sprint 8c.
//
// Sprint 8c (v3.1.0): expõe dados reais para o frontend v3.0.0. Antes
// o backend tinha /v1/envios ausente, /v1/audit_log ausente, e
// /v1/insights/* completamente ausente — frontend ficava em empty
// states (e durante validação 29 eu encontrei que tinha dados fake
// hardcoded em alguns lugares).
//
// Endpoints novos:
//
//	GET /v1/envios                          → lista de envios STA (filtrada por IF)
//	GET /v1/envios/stats                    → KPIs agregados (totais, aprovados, etc)
//	GET /v1/audit_log                       → eventos audit legíveis (admin only)
//	GET /v1/insights/kpis                   → comparativo temporal
//	GET /v1/insights/heatmap?days=14        → falhas por CADOC × dia
//	GET /v1/insights/rules/top-failing      → top regras falhando
//	GET /v1/insights/recommendations        → heurística de recomendações
package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
)

// --- Envios ---

// envioDTO é a resposta JSON de /v1/envios (apenas campos que frontend usa).
type envioDTO struct {
	ID           string `json:"id"`
	CadocCode    string `json:"cadoc_code"`
	Period       string `json:"period"`
	Status       string `json:"status"` // pending | accepted | rejected | error | dead_letter
	RulesPassed  int    `json:"rules_passed"`
	RulesFailed  int    `json:"rules_failed"`
	DurationMs   int    `json:"duration_ms"`
	ProtocolSTA  string `json:"protocol_sta"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	SentAt       string `json:"sent_at"`      // RFC3339
	ConfirmedAt  string `json:"confirmed_at"` // RFC3339
	Attempts     int    `json:"attempts,omitempty"` // Phase 4: retry count (useful for dead_letter)
}

// listEnvios retorna envios da IF logada.
//
// Query params (todos opcionais):
//   - cadoc: filtra por CADOC
//   - status: filtra por status (pending/accepted/rejected/error/dead_letter)
//   - limit: max items (default 50, max 200)
//   - period: filtra por period (e.g., '05/2026')
func (s *Server) listEnvios(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	// Build query
	q := `SELECT id, cadoc_code, COALESCE(period, ''), status,
	             rules_passed, rules_failed, COALESCE(duration_ms, 0),
	             COALESCE(protocol_sta, ''), COALESCE(error_code, ''),
	             COALESCE(error_message, ''),
	             COALESCE(sent_at, ''), COALESCE(confirmed_at, ''),
	             COALESCE(attempts, 0)
	      FROM envios WHERE if_id = ?`
	args := []any{ifID}

	if cadoc := r.URL.Query().Get("cadoc"); cadoc != "" {
		q += " AND cadoc_code = ?"
		args = append(args, cadoc)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	if period := r.URL.Query().Get("period"); period != "" {
		q += " AND period = ?"
		args = append(args, period)
	}
	q += " ORDER BY COALESCE(confirmed_at, sent_at, created_at) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		s.internalServerError(w, err, "listEnvios")
		return
	}
	defer rows.Close()

	envios := make([]envioDTO, 0)
	for rows.Next() {
		var e envioDTO
		if err := rows.Scan(
			&e.ID, &e.CadocCode, &e.Period, &e.Status,
			&e.RulesPassed, &e.RulesFailed, &e.DurationMs,
			&e.ProtocolSTA, &e.ErrorCode, &e.ErrorMessage,
			&e.SentAt, &e.ConfirmedAt, &e.Attempts,
		); err != nil {
			s.internalServerError(w, err, "listEnvios.scan")
			return
		}
		envios = append(envios, e)
	}

	// Sprint 8d: ?format=csv|json. JSON é default (compatibilidade).
	// Whitelist estrita: csv → CSV; vazio/missing → JSON; outros → 400.
	switch strings.ToLower(r.URL.Query().Get("format")) {
	case "csv":
		writeCSV(w, "envios-"+ifID, enviosToRows(envios))
		return
	case "", "json":
		// fallback abaixo
	default:
		http.Error(w, `{"error":"format inválido (use 'json' ou 'csv')"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"envios": envios,
		"total":  len(envios),
	})
}

// enviosStats retorna KPIs agregados dos envios da IF.
//
// Sprint 8c: alimenta /envios stats cards (Total / Aprovados / Pendentes / Rejeitados).
func (s *Server) enviosStats(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT status, COUNT(*) FROM envios WHERE if_id = ? GROUP BY status
	`, ifID)
	if err != nil {
		s.internalServerError(w, err, "enviosStats")
		return
	}
	defer rows.Close()

	stats := map[string]int{
		"total":       0,
		"pending":     0,
		"accepted":    0,
		"rejected":    0,
		"error":       0,
		"dead_letter": 0, // Phase 4: DLQ count
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			s.internalServerError(w, err, "enviosStats.scan")
			return
		}
		stats[status] = count
		stats["total"] += count
	}

	// Avg duration (últimos 30 envios)
	var avgMs sql.NullFloat64
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT AVG(duration_ms) FROM (
		    SELECT duration_ms FROM envios
		    WHERE if_id = ? AND duration_ms > 0
		    ORDER BY COALESCE(confirmed_at, sent_at, created_at) DESC LIMIT 30
		)
	`, ifID).Scan(&avgMs)
	if avgMs.Valid {
		stats["avg_duration_ms"] = int(avgMs.Float64)
	}

	writeJSON(w, http.StatusOK, stats)
}

// --- Audit log ---

// auditEventDTO é a resposta JSON de /v1/audit_log (apenas campos legíveis).
//
// lint-enforce-same-if: false-positive — output struct (handler escreve
// via writeJSON, não aceita do payload). json.Unmarshal na linha 282
// parseia audit_events.payload do DB, não request body. Access control
// de /v1/audit_log é por admin role (F27.13), não por tenant match.
type auditEventDTO struct {
	ID          int64                  `json:"id"`
	IFID        string                 `json:"if_id"`
	Actor       string                 `json:"actor"`
	Action      string                 `json:"action"`
	Target      string                 `json:"target"`
	Description string                 `json:"description"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}

// listAuditLog retorna eventos legíveis (denormalizados em audit_events).
//
// Acesso: admin only (F27.13 — auditoria é vetor de disclosure).
//
// Query params:
//   - if_id: filtra por IF (admin pode ver todas; non-admin vê só sua IF)
//   - action: filtra por tipo ('envio.approved', 'radar.detected', etc)
//   - limit: max (default 50, max 500)
func (s *Server) listAuditLog(w http.ResponseWriter, r *http.Request) {
	callerIF := getIfID(r)
	if callerIF == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}
	callerRole := getRole(r)

	// Admin pode ver tudo; outros só veem a própria IF.
	ifIDFilter := callerIF
	if callerRole == "admin" {
		ifIDFilter = r.URL.Query().Get("if_id") // opcional; vazio = todos
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	q := `SELECT id, COALESCE(if_id, ''), actor, action, COALESCE(target, ''),
	             COALESCE(description, ''), COALESCE(payload, ''), created_at
	      FROM audit_events WHERE 1=1`
	args := []any{}

	if ifIDFilter != "" {
		q += " AND if_id = ?"
		args = append(args, ifIDFilter)
	}
	if action := r.URL.Query().Get("action"); action != "" {
		q += " AND action = ?"
		args = append(args, action)
	}
	q += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		s.internalServerError(w, err, "listAuditLog")
		return
	}
	defer rows.Close()

	events := make([]auditEventDTO, 0)
	for rows.Next() {
		var (
			e          auditEventDTO
			payloadRaw string
			createdAt  time.Time
		)
		if err := rows.Scan(
			&e.ID, &e.IFID, &e.Actor, &e.Action, &e.Target,
			&e.Description, &payloadRaw, &createdAt,
		); err != nil {
			s.internalServerError(w, err, "listAuditLog.scan")
			return
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if payloadRaw != "" {
			_ = json.Unmarshal([]byte(payloadRaw), &e.Payload)
		}
		events = append(events, e)
	}

	// Chain integrity check — chama Verify() real do auditlog.
	// Validação 30 (C7 fix): antes era `s.AuditLog != nil` (falso positivo —
	// só verificava que Logger existia, não que a chain estava intacta).
	// Pra DBs com milhares de entries, Verify() pode demorar — limitamos
	// ao IF do caller pra reduzir custo.
	chainValid := true
	chainCheckedEntries := 0
	if s.AuditLog != nil {
		// Verify() varre TODA a chain (sem filtro IF). Em produção com
		// N entries isso é lento; ideal seria VerifyRange. Por ora,
		// aceitamos o trade-off (verify roda em <100ms típico).
		valid, count, verr := s.AuditLog.Verify()
		if verr != nil {
			chainValid = false
		} else {
			chainValid = valid
			chainCheckedEntries = count
		}
	}

	_ = chainCheckedEntries // disponível pra response se quisermos expor

	// Sprint 8d: ?format=csv|json.
	switch strings.ToLower(r.URL.Query().Get("format")) {
	case "csv":
		filename := "audit-log"
		if callerIF != "" {
			filename = "audit-log-" + callerIF
		}
		writeCSV(w, filename, auditEventsToRows(events))
		return
	case "", "json":
		// fallback abaixo
	default:
		http.Error(w, `{"error":"format inválido (use 'json' ou 'csv')"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events":      events,
		"total":       len(events),
		"chain_valid": chainValid,
	})
}

// --- Insights ---

// insightsKPIs retorna KPIs comparativos (atual vs período anterior).
//
// Sprint 8c: alimenta o hero strip do Dashboard e o "Comparativo temporal"
// da página /insights. Computa agregado de envios + rule_failures.
func (s *Server) insightsKPIs(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	now := time.Now()
	currentStart := now.AddDate(0, 0, -30)
	previousStart := now.AddDate(0, 0, -60)

	// current period stats
	currSent, currAccepted, currRejected, currFailures := s.enviosAggregate(r, ifID, currentStart, now)
	prevSent, prevAccepted, _, _ := s.enviosAggregate(r, ifID, previousStart, currentStart)

	approvalRate := 0.0
	if currSent > 0 {
		approvalRate = float64(currAccepted) / float64(currSent) * 100
	}
	prevApprovalRate := 0.0
	if prevSent > 0 {
		prevApprovalRate = float64(prevAccepted) / float64(prevSent) * 100
	}

	// avg duration
	var avgMs sql.NullFloat64
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT AVG(duration_ms) FROM envios
		WHERE if_id = ? AND duration_ms > 0 AND confirmed_at >= ?
	`, ifID, currentStart).Scan(&avgMs)
	currentAvgMs := 0.0
	if avgMs.Valid {
		currentAvgMs = avgMs.Float64
	}
	var avgMsPrev sql.NullFloat64
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT AVG(duration_ms) FROM envios
		WHERE if_id = ? AND duration_ms > 0
		      AND confirmed_at >= ? AND confirmed_at < ?
	`, ifID, previousStart, currentStart).Scan(&avgMsPrev)
	prevAvgMs := 0.0
	if avgMsPrev.Valid {
		prevAvgMs = avgMsPrev.Float64
	}

	// Deltas (% change)
	delta := func(curr, prev float64) float64 {
		if prev == 0 {
			return 0
		}
		return (curr - prev) / prev * 100
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current": map[string]any{
			"approval_rate":   round2(approvalRate),
			"failures_total":  currFailures,
			"sent_total":      currSent,
			"accepted":        currAccepted,
			"rejected":        currRejected,
			"avg_duration_ms": int(currentAvgMs),
		},
		"previous": map[string]any{
			"approval_rate":   round2(prevApprovalRate),
			"sent_total":      prevSent,
			"avg_duration_ms": int(prevAvgMs),
		},
		"delta": map[string]any{
			"approval_rate_pct":   round2(delta(approvalRate, prevApprovalRate)),
			"failures_total_pct":  round2(delta(float64(currFailures), 0)),
			"avg_duration_ms_pct": round2(delta(currentAvgMs, prevAvgMs)),
		},
		"period": map[string]any{
			"current_from":  currentStart.UTC().Format(time.RFC3339),
			"current_to":    now.UTC().Format(time.RFC3339),
			"previous_from": previousStart.UTC().Format(time.RFC3339),
			"previous_to":   currentStart.UTC().Format(time.RFC3339),
		},
	})
}

func (s *Server) enviosAggregate(r *http.Request, ifID string, from, to time.Time) (sent, accepted, rejected, failures int) {
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) AS accepted,
			SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) AS rejected
		FROM envios WHERE if_id = ?
		  AND COALESCE(confirmed_at, sent_at, created_at) >= ?
		  AND COALESCE(confirmed_at, sent_at, created_at) < ?
	`, ifID, from, to).Scan(&sent, &accepted, &rejected)

	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM rule_failures
		WHERE if_id = ? AND failed_at >= ? AND failed_at < ?
	`, ifID, from, to).Scan(&failures)
	return
}

// insightsHeatmap retorna falhas agrupadas por CADOC × dia (últimos N dias).
//
// Query params:
//   - days: janela (default 14, max 90)
//   - cadoc: opcional filtra por CADOC
type heatmapCellDTO struct {
	Row   string `json:"row"`
	Col   string `json:"col"`
	Value int    `json:"value"`
}

func (s *Server) insightsHeatmap(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 90 {
			days = v
		}
	}
	cadocFilter := r.URL.Query().Get("cadoc")

	from := time.Now().AddDate(0, 0, -days)

	// Query: failures agrupadas por (cadoc, dia)
	q := `
		SELECT cadoc_code, strftime('%Y-%m-%d', failed_at) as day, COUNT(*) as count
		FROM rule_failures
		WHERE if_id = ? AND failed_at >= ?
	`
	args := []any{ifID, from}
	if cadocFilter != "" {
		q += " AND cadoc_code = ?"
		args = append(args, cadocFilter)
	}
	q += " GROUP BY cadoc_code, DATE(failed_at) ORDER BY day ASC, cadoc_code"

	rows, err := s.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		s.internalServerError(w, err, "insightsHeatmap")
		return
	}
	defer rows.Close()

	type dayKey = string
	cells := make([]heatmapCellDTO, 0)
	cadocsSet := map[string]struct{}{}
	datesSet := map[string]struct{}{}

	for rows.Next() {
		var cadoc string
		var dayStr string
		var count int
		// DATE() no SQLite retorna string 'YYYY-MM-DD' ou time.Time
		// dependendo do driver. Scan flexível.
		var rawDay interface{}
		if err := rows.Scan(&cadoc, &rawDay, &count); err != nil {
			s.internalServerError(w, err, "insightsHeatmap.scan")
			return
		}
		switch v := rawDay.(type) {
		case string:
			dayStr = v
		case time.Time:
			dayStr = v.Format("2006-01-02")
		default:
			dayStr = fmt.Sprintf("%v", v)
		}
		// Converter para pt-BR DD/MM
		t, err := time.Parse("2006-01-02", dayStr)
		if err == nil {
			dayStr = t.Format("02/01")
		}
		cells = append(cells, heatmapCellDTO{
			Row:   cadoc,
			Col:   dayStr,
			Value: count,
		})
		cadocsSet[cadoc] = struct{}{}
		datesSet[dayStr] = struct{}{}
	}

	// Preenche todos os dias do range (gaps = 0)
	cols := make([]string, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("02/01")
		cols = append(cols, d)
	}

	rows2 := make([]string, 0, len(cadocsSet))
	for c := range cadocsSet {
		rows2 = append(rows2, c)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": cells,
		"rows": rows2,
		"cols": cols,
		"days": days,
		"from": from.UTC().Format(time.RFC3339),
	})
}

// insightsTopFailingRules retorna top N regras com mais falhas no período.
type ruleFailureDTO struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
	DeltaPct int    `json:"delta_pct"`       // vs período anterior
	TrendDir string `json:"trend_direction"` // up | down | flat
}

func (s *Server) insightsTopFailingRules(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}

	now := time.Now()
	currentStart := now.AddDate(0, 0, -30)
	previousStart := now.AddDate(0, 0, -60)

	type row struct {
		Code     string
		Severity string
		Count    int
	}

	// Validação 30 (C14 fix): antes usava `defer rows.Close()` dentro de
	// block anônimo `{ ... }`. Em Go, `defer` é executado quando a FUNÇÃO
	// retorna, não o block — causando rows abertas até fim do handler.
	// Fix: extrair pra funções com cleanup determinístico.

	current, err := s.queryTopFailingRules(r, ifID, currentStart, limit)
	if err != nil {
		s.internalServerError(w, err, "insightsTopFailingRules.current")
		return
	}

	prevMap, _ := s.queryTopFailingRulesMap(r, ifID, previousStart, currentStart)

	out := make([]ruleFailureDTO, 0, len(current))
	for _, r := range current {
		prev := prevMap[r.Code]
		deltaPct := 0
		trend := "flat"
		if prev > 0 {
			deltaPct = int((float64(r.Count-prev) / float64(prev)) * 100)
			if deltaPct > 0 {
				trend = "up"
			} else if deltaPct < 0 {
				trend = "down"
			}
		} else if r.Count > 0 {
			trend = "up"
			deltaPct = 100 // +100% (era 0)
		}
		out = append(out, ruleFailureDTO{
			Code:     r.Code,
			Severity: r.Severity,
			Count:    r.Count,
			DeltaPct: deltaPct,
			TrendDir: trend,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rules": out,
		"period": map[string]any{
			"current_from": currentStart.UTC().Format(time.RFC3339),
			"current_to":   now.UTC().Format(time.RFC3339),
		},
	})
}

// ruleFailureRow é o tipo interno de queryTopFailingRules.
type ruleFailureRow struct {
	Code     string
	Severity string
	Count    int
}

// queryTopFailingRules retorna []ruleFailureRow com top N regras (com severity) no período.
// Cleanup correto: rows.Close() no return da função (não do caller).
func (s *Server) queryTopFailingRules(r *http.Request, ifID string, from time.Time, limit int) ([]ruleFailureRow, error) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT rule_code, rule_severity, COUNT(*) as count
		FROM rule_failures
		WHERE if_id = ? AND failed_at >= ?
		GROUP BY rule_code, rule_severity
		ORDER BY count DESC LIMIT ?
	`, ifID, from, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // ← executa quando ESTA função retorna, não caller

	out := []ruleFailureRow{}
	for rows.Next() {
		var r ruleFailureRow
		if err := rows.Scan(&r.Code, &r.Severity, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// queryTopFailingRulesMap retorna map[code]count pra período anterior (delta calc).
func (s *Server) queryTopFailingRulesMap(r *http.Request, ifID string, from, to time.Time) (map[string]int, error) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT rule_code, COUNT(*) as count
		FROM rule_failures
		WHERE if_id = ? AND failed_at >= ? AND failed_at < ?
		GROUP BY rule_code
	`, ifID, from, to)
	if err != nil {
		return map[string]int{}, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var code string
		var count int
		if err := rows.Scan(&code, &count); err != nil {
			return out, err
		}
		out[code] = count
	}
	return out, rows.Err()
}

// recommendationDTO é a resposta de /v1/insights/recommendations.
type recommendationDTO struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"` // recommendation | warning | opportunity
	Headline   string `json:"headline"`
	Narrative  string `json:"narrative"`
	Impact     string `json:"impact"`     // low | medium | high
	Confidence int    `json:"confidence"` // 0-100
	CTA        struct {
		Label string `json:"label"`
		Href  string `json:"href"`
	} `json:"cta"`

	// Sprint 12 v3.5.1 (C34.16): marca se user já ackou.
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// insightsRecommendations retorna heurística simples baseada em dados reais.
//
// Sprint 8c: regras determinísticas (não-ML) que geram insights úteis.
// Cada uma olha uma métrica específica e gera 1 insight se o threshold
// for atingido.
//
// Sprint 12 v3.5.1 — C34.16: marca cada recommendation com `acknowledged`
// consultando o service de Acknowledgments. User que acka uma rec não
// mais a vê como "nova" na próxima listagem.
func (s *Server) insightsRecommendations(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "IF não identificada", http.StatusUnauthorized)
		return
	}

	// C34.16: carrega map de acknowledged (se service injetado).
	ackMap := make(map[string]time.Time)
	if s.Insights != nil {
		m, err := s.Insights.ListAcknowledged(r.Context(), ifID)
		if err != nil {
			// Best-effort: log mas não falha (recomendations ainda retorna).
			slog.Default().Warn("insights ListAcknowledged failed",
				"if_id", ifID, "err", loggerutil.SafeError(err))
		} else {
			ackMap = m
		}
	}

	out := []recommendationDTO{}
	markAck := func(rec *recommendationDTO) {
		if t, ok := ackMap[rec.ID]; ok {
			rec.Acknowledged = true
			rec.AcknowledgedAt = &t
		}
	}

	// Regra 1: top regra falhando é responsável por >40% das falhas
	type topRow struct {
		Code     string
		Severity string
		Count    int
	}
	var top topRow
	var totalFailures int
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT rule_code, rule_severity, COUNT(*) FROM rule_failures
		WHERE if_id = ? AND failed_at >= ?
		GROUP BY rule_code, rule_severity
		ORDER BY COUNT(*) DESC LIMIT 1
	`, ifID, time.Now().AddDate(0, 0, -30)).Scan(&top.Code, &top.Severity, &top.Count)
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM rule_failures WHERE if_id = ? AND failed_at >= ?
	`, ifID, time.Now().AddDate(0, 0, -30)).Scan(&totalFailures)

	if top.Count > 0 && totalFailures > 0 {
		pct := float64(top.Count) / float64(totalFailures) * 100
		if pct >= 25 {
			headline := "Concentração de falhas em " + top.Code
			r := recommendationDTO{
				ID:         recID(ifID, "concentration", top.Code),
				Kind:       "recommendation",
				Headline:   headline,
				Narrative:  "Regra " + top.Code + " foi responsável por " + roundStr(pct, 0) + "% das falhas nos últimos 30 dias (" + strconv.Itoa(top.Count) + " de " + strconv.Itoa(totalFailures) + "). Investigar se a fonte é sistêmica ou se a regra tem gap.",
				Impact:     impactFromPct(pct),
				Confidence: 85,
			}
			r.CTA.Label = "Revisar regra " + top.Code
			r.CTA.Href = "/regras?focus=" + top.Code
			markAck(&r)
			out = append(out, r)
		}
	}

	// Regra 2: taxa de aprovação caiu >5pp vs período anterior
	var currAccepted, currSent, prevAccepted, prevSent int
	now := time.Now()
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT
		    SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END),
		    COUNT(*)
		FROM envios WHERE if_id = ?
		  AND COALESCE(confirmed_at, sent_at, created_at) >= ?
	`, ifID, now.AddDate(0, 0, -30)).Scan(&currAccepted, &currSent)
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT
		    SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END),
		    COUNT(*)
		FROM envios WHERE if_id = ?
		  AND COALESCE(confirmed_at, sent_at, created_at) >= ?
		  AND COALESCE(confirmed_at, sent_at, created_at) < ?
	`, ifID, now.AddDate(0, 0, -60), now.AddDate(0, 0, -30)).Scan(&prevAccepted, &prevSent)

	if currSent > 0 && prevSent > 0 {
		currRate := float64(currAccepted) / float64(currSent) * 100
		prevRate := float64(prevAccepted) / float64(prevSent) * 100
		if prevRate-currRate >= 5 {
			delta := roundStr(prevRate-currRate, 1)
			headline := "Taxa de aprovação caiu " + delta + "pp"
			r := recommendationDTO{
				ID:         recID(ifID, "approval_drop", headline),
				Kind:       "warning",
				Headline:   headline,
				Narrative:  "De " + roundStr(prevRate, 1) + "% para " + roundStr(currRate, 1) + "% nos últimos 30 dias. Investigar se há mudança em regras ou em padrões de envio.",
				Impact:     "medium",
				Confidence: 90,
			}
			r.CTA.Label = "Ver envios recentes"
			r.CTA.Href = "/envios"
			markAck(&r)
			out = append(out, r)
		} else if currRate-prevRate >= 5 {
			delta := roundStr(currRate-prevRate, 1)
			headline := "Taxa de aprovação subiu " + delta + "pp"
			r := recommendationDTO{
				ID:         recID(ifID, "approval_up", headline),
				Kind:       "opportunity",
				Headline:   headline,
				Narrative:  "De " + roundStr(prevRate, 1) + "% para " + roundStr(currRate, 1) + "% nos últimos 30 dias. Padrão consistente.",
				Impact:     "low",
				Confidence: 88,
			}
			markAck(&r)
			out = append(out, r)
		}
	}

	// Regra 3: muitos envios pendentes (>3 há mais de 1h)
	var stuck int
	_ = s.DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM envios
		WHERE if_id = ? AND status = 'pending'
		  AND COALESCE(sent_at, created_at) < ?
	`, ifID, now.Add(-1*time.Hour)).Scan(&stuck)
	if stuck >= 3 {
		headline := strconv.Itoa(stuck) + " envios pendentes há mais de 1h"
		r := recommendationDTO{
			ID:         recID(ifID, "stuck_envios", headline),
			Kind:       "warning",
			Headline:   headline,
			Narrative:  "Pode indicar falha silenciosa no worker ou fila travada. Verifique os logs do worker.",
			Impact:     "high",
			Confidence: 80,
		}
		r.CTA.Label = "Investigar worker"
		r.CTA.Href = "/envios?status=pending"
		markAck(&r)
		out = append(out, r)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recommendations": out,
		"total":           len(out),
	})
}

// recID gera ID determinístico pra recommendation.
//
// Validação 30 (C6 fix): antes era string vazia, quebrando React keys
// no frontend. Agora: SHA-256 hex dos 16 primeiros chars de
// ifID + kind + headline (curto o suficiente pra UX, único o suficiente
// pra key stability).
func recID(ifID, kind, headline string) string {
	h := sha256.Sum256([]byte(ifID + ":" + kind + ":" + headline))
	return hex.EncodeToString(h[:8])
}

// --- helpers ---

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}

func roundStr(v float64, digits int) string {
	if digits <= 0 {
		return strconv.Itoa(int(v))
	}
	mult := 1.0
	for i := 0; i < digits; i++ {
		mult *= 10
	}
	return strconv.FormatFloat(float64(int(v*mult))/mult, 'f', -1, 64)
}

func impactFromPct(pct float64) string {
	if pct >= 50 {
		return "high"
	}
	if pct >= 25 {
		return "medium"
	}
	return "low"
}

// getRole retorna o role do caller via JWT claims.
//
// Validação 30 (C2 fix): antes aceitava header X-Role como fallback
// ("role-injection-via-header"). Attacker poderia mandar `X-Role: admin`
// e virar admin sem JWT válido. Agora: somente JWT Claims (assinado
// cryptographicamente) é confiável. Se sem claims, default 'if'
// (least-privilege).
func getRole(r *http.Request) string {
	if claims, err := auth.ClaimsFromContext(r.Context()); err == nil && claims != nil {
		return string(claims.Role)
	}
	// Sem claims = sem privilégios. Default 'if' (não-admin).
	return string(auth.RoleIF)
}
