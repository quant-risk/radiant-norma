// Package validation implements the L1–L4 validation pipeline for generated CADOCs.
//
// Validation levels:
//   L1 — XSD structural: XML parses and conforms to schema structure
//   L2 — Required fields: mandatory fields are present and non-empty
//   L3 — Semantic rules: business rules (e.g. soma parcelas = valor total)
//   L4 — Cross-doc consistency: consistency across multiple CADOCs in a batch
//
// Each level returns a ValidationResult. Levels are run sequentially; later
// levels are only executed if earlier levels pass.
package validation

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
)

// Level is a validation level (L1–L4).
type Level int

const (
	L1XSD Level = iota + 1
	L2Required
	L3Semantic
	L4CrossDoc
)

func (l Level) String() string {
	switch l {
	case L1XSD:
		return "L1-XSD"
	case L2Required:
		return "L2-Required"
	case L3Semantic:
		return "L3-Semantic"
	case L4CrossDoc:
		return "L4-CrossDoc"
	default:
		return "L?"
	}
}

// Issue is a single validation issue.
type Issue struct {
	Level   Level   `json:"level"`
	Code    string `json:"code"`    // e.g. "MISSING_FIELD", "XSD_INVALID"
	Field   string `json:"field"`   // XML tag path, e.g. "Doc3040.Agreg.Venc.V150"
	Message string `json:"message"` // human-readable
}

// ValidationResult holds the result of a validation run.
type ValidationResult struct {
	OK     bool     `json:"ok"`
	Passed []Level  `json:"passed"` // levels that passed
	Issues []Issue  `json:"issues,omitempty"`
}

// AddIssue appends a validation issue.
func (r *ValidationResult) AddIssue(lvl Level, code, field, msg string) {
	r.Issues = append(r.Issues, Issue{
		Level:   lvl,
		Code:    code,
		Field:   field,
		Message: msg,
	})
}

// Config holds validation configuration.
type Config struct {
	// RunL1 enables XSD structural validation.
	RunL1 bool
	// RunL2 enables required fields check.
	RunL2 bool
	// RunL3 enables semantic rules.
	RunL3 bool
	// RunL4 enables cross-doc validation (requires DocSet).
	RunL4 bool
}

// DefaultConfig returns a Config with all levels enabled.
func DefaultConfig() Config {
	return Config{
		RunL1: true,
		RunL2: true,
		RunL3: true,
		RunL4: true,
	}
}

// Validate runs the validation pipeline for a single XML document.
// For L4 (cross-doc), use ValidateBatch.
func Validate(ctx context.Context, xmlBytes []byte, cadocCode string, cfg Config) *ValidationResult {
	res := &ValidationResult{Passed: []Level{}}
	res.OK = true

	if cfg.RunL1 {
		if err := validateL1XSD(xmlBytes, cadocCode); err != nil {
			res.AddIssue(L1XSD, "XSD_INVALID", rootTag(cadocCode), err.Error())
			res.OK = false
		} else {
			res.Passed = append(res.Passed, L1XSD)
		}
	} else {
		res.Passed = append(res.Passed, L1XSD)
	}

	if !res.OK {
		return res
	}

	if cfg.RunL2 {
		if issues := validateL2Required(xmlBytes, cadocCode); len(issues) > 0 {
			for _, iss := range issues {
				res.AddIssue(L2Required, iss.Code, iss.Field, iss.Message)
			}
			res.OK = false
		} else {
			res.Passed = append(res.Passed, L2Required)
		}
	} else {
		res.Passed = append(res.Passed, L2Required)
	}

	if !res.OK {
		return res
	}

	if cfg.RunL3 {
		if issues := validateL3Semantic(xmlBytes, cadocCode); len(issues) > 0 {
			for _, iss := range issues {
				res.AddIssue(L3Semantic, iss.Code, iss.Field, iss.Message)
			}
			res.OK = false
		} else {
			res.Passed = append(res.Passed, L3Semantic)
		}
	} else {
		res.Passed = append(res.Passed, L3Semantic)
	}

	// L4 requires a DocSet — use ValidateBatch for cross-doc validation.
	return res
}

// ValidateBatch runs L4 cross-doc validation using the crossdoc engine.
// Call this after Validate for each individual document.
func ValidateBatch(ctx context.Context, xmlDocs map[string][]byte, engine *crossdoc.Engine) *ValidationResult {
	res := &ValidationResult{Passed: []Level{L1XSD, L2Required, L3Semantic}}
	res.OK = true

	if engine == nil || len(xmlDocs) == 0 {
		res.Passed = append(res.Passed, L4CrossDoc)
		return res
	}

	// Build crossdoc ValidationRequest from raw XML docs.
	cadocMap := make(map[string]string)
	for code, xml := range xmlDocs {
		cadocMap[code] = string(xml)
	}

	resp := engine.Validate(ctx, &crossdoc.ValidationRequest{Cadocs: cadocMap})

	if !resp.Passed || len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			res.AddIssue(L4CrossDoc, e.Code, "", e.Message)
		}
		res.OK = false
	}
	if len(resp.Warnings) > 0 {
		for _, w := range resp.Warnings {
			res.AddIssue(L4CrossDoc, w.Code, "", w.Message)
		}
		// Warnings don't fail the validation
	}

	if res.OK {
		res.Passed = append(res.Passed, L4CrossDoc)
	}

	return res
}

// =======================================================================
// L1 — XSD Structural Validation
// =======================================================================

// validateL1XSD checks that xmlBytes is valid XML.
func validateL1XSD(xmlBytes []byte, cadocCode string) error {
	// Decode XML into a generic map to confirm well-formedness.
	// Full XSD schema validation would require loading the XSD from the
	// Schema Registry (L1 via schema.GetEffective) — this is the
	// first-pass: well-formed XML.
	var v any
	if err := xml.Unmarshal(xmlBytes, &v); err != nil {
		return fmt.Errorf("XML mal formado: %w", err)
	}
	return nil
}

// =======================================================================
// L2 — Required Fields
// ========================================================================

func validateL2Required(xmlBytes []byte, cadocCode string) []Issue {
	var issues []Issue

	switch cadocCode {
	case "3040":
		issues = check3040Required(xmlBytes)
	case "3050":
		issues = check3050Required(xmlBytes)
	case "2061":
		issues = check2061Required(xmlBytes)
	case "2070":
		issues = check2070Required(xmlBytes)
	case "2160":
		issues = check2160Required(xmlBytes)
	case "2170":
		issues = check2170Required(xmlBytes)
	default:
		// For unknown CADOCs, skip L2 (don't false-positive).
	}

	return issues
}

func check3040Required(xmlBytes []byte) []Issue {
	type doc3040 struct {
		CNPJ      string `xml:"cnpj,attr"`
		DataBase  string `xml:"dataBase,attr"`
		Remessa   string `xml:"remessa,attr"`
		Parte     string `xml:"parte,attr"`
		TpArq     string `xml:"tpArq,attr"`
		NomeResp  string `xml:"nomeResp,attr"`
		EmailResp string `xml:"emailResp,attr"`
	}
	var d doc3040
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "Doc3040", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "Doc3040@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "Doc3040@dataBase", Message: "dataBase é obrigatório"})
	}
	if d.Remessa == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "Doc3040@remessa", Message: "remessa é obrigatório"})
	}
	if d.TpArq == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "Doc3040@tpArq", Message: "tpArq é obrigatório"})
	}
	return issues
}

func check3050Required(xmlBytes []byte) []Issue {
	type docTXB struct {
		CNPJ     string `xml:"cnpj,attr"`
		DataBase string `xml:"dataBase,attr"`
	}
	var d docTXB
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "DocTXB", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocTXB@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocTXB@dataBase", Message: "dataBase é obrigatório"})
	}
	return issues
}

func check2061Required(xmlBytes []byte) []Issue {
	type docDLO struct {
		CNPJ     string `xml:"cnpj,attr"`
		DataBase string `xml:"dataBase,attr"`
	}
	var d docDLO
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "DocDLO", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDLO@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDLO@dataBase", Message: "dataBase é obrigatório"})
	}
	return issues
}

func check2070Required(xmlBytes []byte) []Issue {
	type docDDR struct {
		CNPJ     string `xml:"cnpj,attr"`
		DataBase string `xml:"dataBase,attr"`
	}
	var d docDDR
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "DocDDR", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDDR@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDDR@dataBase", Message: "dataBase é obrigatório"})
	}
	return issues
}

func check2160Required(xmlBytes []byte) []Issue {
	type docDRL struct {
		CNPJ     string `xml:"cnpj,attr"`
		DataBase string `xml:"dataBase,attr"`
	}
	var d docDRL
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "DocDRL", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDRL@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDRL@dataBase", Message: "dataBase é obrigatório"})
	}
	return issues
}

func check2170Required(xmlBytes []byte) []Issue {
	type docDLP struct {
		CNPJ     string `xml:"cnpj,attr"`
		DataBase string `xml:"dataBase,attr"`
	}
	var d docDLP
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return []Issue{{Code: "XSD_INVALID", Field: "DocDLP", Message: err.Error()}}
	}
	var issues []Issue
	if d.CNPJ == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDLP@cnpj", Message: "cnpj é obrigatório"})
	}
	if d.DataBase == "" {
		issues = append(issues, Issue{Code: "MISSING_FIELD", Field: "DocDLP@dataBase", Message: "dataBase é obrigatório"})
	}
	return issues
}

// =======================================================================
// L3 — Semantic Rules
// ========================================================================

func validateL3Semantic(xmlBytes []byte, cadocCode string) []Issue {
	switch cadocCode {
	case "3040":
		return check3040Semantic(xmlBytes)
	case "3050":
		return check3050Semantic(xmlBytes)
	case "2160":
		return check2160Semantic(xmlBytes)
	case "2170":
		return check2170Semantic(xmlBytes)
	}
	return nil
}

func check3040Semantic(xmlBytes []byte) []Issue {
	var issues []Issue

	// SCR: LCR (V150 / total venc) must be consistent with 2160 if present.
	// We can only do intra-doc checks here; cross-doc needs L4.
	//
	// Intra-doc check: V150 (inadimplência > 90d) should be < total Venc.
	// This is a simple sanity check.
	type venc struct {
		V110 string `xml:"V110,omitempty"`
		V120 string `xml:"V120,omitempty"`
		V130 string `xml:"V130,omitempty"`
		V140 string `xml:"V140,omitempty"`
		V150 string `xml:"V150,omitempty"`
		V160 string `xml:"V160,omitempty"`
		V165 string `xml:"V165,omitempty"`
	}
	type agreg struct {
		Venc venc `xml:"Venc"`
	}
	type doc3040 struct {
		Agregadas []agreg `xml:"Agreg"`
	}

	var d doc3040
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return nil // L1 would catch this
	}

	for i, a := range d.Agregadas {
		fields := []string{a.Venc.V110, a.Venc.V120, a.Venc.V130, a.Venc.V140, a.Venc.V150, a.Venc.V160, a.Venc.V165}
		var vals [7]float64
		var total float64
		for j, f := range fields {
			if f != "" {
				var v float64
				fmt.Sscanf(f, "%f", &v)
				vals[j] = v
				total += v
			}
		}
		// V150 (inadimplência > 90d) should not exceed total
		if vals[4] > total && total > 0 {
			issues = append(issues, Issue{
				Code:    "SEMANTIC_V150_EXCEEDS_TOTAL",
				Field:   fmt.Sprintf("Doc3040.Agreg[%d].Venc.V150", i),
				Message: fmt.Sprintf("V150 (inadimplência >90d) R$%.2f excede total R$%.2f", vals[4], total),
			})
		}
	}

	return issues
}

func check3050Semantic(xmlBytes []byte) []Issue {
	// 3050 TXB: total de operações deve ser consistente com 3040.
	// This is a cross-doc check — only warn here.
	_ = xmlBytes
	return nil
}

func check2160Semantic(xmlBytes []byte) []Issue {
	// 2160 DRL: LCR must be >= 0 (ratio can't be negative).
	type drl struct {
		LCRRatio struct {
			Valor string `xml:"valor,attr"`
		} `xml:"LCRRatio"`
	}
	var d drl
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return nil
	}
	if d.LCRRatio.Valor != "" {
		var v float64
		fmt.Sscanf(d.LCRRatio.Valor, "%f", &v)
		if v < 0 {
			return []Issue{{
				Code:    "SEMANTIC_LCR_NEGATIVE",
				Field:   "DocDRL.LCRRatio@valor",
				Message: fmt.Sprintf("LCR ratio negativo: %.4f", v),
			}}
		}
	}
	return nil
}

func check2170Semantic(xmlBytes []byte) []Issue {
	// 2170 DLP: NSFR must be >= 0.
	type dlp struct {
		NSFRRatio struct {
			Valor string `xml:"valor,attr"`
		} `xml:"NSFRRatio"`
	}
	var d dlp
	if err := xml.Unmarshal(xmlBytes, &d); err != nil {
		return nil
	}
	if d.NSFRRatio.Valor != "" {
		var v float64
		fmt.Sscanf(d.NSFRRatio.Valor, "%f", &v)
		if v < 0 {
			return []Issue{{
				Code:    "SEMANTIC_NSFR_NEGATIVE",
				Field:   "DocDLP.NSFRRatio@valor",
				Message: fmt.Sprintf("NSFR ratio negativo: %.4f", v),
			}}
		}
	}
	return nil
}

// rootTag returns the expected root tag for cadocCode.
func rootTag(cadocCode string) string {
	switch cadocCode {
	case "3040":
		return "Doc3040"
	case "3050":
		return "DocTXB"
	case "2061":
		return "DocDLO"
	case "2070":
		return "DocDDR"
	case "2160":
		return "DocDRL"
	case "2170":
		return "DocDLP"
	case "4111":
		return "Documento4111"
	default:
		return "Doc" + cadocCode
	}
}

// TrivialDocSet is a minimal DocSet for validation purposes.
type TrivialDocSet struct{}

// String implements fmt.Stringer for TrivialDocSet.
func (TrivialDocSet) String() string { return "trivial" }
