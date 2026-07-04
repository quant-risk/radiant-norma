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
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
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
type ValidationRequest struct {
	CadocCode   string `json:"cadoc_code"`
	DataBase    string `json:"data_base"`    // YYYY-MM-DD
	XML         string `json:"xml"`          // pode ser XML ou JSON (3044)
	ContentType string `json:"content_type"` // "application/xml" ou "application/json"
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
}

// Service é o serviço do Norma Audit.
type Service struct {
	db       *sql.DB
	registry *rules.Registry // registry de regras portadas (Sprint 4+)
}

// New cria um novo Service.
func New(db *sql.DB) *Service {
	return &Service{db: db, registry: rules.Builtin3040()}
}

// SetRegistry permite injetar registry customizado (testes).
func (s *Service) SetRegistry(r *rules.Registry) {
	s.registry = r
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

// Validate é o entrypoint principal: recebe um XML/JSON, retorna erros.
func (s *Service) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error) {
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

		for _, c := range criticas {
			// Respeita flag `enabled` do DB (BACEN marca habilitado?=n para
			// regras que NÃO estão em vigor na data-base corrente).
			// Sem isso, o XML exemplo oficial (Mod=0213 com v150>0) falha.
			if !c.Enabled {
				continue
			}

			ruleErr := s.applyRegra(ctx, c, req, is3040, &cachedDoc)
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

// validateL1Parse faz o parse do XML ou JSON. Falha de parse = erro bloqueante.
func (s *Service) validateL1Parse(req *ValidationRequest) error {
	content := string(req.XML)
	if req.ContentType == "application/json" {
		var v any
		if err := json.Unmarshal([]byte(req.XML), &v); err != nil {
			return fmt.Errorf("JSON não parseia: %w", err)
		}
		return nil
	}
	// default: XML
	// Detecta tag raiz esperada por CADOC
	rootTag := expectedRootTag(req.CadocCode)
	if rootTag == "" {
		rootTag = "Documento"
	}
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		t, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("XML não parseia: %w", err)
		}
		if t == nil {
			break
		}
		if se, ok := t.(xml.StartElement); ok {
			if se.Name.Local != rootTag {
				return fmt.Errorf("esperado elemento raiz <%s> mas tem <%s>", rootTag, se.Name.Local)
			}
			break
		}
	}
	return nil
}

func expectedRootTag(cadoc string) string {
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

	// Regra não implementada — skip (vai virar erro quando tivermos 100% cobertura)
	return nil
}
