// Sprint 54 v3.34.37: SchemaRegistry_v2 — public changelog endpoint.
//
// GET /v1/schemas/{cadoc}/changelog — timeline de versões com changelog.
//
// Resposta: lista de versões com effective_from + changelog (sem fields).
// Usuários conseguem ver o histórico de mudanças sem baixar o layout completo.
//
// Auth: JWT (mesmo dos outros endpoints).
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
