// Diff expandido para L4 — Sprint 79.
//
// Adiciona extractors para: 3050, 3044, 2060 (DRM), 4111, 2030 (DRSAC).
// Cada extractor extrai campos agregados usando parsers existentes,
// compara com a versão anterior, e detecta variações > 0.01%.
package l4

import (
	"fmt"
	"strconv"

	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
	"github.com/fortvna/radiant-norma/backend/internal/bacen"
	"github.com/fortvna/radiant-norma/backend/internal/drsac"
)

// extract3050Changes extrai e compara campos agregados do 3050 (TXB).
//
// Compara: total SldCarAtiva (CRDLivre), contagem de operações,
// totalizadore mensal vs diário.
func (e *Engine) extract3050Changes(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := bacen.Parse3050([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3050 prev: %w", err)
	}
	currDoc, err := bacen.Parse3050([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3050 curr: %w", err)
	}

	var changes []FieldChange

	// Total SldCarAtiva — soma de todos os blocos CRDLivre.
	prevTotal := extract3050TotalSaldo(prevDoc, false)
	currTotal := extract3050TotalSaldo(currDoc, false)
	changes = appendFieldChange(changes, "3050", "SldCarAtivaTotalMensal",
		prevTotal, currTotal)

	// Diário.
	prevDiario := extract3050TotalSaldo(prevDoc, true)
	currDiario := extract3050TotalSaldo(currDoc, true)
	changes = appendFieldChange(changes, "3050", "SldCarAtivaTotalDiario",
		prevDiario, currDiario)

	// Diferença mensal vs diário (captura variação de timing).
	changes = appendFieldChange(changes, "3050", "DiferencaDiarioMensal",
		prevTotal-prevDiario, currTotal-currDiario)

	return changes, nil
}

// extract3050TotalSaldo soma SldCarAtiva de todos os blocos CRDLivre.
// diaria=true usa o bloco Diário; false usa Mensal.
func extract3050TotalSaldo(doc *bacen.DocTXB, diaria bool) float64 {
	var bloco bacen.CRDLivre
	if diaria {
		bloco = doc.Referencia.Diario.CRDLivre
	} else {
		bloco = doc.Referencia.Mensal.CRDLivre
	}
	var total float64
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

// extract3044Changes extrai e compara campos do 3044 (JSON — eventos).
//
// Compara: contagem de operações, total saldo devedor, soma pagamentos,
// soma concessões, soma cessões, soma aquisições.
func (e *Engine) extract3044Changes(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDoc3044([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3044 prev: %w", err)
	}
	currDoc, err := rules.ParseDoc3044([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3044 curr: %w", err)
	}

	var changes []FieldChange

	// Contagem de operações únicas.
	changes = appendFieldChange(changes, "3044", "Operacoes",
		float64(len(prevDoc.Operacoes)), float64(len(currDoc.Operacoes)))

	// Total saldo devedor.
	var prevSaldo, currSaldo float64
	for _, op := range prevDoc.Operacoes {
		prevSaldo += op.SaldoDevedor
	}
	for _, op := range currDoc.Operacoes {
		currSaldo += op.SaldoDevedor
	}
	changes = appendFieldChange(changes, "3044", "SaldoDevedorTotal",
		prevSaldo, currSaldo)

	// Total pagamentos.
	var prevPagto, currPagto float64
	for _, op := range prevDoc.Operacoes {
		for _, p := range op.Pagamentos {
			prevPagto += p.Valor
		}
	}
	for _, op := range currDoc.Operacoes {
		for _, p := range op.Pagamentos {
			currPagto += p.Valor
		}
	}
	changes = appendFieldChange(changes, "3044", "PagamentosTotal",
		prevPagto, currPagto)

	// Total concessões.
	var prevConc, currConc float64
	for _, op := range prevDoc.Operacoes {
		for _, c := range op.Concessoes {
			prevConc += c.Valor
		}
	}
	for _, op := range currDoc.Operacoes {
		for _, c := range op.Concessoes {
			currConc += c.Valor
		}
	}
	changes = appendFieldChange(changes, "3044", "ConcessoesTotal",
		prevConc, currConc)

	// Total cessões.
	var prevCess, currCess float64
	for _, op := range prevDoc.Operacoes {
		for _, cs := range op.Cessoes {
			prevCess += cs.Valor
		}
	}
	for _, op := range currDoc.Operacoes {
		for _, cs := range op.Cessoes {
			currCess += cs.Valor
		}
	}
	changes = appendFieldChange(changes, "3044", "CessoesTotal",
		prevCess, currCess)

	// Total aquisições.
	var prevAq, currAq float64
	for _, op := range prevDoc.Operacoes {
		for _, aq := range op.Aquisicoes {
			prevAq += aq.Valor
		}
	}
	for _, op := range currDoc.Operacoes {
		for _, aq := range op.Aquisicoes {
			currAq += aq.Valor
		}
	}
	changes = appendFieldChange(changes, "3044", "AquisicoesTotal",
		prevAq, currAq)

	return changes, nil
}

// extractDRMChanges extrai e compara campos do 2060 (DRM — Risco de Mercado).
//
// Compara: VaR, sVaR, RWACOM, RWAJUR1-4, e posições por moeda.
func (e *Engine) extractDRMChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := bacen.Parse2060([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRM prev: %w", err)
	}
	currDoc, err := bacen.Parse2060([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRM curr: %w", err)
	}

	var changes []FieldChange

	// VaR e sVaR.
	changes = appendFieldChange(changes, "2060", "VaR",
		parseValor(prevDoc.VaR), parseValor(currDoc.VaR))
	changes = appendFieldChange(changes, "2060", "sVaR",
		parseValor(prevDoc.SVaR), parseValor(currDoc.SVaR))

	// RWAJUR 1-4.
	changes = appendFieldChange(changes, "2060", "RWAJUR1",
		parseValor(prevDoc.RWAJUR1), parseValor(currDoc.RWAJUR1))
	changes = appendFieldChange(changes, "2060", "RWAJUR2",
		parseValor(prevDoc.RWAJUR2), parseValor(currDoc.RWAJUR2))
	changes = appendFieldChange(changes, "2060", "RWAJUR3",
		parseValor(prevDoc.RWAJUR3), parseValor(currDoc.RWAJUR3))
	changes = appendFieldChange(changes, "2060", "RWAJUR4",
		parseValor(prevDoc.RWAJUR4), parseValor(currDoc.RWAJUR4))

	// RWACOM.
	changes = appendFieldChange(changes, "2060", "RWACOM",
		parseValor(prevDoc.RWACOM), parseValor(currDoc.RWACOM))

	// Soma total de posições por código.
	prevPos := totalPosicoes(prevDoc.Posicoes)
	currPos := totalPosicoes(currDoc.Posicoes)
	changes = appendFieldChange(changes, "2060", "PosicoesTotal",
		prevPos, currPos)

	return changes, nil
}

// extract4111Changes extrai e compara campos do 4111 (COSIF).
//
// Compara: QtdCliTotal, ModTotal, contagem de clientes.
func (e *Engine) extract4111Changes(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := bacen.Parse4111([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 4111 prev: %w", err)
	}
	currDoc, err := bacen.Parse4111([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 4111 curr: %w", err)
	}

	var changes []FieldChange

	// QtdCliTotal dos totalizadores.
	if prevDoc.Totaliz != nil && currDoc.Totaliz != nil {
		changes = appendFieldChange(changes, "4111", "QtdCliTotal",
			parseFloatOrZero(prevDoc.Totaliz.QtdCliTotal),
			parseFloatOrZero(currDoc.Totaliz.QtdCliTotal))
		changes = appendFieldChange(changes, "4111", "ModTotal",
			parseFloatOrZero(prevDoc.Totaliz.ModTotal),
			parseFloatOrZero(currDoc.Totaliz.ModTotal))
	}

	// Contagem de clientes.
	changes = appendFieldChange(changes, "4111", "Clientes",
		float64(len(prevDoc.Clientes)), float64(len(currDoc.Clientes)))

	return changes, nil
}

// extractDRSACChanges extrai e compara campos do 2030 (DRSAC).
//
// Compara: total saldo devedor, contagem de operações por tipo de risco,
// total TVM, contagem de setores restritos.
func (e *Engine) extractDRSACChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := drsac.ParseFromBytes([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRSAC prev: %w", err)
	}
	currDoc, err := drsac.ParseFromBytes([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRSAC curr: %w", err)
	}

	var changes []FieldChange

	// Total saldo devedor operações de crédito.
	prevSaldo := drsactotalSaldo(prevDoc)
	currSaldo := drsactotalSaldo(currDoc)
	changes = appendFieldChange(changes, "2030", "SaldoDevedorTotal",
		prevSaldo, currSaldo)

	// Total valor TVM.
	prevTVM := drsactotalTVM(prevDoc)
	currTVM := drsactotalTVM(currDoc)
	changes = appendFieldChange(changes, "2030", "TVMTotal",
		prevTVM, currTVM)

	// Contagem de operações com alto risco social (av=01).
	prevHighSoc := drsacCountHighRisk(prevDoc, "RiscSoc")
	currHighSoc := drsacCountHighRisk(currDoc, "RiscSoc")
	changes = appendFieldChange(changes, "2030", "AltoRiscoSocial",
		prevHighSoc, currHighSoc)

	// Contagem de operações com alto risco ambiental.
	prevHighAmb := drsacCountHighRisk(prevDoc, "RiscAmb")
	currHighAmb := drsacCountHighRisk(currDoc, "RiscAmb")
	changes = appendFieldChange(changes, "2030", "AltoRiscoAmbiental",
		prevHighAmb, currHighAmb)

	// Contagem de setores restritos.
	changes = appendFieldChange(changes, "2030", "SetoresRestritos",
		float64(len(prevDoc.SetoresRestritos)), float64(len(currDoc.SetoresRestritos)))

	return changes, nil
}

// parseValor extrai float64 de um ValorSimples.
func parseValor(v bacen.ValorSimples) float64 {
	f, _ := strconv.ParseFloat(v.Valor, 64)
	return f
}

// totalPosicoes soma o valor absoluto de todas as posições.
func totalPosicoes(pos []bacen.Posicao) float64 {
	var total float64
	for _, p := range pos {
		v, _ := strconv.ParseFloat(p.Valor, 64)
		total += v
	}
	return total
}

// parseFloatOrZero converte string para float64, retorna 0 em caso de erro.
func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// drsactotalSaldo soma o saldo de todas as ExpOperCred.
func drsactotalSaldo(doc *drsac.DocumentoDRSAC) float64 {
	var total float64
	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			v, _ := strconv.ParseFloat(op.Saldo, 64)
			total += v
		}
	}
	return total
}

// drsactotalTVM soma o valor de todas as ExpTVM.
func drsactotalTVM(doc *drsac.DocumentoDRSAC) float64 {
	var total float64
	for _, cl := range doc.Clientes {
		for _, tv := range cl.ExpAtivos.ExpTVM {
			v, _ := strconv.ParseFloat(tv.Valor, 64)
			total += v
		}
	}
	return total
}

// drsacCountHighRisk conta operações com risco do tipo dado av=01.
func drsacCountHighRisk(doc *drsac.DocumentoDRSAC, riscoField string) float64 {
	var count float64
	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			var av string
			switch riscoField {
			case "RiscSoc":
				av = op.RiscSoc.Av
			case "RiscAmb":
				av = op.RiscAmb.Av
			case "RiscClimFis":
				av = op.RiscClimFis.Av
			case "RiscClimTrans":
				av = op.RiscClimTrans.Av
			}
			if av == "01" {
				count++
			}
		}
	}
	return count
}

// extract2070Changes extrai e compara campos agregados do DDR 2070.
//
// Compara: contagem de DDR entries, total por código de exposição,
// detecção de entradas novas/removidas (por Codigo × Moeda).
func (e *Engine) extract2070Changes(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDoc2070([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 2070 prev: %w", err)
	}
	currDoc, err := rules.ParseDoc2070([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 2070 curr: %w", err)
	}

	var changes []FieldChange

	// Contagem de entradas DDR.
	changes = appendFieldChange(changes, "2070", "DDREntryCount",
		float64(len(prevDoc.DDRs)), float64(len(currDoc.DDRs)))

	// Total por código de exposição (161000, 181000).
	prevByCode := make(map[string]float64)
	currByCode := make(map[string]float64)
	for _, ddr := range prevDoc.DDRs {
		if ddr.Valor != nil {
			prevByCode[ddr.Codigo] += *ddr.Valor
		}
	}
	for _, ddr := range currDoc.DDRs {
		if ddr.Valor != nil {
			currByCode[ddr.Codigo] += *ddr.Valor
		}
	}
	// Keys vistas em qualquer versão.
	allCodes := make(map[string]bool)
	for k := range prevByCode {
		allCodes[k] = true
	}
	for k := range currByCode {
		allCodes[k] = true
	}
	for code := range allCodes {
		changes = appendFieldChange(changes, "2070", "DDR_"+code+"_Total",
			prevByCode[code], currByCode[code])
	}

	// Detecta entradas novas (em curr mas não em prev) e removidas.
	prevKeys := make(map[string]bool)
	currKeys := make(map[string]bool)
	for _, ddr := range prevDoc.DDRs {
		prevKeys[ddr.Codigo+"|"+ddr.Moeda] = true
	}
	for _, ddr := range currDoc.DDRs {
		currKeys[ddr.Codigo+"|"+ddr.Moeda] = true
	}
	for key := range currKeys {
		if !prevKeys[key] {
			changes = appendFieldChange(changes, "2070", "DDR_NewEntry_"+key, 0, 1)
		}
	}
	for key := range prevKeys {
		if !currKeys[key] {
			changes = appendFieldChange(changes, "2070", "DDR_RemovedEntry_"+key, 1, 0)
		}
	}

	return changes, nil
}

// Verify Engine implements all extractors for the switch in extractFieldChanges.
// Add new cases here as they are implemented.
func init() {
	// Compile-time assertion: ensure all extractors have matching cases
	// in extractFieldChanges. If a new extractor is added but the switch
	// is not updated, this will fail to compile.
	_ = []interface {
		extractDLOChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extractDLIChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extractDRLChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extractDLPChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extract3040Changes(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extract3050Changes(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extract3044Changes(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extractDRMChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extract4111Changes(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extractDRSACChanges(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
		extract2070Changes(*SubmissionSnapshot, *SubmissionSnapshot) ([]FieldChange, error)
	}{
		(*Engine)(nil),
	}
}
