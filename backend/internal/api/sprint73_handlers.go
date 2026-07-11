// Sprint 73 — API endpoints: crossdoc rules, schema listing, generate history.
//
// Endpoints:
//   - GET  /v1/crossdoc/rules     — lista todas regras cross-doc (XD-*) com metadata
//   - GET  /v1/schema             — lista CADOCs disponíveis com versão e complexity
//   - GET  /v1/generate/history    — histórico de gerações do IF autenticado
package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
)

// crossDocRuleInfo descreve uma regra cross-doc para o frontend.
type crossDocRuleInfo struct {
	Code         string   `json:"code"`
	Description  string   `json:"description"`
	Severity     string   `json:"severity"` // E, A, I
	RequiredDocs []string `json:"required_docs"`
}

// crossDocRulesResponse é o response de GET /v1/crossdoc/rules.
type crossDocRulesResponse struct {
	Rules []crossDocRuleInfo `json:"rules"`
	Total int                 `json:"total"`
}

// listCrossDocRules handles GET /v1/crossdoc/rules.
// Lista todas as regras cross-doc (XD-*) registradas no engine.
func (s *Server) listCrossDocRules(w http.ResponseWriter, r *http.Request) {
	if s.CrossDoc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "CROSSDOC_DISABLED",
			"message": "cross-doc engine não disponível",
		})
		return
	}

	rules := s.CrossDoc.Rules()
	out := make([]crossDocRuleInfo, 0, len(rules))
	for _, rule := range rules {
		out = append(out, crossDocRuleInfo{
			Code:         rule.Code(),
			Description:  rule.Description(),
			Severity:     rule.Severity(),
			RequiredDocs: rule.RequiredDocs(),
		})
	}

	writeJSON(w, http.StatusOK, crossDocRulesResponse{
		Rules: out,
		Total: len(out),
	})
}

// schemaInfo descreve um CADOC disponível para geração.
type schemaInfo struct {
	CadocCode         string                    `json:"cadoc_code"`
	LatestVersion     string                    `json:"latest_version,omitempty"`
	EffectiveFrom     string                    `json:"effective_from,omitempty"`
	SourceURI         string                    `json:"source_uri,omitempty"`
	SupportedVersions []string                  `json:"supported_versions"`
	FieldCount        int                       `json:"field_count"`
	Complexity        generator.ComplexityScore `json:"complexity"`
}

// schemaListResponse é o response de GET /v1/schema.
type schemaListResponse struct {
	Schemas []schemaInfo `json:"schemas"`
	Total   int          `json:"total"`
}

// listSchemasV2 handles GET /v1/schema.
// Lista CADOCs disponíveis com metadata de geração.
// Semelhante a GET /v1/schemas mas inclui info de geração (complexidade,
// versões suportadas) útil para o wizard de geração.
func (s *Server) listSchemasV2(w http.ResponseWriter, r *http.Request) {
	cadocs, err := s.cadocsWithCache(r.Context())
	if err != nil {
		s.internalServerError(w, err, "listSchemasV2")
		return
	}

	out := make([]schemaInfo, 0, len(cadocs))
	for _, cadoc := range cadocs {
		g := genRegistry.Get(cadoc)
		if g == nil {
			continue // sem generator → não listar
		}

		var effFrom string
		var sourceURI string
		if s.Schema != nil {
			v, err := s.Schema.GetEffective(cadoc, time.Now())
			if err == nil && v != nil {
				effFrom = v.EffectiveFrom.Format("2006-01-02")
				sourceURI = v.SourceURI
			}
		}

		// Estimate complexity com doc zero (campos mínimos).
		complexity := g.EstimateComplexity(nil)

		out = append(out, schemaInfo{
			CadocCode:         cadoc,
			LatestVersion:     latestVersion(g.SupportedVersions()),
			EffectiveFrom:     effFrom,
			SourceURI:         sourceURI,
			SupportedVersions: g.SupportedVersions(),
			FieldCount:        len(g.RequiredFields()),
			Complexity:        complexity,
		})
	}

	writeJSON(w, http.StatusOK, schemaListResponse{
		Schemas: out,
		Total:   len(out),
	})
}

// latestVersion retorna a versão mais recente de uma lista de versões.
// Assume formato "X.Y" e retorna o último alfabeticamente (X.Y crescente).
func latestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if v > latest {
			latest = v
		}
	}
	return latest
}

// generationHistoryItem é um item no histórico de gerações.
type generationHistoryItem struct {
	ID          string    `json:"id"`
	CadocCode   string    `json:"cadoc_code"`
	DataBase    string    `json:"data_base"`
	GeneratedAt time.Time `json:"generated_at"`
	SHA256      string    `json:"sha256,omitempty"`
	Status      string    `json:"status"` // generated, crossdoc_passed, crossdoc_failed
	Passed      bool      `json:"passed"`
}

// generationHistoryResponse é o response de GET /v1/generate/history.
type generationHistoryResponse struct {
	Items  []generationHistoryItem `json:"items"`
	Page   int                     `json:"page"`
	PerPage int                    `json:"per_page"`
	Total  int                     `json:"total"`
}

// listGenerateHistory handles GET /v1/generate/history.
// Retorna histórico de gerações do IF autenticado.
//
// Query params:
//   - page (default 1)
//   - per_page (default 20, max 100)
//   - cadoc (opcional, filtro por tipo)
func (s *Server) listGenerateHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ifID := ifIDFromRequest(r) // extrai do JWT ou X-IF-ID fallback

	page := intParam(r.URL.Query().Get("page"), 1)
	perPage := intParam(r.URL.Query().Get("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage
	cadocFilter := r.URL.Query().Get("cadoc")

	// Se não tem DB, retorna 501.
	if s.DB == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error":   "DB_NOT_CONFIGURED",
			"message": "histórico não disponível (banco de dados não configurado)",
		})
		return
	}

	// Query envios como source de histórico.
	// Filtra por if_id, ordenando por created_at DESC.
	query := `
		SELECT id, cadoc_code, data_base, created_at, xml_hash, status
		FROM envios
		WHERE if_id = ?
	`
	args := []any{ifID}
	if cadocFilter != "" {
		query += " AND cadoc_code = ?"
		args = append(args, cadocFilter)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var items []generationHistoryItem
	var total int

	// Count total.
	countQuery := `SELECT COUNT(*) FROM envios WHERE if_id = ?`
	countArgs := []any{ifID}
	if cadocFilter != "" {
		countQuery += " AND cadoc_code = ?"
		countArgs = append(countArgs, cadocFilter)
	}
	if err := s.DB.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.internalServerError(w, err, "listGenerateHistory count")
		return
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		s.internalServerError(w, err, "listGenerateHistory select")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item generationHistoryItem
		var xmlHash sql.NullString
		if err := rows.Scan(&item.ID, &item.CadocCode, &item.DataBase, &item.GeneratedAt, &xmlHash, &item.Status); err != nil {
			continue
		}
		if xmlHash.Valid {
			item.SHA256 = xmlHash.String
		}
		item.Passed = item.Status == "validated" || item.Status == "sent" || item.Status == "accepted"
		items = append(items, item)
	}

	if items == nil {
		items = []generationHistoryItem{}
	}

	writeJSON(w, http.StatusOK, generationHistoryResponse{
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

// intParam parseia param de query como int, retorna default se inválido.
func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}

// ifIDFromRequest extrai IF-ID do request (JWT claim ou X-IF-ID header).
func ifIDFromRequest(r *http.Request) string {
	if claims, err := auth.ClaimsFromContext(r.Context()); err == nil && claims != nil && claims.IFID != "" {
		return claims.IFID
	}
	if ifID := r.Header.Get("X-IF-ID"); ifID != "" {
		return ifID
	}
	return ""
}
