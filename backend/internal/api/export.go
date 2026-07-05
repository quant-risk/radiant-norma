// Package api — Sprint 8d: export CSV/JSON.
//
// Adiciona ?format=csv|json nas rotas existentes para permitir
// export de dados. JSON é o default (mantém compatibilidade);
// CSV é opt-in.
//
// CSV segue RFC 4180 (RFC 4180 commas + quote quando necessário).
// Linhas RFC3339 (ISO 8601 com timezone UTC) — Excel/Sheets parseiam OK.
//
// Segurança: valida o param `format` (whitelist) — sem risk de injection.
// Filtros existentes (cadoc, status, period, limit) preservados.

package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// writeJSONOrCSV serializa `v` em JSON ou CSV dependendo do query param.
//
// Whitelist estrita: aceita apenas "json" (default) ou "csv".
// Headers corretos:
//   - JSON: application/json (default)
//   - CSV: text/csv com filename sugestão via Content-Disposition
func writeJSONOrCSV(w http.ResponseWriter, r *http.Request, baseFilename string, rows []map[string]string, v any) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	switch format {
	case "csv":
		writeCSV(w, baseFilename, rows)
	case "json":
		writeJSON(w, http.StatusOK, v)
	default:
		http.Error(w, `{"error":"format inválido (use 'json' ou 'csv')"}`, http.StatusBadRequest)
	}
}

// writeCSV serializa rows em CSV RFC 4180 + headers corretos.
func writeCSV(w http.ResponseWriter, baseFilename string, rows []map[string]string) {
	if len(rows) == 0 {
		// CSV vazio: ainda envia header correto (text/csv)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, baseFilename))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Estabiliza ordem das colunas via sort alfabético (range map é não-determinístico).
	headers := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, baseFilename))
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write(headers) // header row
	for _, row := range rows {
		vals := make([]string, len(headers))
		for i, h := range headers {
			vals[i] = row[h]
		}
		_ = cw.Write(vals)
	}
	cw.Flush()
}

// enviosToRows converte []envioDTO em []map[string]string para CSV.
func enviosToRows(envios []envioDTO) []map[string]string {
	rows := make([]map[string]string, 0, len(envios))
	for _, e := range envios {
		rows = append(rows, map[string]string{
			"id":            e.ID,
			"cadoc_code":    e.CadocCode,
			"period":        e.Period,
			"status":        e.Status,
			"rules_passed":  strconv.Itoa(e.RulesPassed),
			"rules_failed":  strconv.Itoa(e.RulesFailed),
			"duration_ms":   strconv.Itoa(e.DurationMs),
			"protocol_sta":  e.ProtocolSTA,
			"error_code":    e.ErrorCode,
			"error_message": e.ErrorMessage,
			"sent_at":       e.SentAt,
			"confirmed_at":  e.ConfirmedAt,
		})
	}
	return rows
}

// auditEventsToRows converte []auditEventDTO em []map[string]string.
func auditEventsToRows(events []auditEventDTO) []map[string]string {
	rows := make([]map[string]string, 0, len(events))
	for _, e := range events {
		payloadJSON, _ := json.Marshal(e.Payload)
		rows = append(rows, map[string]string{
			"id":          strconv.FormatInt(e.ID, 10),
			"if_id":       e.IFID,
			"actor":       e.Actor,
			"action":      e.Action,
			"target":      e.Target,
			"description": e.Description,
			"payload":     string(payloadJSON),
			"created_at":  e.CreatedAt,
		})
	}
	return rows
}

// radarAlertDTO é a versão flat (strings) de radar.Alert pra CSV.
// radar.Alert tem DetectedAt time.Time — não serializa bem em CSV.
// Aqui convertemos pra string RFC3339 antes de virar map[string]string.
type radarAlertDTO struct {
	ID          int
	CadocCode   string
	Severity    string
	Title       string
	Description string
	SourceURL   string
	DetectedAt  string // RFC3339
	Resolved    bool
}

// alertasToRows converte []radarAlert em []map[string]string (Sprint 8d bônus).
func alertasToRows(alerts []radarAlertDTO) []map[string]string {
	rows := make([]map[string]string, 0, len(alerts))
	for _, a := range alerts {
		rows = append(rows, map[string]string{
			"id":          strconv.Itoa(a.ID),
			"cadoc_code":  a.CadocCode,
			"severity":    a.Severity,
			"title":       a.Title,
			"description": a.Description,
			"source_url":  a.SourceURL,
			"detected_at": a.DetectedAt,
			"resolved":    strconv.FormatBool(a.Resolved),
		})
	}
	return rows
}
