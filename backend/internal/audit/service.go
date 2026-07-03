// Package audit implementa o Sentinel Audit (validação de CADOCs contra regras).
//
// Camadas (cf. PRODUTO_TESE_ROADMAP § 5):
//   L1 — Structural (XSD)
//   L2 — Semantic (críticas do BACEN)
//   L3 — Cross-doc (3040 ↔ 4111 ↔ DRSAC) [Sprint 3+]
//   L4 — Histórico (diff vs base anterior) [Sprint 3+]
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
	"strings"
	"time"
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
	Critica   Critica `json:"critica"`
	Severity  string  `json:"severity"` // E (Erro), A (Aviso), I (Informativo)
	Message   string  `json:"message"`
	XMLLine   int     `json:"xml_line,omitempty"`
}

// ValidationRequest é o input do endpoint /v1/validate.
type ValidationRequest struct {
	CadocCode   string `json:"cadoc_code"`
	DataBase    string `json:"data_base"` // YYYY-MM-DD
	XML         string `json:"xml"`       // pode ser XML ou JSON (3044)
	ContentType string `json:"content_type"` // "application/xml" ou "application/json"
}

// ValidationResponse é o output.
type ValidationResponse struct {
	CadocCode    string            `json:"cadoc_code"`
	DataBase     string            `json:"data_base"`
	XMLHash      string            `json:"xml_hash"`
	Passed       bool              `json:"passed"`
	Errors       []ValidationError `json:"errors"`
	Warnings     []ValidationError `json:"warnings"`
	ExecutedAt   time.Time         `json:"executed_at"`
	DurationMs   int64             `json:"duration_ms"`
}

// Service é o serviço do Sentinel Audit.
type Service struct {
	db *sql.DB
}

// New cria um novo Service.
func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// LoadCriticas retorna todas as críticas habilitadas de um CADOC.
func (s *Service) LoadCriticas(ctx context.Context, cadocCode string) ([]Critica, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cadoc_code, sheet, codigo, regra, descricao, gravidade,
		       data_base_inicio, mensagem_erro, enabled
		FROM criticas
		WHERE cadoc_code = ? AND enabled = 1
		ORDER BY sheet, codigo
	`, cadocCode)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []Critica
	for rows.Next() {
		var c Critica
		var grav sql.NullString
		var dbi sql.NullTime
		if err := rows.Scan(&c.ID, &c.CadocCode, &c.Sheet, &c.Codigo, &c.Regra, &c.Descricao,
			&grav, &dbi, &c.MensagemErro, &c.Enabled); err != nil {
			return nil, err
		}
		if grav.Valid {
			c.Gravidade = grav.String
		}
		if dbi.Valid {
			c.DataBaseInicio = dbi.Time
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
	if err := s.validateL1Parse(req); err != nil {
		resp.Errors = append(resp.Errors, ValidationError{
			Critica:  Critica{Codigo: "L1-PARSE", CadocCode: req.CadocCode},
			Severity: "E",
			Message:  err.Error(),
		})
	}

	// L2 — Regras semânticas (carrega críticas do DB e aplica as implementadas)
	criticas, err := s.LoadCriticas(ctx, req.CadocCode)
	if err != nil {
		resp.Errors = append(resp.Errors, ValidationError{
			Critica:  Critica{Codigo: "L2-LOAD", CadocCode: req.CadocCode},
			Severity: "E",
			Message:  "Erro carregando críticas: " + err.Error(),
		})
	} else {
		for _, c := range criticas {
			if err := s.applyRegra(ctx, c, req); err != nil {
				sev := c.Gravidade
				if sev == "" {
					sev = "A"
				}
				ve := ValidationError{
					Critica:  c,
					Severity: sev,
					Message:  err.Error(),
				}
				if sev == "E" {
					resp.Errors = append(resp.Errors, ve)
				} else {
					resp.Warnings = append(resp.Warnings, ve)
				}
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
// Em Sprint 3, implementamos apenas B01-B05 + algumas heurísticas simples.
func (s *Service) applyRegra(ctx context.Context, c Critica, req *ValidationRequest) error {
	content := string(req.XML)

	// B01-B05 — Regras básicas (Cadoc 3040)
	if c.Codigo == "B01" {
		// B01: arquivo XML deve ser válido (já checado em L1)
		return nil
	}
	if c.Codigo == "B02" {
		// B02: arquivo .ZIP deve ser gerado pelo aplicativo validador — não checamos aqui
		return nil
	}
	if c.Codigo == "B03" {
		// B03: instituição remetente deve possuir autorização — check externo
		return nil
	}
	if c.Codigo == "B04" {
		// B04: codificação deve estar declarada
		if !strings.HasPrefix(strings.TrimSpace(content), "<?xml") {
			return errors.New("arquivo não começa com declaração <?xml")
		}
		return nil
	}
	if c.Codigo == "B05" {
		// B05: arquivo não pode estar vazio/muito pequeno
		if len(req.XML) == 0 {
			return errors.New("arquivo XML está vazio")
		}
		if len(req.XML) < 50 {
			return fmt.Errorf("arquivo XML tem apenas %d bytes", len(req.XML))
		}
		return nil
	}

	// Outras regras — skip por enquanto (Sprint 4)
	return nil
}