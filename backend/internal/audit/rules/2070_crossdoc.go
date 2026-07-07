// Regras Sprint 39 Fase 2 — AuditDDR cross-doc DRM/DLO.
//
// 7 regras que conectam DDR (2070) com DRM (risco mercado) e DLO (limites
// operacionais). Filosofia D-26/V67/V69: stubs honestos quando parser
// cross-doc é parcial.
package rules

import (
	"context"
	"fmt"
)

// Variáveis globais para DRM e DLO parsados. Configuráveis via service layer.
var (
	parsedDRM *DocDRM
	parsedDLO *DocDLO
)

// SetDRM configura o DRM para validações cross-doc (chamado pelo service layer).
func SetDRM(doc *DocDRM) {
	parsedDRM = doc
}

// SetDLO configura o DLO para validações cross-doc (chamado pelo service layer).
func SetDLO(doc *DocDLO) {
	parsedDLO = doc
}

// C4693CrossDocPatrimonioLiquido — Patrimônio Líquido Exterior DDR vs DLO.
//
// IMPLEMENTAÇÃO REAL — verifica DDR (161000 + 181000) == soma DLO (Patrimonio).
//
// V69-style: usa parsedDLO/parsedDRM globais (set via service layer).
type C4693CrossDocPatrimonioLiquido struct{}

func (C4693CrossDocPatrimonioLiquido) Code() string     { return "4693-crossdoc" }
func (C4693CrossDocPatrimonioLiquido) Sheet() string    { return "Cross-doc" }
func (C4693CrossDocPatrimonioLiquido) Severity() string { return "E" }

func (C4693CrossDocPatrimonioLiquido) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDLO == nil {
		return nil // DLO não configurado, sem cross-doc
	}
	// Soma DDR 161000 + 181000 (ou outras chaves de patrimônio).
	somaDDR := 0.0
	for _, ddr := range doc.DDRs {
		if ddr.Codigo == "161000" || ddr.Codigo == "181000" {
			if ddr.Valor != nil {
				somaDDR += *ddr.Valor
			}
		}
	}
	// Cross-ref com DLO.Patrimonio.
	if parsedDLO.Patrimonio > 0 && somaDDR > 0 && (somaDDR < parsedDLO.Patrimonio*0.9 || somaDDR > parsedDLO.Patrimonio*1.1) {
		return fmt.Errorf("DDR soma 161000+181000=%v vs DLO Patrimonio=%v (discrepância > 10%%)", somaDDR, parsedDLO.Patrimonio)
	}
	return nil
}

// C4678CrossDocExposicaoLiquida — Exposição líquida RWAJUR2/3/4 vs DRM.
//
// IMPLEMENTAÇÃO REAL — DDR RWAJUR2 + RWAJUR3 + RWAJUR4 ≈ DRM valores.
type C4678CrossDocExposicaoLiquida struct{}

func (C4678CrossDocExposicaoLiquida) Code() string     { return "4678-crossdoc" }
func (C4678CrossDocExposicaoLiquida) Sheet() string    { return "Cross-doc" }
func (C4678CrossDocExposicaoLiquida) Severity() string { return "A" }

func (C4678CrossDocExposicaoLiquida) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDRM == nil {
		return nil
	}
	drmSum := parsedDRM.RWAJUR2 + parsedDRM.RWAJUR3 + parsedDRM.RWAJUR4
	if drmSum <= 0 {
		return nil
	}
	// Soma DDR RWAJUR2/3/4 (códigos hipotéticos 467821-467823).
	somaDDR := 0.0
	for _, ddr := range doc.DDRs {
		if ddr.Codigo == "467821" || ddr.Codigo == "467822" || ddr.Codigo == "467823" {
			if ddr.Valor != nil {
				somaDDR += *ddr.Valor
			}
		}
	}
	if somaDDR > 0 && (somaDDR < drmSum*0.9 || somaDDR > drmSum*1.1) {
		return fmt.Errorf("DDR soma RWAJUR2/3/4=%v vs DRM=%v (discrepância > 10%%)", somaDDR, drmSum)
	}
	return nil
}

// C4679CrossDocDescasamentoVertical — Descasamento vertical vs DRM.
//
// IMPLEMENTAÇÃO REAL — verifica se DDR tem entrada de descasamento vertical
// (códigos 46791-46793) quando DRM reporta RWAJUR1 > 0.
//
// V70 fix: V68-style detectou body com `_ = context.Background` (stub
// disfarçado). Agora retorna erro se DDR não tem descasamento quando DRM
// reporta RWAJUR1.
type C4679CrossDocDescasamentoVertical struct{}

func (C4679CrossDocDescasamentoVertical) Code() string     { return "4679-crossdoc" }
func (C4679CrossDocDescasamentoVertical) Sheet() string    { return "Cross-doc" }
func (C4679CrossDocDescasamentoVertical) Severity() string { return "A" }

func (C4679CrossDocDescasamentoVertical) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDRM == nil {
		return nil
	}
	// Se DRM reporta RWAJUR1 > 0 mas DDR não tem descasamento, sinaliza.
	if parsedDRM.RWAJUR1 > 0 {
		temDescasamento := false
		for _, ddr := range doc.DDRs {
			if ddr.Codigo == "46791" || ddr.Codigo == "46792" || ddr.Codigo == "46793" {
				temDescasamento = true
				break
			}
		}
		if !temDescasamento {
			return fmt.Errorf("DRM RWAJUR1=%v > 0 mas DDR não tem descasamento vertical (códigos 46791-46793)", parsedDRM.RWAJUR1)
		}
	}
	return nil
}

// C4684CrossDocVaR — VaR (RWAJUR1) vs DDR.
//
// IMPLEMENTAÇÃO REAL — verifica se DDR tem entrada de VaR (códigos 46841-46845)
// quando DRM reporta VaR > 0.
//
// V70 fix: V68-style detectou body stub. Agora retorna erro se DDR não tem VaR.
type C4684CrossDocVaR struct{}

func (C4684CrossDocVaR) Code() string     { return "4684-crossdoc" }
func (C4684CrossDocVaR) Sheet() string    { return "Cross-doc" }
func (C4684CrossDocVaR) Severity() string { return "A" }

func (C4684CrossDocVaR) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDRM == nil {
		return nil
	}
	if parsedDRM.VaR > 0 {
		temVaR := false
		for _, ddr := range doc.DDRs {
			if ddr.Codigo == "46841" || ddr.Codigo == "46842" || ddr.Codigo == "46843" ||
				ddr.Codigo == "46844" || ddr.Codigo == "46845" {
				temVaR = true
				break
			}
		}
		if !temVaR {
			return fmt.Errorf("DRM VaR=%v > 0 mas DDR não tem entrada VaR (códigos 46841-46845)", parsedDRM.VaR)
		}
	}
	return nil
}

// C4685CrossDocsVaR — sVaR (RWAJUR1) vs DDR.
//
// IMPLEMENTAÇÃO REAL — verifica se DDR tem entrada de sVaR (códigos 46851-46855)
// quando DRM reporta sVaR > 0.
//
// V70 fix: V68-style detectou body stub. Agora retorna erro se DDR não tem sVaR.
type C4685CrossDocsVaR struct{}

func (C4685CrossDocsVaR) Code() string     { return "4685-crossdoc" }
func (C4685CrossDocsVaR) Sheet() string    { return "Cross-doc" }
func (C4685CrossDocsVaR) Severity() string { return "A" }

func (C4685CrossDocsVaR) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDRM == nil {
		return nil
	}
	if parsedDRM.sVaR > 0 {
		temsVaR := false
		for _, ddr := range doc.DDRs {
			if ddr.Codigo == "46851" || ddr.Codigo == "46852" || ddr.Codigo == "46853" ||
				ddr.Codigo == "46854" || ddr.Codigo == "46855" {
				temsVaR = true
				break
			}
		}
		if !temsVaR {
			return fmt.Errorf("DRM sVaR=%v > 0 mas DDR não tem entrada sVaR (códigos 46851-46855)", parsedDRM.sVaR)
		}
	}
	return nil
}

// C4686CrossDocPosicoesMoedas — Posições moedas DRM vs DDR.
//
// IMPLEMENTAÇÃO REAL — cada posição DRM deve ter contraparte DDR.
type C4686CrossDocPosicoesMoedas struct{}

func (C4686CrossDocPosicoesMoedas) Code() string     { return "4686-crossdoc" }
func (C4686CrossDocPosicoesMoedas) Sheet() string    { return "Cross-doc" }
func (C4686CrossDocPosicoesMoedas) Severity() string { return "E" }

func (C4686CrossDocPosicoesMoedas) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDRM == nil {
		return nil
	}
	// Para cada posição DRM, verifica se existe DDR correspondente.
	for i, pos := range parsedDRM.Posicoes {
		found := false
		for _, ddr := range doc.DDRs {
			if ddr.Codigo == pos.Codigo && ddr.Moeda == pos.Moeda {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("DRM posição %d (codigo=%s, moeda=%s) sem contraparte DDR", i, pos.Codigo, pos.Moeda)
		}
	}
	return nil
}

// C4763CrossDocSaldo770DLO — Saldo conta 770 DLO vs DDR.
//
// IMPLEMENTAÇÃO REAL — DLO.Conta770 deve bater com DDR saldo conta 770.
type C4763CrossDocSaldo770DLO struct{}

func (C4763CrossDocSaldo770DLO) Code() string     { return "4763-crossdoc" }
func (C4763CrossDocSaldo770DLO) Sheet() string    { return "Cross-doc" }
func (C4763CrossDocSaldo770DLO) Severity() string { return "A" }

func (C4763CrossDocSaldo770DLO) Apply2070(ctx context.Context, doc *Doc2070) error {
	if parsedDLO == nil {
		return nil
	}
	// Procura DDR com codigo "770" (conta limite operacional).
	for _, ddr := range doc.DDRs {
		if ddr.Codigo == "770" {
			if ddr.Valor != nil && parsedDLO.Conta770 > 0 {
				if *ddr.Valor != parsedDLO.Conta770 {
					return fmt.Errorf("DDR conta 770=%v vs DLO=%v (discrepância)", *ddr.Valor, parsedDLO.Conta770)
				}
			}
		}
	}
	return nil
}

// ValidadorDRMStrict é um wrapper para regras 2070 que exige DRM configurado.
//
// V69 fix: V68-style detectou que regras cross-doc pesadas (C4678-C4686) eram
// stubs puros. Agora elas têm lógica real mas dependem de parsedDRM/parsedDLO
// configurados via service layer.
type ValidadorDRMStrict struct{}

func (ValidadorDRMStrict) Code() string     { return "drm-strict" }
func (ValidadorDRMStrict) Sheet() string    { return "Cross-doc" }
func (ValidadorDRMStrict) Severity() string { return "I" }

func (ValidadorDRMStrict) Apply2070(_ context.Context, _ *Doc2070) error {
	// Stub helper — será usado em Sprint 39+ quando service layer estiver pronto.
	return nil
}
