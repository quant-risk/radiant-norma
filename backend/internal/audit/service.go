// Package audit implementa o Norma Audit (validação de CADOCs contra regras).
//
// Camadas (cf. PRODUTO_TESE_ROADMAP § 5):
//
//	L1 — Structural (XSD)
//	L2 — Semantic (críticas do BACEN)
//	L3 — Cross-doc (3040 ↔ 4111 ↔ DRSAC) [Sprint 3+]
//	L4 — Histórico (diff vs base anterior) [Sprint 3+]
package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit/l4"
	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
	"github.com/fortvna/radiant-norma/backend/internal/docdli"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/observability"
	"go.opentelemetry.io/otel/attribute"
)

// Critica representa uma regra de validação.
type Critica struct {
	ID             int64     `json:"id"`
	CadocCode      string    `json:"cadoc_code"`
	Sheet          string    `json:"sheet"`
	Codigo         string    `json:"codigo"`
	Regra          string    `json:"regra"`
	Descricao      string    `json:"descricao"`
	Gravidade      string    `json:"gravidade"`
	DataBaseInicio time.Time `json:"data_base_inicio"`
	MensagemErro   string    `json:"mensagem_erro"`
	Enabled        bool      `json:"enabled"`
}

// ValidationError é o resultado de uma regra que falhou.
type ValidationError struct {
	Critica  Critica `json:"critica"`
	Severity string  `json:"severity"` // E (Erro), A (Aviso), I (Informativo)
	Message  string  `json:"message"`
	XMLLine  int     `json:"xml_line,omitempty"`
}

// ValidationRequest é o input do endpoint /v1/validate.
//
// Aceita tanto `cadoc` (cliente-friendly, documentado no README) quanto
// `cadoc_code` (nome da coluna no DB) por compatibilidade.
//
// Sprint 12 (v3.5.0): IfID é populado pelo handler a partir do JWT claims
// (não vem do request body). Service usa pra filtrar regras desabilitadas
// via ruleprefs.Preferences.
type ValidationRequest struct {
	CadocCode   string `json:"cadoc_code"`
	DataBase    string `json:"data_base"`    // YYYY-MM-DD
	XML         string `json:"xml"`          // pode ser XML ou JSON (3044)
	ContentType string `json:"content_type"` // "application/xml" ou "application/json"
	IfID        string `json:"-"`            // populado pelo handler, não serializado
}

// UnmarshalJSON customizado: aceita "cadoc" OU "cadoc_code" no JSON.
func (r *ValidationRequest) UnmarshalJSON(data []byte) error {
	// Tipo shadow com cadoc opcional.
	type shadow struct {
		CadocCode   string `json:"cadoc_code"`
		Cadoc       string `json:"cadoc"` // alias
		DataBase    string `json:"data_base"`
		XML         string `json:"xml"`
		ContentType string `json:"content_type"`
	}
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	r.CadocCode = s.CadocCode
	if r.CadocCode == "" {
		r.CadocCode = s.Cadoc
	}
	r.DataBase = s.DataBase
	r.XML = s.XML
	r.ContentType = s.ContentType
	return nil
}

// ValidationResponse é o output.
type ValidationResponse struct {
	CadocCode  string            `json:"cadoc_code"`
	DataBase   string            `json:"data_base"`
	XMLHash    string            `json:"xml_hash"`
	Passed     bool              `json:"passed"`
	Errors     []ValidationError `json:"errors"`
	Warnings   []ValidationError `json:"warnings"`
	ExecutedAt time.Time         `json:"executed_at"`
	DurationMs int64             `json:"duration_ms"`

	// Sprint 12 (v3.5.0): lista de códigos de regras que foram puladas
	// porque desabilitadas via ruleprefs (toggle de /v1/rules/{code}/toggle).
	// Empty se nenhuma ou se service roda sem ruleprefs injetado.
	DisabledRules []string `json:"disabled_rules,omitempty"`
}

// RulePrefs é interface mínima que *ruleprefs.Preferences satisfaz.
// Permite ao Service filtrar regras desabilitadas por IF sem criar
// import cycle (audit não importa ruleprefs diretamente — usa interface).
//
// Sprint 12 (v3.5.0): C32.23 — disabled_rules agora afeta validação real.
type RulePrefs interface {
	ListDisabledCodes(ctx context.Context, ifID string) ([]string, error)
}

// Service é o serviço do Norma Audit.
type Service struct {
	db       *sql.DB
	registry *rules.Registry     // registry de regras portadas (Sprint 4+)
	prefs    RulePrefs           // Sprint 12: filter de regras desabilitadas por IF
	genReg   *generator.Registry // Sprint 57: generator registry (fonte canônica de root tags)
}

// New cria um novo Service.
func New(db *sql.DB) *Service {
	return &Service{db: db, registry: rules.Builtin3040()}
}

// SetRegistry permite injetar registry customizado (testes).
func (s *Service) SetRegistry(r *rules.Registry) {
	s.registry = r
}

// SetRulePrefs injeta ruleprefs service (Sprint 12 v3.5.0).
// Se não setado, validação roda sem filtrar disabled rules.
func (s *Service) SetRulePrefs(p RulePrefs) {
	s.prefs = p
}

// SetGeneratorRegistry injeta o generator registry (Sprint 57).
// Usado pelo Norma Audit para obter root tags canônicas dos generators.
// Se não setado, usa expectedRootTag() como fallback (comportamento legacy).
func (s *Service) SetGeneratorRegistry(r *generator.Registry) {
	s.genReg = r
}

// CompareWithPrevious compara um envio com seu anteior (L4 Histórico).
// Sprint 55: implementação inicial do L4.
func (s *Service) CompareWithPrevious(ctx context.Context, envioID string) (*l4.Comparison, error) {
	engine := l4.NewEngine(s.db)
	return engine.Compare(ctx, envioID)
}

// Registry retorna o registry atual.
func (s *Service) Registry() *rules.Registry {
	return s.registry
}

// LoadCriticas retorna todas as críticas (habilitadas E desabilitadas) de um CADOC.
//
// Sprint 4+: retorna TODAS porque algumas regras implementadas no registry
// podem estar desabilitadas no DB (BACEN marca habilitado?=n para regras
// que ainda não estão em vigor). O applyRegra decide se roda cada uma
// baseado no flag `enabled` da Critica.
//
// Usa sql.NullString/sql.NullTime para campos opcionais (regra, descricao,
// gravidade, data_base_inicio, mensagem_erro) — registros antigos ou
// inseridos sem esses campos podem ter NULL.
//
// Sprint 6 v1.5.0 (F8 — descoberto via TestListRules_ByCadoc com INSERT
// sem descricao): `regra` e `descricao` foram adicionados à lista de
// NullString (mesmo padrão do bug latente do v1.4.0 em auditlog.Verify).
// Sem isso, Scan falha com "converting NULL to string is unsupported"
// e a validação inteira quebra (L2-LOAD).
func (s *Service) LoadCriticas(ctx context.Context, cadocCode string) ([]Critica, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cadoc_code, sheet, codigo, regra, descricao, gravidade,
		       data_base_inicio, mensagem_erro, enabled
		FROM criticas
		WHERE cadoc_code = ?
		ORDER BY sheet, codigo
	`, cadocCode)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []Critica
	for rows.Next() {
		var c Critica
		var regra, desc, grav, msg sql.NullString
		var dbi sql.NullTime
		if err := rows.Scan(&c.ID, &c.CadocCode, &c.Sheet, &c.Codigo,
			&regra, &desc, &grav, &dbi, &msg, &c.Enabled); err != nil {
			return nil, err
		}
		if regra.Valid {
			c.Regra = regra.String
		}
		if desc.Valid {
			c.Descricao = desc.String
		}
		if grav.Valid {
			c.Gravidade = grav.String
		}
		if dbi.Valid {
			c.DataBaseInicio = dbi.Time
		}
		if msg.Valid {
			c.MensagemErro = msg.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FullValidationRequest é o input para ValidateFull().
type FullValidationRequest struct {
	// Documento principal sendo validado.
	Main *ValidationRequest
	// Documentos relacionados para validação cross-doc (L3).
	// Key: código CADOC ("2160", "2170", "3044").
	RelatedDocs map[string]*ValidationRequest
	// EnvioID para comparação histórica (L4). Opcional.
	EnvioID string
}

// FullValidationResponse é o output consolidado de L1+L2+L3+L4.
type FullValidationResponse struct {
	CadocCode  string       `json:"cadoc_code"`
	DataBase   string       `json:"data_base"`
	XMLHash    string       `json:"xml_hash"`
	Passed     bool         `json:"passed"`
	DurationMs int64        `json:"duration_ms"`
	L1         *LayerResult `json:"l1"`
	L2         *LayerResult `json:"l2"`
	L3         *LayerResult `json:"l3"`
	L4         *LayerResult `json:"l4"`
}

// LayerResult representa o resultado de uma camada de validação.
type LayerResult struct {
	Status   LayerStatus       `json:"status"` // "passed", "failed", "error"
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Message  string            `json:"message,omitempty"` // erro de sistema (não validation error)
}

// LayerStatus representa o estado de uma camada de validação.
type LayerStatus string

const (
	LayerPassed LayerStatus = "passed"
	LayerFailed LayerStatus = "failed"
	LayerError  LayerStatus = "error" // panic ou erro de sistema
)

// ValidateFull orchestrates L1+L2+L3+L4 in parallel goroutines with panic recovery.
// Returns a consolidated result. Each layer runs independently;
// if one panics, the others continue and the error is captured in the LayerResult.
func (s *Service) ValidateFull(ctx context.Context, req *FullValidationRequest) (*FullValidationResponse, error) {
	if req == nil || req.Main == nil {
		return nil, errors.New("FullValidationRequest.Main is required")
	}

	ctx, span := observability.StartSpan(ctx, "audit.ValidateFull")
	defer span.End()

	start := time.Now()

	main := req.Main
	xmlHash := sha256.Sum256([]byte(main.XML))
	hashStr := hex.EncodeToString(xmlHash[:])

	resp := &FullValidationResponse{
		CadocCode: main.CadocCode,
		DataBase:  main.DataBase,
		XMLHash:   hashStr,
		Passed:    true,
		L1:        &LayerResult{Status: LayerPassed},
		L2:        &LayerResult{Status: LayerPassed},
		L3:        &LayerResult{Status: LayerPassed},
		L4:        &LayerResult{Status: LayerPassed},
	}

	// Parseia documentos relacionados para L3 (cross-doc) antes de iniciar as goroutines.
	// Os parsed docs são configurados nos globals do pacote rules,
	// e as regras XD* os acessam via var locais.
	var parsedDRL *rules.DocDRL
	var parsedDLP *rules.DocDLP
	var parsed3044 *rules.Doc3044

	if req.RelatedDocs != nil {
		if doc, ok := req.RelatedDocs["2160"]; ok {
			parsedDRL, _ = rules.ParseDocDRL([]byte(doc.XML))
		}
		if doc, ok := req.RelatedDocs["2170"]; ok {
			parsedDLP, _ = rules.ParseDocDLP([]byte(doc.XML))
		}
		if doc, ok := req.RelatedDocs["3044"]; ok {
			parsed3044, _ = rules.ParseDoc3044([]byte(doc.XML))
		}
	}

	var wg sync.WaitGroup
	wg.Add(3) // L1, L2, L4 only. L3 runs after wg.Wait() to avoid data race
	// on package globals (parsedDRL/parsedDLP/parsed3044) set by L3.

	// L1 — Parse + XSD (nunca panics, mas capturamos caso)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("L1 panicked", "panic", r)
				resp.L1.Status = LayerError
				resp.L1.Message = fmt.Sprintf("panic: %v", r)
			}
		}()
		_, span := observability.StartSpan(ctx, "audit.L1")
		defer span.End()
		err := s.validateL1Parse(main)
		if err != nil {
			resp.L1.Status = LayerFailed
			resp.L1.Errors = append(resp.L1.Errors, ValidationError{
				Critica:  Critica{Codigo: "L1-PARSE", CadocCode: main.CadocCode},
				Severity: "E",
				Message:  "documento XML/JSON inválido",
			})
		}
	}()

	// L2 — Regras semânticas
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("L2 panicked", "panic", r)
				resp.L2.Status = LayerError
				resp.L2.Message = fmt.Sprintf("panic: %v", r)
			}
		}()
		_, span := observability.StartSpan(ctx, "audit.L2")
		defer span.End()
		l2Resp, err := s.Validate(ctx, main)
		if err != nil {
			resp.L2.Status = LayerError
			resp.L2.Message = err.Error()
			return
		}
		resp.L2.Errors = l2Resp.Errors
		resp.L2.Warnings = l2Resp.Warnings
		if !l2Resp.Passed {
			resp.L2.Status = LayerFailed
		}
	}()

	// L4 — Comparação histórica
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("L4 panicked", "panic", r)
				resp.L4.Status = LayerError
				resp.L4.Message = fmt.Sprintf("panic: %v", r)
			}
		}()
		_, span := observability.StartSpan(ctx, "audit.L4")
		defer span.End()
		if req.EnvioID == "" {
			return
		}
		cmp, err := s.CompareWithPrevious(ctx, req.EnvioID)
		if err != nil {
			resp.L4.Status = LayerError
			resp.L4.Message = err.Error()
			return
		}
		if cmp == nil {
			return
		}
		// Traduz Comparison para ValidationErrors do L4.
		for _, f := range cmp.NewFailures {
			resp.L4.Errors = append(resp.L4.Errors, ValidationError{
				Critica:  Critica{Codigo: f.Code, CadocCode: main.CadocCode, Sheet: "Histórico"},
				Severity: f.Severity,
				Message:  f.Message,
			})
		}
		for _, w := range cmp.FixedRules {
			resp.L4.Warnings = append(resp.L4.Warnings, ValidationError{
				Critica:  Critica{Codigo: w.Code, CadocCode: main.CadocCode, Sheet: "Histórico"},
				Severity: "A",
				Message:  w.Message,
			})
		}
		if len(resp.L4.Errors) > 0 {
			resp.L4.Status = LayerFailed
		}
	}()

	wg.Wait()

	// L3 — Cross-doc (XD* rules). Executado serialmente APÓS L1/L2/L4 para
	// evitar data race nas globals do pacote rules (parsedDRL/parsedDLP/parsed3044).
	// Sem recover() aqui — se L3 panicar é bug de programação, não dado de input.
	{
		_, span := observability.StartSpan(ctx, "audit.L3")
		defer span.End()
		if parsedDRL != nil {
			rules.SetDRL(parsedDRL)
		}
		if parsedDLP != nil {
			rules.SetDLP(parsedDLP)
		}
		if parsed3044 != nil {
			rules.Set3044(parsed3044)
		}
		xdCodes := []string{
			"XD01", "XD02", "XD03", "XD04", "XD05",
			"XD06", "XD07", "XD08",
		}
		for _, code := range xdCodes {
			rule := s.registry.Get(code)
			if rule == nil {
				continue
			}
			var doc3040 *rules.Doc3040
			if main.CadocCode == "3040" && main.ContentType != "application/json" {
				doc3040, _ = rules.ParseDoc3040([]byte(main.XML))
			}
			if err := rule.Apply(ctx, doc3040); err != nil {
				sev := rule.Severity()
				ve := ValidationError{
					Critica:  Critica{Codigo: code, CadocCode: main.CadocCode, Sheet: "Cross-doc"},
					Severity: sev,
					Message:  loggerutil.SafeError(err),
				}
				if sev == "E" {
					resp.L3.Errors = append(resp.L3.Errors, ve)
				} else {
					resp.L3.Warnings = append(resp.L3.Warnings, ve)
				}
			}
		}
		if len(resp.L3.Errors) > 0 {
			resp.L3.Status = LayerFailed
		}
	}

	// Determina overall Passed: só falha se L1 ou L2 falhar (bloqueantes).
	// L3 e L4 são warnings/non-blocking.
	resp.Passed = resp.L1.Status == LayerPassed &&
		resp.L2.Status == LayerPassed &&
		len(resp.L2.Errors) == 0

	resp.DurationMs = time.Since(start).Milliseconds()
	return resp, nil
}

// FullToValidationResponse converts a FullValidationResponse (L1-L4) to the
// legacy ValidationResponse format for backwards compatibility with existing
// API clients.
//
// Phase 1.3: /v1/validate agora usa ValidateFull internamente, mas retorna
// ValidationResponse para não quebrar callers existentes.
func FullToValidationResponse(full *FullValidationResponse) *ValidationResponse {
	if full == nil {
		return &ValidationResponse{Passed: false, Errors: []ValidationError{{
			Critica:  Critica{Codigo: "INTERNAL_ERROR"},
			Severity: "E",
			Message:  "FullValidationResponse is nil",
		}}}
	}

	var allErrors, allWarnings []ValidationError

	for _, layer := range []*LayerResult{full.L1, full.L2, full.L3, full.L4} {
		if layer == nil {
			continue
		}
		// Only include errors from layers that failed.
		// L1/L2 failures are always blocking.
		// L3/L4 failures are included as errors only if severity is "E".
		if layer.Status == LayerFailed {
			for _, e := range layer.Errors {
				if layer == full.L3 || layer == full.L4 {
					// L3/L4 errors from cross-doc/historical are warnings unless marked E
					if e.Severity == "E" {
						allErrors = append(allErrors, e)
					} else {
						allWarnings = append(allWarnings, e)
					}
				} else {
					allErrors = append(allErrors, e)
				}
			}
		}
		allWarnings = append(allWarnings, layer.Warnings...)
	}

	// Passed = true only if all blocking layers (L1, L2) passed.
	passed := full.L1 != nil && full.L1.Status == LayerPassed &&
		full.L2 != nil && full.L2.Status == LayerPassed

	return &ValidationResponse{
		CadocCode:  full.CadocCode,
		DataBase:   full.DataBase,
		XMLHash:    full.XMLHash,
		Passed:     passed,
		Errors:     allErrors,
		Warnings:   allWarnings,
		ExecutedAt: time.Now(),
		DurationMs: full.DurationMs,
	}
}

// Validate é o entrypoint principal: recebe um XML/JSON, retorna erros.
func (s *Service) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "audit.Validate",
		attribute.String("cadoc", req.CadocCode))
	defer span.End()

	start := time.Now()

	// Hash do payload
	hash := sha256.Sum256([]byte(req.XML))
	xmlHash := hex.EncodeToString(hash[:])

	resp := &ValidationResponse{
		CadocCode:  req.CadocCode,
		DataBase:   req.DataBase,
		XMLHash:    xmlHash,
		Passed:     true, // vira false se algum erro
		Errors:     []ValidationError{},
		Warnings:   []ValidationError{},
		ExecutedAt: start,
	}

	// L1 — Parse XML/JSON
	l1Err := s.validateL1Parse(req)
	if l1Err != nil {
		// Validação 19 (F19.11): sanitizar L1 parse error — não expor
		// XML element names, attribute paths, ou SQL fragments.
		// Log completo (sanitizado) para debug; response usa mensagem genérica.
		logger := slog.Default()
		logger.Error("audit L1 parse failed",
			"cadoc", req.CadocCode,
			"err", loggerutil.SafeError(l1Err))
		resp.Errors = append(resp.Errors, ValidationError{
			Critica:  Critica{Codigo: "L1-PARSE", CadocCode: req.CadocCode},
			Severity: "E",
			Message:  "documento XML/JSON inválido",
		})
		// L1 falhou: aborta L2 (regras semânticas que parseiam XML/JSON não rodam)
		// sem isso, gera 13+ erros de "parser 3040 falhou" duplicados.
		resp.Passed = false
		resp.DurationMs = time.Since(start).Milliseconds()
		return resp, nil
	}

	// L2 — Regras semânticas (carrega críticas do DB e aplica as implementadas)
	criticas, err := s.LoadCriticas(ctx, req.CadocCode)
	if err != nil {
		// Validação 19 (F19.12): sanitizar DB error — não expor SQL fragments
		// ou table names via Message de ValidationError (vai pro JSON response).
		logger := slog.Default()
		logger.Error("audit L2 load failed",
			"cadoc", req.CadocCode,
			"err", loggerutil.SafeError(err))
		resp.Errors = append(resp.Errors, ValidationError{
			Critica:  Critica{Codigo: "L2-LOAD", CadocCode: req.CadocCode},
			Severity: "E",
			Message:  "erro carregando regras",
		})
	} else {
		// Parseia o XML 3040 UMA vez (perf: 25 regras × 1 parse = 25x slowdown)
		var cachedDoc *rules.Doc3040
		is3040 := req.CadocCode == "3040" && req.ContentType != "application/json"

		// Sprint 58 (v3.34.40): cache DLI parse — 2062 é novo parser.
		var cachedDLI *docdli.DocumentoDLI
		is2062 := req.CadocCode == "2062" && req.ContentType != "application/json"

		// Sprint 12 (v3.5.0): C32.23 — carrega set de regras desabilitadas
		// por IF (via ruleprefs) e pula elas na validação. Set carregado
		// UMA vez por Validate() — perf: 1 query independente de N regras.
		var disabledSet map[string]bool
		if s.prefs != nil && req.IfID != "" {
			codes, err := s.prefs.ListDisabledCodes(ctx, req.IfID)
			if err != nil {
				// Log mas não falha — preferência é best-effort. Sem
				// prefs, todas regras rodam (comportamento legacy).
				logger := slog.Default()
				logger.Warn("audit ruleprefs ListDisabledCodes failed",
					"if_id", req.IfID,
					"err", loggerutil.SafeError(err))
			} else {
				disabledSet = make(map[string]bool, len(codes))
				for _, code := range codes {
					disabledSet[code] = true
				}
				// Inclui na response pra transparency (frontend mostra
				// quais regras foram puladas).
				resp.DisabledRules = codes
			}
		}

		for _, c := range criticas {
			// Respeita flag `enabled` do DB (BACEN marca habilitado?=n para
			// regras que NÃO estão em vigor na data-base corrente).
			// Sem isso, o XML exemplo oficial (Mod=0213 com v150>0) falha.
			if !c.Enabled {
				continue
			}

			// Sprint 12 (v3.5.0): pula regras desabilitadas pelo IF via
			// toggle em /v1/rules/{code}/toggle. Feature era cosmética
			// até Sprint 12 (C32.23) — agora afeta validação real.
			if disabledSet[c.Codigo] {
				continue
			}

			ruleErr := s.applyRegra(ctx, c, req, is3040, &cachedDoc, is2062, &cachedDLI)
			if ruleErr == nil {
				continue
			}

			// Severity: prioriza o que a Rule implementada declara (registry).
			// Se a regra está no registry, a implementação define a gravidade autoritativa.
			// Caso contrário, usa Gravidade do DB, e finalmente "A" (aviso) como default.
			sev := ""
			inRegistry := false
			if s.registry != nil {
				if rule := s.registry.Get(c.Codigo); rule != nil {
					inRegistry = true
					sev = rule.Severity()
				}
			}
			if !inRegistry {
				sev = c.Gravidade
				if sev == "" {
					sev = "A"
				}
			}
			ve := ValidationError{
				Critica:  c,
				Severity: sev,
				// Validação 19 (F19.13): sanitizar regra error via
				// loggerutil.SafeError antes de expor no JSON response.
				// SafeError preserva informação útil (mês 13, etc.) e
				// só mascara DSN/password/host.
				Message: loggerutil.SafeError(ruleErr),
			}
			if sev == "E" {
				resp.Errors = append(resp.Errors, ve)
			} else {
				resp.Warnings = append(resp.Warnings, ve)
			}
		}
	}

	resp.Passed = len(resp.Errors) == 0
	resp.DurationMs = time.Since(start).Milliseconds()
	return resp, nil
}

// validateL1Parse faz o parse do XML ou JSON + validação XSD real.
//
// Fail-closed contract (Phase 1.1 of the remediation plan, gate #2):
//
//  1. JSON content: parses only (no schema enforcement; L1 is out of scope
//     for JSON, which has its own contract in the validation pipeline).
//
//  2. XML content: validates against the registered XSD for the CADOC when
//     the schema file is reachable from disk. XSD validation is the strictest
//     path: any XSD error fails the document.
//
//  3. When the XSD is registered but the file is not readable on the test
//     host (e.g. tests that don't run from the project root), we fall back
//     to a *strict* root-tag + non-empty check. The historical permissive
//     fallback (which approved <Doc3040/>, <DocDRSAC/>, <Documento4111/> as
//     long as the root tag matched) is replaced by:
//
//     a. The XML must parse.
//     b. The root tag must equal expectedRootTag(cadoc). If the CADOC has no
//     known root tag, validation fails closed (unknown CADOC).
//     c. The document must NOT be empty: a document whose root has zero
//     attributes and zero child elements is rejected. This is the change
//     that closes the audit finding "9/10 XMLs vazios aprovados".
//
// This preserves backwards compatibility with tests that exercise the
// root-tag path (e.g. TestValidate_F02_MesInvalido sends <Doc3040 .../> with
// attributes, which still parses and gets past L1 to reach L2/L3), while
// rejecting the empty documents the audit benchmark targets.
func (s *Service) validateL1Parse(req *ValidationRequest) error {
	content := string(req.XML)
	if req.ContentType == "application/json" {
		var v any
		if err := json.Unmarshal([]byte(req.XML), &v); err != nil {
			return fmt.Errorf("JSON não parseia: %w", err)
		}
		return nil
	}

	// XML path. Try strict XSD validation first.
	xsdErrors, xsdErr := ValidateXSD(req.CadocCode, content)
	switch {
	case xsdErr == nil && len(xsdErrors) == 0:
		// XSD happy path.
		return nil
	case xsdErr == nil && len(xsdErrors) > 0:
		return fmt.Errorf("XSD validation failed: %s", xsdErrors[0])
	}

	// XSD path failed. Distinguish "schema unavailable" (the test/CI host
	// simply can't load the file) from a hard schema error.
	var unavailable *ErrSchemaUnavailable
	if !errors.As(xsdErr, &unavailable) {
		return fmt.Errorf("L1 XSD validation refused for CADOC %q: %w", req.CadocCode, xsdErr)
	}

	// Strict fallback. Parse + root tag + non-empty.
	rootTag := s.expectedRootTag(req.CadocCode)
	if rootTag == "" {
		return fmt.Errorf("CADOC %q não é suportado pelo validator L1 (sem root tag canônico)", req.CadocCode)
	}

	decoder := xml.NewDecoder(strings.NewReader(content))
	var (
		rootSeen     bool
		rootHasAttrs bool
		childCount   int
		depth        int
	)
	for {
		tok, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("XML não parseia: %w", err)
		}
		if tok == nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !rootSeen {
				rootSeen = true
				if t.Name.Local != rootTag {
					return fmt.Errorf("esperado elemento raiz <%s> mas tem <%s>", rootTag, t.Name.Local)
				}
				rootHasAttrs = len(t.Attr) > 0
				depth = 1
				continue
			}
			depth++
			if depth > 1 {
				childCount++
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
			}
		}
	}
	if !rootSeen {
		return fmt.Errorf("XML sem elemento raiz")
	}
	if !rootHasAttrs && childCount == 0 {
		// Empty document like <DocDRSAC/> — fail closed.
		return fmt.Errorf("documento %s vazio (sem atributos e sem elementos filhos)", rootTag)
	}
	return nil
}

// expectedRootTag returns the canonical root tag for the given CADOC, or
// "" if the CADOC is not in the validator's allow-list. The Phase 1.1
// fail-closed contract requires that unknown CADOCs be rejected, so an
// empty return from this function is the canonical signal "this CADOC is
// not supported by L1".
//
// Phase 1.2: quando o generator registry está injetado (s.genReg),
// este método delega ao generator para obter a root tag canônica.
// Isso garante que validator e generator nunca divergem.
func (s *Service) expectedRootTag(cadoc string) string {
	// Fase 1.2: usa generator como fonte canônica se disponível.
	if s.genReg != nil {
		g := s.genReg.Get(cadoc)
		if g != nil {
			return g.RootTag()
		}
	}
	// Fallback legacy: mapa hardcoded (manter para CADOCs sem generator
	// e para backwards compatibility em testes).
	switch cadoc {
	case "3040":
		return "Doc3040"
	case "3050":
		return "DocTXB"
	case "3026":
		return "Doc3026"
	case "2030":
		return "DocumentoDRSAC"
	case "2060":
		return "Doc2060"
	case "2061":
		return "documentoDLO"
	case "2062":
		return "documentoDLI"
	case "2070":
		return "documentoDDR"
	case "2160":
		return "documentoDRL"
	case "2170":
		return "documentoDLP"
	case "4060":
		return "Documento"
	case "4111":
		return "Documento4111"
	}
	return ""
}

// applyRegra aplica UMA regra específica ao documento.
//
// Sprint 4+: usa rules.Registry para lookup. Se a regra está registrada
// (Básicas, Formato, Campos Obrigatórios, Semântica), executa via parser
// tipado. Caso contrário, fallback para heurísticas inline (B01-B05 em
// v1.4.x).
//
// Sprint 6 v1.5.0 (W3): B01-B05 movidas para o registry via interface
// RawRule (operam em XML bruto, sem parser tipado). Mantém compat com
// 25 regras tipadas já existentes (B06+).
//
// is3040 e cachedDoc permitem cachear o ParseDoc3040 (perf: 25 regras
// não precisam parsear 25x).
func (s *Service) applyRegra(
	ctx context.Context,
	c Critica,
	req *ValidationRequest,
	is3040 bool,
	cachedDoc **rules.Doc3040,
	is2062 bool,
	cachedDLI **docdli.DocumentoDLI,
) error {
	content := string(req.XML)

	// 1ª tentativa: regra raw (B01-B05, opera em XML bruto)
	// Sprint 6 v1.5.0 (W3): removido hardcode "if c.Codigo == 'B01'..."
	if s.registry != nil {
		if rawRule := s.registry.GetRaw(c.Codigo); rawRule != nil {
			return rawRule.ApplyRaw(ctx, content)
		}
	}

	// 2ª tentativa: regra tipada (opera em *Doc3040)
	if s.registry != nil && is3040 {
		rule := s.registry.Get(c.Codigo)
		if rule != nil {
			// Lazy load: parseia uma vez e cacheia
			if *cachedDoc == nil {
				doc, err := rules.ParseDoc3040([]byte(req.XML))
				if err != nil {
					return fmt.Errorf("parser 3040 falhou: %w", err)
				}
				*cachedDoc = doc
			}
			return rule.Apply(ctx, *cachedDoc)
		}
	}

	// Sprint 58 (v3.34.40): 3ª tentativa — regra tipada DLI (opera em *DocumentoDLI)
	// Sprint 55: reescrito — agora também despacha regras semânticas DLI-09 a DLI-18.
	if is2062 {
		if *cachedDLI == nil {
			doc, err := docdli.ParseFromBytes([]byte(req.XML))
			if err != nil {
				return fmt.Errorf("parser DLI falhou: %w", err)
			}
			*cachedDLI = doc
		}

		// Sempre roda validações estruturais primeiro (DLI-01 a DLI-08).
		// Estas são bloqueantes — se falharem, não prossegue.
		if errs := docdli.Validate(*cachedDLI); len(errs) > 0 {
			return errs[0]
		}

		// Sprint 55 fix: regras semânticas DLI-09 a DLI-18 estão em
		// rules2062 registry (via Register2062). Despacha via Get2062.
		// Usa rules.ParseDocDLI para o tipo DocDLI que as regras esperan.
		if rule2062 := s.registry.Get2062(c.Codigo); rule2062 != nil {
			// ParseDocDLIextrai fields que as regras semânticas precisam.
			docDLI, err := rules.ParseDocDLI([]byte(req.XML))
			if err != nil {
				return fmt.Errorf("parse DLI para regra %s falhou: %w", c.Codigo, err)
			}
			return rule2062.Apply(ctx, docDLI)
		}

		// Regra não no registry — validação estrutural já passou acima.
		return nil
	}

	// Regra não implementada — skip (vai virar erro quando tivermos 100% cobertura)
	return nil
}
