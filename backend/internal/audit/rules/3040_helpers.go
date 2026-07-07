// Helpers Sprint 37 — Audit3040 Fase 3.
//
// Validadores reutilizáveis para regras I06-I15, A16-A30, S71-S90.
// Centralizar evita duplicação entre regras e mantém a lista de UFs
// e modalidades em um único lugar.
package rules

import (
	"strings"
)

// UFsBrasileiras são as 27 UFs válidas (26 estados + DF) + "EX" (exterior).
// "EX" cobre Localiz de operações internacionais.
var UFsBrasileiras = map[string]bool{
	"AC": true, "AL": true, "AM": true, "AP": true, "BA": true,
	"CE": true, "DF": true, "ES": true, "GO": true, "MA": true,
	"MG": true, "MS": true, "MT": true, "PA": true, "PB": true,
	"PE": true, "PI": true, "PR": true, "RJ": true, "RN": true,
	"RO": true, "RR": true, "RS": true, "SC": true, "SE": true,
	"SP": true, "TO": true, "EX": true,
}

// validarUF retorna true se uf é uma UF brasileira válida ou "EX".
func validarUF(uf string) bool {
	return UFsBrasileiras[uf]
}

// validarIPOC retorna true se ipoc é alfanumérico com 8-20 caracteres.
// IPOC = Identificador Padronizado de Operações de Crédito (BACEN).
func validarIPOC(ipoc string) bool {
	if len(ipoc) < 8 || len(ipoc) > 20 {
		return false
	}
	for _, c := range ipoc {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

// modNatuOpValidas é a combinação regulamentar de Modalidade × NatuOp.
// Mapeamento parcial (top 30 combinações usadas); extend conforme necessário.
//
// Referência: catálogo BACEN SCR3040_críticas (Básicas + Semantica).
var modNatuOpValidas = map[string]bool{
	// 0201-0213: Crédito (NatuOp 01=própria, 02=cobrados)
	"0201|01": true, "0201|02": true,
	"0202|01": true, "0202|02": true,
	"0203|01": true, "0203|02": true,
	"0204|01": true, "0204|02": true,
	"0205|01": true, "0205|02": true,
	"0206|01": true, "0206|02": true,
	"0207|01": true, "0207|02": true,
	"0208|01": true, "0208|02": true,
	"0209|01": true, "0209|02": true,
	"0210|01": true, "0210|02": true,
	"0211|01": true, "0211|02": true,
	"0212|01": true, "0212|02": true,
	"0213|01": true, "0213|02": true,
	// 0271-0272: BNDES (NatuOp 01=própria)
	"0271|01": true, "0272|01": true,
	// 0301-0307: Outros créditos
	"0301|01": true, "0301|02": true,
	"0302|01": true, "0302|02": true,
	"0303|01": true, "0303|02": true,
	"0304|01": true, "0304|02": true,
	"0305|01": true, "0305|02": true,
	"0306|01": true, "0306|02": true,
	"0307|01": true, "0307|02": true,
	// 0501-0511: Rural
	"0501|01": true, "0502|01": true,
	// 0601-0613: Habitacional
	"0601|01": true, "0602|01": true,
	// 0701-0715: Leasing
	"0701|01": true, "0702|01": true,
}

// validarModNatuOp retorna true se combinação Mod × NatuOp é regulamentar.
func validarModNatuOp(mod, natuOp string) bool {
	key := mod + "|" + natuOp
	return modNatuOpValidas[key]
}

// validarPerc retorna true se perc está em [0, 100].
func validarPerc(perc float64) bool {
	return perc >= 0 && perc <= 100
}

// validarNatuOp retorna true se natuOp é 01, 02 ou 03 (própria, cobrados, outros).
func validarNatuOp(natuOp string) bool {
	return natuOp == "01" || natuOp == "02" || natuOp == "03"
}

// isVencimentoOrdemCronologica retorna true se V110 < V120 < V150 < V160 < V165.
// Assume valores não-negativos.
func isVencimentoOrdemCronologica(v Vencimentos) bool {
	v110 := parseNum(v.V110)
	v120 := parseNum(v.V120)
	v150 := parseNum(v.V150)
	v160 := parseNum(v.V160)
	v165 := parseNum(v.V165)
	// V110 (até 14d) < V120 (15-30d) < V150 (31-60d) < V160 (61-90d) < V165 (>90d)
	if v110 > v120 {
		return false
	}
	if v120 > v150 {
		return false
	}
	if v150 > v160 {
		return false
	}
	if v160 > v165 {
		return false
	}
	return true
}

// isFaixaVlrValida retorna true se faixa é 01-13 (faixas BACEN 3040).
func isFaixaVlrValida(faixa string) bool {
	switch faixa {
	case "01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12", "13":
		return true
	}
	return false
}

// isModValida retorna true se mod tem 4 dígitos numéricos.
func isModValida(mod string) bool {
	if len(mod) != 4 {
		return false
	}
	for _, c := range mod {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// upperTrim remove whitespace e converte para uppercase.
// Usado para normalizar UF e IPOC antes de validar.
func upperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
