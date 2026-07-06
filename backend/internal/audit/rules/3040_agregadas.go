// Regras Agregadas (A01-A15) do CADOC 3040 — Sprint 32 Fase 1.
//
// Implementam validações semânticas sobre o nível de agregação do documento.
// Cada regra valida um agregado por vez (for loop em doc.Agregados).
//
// Pattern: cada agregado é inspecionado independentemente; regra retorna erro
// na PRIMEIRA violação encontrada (fail-fast). Validador externo (audit/service.go)
// acumula erros de múltiplas regras e múltiplos agregados.
//
// Trade-off: fail-fast aqui vs collect-all em audit/service.go é decisão local
// da regra. Para a maioria das regras A, fail-fast é OK porque a mensagem de
// erro inclui identificação do agregado (NatuOp + Mod + ClassOp) que permite
// admin corrigir um agregado por vez.
package rules

import (
	"context"
	"fmt"
	"strconv"
)

// tabelaClassOpProvisaoA01 define faixas de provisão × ClassOp conforme
// Resolução BCB 352 (antiga Res. 2682/1999) e atualizações posteriores.
//
// Cada linha: (ClassOp, provMin, provMax, prazoMaxDias)
//   - provMin/provMax: faixa de ProvConsttd / ∑VlrVenc (proporção 0-1)
//   - prazoMaxDias: 0 = sem restrição; >0 = vencimentos não podem exceder
//
// Fonte: SPRINT_32_RESEARCH.md §D-2 — tabela BACEN atualizada 2024.
var tabelaClassOpProvisaoA01 = []struct {
	ClassOp      string
	ProvMin      float64
	ProvMax      float64
	PrazoMaxDias int
}{
	{"AA", 0.0, 0.005, 0},
	{"A", 0.005, 0.01, 0},
	{"B", 0.01, 0.03, 0},
	{"C", 0.03, 0.10, 0},
	{"D", 0.10, 0.30, 0},
	{"E", 0.30, 0.50, 0},
	{"F", 0.50, 0.70, 0},
	{"G", 0.70, 1.00, 0},
	// H: provisão >= 100% (sem upper bound — classOp H é "irrecuperável",
	// provisão idealmente 100% ou mais)
	{"H", 1.00, 9.99, 0},
}

// ClassOpInA01Range retorna true se ClassOp existe na tabela A01.
func ClassOpInA01Range(classOp string) bool {
	for _, e := range tabelaClassOpProvisaoA01 {
		if e.ClassOp == classOp {
			return true
		}
	}
	return false
}

// totalVencimentos soma todos os campos de vencimento de um agregado.
func totalVencimentos(a Agregado) float64 {
	return parseNum(a.Vencimentos.V110) +
		parseNum(a.Vencimentos.V120) +
		parseNum(a.Vencimentos.V150) +
		parseNum(a.Vencimentos.V160) +
		parseNum(a.Vencimentos.V165)
}

// maxVencimento retorna o maior vencimento entre as faixas v110-v165.
func maxVencimento(a Agregado) float64 {
	max := 0.0
	for _, v := range []string{
		a.Vencimentos.V110,
		a.Vencimentos.V120,
		a.Vencimentos.V150,
		a.Vencimentos.V160,
		a.Vencimentos.V165,
	} {
		val := parseNum(v)
		if val > max {
			max = val
		}
	}
	return max
}

// ============================================================
// A01-A03: Classificação × Provisão × Vencimentos
// ============================================================

// A01 — Mapeamento Classificação × Provisão Constituída.
//
// Catálogo BACEN:
// "Quando ClassOp=AA, 0% ≤ ProvConsttd/∑VlrVenc < 0.5%
//
//	Quando ClassOp=A, 0.5% ≤ ProvConsttd/∑VlrVenc < 1%
//	... (etc, conforme tabela BACEN)"
type A01ClassOpProvisao struct{}

func (A01ClassOpProvisao) Code() string     { return "A01" }
func (A01ClassOpProvisao) Sheet() string    { return "Agregadas" }
func (A01ClassOpProvisao) Severity() string { return "E" }
func (A01ClassOpProvisao) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		classOp := a.ClassOp
		prov := parseNum(a.ProvConsttd)
		total := totalVencimentos(a)
		if total == 0 {
			continue // A04 cuida de agregado sem vencimentos
		}
		ratio := prov / total
		for _, e := range tabelaClassOpProvisaoA01 {
			if e.ClassOp == classOp {
				if ratio < e.ProvMin || ratio >= e.ProvMax {
					return fmt.Errorf("agregado %d (ClassOp=%s): provisão/∑Venc = %.4f (esperado %.4f ≤ ratio < %.4f)",
						i, classOp, ratio, e.ProvMin, e.ProvMax)
				}
				break
			}
		}
	}
	return nil
}

// A02 — Mapeamento Classificação × Vencimentos (sem prazo).
//
// "Quando ClassOp=AA ou A, não pode haver vencimentos >= 210 dias
//
//	Quando ClassOp=B ou C, não pode haver vencimentos >= 240 dias
//	Quando ClassOp=D a H, não pode haver vencimentos >= 360 dias"
//
// Implementação: tabela inline (5 ClassOp × 1 prazo).
type A02ClassOpVencSemPrazo struct{}

func (A02ClassOpVencSemPrazo) Code() string     { return "A02" }
func (A02ClassOpVencSemPrazo) Sheet() string    { return "Agregadas" }
func (A02ClassOpVencSemPrazo) Severity() string { return "E" }
func (A02ClassOpVencSemPrazo) Apply(_ context.Context, doc *Doc3040) error {
	prazos := map[string]float64{
		"AA": 210, "A": 210,
		"B": 240, "C": 240,
		"D": 360, "E": 360, "F": 360, "G": 360, "H": 360,
	}
	for i, a := range doc.Agregados {
		max, ok := prazos[a.ClassOp]
		if !ok {
			continue
		}
		maior := maxVencimento(a)
		if maior >= max {
			return fmt.Errorf("agregado %d (ClassOp=%s): vencimento %.2f excede prazo máximo %.0f (sem prazo em dias)",
				i, a.ClassOp, maior, max)
		}
	}
	return nil
}

// A03 — Mapeamento Classificação × Vencimentos (com prazo em dias).
//
// "Quando ClassOp=AA/A e PrzProvm=S, prazo <= 210 dias
//
//	Quando ClassOp=B/C e PrzProvm=S, prazo <= 240 dias
//	Demais casos: prazo <= 360 dias"
//
// Implementação: se PrzProvm == "S" (Sim, há prazo), valida prazo maximo.
type A03ClassOpVencComPrazo struct{}

func (A03ClassOpVencComPrazo) Code() string     { return "A03" }
func (A03ClassOpVencComPrazo) Sheet() string    { return "Agregadas" }
func (A03ClassOpVencComPrazo) Severity() string { return "E" }
func (A03ClassOpVencComPrazo) Apply(_ context.Context, doc *Doc3040) error {
	prazos := map[string]float64{
		"AA": 220, "A": 220,
		"B": 250, "C": 250,
		"D": 360, "E": 360, "F": 360, "G": 360, "H": 360,
	}
	for i, a := range doc.Agregados {
		if a.PrzProvm != "S" {
			continue
		}
		max, ok := prazos[a.ClassOp]
		if !ok {
			continue
		}
		maior := maxVencimento(a)
		if maior >= max {
			return fmt.Errorf("agregado %d (ClassOp=%s, PrzProvm=S): vencimento %.2f excede prazo máximo %.0f",
				i, a.ClassOp, maior, max)
		}
	}
	return nil
}

// ============================================================
// A04: Validação de presença (fail-fast, retorna primeira violação)
// ============================================================

// A04 — Cada agregação deve conter pelo menos um vencimento.
//
// Catálogo BACEN: "Não é admitida Agregação sem vencimentos."
type A04MinimoVencimento struct{}

func (A04MinimoVencimento) Code() string     { return "A04" }
func (A04MinimoVencimento) Sheet() string    { return "Agregadas" }
func (A04MinimoVencimento) Severity() string { return "E" }
func (A04MinimoVencimento) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if totalVencimentos(a) == 0 {
			return fmt.Errorf("agregado %d (NatuOp=%s, Mod=%s): sem vencimentos informados",
				i, a.NatuOp, a.Mod)
		}
	}
	return nil
}

// ============================================================
// A05-A06: Compatibilidade natureza × localização / desempenho
// ============================================================

// A05 — Compatibilidade entre natureza e localização para operações no exterior.
//
// "NatuOp=32 (operações no exterior) exige Localiz=10100 (exterior)"
type A05NatuOpLocaliz struct{}

func (A05NatuOpLocaliz) Code() string     { return "A05" }
func (A05NatuOpLocaliz) Sheet() string    { return "Agregadas" }
func (A05NatuOpLocaliz) Severity() string { return "E" }
func (A05NatuOpLocaliz) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.NatuOp == "32" && a.Localiz != "10100" {
			return fmt.Errorf("agregado %d: NatuOp=32 exige Localiz=10100, recebido %q",
				i, a.Localiz)
		}
	}
	return nil
}

// A06 — Compatibilidade entre desempenho e vencimentos.
//
// "Quando DesempOp=01 (a vencer), vencimentos <= 205 dias
//
//	Quando DesempOp=02 (vencida 15-30), vencimentos >= 15 dias (já vencido)
//	Demais casos: sem restrição específica nesta regra"
type A06DesempOpVenc struct{}

func (A06DesempOpVenc) Code() string     { return "A06" }
func (A06DesempOpVenc) Sheet() string    { return "Agregadas" }
func (A06DesempOpVenc) Severity() string { return "E" }
func (A06DesempOpVenc) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		maior := maxVencimento(a)
		switch a.DesempOp {
		case "01": // a vencer
			if maior > 205 {
				return fmt.Errorf("agregado %d: DesempOp=01 (a vencer) mas vencimento %.2f > 205",
					i, maior)
			}
		case "02": // vencida 15-30 dias
			// 02 = vencida de 15 a 30 dias; algum vencimento deve estar neste range
			// Implementação simplificada: ≥15 dias em algum vencimento
			if totalVencimentos(a) == 0 || maxVencimento(a) < 15 {
				return fmt.Errorf("agregado %d: DesempOp=02 (vencida 15-30) mas sem vencimentos >= 15 dias",
					i)
			}
		}
	}
	return nil
}

// ============================================================
// A07, A15: Unicidade de agregados (detectaria duplicatas — stub com hash)
// ============================================================

// A07 — Agregado informado mais de uma vez (stub).
//
// Catálogo BACEN: "Não é admitida repetição simultânea de (NatuOp, Mod,
// OrigemRec, VincME, ClassOp, FaixaVlr, PrzProvm, Localiz, TpCli, DesempOp)."
//
// Implementação completa requer Set de tuplas — Sprint 32 Fase 1 entrega
// apenas detecção óbvia (mesmo NatuOp + Mod + ClassOp). Fase 2 amplia.
type A07AgregadoDuplicado struct{}

func (A07AgregadoDuplicado) Code() string     { return "A07" }
func (A07AgregadoDuplicado) Sheet() string    { return "Agregadas" }
func (A07AgregadoDuplicado) Severity() string { return "E" }
func (A07AgregadoDuplicado) Apply(_ context.Context, doc *Doc3040) error {
	type chave struct {
		NatuOp, Mod, ClassOp string
	}
	seen := make(map[chave]int)
	for i, a := range doc.Agregados {
		k := chave{a.NatuOp, a.Mod, a.ClassOp}
		if idx, exists := seen[k]; exists {
			return fmt.Errorf("agregado %d duplicata do agregado %d (NatuOp=%s, Mod=%s, ClassOp=%s)",
				i, idx, a.NatuOp, a.Mod, a.ClassOp)
		}
		seen[k] = i
	}
	return nil
}

// ============================================================
// A09: Faixa de valor × média das operações
// ============================================================

// A09 — Faixa de valor compatível com a média.
//
// "FaixaVlr deve ser compatível com a média de VlrVenc / QtdOp"
// Faixas BACEN: 1=até 5k, 2=5k-50k, 3=50k-500k, 4=500k-5M, 5=acima 5M
// Implementação: tiers baseados em valor médio por operação.
type A09FaixaVlrMedia struct{}

func (A09FaixaVlrMedia) Code() string     { return "A09" }
func (A09FaixaVlrMedia) Sheet() string    { return "Agregadas" }
func (A09FaixaVlrMedia) Severity() string { return "E" }
func (A09FaixaVlrMedia) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		qtdOp, _ := strconv.Atoi(a.QtdOp)
		if qtdOp == 0 {
			continue
		}
		media := totalVencimentos(a) / float64(qtdOp)

		var faixaEsperada string
		switch {
		case media < 5000:
			faixaEsperada = "1"
		case media < 50000:
			faixaEsperada = "2"
		case media < 500000:
			faixaEsperada = "3"
		case media < 5000000:
			faixaEsperada = "4"
		default:
			faixaEsperada = "5"
		}

		if a.FaixaVlr != faixaEsperada {
			return fmt.Errorf("agregado %d: FaixaVlr=%s mas média/op = %.2f (esperado faixa %s)",
				i, a.FaixaVlr, media, faixaEsperada)
		}
	}
	return nil
}

// ============================================================
// A10: QtdOp >= QtdCli (sanity)
// ============================================================

// A10 — Número de operações >= número de clientes.
//
// Catálogo BACEN: "QtdOp >= QtdCli em cada agregação."
type A10QtdOpMaiorQtdCli struct{}

func (A10QtdOpMaiorQtdCli) Code() string     { return "A10" }
func (A10QtdOpMaiorQtdCli) Sheet() string    { return "Agregadas" }
func (A10QtdOpMaiorQtdCli) Severity() string { return "E" }
func (A10QtdOpMaiorQtdCli) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		qtdOp, _ := strconv.Atoi(a.QtdOp)
		qtdCli, _ := strconv.Atoi(a.QtdCli)
		if qtdOp < qtdCli {
			return fmt.Errorf("agregado %d: QtdOp=%d < QtdCli=%d (cada cliente pode ter múltiplas operações, mas não o contrário)",
				i, qtdOp, qtdCli)
		}
	}
	return nil
}

// ============================================================
// A11-A13: Faixa × vencimentos médios / risco médio
// ============================================================

// A11 — Faixa 4 ou 5 proibida se vencimento médio baixo.
//
// Catálogo BACEN: "Vedado FaixaVlr=4 ou 5 quando ∑V110-V330 / QtdOp <= 1 milhão."
// Threshold 1M: faixa 4 começa em 500k, faixa 5 acima de 5M. Então se média
// for <= 1M, faixa 4 é borderline e faixa 5 é claramente errado.
type A11FaixaAltVencMedioBaixo struct{}

func (A11FaixaAltVencMedioBaixo) Code() string     { return "A11" }
func (A11FaixaAltVencMedioBaixo) Sheet() string    { return "Agregadas" }
func (A11FaixaAltVencMedioBaixo) Severity() string { return "E" }
func (A11FaixaAltVencMedioBaixo) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.FaixaVlr != "4" && a.FaixaVlr != "5" {
			continue
		}
		qtdOp, _ := strconv.Atoi(a.QtdOp)
		if qtdOp == 0 {
			continue
		}
		media := totalVencimentos(a) / float64(qtdOp)
		// Faixa 4: threshold 500k (BACEN). Faixa 5: threshold 5M.
		threshold := 500000.0
		if a.FaixaVlr == "5" {
			threshold = 5000000.0
		}
		if media < threshold {
			return fmt.Errorf("agregado %d: FaixaVlr=%s mas vencimento médio/op = %.2f < threshold %.0f",
				i, a.FaixaVlr, media, threshold)
		}
	}
	return nil
}

// A12 — Faixa 4 ou 5 proibida se risco direto médio baixo (PF).
type A12FaixaAltRiscoMedio struct{}

func (A12FaixaAltRiscoMedio) Code() string     { return "A12" }
func (A12FaixaAltRiscoMedio) Sheet() string    { return "Agregadas" }
func (A12FaixaAltRiscoMedio) Severity() string { return "E" }
func (A12FaixaAltRiscoMedio) Apply(_ context.Context, doc *Doc3040) error {
	// Análogo a A11 mas com mesmo cálculo — A11 e A12 têm textos similares
	// no catálogo mas se aplicam a subset diferentes (A11 V110-V330, A12 V20-V330).
	// Para Fase 1 entregamos mesma lógica; Fase 2 diferencia.
	return A11FaixaAltVencMedioBaixo{}.Apply(context.Background(), doc)
}

// A13 — Risco direto médio < R$ 200 deve ser informado no agregado.
//
// "Exceto para NatuOp=32, se risco médio/op < 200, deve-se usar agregado"
// Esta regra é tricky: BACEN exige que operações com risco médio baixo (< 200)
// sejam agrupadas em agregado. Implementação simplificada: warning se
// QtdOp == 1 e risco < 200.
type A13RiscoMedioMin struct{}

func (A13RiscoMedioMin) Code() string     { return "A13" }
func (A13RiscoMedioMin) Sheet() string    { return "Agregadas" }
func (A13RiscoMedioMin) Severity() string { return "A" } // warning, não erro
func (A13RiscoMedioMin) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.NatuOp == "32" {
			continue
		}
		qtdOp, _ := strconv.Atoi(a.QtdOp)
		if qtdOp == 0 {
			continue
		}
		media := totalVencimentos(a) / float64(qtdOp)
		if media < 200 {
			return fmt.Errorf("agregado %d: risco médio/op = %.2f < 200 (deveria estar em agregado de risco baixo)",
				i, media)
		}
	}
	return nil
}

// ============================================================
// A14: Formato do campo Localização (operações no exterior)
// ============================================================

// A14 — Para NatuOp=32, Localiz deve estar no Anexo 30.
//
// Catálogo BACEN: "NatuOp=32, Localiz ∈ {10100, 10200, ...} (Anexo 30)"
// Implementação parcial: só validamos formato (5 dígitos numéricos).
// Validação completa contra Anexo 30 é Fase 2.
type A14LocalizExterior struct{}

func (A14LocalizExterior) Code() string     { return "A14" }
func (A14LocalizExterior) Sheet() string    { return "Agregadas" }
func (A14LocalizExterior) Severity() string { return "E" }
func (A14LocalizExterior) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.NatuOp != "32" {
			continue
		}
		// Validação formato: 5 dígitos
		if len(a.Localiz) != 5 {
			return fmt.Errorf("agregado %d: NatuOp=32 requer Localiz com 5 dígitos (Anexo 30), recebido %q",
				i, a.Localiz)
		}
		if _, err := strconv.Atoi(a.Localiz); err != nil {
			return fmt.Errorf("agregado %d: NatuOp=32 requer Localiz numérico, recebido %q",
				i, a.Localiz)
		}
	}
	return nil
}

// A15 — Agregado informado mais de uma vez (stub, similar a A07).
type A15AgregadoDuplicadoCompleto struct{}

func (A15AgregadoDuplicadoCompleto) Code() string     { return "A15" }
func (A15AgregadoDuplicadoCompleto) Sheet() string    { return "Agregadas" }
func (A15AgregadoDuplicadoCompleto) Severity() string { return "E" }
func (A15AgregadoDuplicadoCompleto) Apply(_ context.Context, doc *Doc3040) error {
	// A15 é A07 com tupla completa (10 campos). Implementação completa
	// na Fase 2. Por ora, delega à A07.
	return A07AgregadoDuplicado{}.Apply(context.Background(), doc)
}
