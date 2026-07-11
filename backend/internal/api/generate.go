// Package api — handlers de geração de CADOCs.
//
// Endpoints:
//   - POST /v1/generate/{cadoc}  — gera XML a partir de CanonicalDocument
//   - POST /v1/generate/{cadoc}/sources — configura fonte de dados
//   - GET  /v1/generate/{cadoc}/fields — campos necessários para este CADOC
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	gen2030 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2030"
	gen2060 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2060"
	gen2061 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2061"
	gen2062 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2062"
	gen2070 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2070"
	gen2160 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2160"
	gen2170 "github.com/fortvna/radiant-norma/backend/internal/generator/gen2170"
	gen3040 "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	gen3050 "github.com/fortvna/radiant-norma/backend/internal/generator/gen3050"
	gen4111 "github.com/fortvna/radiant-norma/backend/internal/generator/gen4111"
	"github.com/fortvna/radiant-norma/backend/internal/ingest"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/go-chi/chi/v5"
)

// GeneratorRegistry é o registry global de generators.
var genRegistry = generator.NewRegistry()

func init() {
	genRegistry.Register(gen2030.New())
	genRegistry.Register(gen2060.New())
	genRegistry.Register(gen2061.New())
	genRegistry.Register(gen2062.New())
	genRegistry.Register(gen2070.New())
	genRegistry.Register(gen2160.New())
	genRegistry.Register(gen2170.New())
	genRegistry.Register(gen3040.New())
	genRegistry.Register(gen3050.New())
	genRegistry.Register(gen4111.New())
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
	if req.VersaoLayout != "" {
		doc.VersaoLayout = req.VersaoLayout
	} else {
		// Use generator's default version instead of hardcoded "3.2".
		doc.VersaoLayout = g.SupportedVersions()[0]
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
		// Validation/creation failures → 422 com detalhes.
		// Systemic failures (DB, etc.) → 500.
		writeJSON(w, http.StatusUnprocessableEntity, GenerateResponse{
			CadocCode: cadoc,
			DataBase:  dataBase,
			Generated: &generator.GeneratedDoc{
				Errors: []generator.GenError{{
					Code:    "GENERATION_FAILED",
					Message: err.Error(),
				}},
			},
			Status:  "error",
			Message: fmt.Sprintf("falha ao gerar %s: %v", cadoc, err),
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

// generateBatch handles POST /v1/generate/batch.
func (s *Server) generateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BODY_READ_ERROR", "falha ao ler body")
		return
	}

	var req BatchGenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON_PARSE_ERROR", fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	if len(req.Cadocs) == 0 {
		writeError(w, http.StatusBadRequest, "EMPTY_CADOCS", "nenhum CADOC fornecido")
		return
	}

	results := make([]BatchResult, 0, len(req.Cadocs))
	successfulXMLs := make(map[string]string)

	for _, cadocReq := range req.Cadocs {
		cadocCode := cadocReq.CadocCode

		if err := ValidateCadocCode(cadocCode); err != nil {
			results = append(results, BatchResult{
				CadocCode: cadocCode,
				Errors: []generator.GenError{{
					Code:    "INVALID_CADOC",
					Message: err.Error(),
				}},
				Status: "error",
			})
			continue
		}

		g := genRegistry.Get(cadocCode)
		if g == nil {
			results = append(results, BatchResult{
				CadocCode: cadocCode,
				Errors: []generator.GenError{{
					Code:    "GENERATOR_NOT_FOUND",
					Message: fmt.Sprintf("generator para CADOC %s não encontrado", cadocCode),
				}},
				Status: "error",
			})
			continue
		}

		dataBase := cadocReq.DataBase
		if dataBase.IsZero() {
			dataBase = time.Now()
			if dataBase.Day() > 25 {
				dataBase = dataBase.AddDate(0, 1, -dataBase.Day()+1)
			} else {
				dataBase = time.Date(dataBase.Year(), dataBase.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
		}

		doc := canonical.NewCanonical(cadocReq.IFID, dataBase, canonical.CadocType(cadocCode))
		if cadocReq.VersaoLayout != "" {
			doc.VersaoLayout = cadocReq.VersaoLayout
		} else {
			doc.VersaoLayout = g.SupportedVersions()[0]
		}
		doc.Header.CNPJ = cadocReq.CNPJ
		doc.Header.NomeIF = cadocReq.NomeIF
		doc.Header.DataHoraGeracao = time.Now()
		doc.Extra = cadocReq.Extra
		doc.Participantes = cadocReq.Participantes
		doc.Operacoes = cadocReq.Operacoes
		doc.Metadata.SourceAdapter = cadocReq.Source

		if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims != nil {
			doc.Metadata.GeneratedBy = claims.IFID
		}

		generated, err := g.Generate(ctx, doc, dataBase)
		if err != nil {
			slog.Error("batch generate", "cadoc", cadocCode, "err", err)
			results = append(results, BatchResult{
				CadocCode: cadocCode,
				Generated: &generator.GeneratedDoc{
					Errors: []generator.GenError{{
						Code:    "GENERATION_FAILED",
						Message: err.Error(),
					}},
				},
				Status: "error",
			})
			continue
		}

		results = append(results, BatchResult{
			CadocCode: cadocCode,
			Generated: generated,
			Status:    "ok",
		})

		if generated != nil && len(generated.XML) > 0 {
			successfulXMLs[cadocCode] = string(generated.XML)
		}
	}

	response := BatchGenerateResponse{
		Results: results,
		Passed:  true,
		Message: "batch generation completed",
	}

	// Run cross-doc validation if requested and 2+ CADOCs succeeded
	if req.RunCrossDoc && len(successfulXMLs) >= 2 && s.CrossDoc != nil {
		crossResp := s.CrossDoc.Validate(ctx, &crossdoc.ValidationRequest{
			Cadocs: successfulXMLs,
		})

		for _, err := range crossResp.Errors {
			response.CrossDocErrors = append(response.CrossDocErrors, CrossDocError{
				Code:     err.Code,
				Severity: "error",
				Message:  err.Message,
			})
		}
		for _, warn := range crossResp.Warnings {
			response.CrossDocWarnings = append(response.CrossDocWarnings, CrossDocError{
				Code:     warn.Code,
				Severity: "warning",
				Message:  warn.Message,
			})
		}
		response.Passed = crossResp.Passed
		if !crossResp.Passed {
			response.Message = "batch generation completed with cross-doc errors"
		}
	} else if req.RunCrossDoc && len(successfulXMLs) < 2 {
		response.Message = "cross-doc validation skipped: less than 2 CADOCs generated successfully"
		response.Passed = true // skipped ≠ failed
	}

	// HTTP semantics: 200 = request processed, 422 = semantically invalid.
	// Only gate on 422 when cross-doc was actually requested and failed.
	// Skipped (no cross-doc flag, or <2 docs) does NOT trigger 422.
	if req.RunCrossDoc && len(successfulXMLs) >= 2 && !response.Passed {
		writeJSON(w, http.StatusUnprocessableEntity, response)
		return
	}
	writeJSON(w, http.StatusOK, response)
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
		if errors.Is(err, ingest.ErrNotImplemented) {
			writeError(w, http.StatusNotImplemented,
				"ADAPTER_NOT_IMPLEMENTED",
				fmt.Sprintf("adapter %s ainda não implementado", cfg.Type))
			return
		}
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

// parseUploadedFile handles POST /v1/generate/file/parse.
// Accepts multipart form data with a CSV/XLSX file and returns a parsed
// CanonicalDocument. Used by the Wizard UI for the file → canonical step.
func (s *Server) parseUploadedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50 MB max
		writeError(w, http.StatusBadRequest, "MULTIPART_PARSE_ERROR",
			fmt.Sprintf("falha ao processar multipart: %v", err))
		return
	}

	// Get cadoc type from form field.
	cadoc := r.FormValue("cadoc")
	if cadoc == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CADOC", "campo cadoc é obrigatório")
		return
	}
	if err := ValidateCadocCode(cadoc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CADOC", err.Error())
		return
	}
	if genRegistry.Get(cadoc) == nil {
		writeError(w, http.StatusBadRequest, "UNKNOWN_CADOC",
			fmt.Sprintf("nenhum generator registrado para CADOC %s", cadoc))
		return
	}

	// Get file from multipart.
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_FILE", "campo file é obrigatório")
		return
	}
	defer file.Close()

	// Determine format from filename extension.
	format := r.FormValue("format")
	if format == "" {
		switch {
		case strings.HasSuffix(header.Filename, ".xlsx"), strings.HasSuffix(header.Filename, ".xls"):
			format = "xlsx"
		default:
			format = "csv"
		}
	}
	if format != "csv" && format != "xlsx" {
		writeError(w, http.StatusBadRequest, "INVALID_FORMAT",
			fmt.Sprintf("formato %q não suportado (suporta: csv, xlsx)", format))
		return
	}

	// Parse data_base.
	dataBaseStr := r.FormValue("data_base")
	var dataBase time.Time
	if dataBaseStr != "" {
		dataBase, err = time.Parse("2006-01-02", dataBaseStr)
		if err != nil {
			dataBase, err = time.Parse("2006-01", dataBaseStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "INVALID_DATA_BASE",
					fmt.Sprintf("data_base inválida: %v", err))
				return
			}
		}
	} else {
		dataBase = time.Now()
	}

	// Copy uploaded file to a temp location for the FileAdapter.
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "radiant-parse-*.tmp")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TEMP_FILE_ERROR",
			fmt.Sprintf("falha ao criar arquivo temporário: %v", err))
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		writeError(w, http.StatusInternalServerError, "TEMP_FILE_WRITE_ERROR",
			fmt.Sprintf("falha ao escrever arquivo temporário: %v", err))
		return
	}

	hasHeader := r.FormValue("has_header") != "false"

	cfg := ingest.SourceConfig{
		Name: cadoc,
		File: &ingest.FileConfig{
			Path:      tmpPath,
			Format:    format,
			HasHeader: hasHeader,
		},
	}

	adapter := ingest.GetAdapter(ingest.SourceFile)
	if adapter == nil {
		writeError(w, http.StatusInternalServerError, "ADAPTER_NOT_FOUND",
			"FileAdapter não registrado")
		return
	}

	parsed, err := adapter.Fetch(ctx, cfg, cadoc, dataBase)
	if err != nil {
		if errors.Is(err, ingest.ErrNotImplemented) {
			writeError(w, http.StatusNotImplemented, "ADAPTER_NOT_IMPLEMENTED",
				"FileAdapter ainda não implementado")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "PARSE_ERROR",
			fmt.Sprintf("falha ao parsear arquivo: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, FileParseResponse{
		CadocCode: cadoc,
		DataBase:  dataBase,
		Document:  parsed,
		Status:    "ok",
		Message:   fmt.Sprintf("arquivo %s parseado com sucesso", header.Filename),
	})
}

// --- Request/Response types ---

// GenerateRequest é o body do POST /v1/generate/{cadoc}.
type GenerateRequest struct {
	CadocCode     string                   `json:"cadoc_code"`
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

// FileParseResponse é o response do POST /v1/generate/file/parse.
type FileParseResponse struct {
	CadocCode string                       `json:"cadoc_code"`
	DataBase  time.Time                    `json:"data_base"`
	Document  *canonical.CanonicalDocument `json:"document"`
	Status    string                       `json:"status"`
	Message   string                       `json:"message"`
}

// writeError escreve um JSON de erro com código e mensagem.
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "message": msg})
}

// --- Batch generation types ---

// BatchGenerateRequest é o body do POST /v1/generate/batch.
type BatchGenerateRequest struct {
	Cadocs     []GenerateRequest `json:"cadocs"`
	RunCrossDoc bool             `json:"run_crossdoc,omitempty"`
}

// BatchGenerateResponse é o response do POST /v1/generate/batch.
type BatchGenerateResponse struct {
	Results          []BatchResult    `json:"results"`
	CrossDocErrors   []CrossDocError  `json:"crossdoc_errors,omitempty"`
	CrossDocWarnings []CrossDocError  `json:"crossdoc_warnings,omitempty"`
	Passed           bool             `json:"passed"`
	Message          string          `json:"message"`
}

// BatchResult representa o resultado da geração de um único CADOC no batch.
type BatchResult struct {
	CadocCode string                  `json:"cadoc_code"`
	Generated *generator.GeneratedDoc `json:"generated,omitempty"`
	Errors    []generator.GenError    `json:"errors,omitempty"`
	Warnings  []generator.GenWarning  `json:"warnings,omitempty"`
	Status    string                  `json:"status"`
}

// CrossDocError representa um erro ou warning do cross-doc validation.
type CrossDocError struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
