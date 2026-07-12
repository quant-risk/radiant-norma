// Sprint 73 — API endpoints: crossdoc rules, schema listing, generate history.
//
// Endpoints:
//   - GET  /v1/crossdoc/rules     — lista todas regras cross-doc (XD-*) com metadata
//   - GET  /v1/schema             — lista CADOCs disponíveis com versão e complexity
//   - GET  /v1/generate/history   — histórico de gerações do IF autenticado
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"golang.org/x/sync/singleflight"
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
	Total int                `json:"total"`
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

// schemaInfoCache é um cache in-memory (5min TTL) para listSchemasV2.
//
// Sprint 75: Evita N queries (1 por CADOC) ao Schema.Registry.
// O cache expira a cada 5min — dados de schema_versions mudam raramente.
type SchemaInfoCache struct {
	mu       sync.Mutex
	resp     *schemaListResponse
	cachedAt time.Time
	ttl      time.Duration
	sf       singleflight.Group
}

// NewSchemaInfoCache cria um cache com TTL de 5 minutos.
func NewSchemaInfoCache() *SchemaInfoCache {
	return &SchemaInfoCache{ttl: 5 * time.Minute}
}

// Invalidate limpa o cache (usado quando um novo schema é inserido).
func (c *SchemaInfoCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resp = nil
	c.cachedAt = time.Time{}
}

// GetOrFetch retorna cache se válido, senão chama fetch() e cacheia.
// Sempre retorna deep copy do slice Schemas para evitar aliasing com o cache.
func (c *SchemaInfoCache) GetOrFetch(fetch func() (*schemaListResponse, error)) (*schemaListResponse, error) {
	c.mu.Lock()
	if c.resp != nil && time.Since(c.cachedAt) < c.ttl {
		out := *c.resp
		schemas := make([]schemaInfo, len(c.resp.Schemas))
		copy(schemas, c.resp.Schemas)
		out.Schemas = schemas
		c.mu.Unlock()
		return &out, nil
	}
	c.mu.Unlock()

	v, err, _ := c.sf.Do("schemaInfo", func() (any, error) {
		c.mu.Lock()
		// Re-check após acquire
		if c.resp != nil && time.Since(c.cachedAt) < c.ttl {
			out := *c.resp
			schemas := make([]schemaInfo, len(c.resp.Schemas))
			copy(schemas, c.resp.Schemas)
			out.Schemas = schemas
			c.mu.Unlock()
			return &out, nil
		}
		c.mu.Unlock()

		result, err := fetch()
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.resp = result
		c.cachedAt = time.Now()
		c.mu.Unlock()
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	// Deep copy do resultado do singleflight antes de retornar.
	schemas := make([]schemaInfo, len(v.(*schemaListResponse).Schemas))
	copy(schemas, v.(*schemaListResponse).Schemas)
	out := &schemaListResponse{Schemas: schemas, Total: v.(*schemaListResponse).Total}
	return out, nil
}

// listSchemasV2 handles GET /v1/schema.
// Lista CADOCs disponíveis com metadata de geração.
// Semelhante a GET /v1/schemas mas inclui info de geração (complexidade,
// versões suportadas) útil para o wizard de geração.
//
// Sprint 75: Usa SchemaInfoCache (5min TTL) para evitar N queries ao DB
// (1 GetEffective por CADOC). Cache é por Server instance.
func (s *Server) listSchemasV2(w http.ResponseWriter, r *http.Request) {
	if s.SchemaInfoCache == nil {
		s.listSchemasV2NoCache(w, r)
		return
	}

	resp, err := s.SchemaInfoCache.GetOrFetch(func() (*schemaListResponse, error) {
		return s.buildSchemaInfoList()
	})
	if err != nil {
		s.internalServerError(w, err, "listSchemasV2")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildSchemaInfoList constrói a lista de schemas (sem cache).
// Chamado por GetOrFetch no cache miss.
func (s *Server) buildSchemaInfoList() (*schemaListResponse, error) {
	cadocs, err := s.cadocsWithCacheWithoutLock()
	if err != nil {
		return nil, err
	}

	out := make([]schemaInfo, 0, len(cadocs))
	for _, cadoc := range cadocs {
		g := genRegistry.Get(cadoc)
		if g == nil {
			continue
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

		complexity := g.EstimateComplexity(canonical.NewCanonical("", time.Now(), ""))
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
	return &schemaListResponse{Schemas: out, Total: len(out)}, nil
}

// listSchemasV2NoCache usado quando não há cache (ex: tests).
func (s *Server) listSchemasV2NoCache(w http.ResponseWriter, r *http.Request) {
	resp, err := s.buildSchemaInfoList()
	if err != nil {
		s.internalServerError(w, err, "listSchemasV2NoCache")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// cadocsWithCacheWithoutLock versão sem request context — para uso dentro
// do GetOrFetch (singleflight). Usa CadocListCache se disponível.
func (s *Server) cadocsWithCacheWithoutLock() ([]string, error) {
	if s.Schema == nil {
		return []string{}, nil
	}
	if s.CadocListCache != nil {
		return s.CadocListCache.GetOrFetch(func() ([]string, error) {
			return s.Schema.ListCadocs(context.Background())
		})
	}
	return s.Schema.ListCadocs(context.Background())
}

// latestVersion retorna a versão mais recente de uma lista de versões.
// Usa comparação semântica (major.minor) para ordernar.
// Funciona corretamente com versões como "3.9" vs "3.10" onde string compare
// falharia (alfabeticamente "3.9" > "3.10" porque '9' > '1').
func latestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if compareVersion(v, latest) > 0 {
			latest = v
		}
	}
	return latest
}

// compareVersion retorna -1, 0, +1 comparando v1 e v2 semanticamente.
// Suporta formato "X.Y" ou "X.Y.Z". Usa major.minor para comparação primária.
func compareVersion(v1, v2 string) int {
	p1 := splitVersion(v1)
	p2 := splitVersion(v2)
	major1, minor1 := p1[0], p1[1]
	major2, minor2 := p2[0], p2[1]
	if major1 != major2 {
		if major1 > major2 {
			return 1
		}
		return -1
	}
	if minor1 != minor2 {
		if minor1 > minor2 {
			return 1
		}
		return -1
	}
	return 0
}

// splitVersion divide "X.Y" ou "X.Y.Z" em [major, minor, patch].
// Usa strings.Split + strconv.Atoi — óbvio e correto.
func splitVersion(v string) [3]int {
	var parts [3]int
	segs := strings.Split(v, ".")
	if len(segs) >= 1 && segs[0] != "" {
		parts[0], _ = strconv.Atoi(segs[0])
	}
	if len(segs) >= 2 && segs[1] != "" {
		parts[1], _ = strconv.Atoi(segs[1])
	}
	if len(segs) >= 3 && segs[2] != "" {
		parts[2], _ = strconv.Atoi(segs[2])
	}
	return parts
}

// generationHistoryItem é um item no histórico de gerações.
type generationHistoryItem struct {
	ID          string    `json:"id"`
	CadocCode   string    `json:"cadoc_code"`
	DataBase    string    `json:"data_base"`
	GeneratedAt time.Time `json:"generated_at"`
	SHA256      string    `json:"sha256,omitempty"`
	// Status: pending, validated, sent, accepted, rejected, error, processing, dead_letter.
	// Passed=true para: validated, sent, accepted.
	Status string `json:"status"`
	Passed bool   `json:"passed"`
}

// generationHistoryResponse é o response de GET /v1/generate/history.
type generationHistoryResponse struct {
	Items   []generationHistoryItem `json:"items"`
	Page    int                     `json:"page"`
	PerPage int                     `json:"per_page"`
	Total   int                     `json:"total"`
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
	ifID := getIfID(r) // extrai do JWT ou X-IF-ID fallback

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

	// Se não tem DB, retorna 503.
	if s.DB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
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
		var xmlHash string
		if err := rows.Scan(&item.ID, &item.CadocCode, &item.DataBase, &item.GeneratedAt, &xmlHash, &item.Status); err != nil {
			slog.Warn("listGenerateHistory: scan error, dropping row", "err", err)
			continue
		}
		item.SHA256 = xmlHash
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
