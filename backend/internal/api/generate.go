// Package api — handlers de geração de CADOCs.
//
// Endpoints:
//   - POST /v1/generate/{cadoc}  — gera XML a partir de CanonicalDocument
//   - POST /v1/generate/{cadoc}/sources — configura fonte de dados
//   - GET  /v1/generate/{cadoc}/fields — campos necessários para este CADOC
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	gen3040 "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	"github.com/fortvna/radiant-norma/backend/internal/ingest"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/go-chi/chi/v5"
)

// GeneratorRegistry é o registry global de generators.
var genRegistry = generator.NewRegistry()

func init() {
	genRegistry.Register(gen3040.New())
}

// generateCadoc handles POST /v1/generate/{cadoc}.
func (s *Server) generateCadoc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cadoc := chi.URLParam(r, "cadoc")

	if err := ValidateCadocCode(cadoc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CADOC", err.Error())
		return
	}

	g := genRegistry.Get(cadoc)
	if g == nil {
		writeError(w, http.StatusNotFound, "GENERATOR_NOT_FOUND",
			fmt.Sprintf("generator para CADOC %s não encontrado", cadoc))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BODY_READ_ERROR", "falha ao ler body")
		return
	}

	var req GenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON_PARSE_ERROR", fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	dataBase := req.DataBase
	if dataBase.IsZero() {
		dataBase = time.Now()
		if dataBase.Day() > 25 {
			dataBase = dataBase.AddDate(0, 1, -dataBase.Day()+1)
		} else {
			dataBase = time.Date(dataBase.Year(), dataBase.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
	}

	doc := canonical.NewCanonical(req.IFID, dataBase, canonical.CadocType(cadoc))
	doc.VersaoLayout = req.VersaoLayout
	if doc.VersaoLayout == "" {
		doc.VersaoLayout = "3.2"
	}
	doc.Header.CNPJ = req.CNPJ
	doc.Header.NomeIF = req.NomeIF
	doc.Header.DataHoraGeracao = time.Now()
	doc.Extra = req.Extra
	doc.Participantes = req.Participantes
	doc.Operacoes = req.Operacoes
	doc.Metadata.SourceAdapter = req.Source

	if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims != nil {
		doc.Metadata.GeneratedBy = claims.IFID
	}

	generated, err := g.Generate(ctx, doc, dataBase)
	if err != nil {
		slog.Error("generate", "cadoc", cadoc, "err", err)
		writeError(w, http.StatusInternalServerError, "GENERATE_ERROR", fmt.Sprintf("falha ao gerar %s: %v", cadoc, err))
		return
	}

	if len(generated.Errors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, GenerateResponse{
			CadocCode: cadoc,
			DataBase:  dataBase,
			Generated: generated,
			Status:    "error",
			Message:   "documento não pode ser gerado — campos obrigatórios faltantes",
		})
		return
	}

	writeJSON(w, http.StatusOK, GenerateResponse{
		CadocCode: cadoc,
		DataBase:  dataBase,
		Generated: generated,
		Status:    "ok",
		Message:   fmt.Sprintf("%s gerado com sucesso", cadoc),
	})
}

// listGenerateFields handles GET /v1/generate/{cadoc}/fields.
func (s *Server) listGenerateFields(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")

	if err := ValidateCadocCode(cadoc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CADOC", err.Error())
		return
	}

	g := genRegistry.Get(cadoc)
	if g == nil {
		writeError(w, http.StatusNotFound, "GENERATOR_NOT_FOUND",
			fmt.Sprintf("generator para CADOC %s não encontrado", cadoc))
		return
	}

	fields := g.RequiredFields()
	complexity := g.EstimateComplexity(&canonical.CanonicalDocument{CadocCode: canonical.CadocType(cadoc)})

	writeJSON(w, http.StatusOK, FieldsResponse{
		CadocCode:  cadoc,
		Fields:     fields,
		Versions:   g.SupportedVersions(),
		Complexity: complexity,
	})
}

// ingestSources handles POST /v1/generate/{cadoc}/sources.
func (s *Server) ingestSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cadoc := chi.URLParam(r, "cadoc")

	if err := ValidateCadocCode(cadoc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CADOC", err.Error())
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BODY_READ_ERROR", "falha ao ler body")
		return
	}

	var cfg ingest.SourceConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "JSON_PARSE_ERROR", fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	adapter := ingest.GetAdapter(cfg.Type)
	if adapter == nil {
		writeError(w, http.StatusBadRequest, "UNKNOWN_ADAPTER",
			fmt.Sprintf("adapter de tipo %s não encontrado", cfg.Type))
		return
	}

	if err := adapter.ValidateConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", err.Error())
		return
	}

	if err := adapter.HealthCheck(ctx, cfg); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, SourceConfigResponse{
			CadocCode:  cadoc,
			SourceType: cfg.Type,
			SourceName: cfg.Name,
			Fields:     nil,
			Status:     "error",
			Message:    fmt.Sprintf("healthcheck falhou: %v", err),
		})
		return
	}

	fields := adapter.DescribeFields(cadoc)

	writeJSON(w, http.StatusOK, SourceConfigResponse{
		CadocCode:  cadoc,
		SourceType: cfg.Type,
		SourceName: cfg.Name,
		Fields:     fields,
		Status:     "ok",
		Message:    "fonte configurada e acessível",
	})
}

// listSourceAdapters handles GET /v1/generate/adapters.
func (s *Server) listSourceAdapters(w http.ResponseWriter, r *http.Request) {
	adapters := ingest.ListAdapters()
	var out []AdapterInfo
	for _, a := range adapters {
		out = append(out, AdapterInfo{Type: a.Type(), Name: a.Name()})
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Request/Response types ---

// GenerateRequest é o body do POST /v1/generate/{cadoc}.
type GenerateRequest struct {
	IFID          string                   `json:"if_id"`
	CNPJ          string                   `json:"cnpj"`
	NomeIF        string                   `json:"nome_if"`
	VersaoLayout  string                   `json:"versao_layout"`
	DataBase      time.Time                `json:"data_base"`
	Extra         map[string]any           `json:"extra,omitempty"`
	Participantes []canonical.Participante `json:"participantes,omitempty"`
	Operacoes     []canonical.Operacao     `json:"operacoes,omitempty"`
	Source        string                   `json:"source,omitempty"`
}

// GenerateResponse é o response do POST /v1/generate/{cadoc}.
type GenerateResponse struct {
	CadocCode string                  `json:"cadoc_code"`
	DataBase  time.Time               `json:"data_base"`
	Generated *generator.GeneratedDoc `json:"generated"`
	Status    string                  `json:"status"`
	Message   string                  `json:"message"`
}

// FieldsResponse é o response do GET /v1/generate/{cadoc}/fields.
type FieldsResponse struct {
	CadocCode  string                    `json:"cadoc_code"`
	Fields     []schema.Field            `json:"fields"`
	Versions   []string                  `json:"versions"`
	Complexity generator.ComplexityScore `json:"complexity"`
}

// SourceConfigResponse é o response do POST /v1/generate/{cadoc}/sources.
type SourceConfigResponse struct {
	CadocCode  string                   `json:"cadoc_code"`
	SourceType ingest.SourceType        `json:"source_type"`
	SourceName string                   `json:"source_name"`
	Fields     []ingest.FieldDescriptor `json:"fields"`
	Status     string                   `json:"status"`
	Message    string                   `json:"message"`
}

// AdapterInfo descreve um adapter disponível.
type AdapterInfo struct {
	Type ingest.SourceType `json:"type"`
	Name string            `json:"name"`
}

// writeError escreve um JSON de erro com código e mensagem.
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "message": msg})
}
