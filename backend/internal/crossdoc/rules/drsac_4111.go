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
package rules

import (
	"context"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/doc4111"
	"github.com/fortvna/radiant-norma/backend/internal/drsac"
)

// ============================================================
// Helpers — extrair SCR data do 3040 (DocSet → map[string]SCRData)
// ============================================================

// scrDataFrom3040 extrai map de IPOC → SCRData do XML do 3040.
func scrDataFrom3040(xml3040 string) map[string]drsac.SCRData {
	result := make(map[string]drsac.SCRData)

	// Estrutura 3040: <Agreg><IPOC>...</IPOC><CNPJ>...</CNPJ><Saldo>...</Saldo>...
	decoder := xml.NewDecoder(strings.NewReader(xml3040))
	var currentIPOC, currentCNPJ, currentSaldo string
	var hasHighRisk, hasCollateral, isGreen bool
	var inAgreg bool

	for {
		tok, err := decoder.Token()
		if err != nil || tok == nil {
			break
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "Agreg" {
				inAgreg = true
				currentIPOC, currentCNPJ, currentSaldo = "", "", ""
				hasHighRisk, hasCollateral, isGreen = false, false, false
			}
			if inAgreg {
				switch tok.Name.Local {
				case "IPOC":
					if t2, _ := decoder.Token(); t2 != nil {
						if cd, ok := t2.(xml.CharData); ok {
							currentIPOC = strings.TrimSpace(string(cd))
						}
					}
				case "CNPJ":
					if t2, _ := decoder.Token(); t2 != nil {
						if cd, ok := t2.(xml.CharData); ok {
							currentCNPJ = strings.TrimSpace(string(cd))
						}
					}
				case "Saldo":
					if t2, _ := decoder.Token(); t2 != nil {
						if cd, ok := t2.(xml.CharData); ok {
							currentSaldo = strings.TrimSpace(string(cd))
						}
					}
				}
			}
		case xml.EndElement:
			if tok.Name.Local == "Agreg" && inAgreg {
				inAgreg = false
				if currentIPOC != "" {
					result[currentIPOC] = drsac.SCRData{
						Saldo:             currentSaldo,
						CNAE:              "",
						HasCliente:        currentCNPJ != "",
						HasHighRiskFlag:   hasHighRisk,
						HasCollateral:     hasCollateral,
						IsGreenInstrument: isGreen,
					}
				}
			}
		}
	}

	// Extrai total TVM do 3040 (tag especial)
	tvmTotal := extractTVMTotal3040(xml3040)
	if tvmTotal != "" {
		result["_TVM_TOTAL"] = drsac.SCRData{Saldo: tvmTotal}
	}

	return result
}

// extractTVMTotal3040 extrai o saldo total de TVM do 3040.
func extractTVMTotal3040(xml3040 string) string {
	// Procura tag <Saldo> dentro de <Agreg type="TVM"> ou similar
	// Usa pattern simples baseado em texto
	openTag := "<TVM>"
	closeTag := "</TVM>"
	idx := strings.Index(xml3040, openTag)
	if idx == -1 {
		return ""
	}
	end := strings.Index(xml3040[idx:], closeTag)
	if end == -1 {
		return ""
	}
	content := xml3040[idx+len(openTag) : idx+end]
	// Extrai último Saldo encontrado
	saldo := crossdoc.ExtractTextBetween(content, "Saldo")
	return saldo
}

// ============================================================
// DRSAC adapters — wrappers em torno de drsac.ValidateCrossRefs
// ============================================================

// XD-DR01 — IPOC de operação no DRSAC deve existir no SCR (3040).
type XDDR01IPOCExistsInSCR struct{}

func (XDDR01IPOCExistsInSCR) Code() string { return "XD-DR01" }
func (XDDR01IPOCExistsInSCR) Description() string {
	return "IPOC de operação no DRSAC (2030) deve existir no SCR (3040)"
}
func (XDDR01IPOCExistsInSCR) Severity() string { return "E" }
func (XDDR01IPOCExistsInSCR) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR01IPOCExistsInSCR) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return crossdoc.NewError("XD-DR01", "E", "documentos 2030 ou 3040 ausentes")
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return crossdoc.NewError("XD-DR01", "E", "falha ao parsear 2030: "+err.Error())
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	for _, r := range results {
		if r.Code == "XD-DR01" {
			return crossdoc.NewError(r.Code, r.Severity, r.Message)
		}
	}
	return nil
}

// XD-DR02 — Saldo reportado no DRSAC deve ser consistente com SCR.
type XDDR02SaldoConsistente struct{}

func (XDDR02SaldoConsistente) Code() string { return "XD-DR02" }
func (XDDR02SaldoConsistente) Description() string {
	return "Saldo DRSAC (2030) diverge mais de 10% do saldo SCR (3040) para mesmo IPOC"
}
func (XDDR02SaldoConsistente) Severity() string { return "A" }
func (XDDR02SaldoConsistente) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR02SaldoConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil // skip silently if docs missing
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	var msgs []string
	for _, r := range results {
		if r.Code == "XD-DR02" {
			msgs = append(msgs, r.Message)
		}
	}
	if len(msgs) > 0 {
		return crossdoc.NewError("XD-DR02", "A", strings.Join(msgs, "; "))
	}
	return nil
}

// XD-DR03 — CNPJ do cliente no DRSAC deve existir no SCR.
type XDDR03ClienteExisteNoSCR struct{}

func (XDDR03ClienteExisteNoSCR) Code() string { return "XD-DR03" }
func (XDDR03ClienteExisteNoSCR) Description() string {
	return "Cliente do DRSAC (2030) não encontrado no SCR (3040) para a mesma data-base"
}
func (XDDR03ClienteExisteNoSCR) Severity() string { return "E" }
func (XDDR03ClienteExisteNoSCR) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR03ClienteExisteNoSCR) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	for _, r := range results {
		if r.Code == "XD-DR03" {
			return crossdoc.NewError(r.Code, r.Severity, r.Message)
		}
	}
	return nil
}

// XD-DR04 — Setor CNAE no DRSAC deve ser consistente com SCR.
type XDDR04SetorCNAEConsistente struct{}

func (XDDR04SetorCNAEConsistente) Code() string { return "XD-DR04" }
func (XDDR04SetorCNAEConsistente) Description() string {
	return "CNAE do setor DRSAC (2030) diverge da classificação no SCR (3040)"
}
func (XDDR04SetorCNAEConsistente) Severity() string { return "A" }
func (XDDR04SetorCNAEConsistente) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR04SetorCNAEConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	var msgs []string
	for _, r := range results {
		if r.Code == "XD-DR04" {
			msgs = append(msgs, r.Message)
		}
	}
	if len(msgs) > 0 {
		return crossdoc.NewError("XD-DR04", "A", strings.Join(msgs, "; "))
	}
	return nil
}

// XD-DR05 — Alto risco social (av=01) no DRSAC deve ter flag no SCR.
type XDDR05RiscoSocialAlto struct{}

func (XDDR05RiscoSocialAlto) Code() string { return "XD-DR05" }
func (XDDR05RiscoSocialAlto) Description() string {
	return "Operação com risco social alto (av=01) no DRSAC sem flag correspondente no SCR"
}
func (XDDR05RiscoSocialAlto) Severity() string { return "A" }
func (XDDR05RiscoSocialAlto) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR05RiscoSocialAlto) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	var msgs []string
	for _, r := range results {
		if r.Code == "XD-DR05" {
			msgs = append(msgs, r.Message)
		}
	}
	if len(msgs) > 0 {
		return crossdoc.NewError("XD-DR05", "A", strings.Join(msgs, "; "))
	}
	return nil
}

// XD-DR06 — Risco ambiental (av=01 ou 02) no DRSAC deve constar no SCR.
type XDDR06RiscoAmbiental struct{}

func (XDDR06RiscoAmbiental) Code() string { return "XD-DR06" }
func (XDDR06RiscoAmbiental) Description() string {
	return "Operação com risco ambiental no DRSAC sem menção no SCR"
}
func (XDDR06RiscoAmbiental) Severity() string { return "A" }
func (XDDR06RiscoAmbiental) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR06RiscoAmbiental) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	var msgs []string
	for _, r := range results {
		if r.Code == "XD-DR06" {
			msgs = append(msgs, r.Message)
		}
	}
	if len(msgs) > 0 {
		return crossdoc.NewError("XD-DR06", "A", strings.Join(msgs, "; "))
	}
	return nil
}

// XD-DR07 — Total de exposição em TVM no DRSAC deve ser consistente com SCR.
type XDDR07TotalTVMConsistente struct{}

func (XDDR07TotalTVMConsistente) Code() string { return "XD-DR07" }
func (XDDR07TotalTVMConsistente) Description() string {
	return "Total de exposição TVM no DRSAC diverge mais de 15% do SCR"
}
func (XDDR07TotalTVMConsistente) Severity() string { return "A" }
func (XDDR07TotalTVMConsistente) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR07TotalTVMConsistente) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	for _, r := range results {
		if r.Code == "XD-DR07" {
			return crossdoc.NewError(r.Code, r.Severity, r.Message)
		}
	}
	return nil
}

// XD-DR08 — Contribuição positiva no DRSAC deve ter instrumento verde no SCR.
type XDDR08ContribPositivaGreen struct{}

func (XDDR08ContribPositivaGreen) Code() string { return "XD-DR08" }
func (XDDR08ContribPositivaGreen) Description() string {
	return "Operação com contribuição positiva sem instrumento verde registrado no SCR"
}
func (XDDR08ContribPositivaGreen) Severity() string { return "I" }
func (XDDR08ContribPositivaGreen) RequiredDocs() []string {
	return []string{"2030", "3040"}
}
func (XDDR08ContribPositivaGreen) Apply(ctx context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml3040 := docs.Get("3040")
	if xml2030 == "" || xml3040 == "" {
		return nil
	}

	doc, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	scrData := scrDataFrom3040(xml3040)
	results := drsac.ValidateCrossRefs(doc, scrData)

	var msgs []string
	for _, r := range results {
		if r.Code == "XD-DR08" {
			msgs = append(msgs, r.Message)
		}
	}
	if len(msgs) > 0 {
		return crossdoc.NewError("XD-DR08", "I", strings.Join(msgs, "; "))
	}
	return nil
}

// ============================================================
// 4111 structural cross-doc rules
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

	cnpj4111 := crossdoc.ExtractTextBetween(xml4111, "cnpj")
	if cnpj4111 == "" {
		return nil
	}

	// 3040 tem CNPJ no atributo do root element
	cnpj3040 := extractCNPJ3040(xml3040)
	if cnpj3040 == "" {
		return nil
	}

	if cnpj4111 != cnpj3040 {
		return crossdoc.NewError("XD-4111-01", "E",
			fmt.Sprintf("CNPJ 4111=%s difere do CNPJ 3040=%s", cnpj4111, cnpj3040))
	}
	return nil
}

// extractCNPJ3040 extrai o CNPJ do atributo root do 3040.
func extractCNPJ3040(xml3040 string) string {
	// <Documento3040 cnpj="12345678" ...
	re := regexp.MustCompile(`cnpj="(\d{8,14})"`)
	m := re.FindStringSubmatch(xml3040)
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

	clients4111 := crossdoc.ExtractSumOfTag(xml4111, "Cliente", "QtdCli")
	ops3040 := crossdoc.ExtractSumOfTag(xml3040, "Agreg", "QtdOp")

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
	if xml4111 == "" {
		return nil
	}

	// Conta clientes com indicação inadimplente
	inad4111 := countInadimplente4111(xml4111)
	if inad4111 == 0 {
		return nil // sem inadimplentes → regra não se aplica
	}

	// Conta registros com v150>0 no 3040
	xml3040 := docs.Get("3040")
	v150Count := countV1503040(xml3040)

	if v150Count == 0 && inad4111 > 0 {
		return crossdoc.NewError("XD-4111-03", "A",
			fmt.Sprintf("4111 reporta %d clientes inadimplentes mas 3040 não tem v150>0", inad4111))
	}
	return nil
}

// countInadimplente4111 conta clientes com indicação de inadimplência.
func countInadimplente4111(xml4111 string) int {
	count := 0
	decoder := xml.NewDecoder(strings.NewReader(xml4111))
	for {
		tok, err := decoder.Token()
		if err != nil || tok == nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "Modalidade" {
			var indicacao string
			for _, attr := range se.Attr {
				if attr.Name.Local == "indicacao" {
					indicacao = attr.Value
					break
				}
			}
			if indicacao == "S" || indicacao == "s" {
				count++
			}
		}
	}
	return count
}

// countV1503040 conta ocorrências no 3040 onde v150 > 0.
func countV1503040(xml3040 string) int {
	// v150 é valor vencido > 90 dias — conta tags <V150> com valor > 0
	count := 0
	idx := 0
	for {
		tag := "<V150>"
		i := strings.Index(xml3040[idx:], tag)
		if i == -1 {
			break
		}
		pos := idx + i + len(tag)
		end := strings.Index(xml3040[pos:], "</V150>")
		if end == -1 {
			break
		}
		val := strings.TrimSpace(xml3040[pos : pos+end])
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil && f > 0 {
			count++
		}
		idx = pos + end + 1
	}
	return count
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

	db4111 := crossdoc.ExtractTextBetween(xml4111, "dataBase")
	if db4111 == "" {
		return nil
	}

	db3040 := crossdoc.ExtractTextBetween(xml3040, "dataBase")
	if db3040 == "" {
		return nil
	}

	if db4111 != db3040 {
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

	clients4111 := crossdoc.ExtractSumOfTag(xml4111, "Cliente", "QtdCli")
	ops3040 := crossdoc.ExtractSumOfTag(xml3040, "Agreg", "QtdOp")

	// Se 4111 é zerado mas 3040 tem ops → possível inconsistência
	if clients4111 == 0 && ops3040 > 0 {
		return crossdoc.NewError("XD-4111-05", "A",
			fmt.Sprintf("4111 reportado zerado (0 clientes) mas 3040 tem %.0f operações — verificar",
				ops3040))
	}
	return nil
}

var (
	_ = drsac.ParseFromBytes // used implicitly via adapters
	_ = doc4111.ParseFromBytes
	_ = crossdoc.ExtractTextBetween
	_ = crossdoc.ExtractSumOfTag
	_ = crossdoc.CountTag
)
