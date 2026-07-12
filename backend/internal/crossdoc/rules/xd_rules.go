// Sprint 52 v3.34.33: Regras cross-doc XD02, XD03, XD06–XD12.
//
// Contexto:
//   XD02: 3040 ↔ 3050 (total de operações/participantes consistente)
//   XD03: 2160 ↔ 2170 (LCR ≥ 100% consistência com NSFR)
//   XD06: 3050 ↔ 2160 (APR em LCR — Operating Credit vs Liquidity)
//   XD07: 3040 ↔ 4111 ↔ 3050 (triangulação completa)
//   XD08: 2061 ↔ 2070 (limites operacionais vs requerimento capital)
//   XD09: 2160 ↔ 3040 (liquidez × risco de crédito)
//   XD10: DRSAC ↔ 3040 (ESG score × taxa de inadimplência)
//   XD11: DRSAC ↔ 4111 (operações ESG-classified)
//   XD12: Consistência de data-base entre todos CADOCs do envío
package rules

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/fortvna/radiant-norma/backend/internal/bacen"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/doc4111"
	"github.com/fortvna/radiant-norma/backend/internal/drsac"
)

// ============================================================
// XD02 — 3040 ↔ 3050: total de operações/participantes
// ============================================================

// XD02TotalOperacoes3040vs3050 verifica que a quantidade total de
// operações ativas no 3040 é aproximadamente consistente (±20%) com
// o total de operações no 3050 (somando SldCarAtiva de todas
// submodalidades em CRDLivre).
type XD02TotalOperacoes3040vs3050 struct{}

func (XD02TotalOperacoes3040vs3050) Code() string          { return "XD02" }
func (XD02TotalOperacoes3040vs3050) Description() string {
	return "Total de operações 3040 deve ser consistente (±20%) com 3050"
}
func (XD02TotalOperacoes3040vs3050) Severity() string { return "A" }
func (XD02TotalOperacoes3040vs3050) RequiredDocs() []string {
	return []string{"3040", "3050"}
}

func (XD02TotalOperacoes3040vs3050) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml3040 := docs.Get("3040")
	xml3050 := docs.Get("3050")
	if xml3040 == "" || xml3050 == "" {
		return nil
	}

	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}
	doc3050, err := bacen.Parse3050([]byte(xml3050))
	if err != nil {
		return nil
	}
	ops3040 := doc3040.QtdOpTotal()
	ops3050 := extract3050TotalOperations(doc3050)

	if ops3040 == 0 || ops3050 == 0 {
		return nil
	}
	ratio := math.Abs(ops3040 - ops3050) / math.Max(ops3040, ops3050)
	if ratio > 0.20 {
		return crossdoc.NewError("XD02", "A",
			fmt.Sprintf("discrepância %.0f%% entre ops 3040 (%.0f) e 3050 (%.0f) — tol. 20%%",
				ratio*100, ops3040, ops3050))
	}
	return nil
}

// extract3050TotalOperations soma SldCarAtiva de todas as submodalidades
// CRDLivre (Pessoa Jurídica + Física, Flu/Pre/Vc/Ind).
func extract3050TotalOperations(doc *bacen.DocTXB) float64 {
	var total float64
	addSaldo := func(blocks []bacen.SubModalidade) {
		for _, b := range blocks {
			v, _ := strconv.ParseFloat(b.SldCarAtiva, 64)
			total += v
		}
	}
	addSaldo(doc.Referencia.Diario.CRDLivre.PesJuridica.Flu)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesJuridica.Pre)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesJuridica.Vc)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesJuridica.Ind)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesFisica.Flu)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesFisica.Pre)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesFisica.Vc)
	addSaldo(doc.Referencia.Diario.CRDLivre.PesFisica.Ind)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesJuridica.Flu)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesJuridica.Pre)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesJuridica.Vc)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesJuridica.Ind)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesFisica.Flu)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesFisica.Pre)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesFisica.Vc)
	addSaldo(doc.Referencia.Mensal.CRDLivre.PesFisica.Ind)
	return total
}

// ============================================================
// XD03 — 2160 ↔ 2170: LCR vs NSFR consistência
// ============================================================

// XD03LCRvsNSFR verifica que o ratio LCR (2160) ≥ 100% E que o
// NSFR (2170) ≥ 100% — ambos são requisitos regulatórios.
// Além disso, valida consistência de data-base entre os dois.
type XD03LCRvsNSFR struct{}

func (XD03LCRvsNSFR) Code() string          { return "XD03" }
func (XD03LCRvsNSFR) Description() string {
	return "LCR (2160) ≥ 100%% E NSFR (2170) ≥ 100%% — consistência de data-base"
}
func (XD03LCRvsNSFR) Severity() string { return "E" }
func (XD03LCRvsNSFR) RequiredDocs() []string {
	return []string{"2160", "2170"}
}

func (XD03LCRvsNSFR) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml2160 := docs.Get("2160")
	xml2170 := docs.Get("2170")
	if xml2160 == "" || xml2170 == "" {
		return nil
	}

	doc2160, err := bacen.Parse2160([]byte(xml2160))
	if err != nil {
		return nil
	}
	doc2170, err := bacen.Parse2170([]byte(xml2170))
	if err != nil {
		return nil
	}

	// Consistência de data-base.
	if doc2160.DataBase != "" && doc2170.DataBase != "" &&
		normalizeDate(doc2160.DataBase) != normalizeDate(doc2170.DataBase) {
		return crossdoc.NewError("XD03", "E",
			fmt.Sprintf("dataBase 2160=%s difere de 2170=%s",
				doc2160.DataBase, doc2170.DataBase))
	}

	// LCR ≥ 100%.
	lcr := parseRatio(doc2160.LCRRatio.Valor)
	if lcr >= 0 && lcr < 1.0 {
		return crossdoc.NewError("XD03", "E",
			fmt.Sprintf("LCR=%.2f%% < 100%% (mínimo regulatório)", lcr*100))
	}

	// NSFR ≥ 100%.
	nsfr := parseRatio(doc2170.NSFRRatio.Valor)
	if nsfr >= 0 && nsfr < 1.0 {
		return crossdoc.NewError("XD03", "E",
			fmt.Sprintf("NSFR=%.2f%% < 100%% (mínimo regulatório)", nsfr*100))
	}
	return nil
}

// ============================================================
// XD06 — 3050 ↔ 2160: APR em LCR (taxa média × saldo vs HQLA)
// ============================================================

// XD06APRemLCR verifica que o total de APR (Average Prime Rate,
// proxies via saldo ativo em 3050) é consistente com o valor do
// HQLA reportado no 2160 — grandes volumes de crédito devem
// gerar estoques de liquidez proporcionais.
type XD06APRemLCR struct{}

func (XD06APRemLCR) Code() string          { return "XD06" }
func (XD06APRemLCR) Description() string {
	return "Volume de crédito ativo (3050) deve ser consistente com HQLA (2160)"
}
func (XD06APRemLCR) Severity() string { return "A" }
func (XD06APRemLCR) RequiredDocs() []string {
	return []string{"3050", "2160"}
}

func (XD06APRemLCR) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml3050 := docs.Get("3050")
	xml2160 := docs.Get("2160")
	if xml3050 == "" || xml2160 == "" {
		return nil
	}

	doc3050, err := bacen.Parse3050([]byte(xml3050))
	if err != nil {
		return nil
	}
	doc2160, err := bacen.Parse2160([]byte(xml2160))
	if err != nil {
		return nil
	}

	credito3050 := extract3050TotalOperations(doc3050)
	hqla2160, _ := strconv.ParseFloat(doc2160.HQLA.Valor, 64)

	// Sanity: se há Crédito > 0 mas HQLA = 0, alerta.
	if credito3050 > 0 && hqla2160 == 0 {
		return crossdoc.NewError("XD06", "A",
			fmt.Sprintf("crédito ativo %.0f na 3050 mas HQLA=0 na 2160 — verificar cobertura de liquidez",
				credito3050))
	}
	return nil
}

// ============================================================
// XD07 — 3040 ↔ 4111 ↔ 3050: triangulação completa
// ============================================================

// XD07Triangulacao304041113050 verifica que não há contradição
// tripla: se 4111 reporta clientes inadimplentes, 3040 deve ter
// v150>0 E 3050 deve reportar variação negativa de saldo.
type XD07Triangulacao304041113050 struct{}

func (XD07Triangulacao304041113050) Code() string          { return "XD07" }
func (XD07Triangulacao304041113050) Description() string {
	return "Triangulação 3040↔4111↔3050: inadimplência consistente"
}
func (XD07Triangulacao304041113050) Severity() string { return "E" }
func (XD07Triangulacao304041113050) RequiredDocs() []string {
	return []string{"3040", "4111", "3050"}
}

func (XD07Triangulacao304041113050) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml4111 := docs.Get("4111")
	xml3040 := docs.Get("3040")
	xml3050 := docs.Get("3050")
	if xml4111 == "" || xml3040 == "" || xml3050 == "" {
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

	// Regra 1: se 4111 tem inadimplentes, 3040 deve ter v150>0.
	if doc4111.HasModalidadeInadimplente(d4111) && doc3040.CountV150Gt0() == 0 {
		return crossdoc.NewError("XD07", "E",
			"4111 reporta inadimplentes mas 3040 não tem v150>0")
	}

	// Regra 2: se 4111 tem inadimplentes, saldo mensal ≤ saldo diário.
	doc3050, err := bacen.Parse3050([]byte(xml3050))
	if err != nil {
		return nil
	}
	if doc4111.HasModalidadeInadimplente(d4111) {
		diario := extract3050TotalSaldo(doc3050, true)
		mensal := extract3050TotalSaldo(doc3050, false)
		if mensal > diario && mensal > 0 && diario > 0 {
			return crossdoc.NewError("XD07", "A",
				"4111 reporta inadimplentes mas saldo mensal > diário na 3050 — verificar consistência")
		}
	}
	return nil
}

// extract3050TotalSaldo retorna total SldCarAtiva de uma frequência.
func extract3050TotalSaldo(doc *bacen.DocTXB, diario bool) float64 {
	var total float64
	var bloco bacen.CRDLivre
	if diario {
		bloco = doc.Referencia.Diario.CRDLivre
	} else {
		bloco = doc.Referencia.Mensal.CRDLivre
	}
	addSaldo := func(blocks []bacen.SubModalidade) {
		for _, b := range blocks {
			v, _ := strconv.ParseFloat(b.SldCarAtiva, 64)
			total += v
		}
	}
	addSaldo(bloco.PesJuridica.Flu)
	addSaldo(bloco.PesJuridica.Pre)
	addSaldo(bloco.PesJuridica.Vc)
	addSaldo(bloco.PesJuridica.Ind)
	addSaldo(bloco.PesFisica.Flu)
	addSaldo(bloco.PesFisica.Pre)
	addSaldo(bloco.PesFisica.Vc)
	addSaldo(bloco.PesFisica.Ind)
	return total
}

// ============================================================
// XD08 — 2061 ↔ 2070: limites operacionais vs requerimento capital
// ============================================================

// XD08LimitesvsCapital verifica que o LimiteTotal (2061) é
// consistente com o total de DDR (2070). Relação: se há requerimento
// DDR > 0, limites devem ser reportados (> 0).
type XD08LimitesvsCapital struct{}

func (XD08LimitesvsCapital) Code() string          { return "XD08" }
func (XD08LimitesvsCapital) Description() string {
	return "Limite operacional 2061 deve ser consistente com requerimento capital 2070"
}
func (XD08LimitesvsCapital) Severity() string { return "A" }
func (XD08LimitesvsCapital) RequiredDocs() []string {
	return []string{"2061", "2070"}
}

func (XD08LimitesvsCapital) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml2061 := docs.Get("2061")
	xml2070 := docs.Get("2070")
	if xml2061 == "" || xml2070 == "" {
		return nil
	}

	doc2061, err := bacen.Parse2061([]byte(xml2061))
	if err != nil {
		return nil
	}
	doc2070, err := bacen.Parse2070([]byte(xml2070))
	if err != nil {
		return nil
	}

	limiteTotal, _ := strconv.ParseFloat(doc2061.LimiteTotal.Valor, 64)
	var totalDDR float64
	for _, d := range doc2070.DDRs {
		v, _ := strconv.ParseFloat(d.Valor, 64)
		totalDDR += v
	}

	// Se há requerimento DDR > 0, limites devem ser reportados.
	if totalDDR > 0 && limiteTotal == 0 {
		return crossdoc.NewError("XD08", "A",
			fmt.Sprintf("DDR reporta requerimento %.2f mas LimiteTotal=0 no 2061", totalDDR))
	}

	// Consistência de data-base.
	if doc2061.DataBase != "" && doc2070.DataBase != "" &&
		normalizeDate(doc2061.DataBase) != normalizeDate(doc2070.DataBase) {
		return crossdoc.NewError("XD08", "E",
			fmt.Sprintf("dataBase 2061=%s difere de 2070=%s",
				doc2061.DataBase, doc2070.DataBase))
	}
	return nil
}

// ============================================================
// XD09 — 2160 ↔ 3040: liquidez × risco de crédito
// ============================================================

// XD09LiquidezvsRisco verifica que o HQLA (2160) é suficiente em
// relação ao total de exposição no 3040. Se exposição > 100M e
// HQLA = 0 → alerta.
type XD09LiquidezvsRisco struct{}

func (XD09LiquidezvsRisco) Code() string          { return "XD09" }
func (XD09LiquidezvsRisco) Description() string {
	return "HQLA (2160) deve ser positivo quando exposição crédito (3040) é significativa"
}
func (XD09LiquidezvsRisco) Severity() string { return "A" }
func (XD09LiquidezvsRisco) RequiredDocs() []string {
	return []string{"2160", "3040"}
}

func (XD09LiquidezvsRisco) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml2160 := docs.Get("2160")
	xml3040 := docs.Get("3040")
	if xml2160 == "" || xml3040 == "" {
		return nil
	}

	doc2160, err := bacen.Parse2160([]byte(xml2160))
	if err != nil {
		return nil
	}
	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}

	hqla, _ := strconv.ParseFloat(doc2160.HQLA.Valor, 64)
	exposicao3040 := extract3040TotalSaldo(doc3040)

	if exposicao3040 > 100_000_000 && hqla == 0 {
		return crossdoc.NewError("XD09", "A",
			fmt.Sprintf("exposição 3040=%.0f (>100M) mas HQLA=0 na 2160", exposicao3040))
	}
	return nil
}

// extract3040TotalSaldo soma saldos de todas as Agregadas no 3040.
func extract3040TotalSaldo(doc *bacen.Doc3040) float64 {
	var total float64
	for _, a := range doc.Agregadas {
		v, _ := strconv.ParseFloat(a.Saldo, 64)
		total += v
	}
	return total
}

// ============================================================
// XD10 — DRSAC ↔ 3040: ESG score × taxa de inadimplência
// ============================================================

// XD10ESGvsInadimplencia verifica que operações com classificação
// de alto risco social (av=01) ou ambiental (av=01/02) no DRSAC
// correspondem a registros no 3040. Usa ipocsDRSAC × scrData cross-check.
// Note: HasHighRiskFlag e HasCollateral em SCRData não são populados
// pelo helper scrDataFrom3040 (pré-existência) — esta regra checa
// presença de saldo como proxy.
type XD10ESGvsInadimplencia struct{}

func (XD10ESGvsInadimplencia) Code() string          { return "XD10" }
func (XD10ESGvsInadimplencia) Description() string {
	return "Alto risco ESG (DRSAC) deve corresponder a exposição no 3040"
}
func (XD10ESGvsInadimplencia) Severity() string { return "A" }
func (XD10ESGvsInadimplencia) RequiredDocs() []string {
	return []string{"2030", "3040"}
}

func (XD10ESGvsInadimplencia) Apply(_ context.Context, docs *crossdoc.DocSet) error {
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

	// Itera todas operações de crédito do DRSAC.
	for _, cl := range doc2030.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC == "" {
				continue
			}
			// Alto risco social (av=01) ou ambiental (av=01/02).
			highRisk := op.RiscSoc.Av == "01" || op.RiscAmb.Av == "01" || op.RiscAmb.Av == "02"
			if !highRisk {
				continue
			}
			scr, ok := scrData[op.IPOC]
			if !ok {
				continue
			}
			// Proxy: se SCR tem saldo, operação existe no SCR.
			// Alerta se alto risco ESG mas sem saldo no SCR.
			saldo, _ := strconv.ParseFloat(scr.Saldo, 64)
			if saldo == 0 {
				return crossdoc.NewError("XD10", "A",
					fmt.Sprintf("IPOC %s: alto risco ESG (%s/%s) sem saldo no SCR",
						op.IPOC, op.RiscSoc.Av, op.RiscAmb.Av))
			}
		}
	}
	return nil
}

// ============================================================
// XD11 — DRSAC ↔ 4111: operações ESG-classified
// ============================================================

// XD11ESGvs4111 verifica que operações classificadas como alto
// risco ambiental/social no DRSAC aparecem no 4111.
type XD11ESGvs4111 struct{}

func (XD11ESGvs4111) Code() string          { return "XD11" }
func (XD11ESGvs4111) Description() string {
	return "Operações com alto risco ESG (DRSAC) devem constar no 4111"
}
func (XD11ESGvs4111) Severity() string { return "A" }
func (XD11ESGvs4111) RequiredDocs() []string {
	return []string{"2030", "4111"}
}

func (XD11ESGvs4111) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml2030 := docs.Get("2030")
	xml4111 := docs.Get("4111")
	if xml2030 == "" || xml4111 == "" {
		return nil
	}

	doc2030, err := drsac.ParseFromBytes([]byte(xml2030))
	if err != nil {
		return nil
	}

	// Extrai CNPJs do 4111 (raiz 8 dígitos).
	cnpjs4111 := extractCNPJs4111([]byte(xml4111))
	if len(cnpjs4111) == 0 {
		return nil
	}

	for _, cl := range doc2030.Clientes {
		cnpj := strings.TrimSpace(cl.Ident)
		if cnpj == "" {
			continue
		}
		raiz := cnpj
		if len(raiz) > 8 {
			raiz = raiz[:8]
		}
		found := false
		for _, c := range cnpjs4111 {
			r := c
			if len(r) > 8 {
				r = r[:8]
			}
			if r == raiz {
				found = true
				break
			}
		}
		// Para cada operação do cliente com alto risco ESG.
		for _, op := range cl.ExpAtivos.ExpOperCred {
			highRisk := op.RiscSoc.Av == "01" || op.RiscAmb.Av == "01" || op.RiscAmb.Av == "02"
			if !highRisk {
				continue
			}
			if !found {
				return crossdoc.NewError("XD11", "A",
					fmt.Sprintf("IPOC %s: alto risco ESG sem correspondente CNPJ no 4111", op.IPOC))
			}
		}
	}
	return nil
}

// extractCNPJs4111 extrai CNPJs de clientes do 4111 (raiz 8 dígitos).
func extractCNPJs4111(data []byte) []string {
	re := regexp.MustCompile(`(?i)cnpj="(\d+)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	var cnpjs []string
	for _, m := range matches {
		if len(m) >= 2 {
			cnpjs = append(cnpjs, m[1])
		}
	}
	return cnpjs
}

// ============================================================
// XD12 — Consistência de data-base entre todos CADOCs
// ============================================================

// XD12DataBaseConsistente verifica que todos os CADOCs presentes
// no DocSet compartilham a mesma data-base. Diferenças de
// data-base são erros regulatórios graves.
type XD12DataBaseConsistente struct{}

func (XD12DataBaseConsistente) Code() string          { return "XD12" }
func (XD12DataBaseConsistente) Description() string {
	return "Todos os CADOCs devem ter a mesma data-base no mesmo envío"
}
func (XD12DataBaseConsistente) Severity() string { return "E" }
func (XD12DataBaseConsistente) RequiredDocs() []string {
	return []string{"2030", "3040", "3050"} // minimal set
}

func (XD12DataBaseConsistente) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	type dbEntry struct {
		cadoc string
		db    string
	}
	var entries []dbEntry

	for cadoc := range docs.Cadocs {
		xml := docs.Get(cadoc)
		if xml == "" {
			continue
		}
		db := extractRootAttr(xml, "dataBase")
		if db == "" {
			continue
		}
		entries = append(entries, dbEntry{cadoc: cadoc, db: db})
	}

	if len(entries) < 2 {
		return nil
	}

	ref := entries[0]
	for _, e := range entries[1:] {
		if normalizeDate(e.db) != normalizeDate(ref.db) {
			return crossdoc.NewError("XD12", "E",
				fmt.Sprintf("dataBase不一致: %s=%s, %s=%s",
					ref.cadoc, ref.db, e.cadoc, e.db))
		}
	}
	return nil
}

// ============================================================
// Helpers
// ============================================================

// normalizeDate returns YYYY-MM prefix for comparison.
func normalizeDate(s string) string {
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}

// parseRatio converte string percentual "123.45%" → 1.2345.
func parseRatio(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	v, _ := strconv.ParseFloat(s, 64)
	if v > 10 {
		// Assume it's a percentage value (e.g., "150%" stored as "150")
		v = v / 100
	}
	return v
}
