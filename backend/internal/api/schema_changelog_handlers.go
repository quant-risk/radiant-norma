// Sprint 54 v3.34.37: SchemaRegistry_v2 — public changelog endpoint.
//
// GET /v1/schemas/{cadoc}/changelog — timeline de versões com changelog.
// Query ?format=structured retorna diff machine-parseable (Sprint 54 enhancement).
//
// Resposta: lista de versões com effective_from + changelog (sem fields).
// Usuários conseguem ver o histórico de mudanças sem baixar o layout completo.
//
// Auth: JWT (mesmo dos outros endpoints).
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
)

// schemaChangelogEntry é uma entrada no timeline de changelog.
type schemaChangelogEntry struct {
	VersionID     int64  `json:"id"`
	EffectiveFrom string `json:"effective_from"`
	SourceURI     string `json:"source_uri,omitempty"`
	Changelog     string `json:"changelog,omitempty"`
}

// schemaChangelogTimeline é a resposta de GET /schemas/{cadoc}/changelog.
type schemaChangelogTimeline struct {
	Cadoc   string                 `json:"cadoc"`
	Entries []schemaChangelogEntry `json:"entries"`
	Total   int                    `json:"total"`
}

// listSchemaChangelog handles GET /v1/schemas/{cadoc}/changelog.
// Query ?format=structured retorna StructuredChangelog (machine-parseable).
func (s *Server) listSchemaChangelog(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	if err := ValidateCadocCode(cadoc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	versions, err := s.Schema.List(cadoc)
	if err != nil {
		s.internalServerError(w, err, "listSchemaChangelog")
		return
	}

	if r.URL.Query().Get("format") == "structured" {
		s.writeStructuredChangelog(w, cadoc, versions)
		return
	}

	entries := make([]schemaChangelogEntry, 0, len(versions))
	for _, v := range versions {
		entries = append(entries, schemaChangelogEntry{
			VersionID:     v.ID,
			EffectiveFrom: v.EffectiveFrom.Format("2006-01-02"),
			SourceURI:     v.SourceURI,
			Changelog:     v.Changelog,
		})
	}

	writeJSON(w, http.StatusOK, schemaChangelogTimeline{
		Cadoc:   cadoc,
		Entries: entries,
		Total:   len(entries),
	})
}

// writeStructuredChangelog writes a structured diff between consecutive versions.
func (s *Server) writeStructuredChangelog(w http.ResponseWriter, cadoc string, versions []schema.Version) {
	if len(versions) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"cadoc": cadoc, "versions": []any{}})
		return
	}

	type versionDiff struct {
		schema.Version
		Diff *schema.StructuredChangelog `json:"diff,omitempty"`
	}

	out := make([]versionDiff, 0, len(versions))
	for i, v := range versions {
		var prevFields []schema.Field
		var prevEff string
		if i+1 < len(versions) {
			prevEff = versions[i+1].EffectiveFrom.Format("2006-01-02")
			prevFields = versions[i+1].Fields
		}
		vd := versionDiff{Version: v}
		vd.Diff = schema.ComputeStructuredChangelog(
			cadoc,
			prevFields,
			v.Fields,
			prevEff,
			v.EffectiveFrom.Format("2006-01-02"),
			v.EffectiveFrom.Format("2006-01-02"),
		)
		out = append(out, vd)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"cadoc":    cadoc,
		"versions": out,
	})
}

// publishSchema handles POST /v1/admin/schemas (internal, admin-only).
// Insere nova versão de schema e invalida caches.
func (s *Server) publishSchema(w http.ResponseWriter, r *http.Request) {
	if !s.AdminAuth.IsAdmin(r) {
		http.Error(w, `{"error":"admin required"}`, http.StatusUnauthorized)
		return
	}
	var req schemaInsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	effFrom, err := time.Parse("2006-01-02", req.EffectiveFrom)
	if err != nil {
		http.Error(w, "effective_from must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	v := &schema.Version{
		CadocCode:     req.CadocCode,
		EffectiveFrom: effFrom,
		SourceURI:     req.SourceURI,
		XSD:           req.XSD,
		Fields:        req.Fields,
	}
	if err := s.Schema.Insert(v); err != nil {
		s.internalServerError(w, err, "publishSchema")
		return
	}
	// Sprint 54: invalidate caches so next listSchemas call picks up the new version.
	if s.SchemaInfoCache != nil {
		s.SchemaInfoCache.Invalidate()
	}
	if s.CadocListCache != nil {
		s.CadocListCache.Invalidate()
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": v.ID, "cadoc": v.CadocCode})
}

// getSchemaVersion handles GET /v1/schemas/{cadoc}/versions/{versionId}.
// Retorna versão específica (para diff visual no frontend).
func (s *Server) getSchemaVersion(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	versionID, err := strconv.ParseInt(chi.URLParam(r, "versionId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid versionId", http.StatusBadRequest)
		return
	}
	v, err := s.Schema.GetByID(versionID)
	if err != nil {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}
	if v.CadocCode != cadoc {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

type schemaInsertRequest struct {
	CadocCode     string          `json:"cadoc_code"`
	EffectiveFrom string          `json:"effective_from"`
	SourceURI     string          `json:"source_uri,omitempty"`
	XSD           string          `json:"xsd,omitempty"`
	Fields        []schema.Field  `json:"fields"`
}
