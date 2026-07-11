// Regras cross-document DRSAC (2030) ↔ 3040 (SCR) ↔ 4111.
//
// Este file implementa adapters que conectam os packages drsac e doc4111
// ao engine crossdoc.
//
// DRSAC: wrappers em torno de drsac.ValidateCrossRefs que extraem SCR data
// do 3040 contido no DocSet.
//
// 4111: regras estruturais 4111 ↔ 3040.
//
// Sprint 52 v3.34.33: integração DRSAC e 4111 no cross-doc engine.
// Sprint 72: refatorado para usar bacen.Doc3040 via xml.Unmarshal,
// eliminando string-scraping com regex frágil.
package rules

import (
	"context"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/fortvna/radiant-norma/backend/internal/bacen"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/doc4111"
	"github.com/fortvna/radiant-norma/backend/internal/drsac"
)

// ============================================================
// Helpers — extrair SCR data do 3040 via typed unmarshal
// ============================================================

// scrDataFrom3040 extrai map de IPOC → SCRData do XML do 3040.
// Agora usa bacen.Parse3040 (xml.Unmarshal) em vez de token-scanning.
func scrDataFrom3040(xml3040 string) map[string]drsac.SCRData {
	doc, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return make(map[string]drsac.SCRData)
	}

	result := make(map[string]drsac.SCRData)
	for _, a := range doc.Agregadas {
		if a.IPOC == "" {
			continue
		}
		result[a.IPOC] = drsac.SCRData{
			Saldo:      a.Saldo,
			CNAE:       "",
			HasCliente: a.CNPJSCR != "",
		}
	}
	return result
}

// extractTVMTotal3040 extrai o saldo total de TVM do 3040 via typed struct.
// Usa xml.Unmarshal para encontrar <TVM> dentro do Doc3040.
func extractTVMTotal3040(xml3040 string) string {
	// xml.Unmarshal direto não preserva TVM que não está no Doc3040 struct.
	// Fallback: token-based scan para <TVM>...</TVM>.
	type tvrWrapper struct {
		XMLName xml.Name `xml:"Doc3040"`
		TVM     struct {
			Saldo []string `xml:"Saldo"`
		} `xml:"TVM"`
	}
	var w tvrWrapper
	if err := xml.Unmarshal([]byte(xml3040), &w); err != nil {
		return ""
	}
	if len(w.TVM.Saldo) == 0 {
		return ""
	}
	// Retorna o último saldo (tipicamente o total).
	return w.TVM.Saldo[len(w.TVM.Saldo)-1]
}

// ============================================================
// 4111 structural cross-doc rules — usam bacen.Doc3040 via Parse3040
// ============================================================

// XD-4111-01 — CNPJ do 4111 deve bater com CNPJ do 3040 (mesmo IF).
type XD4111CNPJConsistente struct{}

func (XD4111CNPJConsistente) Code() string { return "XD-4111-01" }
func (XD4111CNPJConsistente) Description() string {
	return "CNPJ do 4111 deve ser consistente com CNPJ do 3040 (mesmo IF)"
}
func (XD4111CNPJConsistente) Severity() string { return "E" }
func (XD4111CNPJConsistente) RequiredDocs() []string {
	return []string{"4111", "3040"}
}
func (XD4111CNPJConsistente) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	if xml4111 == "" || xml3040 == "" {
		return nil
	}

	cnpj4111 := extractRootAttr(xml4111, "cnpj")
	if cnpj4111 == "" {
		return nil
	}

	// Usa bacen.Parse3040 (typed unmarshal) em vez de regex.
	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil // parsing failure → skip
	}

	// CNPJ pode vir em diferentes tamanhos (8 ou 14 dígitos).
	// Normaliza para os 8 primeiros dígitos (raiz) para comparação.
	raiz4111 := cnpj4111
	if len(raiz4111) > 8 {
		raiz4111 = raiz4111[:8]
	}
	raiz3040 := doc3040.CNPJ
	if len(raiz3040) > 8 {
		raiz3040 = raiz3040[:8]
	}

	if raiz4111 != raiz3040 {
		return crossdoc.NewError("XD-4111-01", "E",
			fmt.Sprintf("CNPJ 4111=%s difere do CNPJ 3040=%s (raiz 8 dígitos)",
				cnpj4111, doc3040.CNPJ))
	}
	return nil
}

// extractRootAttr extrai atributo do elemento root de um XML.
// Case-insensitive: aceita dataBase=, DataBase=, DATABASE= etc.
func extractRootAttr(xmlContent, attrName string) string {
	// (?i) = case-insensitive.
	re := regexp.MustCompile(`(?i)<[\w:]+[^>]*\s` + regexp.QuoteMeta(attrName) + `="([^"]*)"`)
	m := re.FindStringSubmatch(xmlContent)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// XD-4111-02 — Total de clientes no 4111 deve ser consistente com ops no 3040 (±10%).
type XD4111TotalClientesvsOps struct{}

func (XD4111TotalClientesvsOps) Code() string { return "XD-4111-02" }
func (XD4111TotalClientesvsOps) Description() string {
	return "Total de clientes no 4111 deve ser próximo (±10%) do total de operações no 3040"
}
func (XD4111TotalClientesvsOps) Severity() string { return "A" }
func (XD4111TotalClientesvsOps) RequiredDocs() []string {
	return []string{"4111", "3040"}
}
func (XD4111TotalClientesvsOps) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	if xml4111 == "" || xml3040 == "" {
		return nil
	}

	// 4111: usa doc4111.ParseFromBytes (typed).
	d4111, err := doc4111.ParseFromBytes([]byte(xml4111))
	if err != nil {
		return nil
	}

	// 3040: usa bacen.Parse3040 (typed).
	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}

	clients4111 := doc4111.ExtractQtdTotal(d4111)
	ops3040 := doc3040.QtdOpTotal()

	if clients4111 == 0 || ops3040 == 0 {
		return nil
	}

	diff := clients4111 - ops3040
	if diff < 0 {
		diff = -diff
	}
	ratio := diff / clients4111
	if ratio > 0.10 {
		return crossdoc.NewError("XD-4111-02", "A",
			fmt.Sprintf("discrepância %.1f%% entre clientes 4111 (%.0f) e ops 3040 (%.0f) — tol. 10%%",
				ratio*100, clients4111, ops3040))
	}
	return nil
}

// XD-4111-03 — Clientes inadimplentes no 4111 devem ter correspondência no 3040.
type XD4111Inadimplentesvs3040 struct{}

func (XD4111Inadimplentesvs3040) Code() string { return "XD-4111-03" }
func (XD4111Inadimplentesvs3040) Description() string {
	return "Clientes com indicação de inadimplência no 4111 devem constar no 3040 com v150>0"
}
func (XD4111Inadimplentesvs3040) Severity() string { return "A" }
func (XD4111Inadimplentesvs3040) RequiredDocs() []string {
	return []string{"4111", "3040"}
}
func (XD4111Inadimplentesvs3040) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	if xml4111 == "" || xml3040 == "" {
		return nil
	}

	d4111, err := doc4111.ParseFromBytes([]byte(xml4111))
	if err != nil {
		return nil
	}

	if !doc4111.HasModalidadeInadimplente(d4111) {
		return nil // sem inadimplentes → regra não se aplica
	}

	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}

	if doc3040.CountV150Gt0() == 0 {
		return crossdoc.NewError("XD-4111-03", "A",
			"4111 reporta clientes inadimplentes mas 3040 não tem v150>0")
	}
	return nil
}

// XD-4111-04 — Data-base do 4111 deve bater com data-base do 3040.
type XD4111DataBaseConsistente struct{}

func (XD4111DataBaseConsistente) Code() string { return "XD-4111-04" }
func (XD4111DataBaseConsistente) Description() string {
	return "Data-base do 4111 deve ser igual à data-base do 3040"
}
func (XD4111DataBaseConsistente) Severity() string { return "E" }
func (XD4111DataBaseConsistente) RequiredDocs() []string {
	return []string{"4111", "3040"}
}
func (XD4111DataBaseConsistente) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	if xml4111 == "" || xml3040 == "" {
		return nil
	}

	// Extrai dataBase do root de cada documento (atributo, case-insensitive).
	db4111 := extractRootAttr(xml4111, "dataBase")
	if db4111 == "" {
		return nil
	}
	db3040 := extractRootAttr(xml3040, "dataBase")
	if db3040 == "" {
		return nil
	}

	// Normaliza: ambos os formatos usam YYYY-MM-DD ou YYYY-MM.
	// Extrai o prefixo YYYY-MM para comparação.
	normalizeYM := func(s string) string {
		if len(s) >= 7 {
			return s[:7]
		}
		return s
	}

	if normalizeYM(db4111) != normalizeYM(db3040) {
		return crossdoc.NewError("XD-4111-04", "E",
			fmt.Sprintf("dataBase 4111=%s difere da dataBase 3040=%s", db4111, db3040))
	}
	return nil
}

// XD-4111-05 — 4111 zerado (sem clientes) deve ser consistente com 3040 zerado.
type XD4111Zeradovs3040 struct{}

func (XD4111Zeradovs3040) Code() string { return "XD-4111-05" }
func (XD4111Zeradovs3040) Description() string {
	return "Se 4111 é zerado (sem clientes), 3040 também deve não ter operações ativas"
}
func (XD4111Zeradovs3040) Severity() string { return "A" }
func (XD4111Zeradovs3040) RequiredDocs() []string {
	return []string{"4111", "3040"}
}
func (XD4111Zeradovs3040) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	if xml4111 == "" || xml3040 == "" {
		return nil
	}

	d4111, err := doc4111.ParseFromBytes([]byte(xml4111))
	if err != nil {
		return nil
	}
	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}

	clients4111 := doc4111.ExtractQtdTotal(d4111)
	ops3040 := doc3040.QtdOpTotal()

	if clients4111 == 0 && ops3040 > 0 {
		return crossdoc.NewError("XD-4111-05", "A",
			fmt.Sprintf("4111 reportado zerado (0 clientes) mas 3040 tem %.0f operações — verificar",
				ops3040))
	}
	return nil
}

// ============================================================
// DRSAC adapter wrappers — implementam crossdoc.CrossDocRule
// usando drsac.ValidateCrossRefs + scrDataFrom3040.
//
// Estes adapters permitem que as regras XD-DR01~08 (definidas
// no package drsac) sejam registradas no crossdoc.Registry.
//
// Sprint 72: cada adapter filtra os resultados de ValidateCrossRefs
// pelo seu próprio código.
// ============================================================

// applyDRSAC executa ValidateCrossRefs e retorna erro para um código específico.
func applyDRSAC(ctx context.Context, docs *crossdoc.DocSet, ruleCode string) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}
	doc2030, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}
	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc2030, scrData)
	for _, r := range results {
		if r.Code == ruleCode {
			return crossdoc.NewError(r.Code, r.Severity, r.Message)
		}
	}
	return nil
}

// XDDR01IPOCExistsInSCR — XD-DR01.
type XDDR01IPOCExistsInSCR struct{}

func (XDDR01IPOCExistsInSCR) Code() string { return "XD-DR01" }
func (XDDR01IPOCExistsInSCR) Description() string {
	return "IPOC de operação no DRSAC deve existir no SCR (3040)"
}
func (XDDR01IPOCExistsInSCR) Severity() string       { return "E" }
func (XDDR01IPOCExistsInSCR) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR01IPOCExistsInSCR) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR01")
}

// XDDR02SaldoConsistente — XD-DR02.
type XDDR02SaldoConsistente struct{}

func (XDDR02SaldoConsistente) Code() string { return "XD-DR02" }
func (XDDR02SaldoConsistente) Description() string {
	return "Saldo DRSAC diverge mais de 10%% do saldo SCR para mesmo IPOC"
}
func (XDDR02SaldoConsistente) Severity() string       { return "A" }
func (XDDR02SaldoConsistente) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR02SaldoConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR02")
}

// XDDR03ClienteExisteNoSCR — XD-DR03.
type XDDR03ClienteExisteNoSCR struct{}

func (XDDR03ClienteExisteNoSCR) Code() string { return "XD-DR03" }
func (XDDR03ClienteExisteNoSCR) Description() string {
	return "Cliente do DRSAC não encontrado no SCR para a mesma data-base"
}
func (XDDR03ClienteExisteNoSCR) Severity() string       { return "E" }
func (XDDR03ClienteExisteNoSCR) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR03ClienteExisteNoSCR) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR03")
}

// XDDR04SetorCNAEConsistente — XD-DR04.
type XDDR04SetorCNAEConsistente struct{}

func (XDDR04SetorCNAEConsistente) Code() string { return "XD-DR04" }
func (XDDR04SetorCNAEConsistente) Description() string {
	return "Setor CNAE no DRSAC deve ser consistente com classificação no SCR"
}
func (XDDR04SetorCNAEConsistente) Severity() string       { return "A" }
func (XDDR04SetorCNAEConsistente) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR04SetorCNAEConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR04")
}

// XDDR05RiscoSocialAlto — XD-DR05.
type XDDR05RiscoSocialAlto struct{}

func (XDDR05RiscoSocialAlto) Code() string { return "XD-DR05" }
func (XDDR05RiscoSocialAlto) Description() string {
	return "Alto risco social no DRSAC deve ter flag correspondente no SCR"
}
func (XDDR05RiscoSocialAlto) Severity() string       { return "A" }
func (XDDR05RiscoSocialAlto) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR05RiscoSocialAlto) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR05")
}

// XDDR06RiscoAmbiental — XD-DR06.
type XDDR06RiscoAmbiental struct{}

func (XDDR06RiscoAmbiental) Code() string { return "XD-DR06" }
func (XDDR06RiscoAmbiental) Description() string {
	return "Risco ambiental no DRSAC deve constar no SCR"
}
func (XDDR06RiscoAmbiental) Severity() string       { return "A" }
func (XDDR06RiscoAmbiental) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR06RiscoAmbiental) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR06")
}

// XDDR07TotalTVMConsistente — XD-DR07.
type XDDR07TotalTVMConsistente struct{}

func (XDDR07TotalTVMConsistente) Code() string { return "XD-DR07" }
func (XDDR07TotalTVMConsistente) Description() string {
	return "Total de exposição TVM no DRSAC deve ser consistente com SCR"
}
func (XDDR07TotalTVMConsistente) Severity() string       { return "A" }
func (XDDR07TotalTVMConsistente) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR07TotalTVMConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR07")
}

// XDDR08ContribPositivaGreen — XD-DR08.
type XDDR08ContribPositivaGreen struct{}

func (XDDR08ContribPositivaGreen) Code() string { return "XD-DR08" }
func (XDDR08ContribPositivaGreen) Description() string {
	return "Contribuição positiva sem instrumento verde registrado no SCR"
}
func (XDDR08ContribPositivaGreen) Severity() string       { return "I" }
func (XDDR08ContribPositivaGreen) RequiredDocs() []string { return []string{"2030", "3040"} }
func (XDDR08ContribPositivaGreen) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	return applyDRSAC(ctx, docs, "XD-DR08")
}

var (
	_ = drsac.ParseFromBytes // used implicitly via adapters
	_ = doc4111.ParseFromBytes
	_ = bacen.Parse3040
	_ = crossdoc.ExtractTextBetween
	_ = crossdoc.ExtractSumOfTag
	_ = crossdoc.CountTag
	_ = strings.TrimSpace
)
