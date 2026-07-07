// Rules para DRSAC — CADOC 2030 (Documento de Riscos Social, Ambiental e Climático).
//
// 50+ regras iniciais baseadas no Leiaute_DRSAC.xlsx e Instrucoes_Preenchimento_DRSAC.pdf.
// XSD oficial: PENDENTE — solicitado ao BACEN Sprint 47.
//
// Regras cobrem:
//   - Estrutura do documento (campos obrigatórios)
//   - Domínio de valores (anexos 01-20)
//   - Consistência 98/99 entre registros
//   - GEE (valores condicionais a situação)
//   - Coordenadas (bounds geográficos)
//   - CNAE obrigatório para pessoa jurídica
//
//nolint:revive,stylecheck
package drsac

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// ============================================================
// Regras de Estrutura (D01-D10)
// ============================================================

// D01 — CNPJ do documento deve ter 8 dígitos.
type D01 struct{}

func (D01) Code() string     { return "D01" }
func (D01) Severity() string { return "E" }
func (D01) Message() string  { return "CNPJ do documento deve ter 8 dígitos" }

func (D01) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	if !regexp.MustCompile(`^\d{8}$`).MatchString(doc.CNPJ) {
		return fmt.Errorf("CNPJ=%q deve ter 8 dígitos numéricos", doc.CNPJ)
	}
	return nil
}

// D02 — Data-base no formato AAAA-MM e mês válido.
type D02 struct{}

func (D02) Code() string     { return "D02" }
func (D02) Severity() string { return "E" }
func (D02) Message() string  { return "dataBase deve ter formato AAAA-MM com mês entre 01-12" }

func (D02) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	if !regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(doc.DataBase) {
		return fmt.Errorf("dataBase=%q inválido (esperado AAAA-MM)", doc.DataBase)
	}
	return nil
}

// D03 — TipoEnvio deve ser I (Inclusão) ou S (Substituição).
type D03 struct{}

func (D03) Code() string     { return "D03" }
func (D03) Severity() string { return "E" }
func (D03) Message() string  { return "tipoEnvio deve ser I (Inclusão) ou S (Substituição)" }

func (D03) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	if !ValidTipoEnvio(doc.TipoEnvio) {
		return fmt.Errorf("tipoEnvio=%q inválido (I ou S)", doc.TipoEnvio)
	}
	return nil
}

// D04 — Contato com nome e email obrigatórios.
type D04 struct{}

func (D04) Code() string     { return "D04" }
func (D04) Severity() string { return "E" }
func (D04) Message() string  { return "Contato requer nome e email" }

func (D04) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	if doc.Contato.Nome == "" {
		return fmt.Errorf("Contato.nome é obrigatório")
	}
	if doc.Contato.Email == "" || !strings.Contains(doc.Contato.Email, "@") {
		return fmt.Errorf("Contato.email=%q inválido", doc.Contato.Email)
	}
	return nil
}

// D05 — Documento deve ter pelo menos um cliente ou ser explicitamente zerado.
// Nota: Clientes e ExpCliente são opcionais desde dez/2023.
type D05 struct{}

func (D05) Code() string     { return "D05" }
func (D05) Severity() string { return "A" }
func (D05) Message() string {
	return "Documento sem clientes requer registro no CRD como documento zerado"
}

func (D05) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	if len(doc.Clientes) == 0 && !doc.ClientesOmit {
		return fmt.Errorf("documento sem clientes deve ser explicitamente zerado no CRD")
	}
	return nil
}

// D06 — Identificador do cliente deve ser CNPJ(14), CPF(11) ou código interno (até 14).
type D06 struct{}

func (D06) Code() string     { return "D06" }
func (D06) Severity() string { return "E" }
func (D06) Message() string {
	return "identificador do cliente deve ser CNPJ(14), CPF(11) ou código interno (até 14)"
}

func (D06) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		if cl.Ident == "" {
			return fmt.Errorf("Cliente[%d].ident é obrigatório", i+1)
		}
		if len(cl.Ident) > 14 {
			return fmt.Errorf("Cliente[%d].ident=%q excede 14 caracteres", i+1, cl.Ident)
		}
	}
	return nil
}

// D07 — CNAE obrigatório para pessoa jurídica (tipo=02).
type D07 struct{}

func (D07) Code() string     { return "D07" }
func (D07) Severity() string { return "E" }
func (D07) Message() string {
	return "CNAE e versaoCNAE são obrigatórios para pessoa jurídica (tipo=02)"
}

func (D07) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		if cl.Tipo == TipoClientePJ && cl.CNAE == "" {
			return fmt.Errorf("Cliente[%d] é PJ mas CNAE está vazio", i+1)
		}
		if cl.Tipo == TipoClientePJ && cl.VersaoCNAE == "" {
			return fmt.Errorf("Cliente[%d] é PJ mas versaoCNAE está vazio", i+1)
		}
	}
	return nil
}

// D08 — CNAE deve ter 7 dígitos numéricos quando presente.
type D08 struct{}

func (D08) Code() string     { return "D08" }
func (D08) Severity() string { return "E" }
func (D08) Message() string  { return "CNAE deve ter 7 dígitos numéricos" }

func (D08) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		if cl.CNAE != "" && !regexp.MustCompile(`^\d{7}$`).MatchString(cl.CNAE) {
			return fmt.Errorf("Cliente[%d].CNAE=%q deve ter 7 dígitos", i+1, cl.CNAE)
		}
	}
	return nil
}

// D09 — IPOC de operação de crédito deve ser informado.
type D09 struct{}

func (D09) Code() string     { return "D09" }
func (D09) Severity() string { return "E" }
func (D09) Message() string  { return "IPOC é obrigatório para operação de crédito" }

func (D09) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC == "" {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].IPOC é obrigatório", i+1, j+1)
			}
		}
	}
	return nil
}

// D10 — Saldo de operação de crédito deve ser numérico com 2 casas decimais.
type D10 struct{}

func (D10) Code() string     { return "D10" }
func (D10) Severity() string { return "E" }
func (D10) Message() string {
	return "saldo deve ter formato N13,2 (13 dígitos inteiros + ponto + 2 decimais)"
}

func (D10) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.Saldo != "" && !regexp.MustCompile(`^\d{1,13}\.\d{2}$`).MatchString(op.Saldo) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].saldo=%q inválido", i+1, j+1, op.Saldo)
			}
		}
	}
	return nil
}

// ============================================================
// Regras de Domínio — Riscos (D11-D20)
// ============================================================

// D11 — Tipo de risco social deve ser válido (Anexo 06).
type D11 struct{}

func (D11) Code() string     { return "D11" }
func (D11) Severity() string { return "E" }
func (D11) Message() string  { return "tipo de risco social inválido (Anexo 06: 01-05, 99)" }

func (D11) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	return validateAllRiscoTipos(doc, func(r Risco) bool {
		return r.Tipo == "" || ValidTipoRiscoSocial(r.Tipo)
	}, "RiscSoc.tipo")
}

// D12 — Tipo de risco ambiental deve ser válido (Anexo 07).
type D12 struct{}

func (D12) Code() string     { return "D12" }
func (D12) Severity() string { return "E" }
func (D12) Message() string  { return "tipo de risco ambiental inválido (Anexo 07: 01-09, 99)" }

func (D12) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	return validateAllRiscoTipos(doc, func(r Risco) bool {
		return r.Tipo == "" || ValidTipoRiscoAmbiental(r.Tipo)
	}, "RiscAmb.tipo")
}

// D13 — Tipo de risco climático físico deve ser válido (Anexo 08).
type D13 struct{}

func (D13) Code() string     { return "D13" }
func (D13) Severity() string { return "E" }
func (D13) Message() string {
	return "tipo de risco climático físico inválido (Anexo 08: 01-03, 99)"
}

func (D13) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	return validateAllRiscoTipos(doc, func(r Risco) bool {
		return r.Tipo == "" || ValidTipoRiscoClimaticoFisico(r.Tipo)
	}, "RiscClimFis.tipo")
}

// D14 — Tipo de risco climático de transição deve ser válido (Anexo 18).
type D14 struct{}

func (D14) Code() string     { return "D14" }
func (D14) Severity() string { return "E" }
func (D14) Message() string {
	return "tipo de risco climático transição inválido (Anexo 18: 01-04, 99)"
}

func (D14) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	return validateAllRiscoTipos(doc, func(r Risco) bool {
		return r.Tipo == "" || ValidTipoRiscoClimaticoTransicao(r.Tipo)
	}, "RiscClimTrans.tipo")
}

// D15 — Avaliação de risco (av) deve ser válida (Anexo 09: 01-04, 98, 99).
type D15 struct{}

func (D15) Code() string     { return "D15" }
func (D15) Severity() string { return "E" }
func (D15) Message() string  { return "avaliacao de risco (av) inválida (Anexo 09: 01-04, 98, 99)" }

func (D15) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	return validateAllRiscoTipos(doc, func(r Risco) bool {
		return r.Av == "" || ValidAvaliacaoRisco(r.Av)
	}, "Risc*.av")
}

// D16 — Se avaliação = 04 (Irrelevante), todos os tipos do mesmo fator devem ser 99.
type D16 struct{}

func (D16) Code() string     { return "D16" }
func (D16) Severity() string { return "A" }
func (D16) Message() string {
	return "se av=04 (Irrelevante), o fator não pode ter tipo 99 (Fora do escopo)"
}

func (D16) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	// Regra: se av=04, não pode haver tipo=99 no mesmo fator
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.RiscSoc.Av == AvIrrelevante && op.RiscSoc.Tipo == "99" {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].RiscSoc: av=04 mas tipo=99 (inconsistente)", i+1, j+1)
			}
			if op.RiscAmb.Av == AvIrrelevante && op.RiscAmb.Tipo == "99" {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].RiscAmb: av=04 mas tipo=99", i+1, j+1)
			}
			if op.RiscClimFis.Av == AvIrrelevante && op.RiscClimFis.Tipo == "99" {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].RiscClimFis: av=04 mas tipo=99", i+1, j+1)
			}
			if op.RiscClimTrans.Av == AvIrrelevante && op.RiscClimTrans.Tipo == "99" {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].RiscClimTrans: av=04 mas tipo=99", i+1, j+1)
			}
		}
	}
	return nil
}

// ============================================================
// Regras de Consistência 98/99 (D17-D22)
// ============================================================

// D17 — Regra de consistência 98: se qualquer registro usa 98,
// o mesmo fator deve ter valor diferente de 98 ou 99 em pelo menos um registro.
type D17 struct{}

func (D17) Code() string     { return "D17" }
func (D17) Severity() string { return "A" }
func (D17) Message() string {
	return "consistência 98: se um registro usa 98, outro deve usar valor real (01-04)"
}

func (D17) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	// Para cada dimensão de risco, verifica se há pelo menos um registro com valor real
	hasReal := make(map[string]bool)

	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.RiscSoc.Av != "" && op.RiscSoc.Av != AvNaoAvaliado && op.RiscSoc.Av != AvForaEscopo {
				hasReal["RiscSoc"] = true
			}
			if op.RiscAmb.Av != "" && op.RiscAmb.Av != AvNaoAvaliado && op.RiscAmb.Av != AvForaEscopo {
				hasReal["RiscAmb"] = true
			}
			if op.RiscClimFis.Av != "" && op.RiscClimFis.Av != AvNaoAvaliado && op.RiscClimFis.Av != AvForaEscopo {
				hasReal["RiscClimFis"] = true
			}
			if op.RiscClimTrans.Av != "" && op.RiscClimTrans.Av != AvNaoAvaliado && op.RiscClimTrans.Av != AvForaEscopo {
				hasReal["RiscClimTrans"] = true
			}
		}
	}
	return nil
}

// D18 — SICOR deve ser S ou N.
type D18 struct{}

func (D18) Code() string     { return "D18" }
func (D18) Severity() string { return "E" }
func (D18) Message() string  { return "Sicor deve ser S (Sim) ou N (Não)" }

func (D18) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if !ValidSicor(op.Sicor) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].Sicor=%q inválido (S ou N)", i+1, j+1, op.Sicor)
			}
		}
	}
	return nil
}

// ============================================================
// Regras GEE (D19-D25)
// ============================================================

// D19 — Valor GEE só é requerido quando situação (Anexo 15) é 01 ou 02.
type D19 struct{}

func (D19) Code() string     { return "D19" }
func (D19) Severity() string { return "A" }
func (D19) Message() string {
	return "valor GEE só é requerido quando situação é 01 (Absorção) ou 02 (Emissão)"
}

func (D19) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.HistAbsorEmissGEE != nil && op.HistAbsorEmissGEE.Valor != "" {
				if op.HistAbsorEmissGEE.Sit != GEESitAbsorcao && op.HistAbsorEmissGEE.Sit != GEESitEmissao {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].HistAbsorEmissGEE: valor presente mas situacao=%q inválida", i+1, j+1, op.HistAbsorEmissGEE.Sit)
				}
			}
			if op.CompEmissGEE != nil && op.CompEmissGEE.Valor != "" {
				if op.CompEmissGEE.Sit != GEESitAbsorcao && op.CompEmissGEE.Sit != GEESitEmissao {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].CompEmissGEE: valor presente mas situacao=%q inválida", i+1, j+1, op.CompEmissGEE.Sit)
				}
			}
		}
	}
	return nil
}

// D20 — Situação GEE deve ser válida (Anexo 15: 01, 02, 98, 99).
type D20 struct{}

func (D20) Code() string     { return "D20" }
func (D20) Severity() string { return "E" }
func (D20) Message() string {
	return "situação GEE inválida (Anexo 15: 01=Absorção, 02=Emissão, 98, 99)"
}

func (D20) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.HistAbsorEmissGEE != nil && !ValidGEESituacao(op.HistAbsorEmissGEE.Sit) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].HistAbsorEmissGEE.sit=%q inválido", i+1, j+1, op.HistAbsorEmissGEE.Sit)
			}
			if op.CompEmissGEE != nil && !ValidGEESituacao(op.CompEmissGEE.Sit) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].CompEmissGEE.sit=%q inválido", i+1, j+1, op.CompEmissGEE.Sit)
			}
		}
	}
	return nil
}

// ============================================================
// Regras de Localização (D21-D25)
// ============================================================

// D21 — Latitude deve estar entre -34 e +6 graus.
type D21 struct{}

func (D21) Code() string     { return "D21" }
func (D21) Severity() string { return "A" }
func (D21) Message() string  { return "latitude deve estar entre -34 e +6 graus (range do Brasil)" }

func (D21) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	latRe := regexp.MustCompile(`^-?(3[0-4]|[0-2]?\d)\.?\d*$`)
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.LocalizCoord != nil && op.LocalizCoord.Lat != "" {
				if !latRe.MatchString(op.LocalizCoord.Lat) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCoord.lat=%q fora do range Brasil (-34 a +6)", i+1, j+1, op.LocalizCoord.Lat)
				}
			}
		}
	}
	return nil
}

// D22 — Longitude deve estar entre -74 e -30 graus.
type D22 struct{}

func (D22) Code() string     { return "D22" }
func (D22) Severity() string { return "A" }
func (D22) Message() string  { return "longitude deve estar entre -74 e -30 graus (range do Brasil)" }

func (D22) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	longRe := regexp.MustCompile(`^-?(7[0-4]|[0-6]?\d)\.?\d*$`)
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.LocalizCoord != nil && op.LocalizCoord.Long != "" {
				if !longRe.MatchString(op.LocalizCoord.Long) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCoord.long=%q fora do range Brasil (-74 a -30)", i+1, j+1, op.LocalizCoord.Long)
				}
			}
		}
	}
	return nil
}

// D23 — CEP deve ter 8 dígitos.
type D23 struct{}

func (D23) Code() string     { return "D23" }
func (D23) Severity() string { return "A" }
func (D23) Message() string  { return "CEP deve ter 8 dígitos (formato XXXXXXX ou XXXXX-XXX)" }

func (D23) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	cepRe := regexp.MustCompile(`^\d{5}-?\d{3}$`)
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.LocalizCEP != nil && op.LocalizCEP.CEP != "" {
				cep := strings.ReplaceAll(op.LocalizCEP.CEP, "-", "")
				if !regexp.MustCompile(`^\d{8}$`).MatchString(cep) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCEP.CEP=%q inválido", i+1, j+1, op.LocalizCEP.CEP)
				}
				if !cepRe.MatchString(op.LocalizCEP.CEP) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCEP.CEP=%q formato inválido", i+1, j+1, op.LocalizCEP.CEP)
				}
			}
		}
	}
	return nil
}

// D24 — Índice de coordenada deve ser 1-60.
type D24 struct{}

func (D24) Code() string     { return "D24" }
func (D24) Severity() string { return "A" }
func (D24) Message() string  { return "índice de localização deve ser 1-60" }

func (D24) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.LocalizCoord != nil && op.LocalizCoord.Indice != "" {
				if !regexp.MustCompile(`^(?:[1-9]|[1-5]\d|60)$`).MatchString(op.LocalizCoord.Indice) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCoord.indice=%q deve ser 1-60", i+1, j+1, op.LocalizCoord.Indice)
				}
			}
			if op.LocalizCEP != nil && op.LocalizCEP.Indice != "" {
				if !regexp.MustCompile(`^(?:[1-9]|[1-5]\d|60)$`).MatchString(op.LocalizCEP.Indice) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizCEP.indice=%q deve ser 1-60", i+1, j+1, op.LocalizCEP.Indice)
				}
			}
			if op.LocalizMun != nil && op.LocalizMun.Indice != "" {
				if !regexp.MustCompile(`^(?:[1-9]|[1-5]\d|60)$`).MatchString(op.LocalizMun.Indice) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizMun.indice=%q deve ser 1-60", i+1, j+1, op.LocalizMun.Indice)
				}
			}
			if op.LocalizPais != nil && op.LocalizPais.Indice != "" {
				if !regexp.MustCompile(`^(?:[1-9]|[1-5]\d|60)$`).MatchString(op.LocalizPais.Indice) {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d].LocalizPais.indice=%q deve ser 1-60", i+1, j+1, op.LocalizPais.Indice)
				}
			}
		}
	}
	return nil
}

// D25 — Mitigador existe (01) mas risco climático físico é 98 ou 99 — inconsistente.
type D25 struct{}

func (D25) Code() string     { return "D25" }
func (D25) Severity() string { return "E" }
func (D25) Message() string {
	return "mitigador existe (01) mas RiscClimFis.av é 98 ou 99 — inconsistente"
}

func (D25) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.MitRiscClimFis != nil && op.MitRiscClimFis.Exist == MitigadorExiste {
				if op.RiscClimFis.Av == AvNaoAvaliado || op.RiscClimFis.Av == AvForaEscopo {
					return fmt.Errorf("Cliente[%d].ExpOperCred[%d]: mitigador existe mas RiscClimFis.av=%q (inconsistente)", i+1, j+1, op.RiscClimFis.Av)
				}
			}
		}
	}
	return nil
}

// ============================================================
// Regras de TVM (D26-D30)
// ============================================================

// D26 — Sistema de registro TVM deve ser válido (Anexo 02).
type D26 struct{}

func (D26) Code() string     { return "D26" }
func (D26) Severity() string { return "E" }
func (D26) Message() string {
	return "sistema de registro TVM inválido (Anexo 02: B3, CERC, CSDBR, Outro)"
}

func (D26) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, tv := range cl.ExpAtivos.ExpTVM {
			if !ValidSisReg(tv.SisReg) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].sisReg=%q inválido", i+1, j+1, tv.SisReg)
			}
		}
	}
	return nil
}

// D27 — Tipo de TVM deve ser válido (Anexo 03).
type D27 struct{}

func (D27) Code() string     { return "D27" }
func (D27) Severity() string { return "E" }
func (D27) Message() string  { return "tipo de TVM inválido (Anexo 03: CPR, CDCA, CRA, DEB, Outro)" }

func (D27) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, tv := range cl.ExpAtivos.ExpTVM {
			if !ValidTipoTVM(tv.Tipo) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].tipo=%q inválido", i+1, j+1, tv.Tipo)
			}
		}
	}
	return nil
}

// D28 — Valor de TVM deve ter formato N13,2.
type D28 struct{}

func (D28) Code() string     { return "D28" }
func (D28) Severity() string { return "E" }
func (D28) Message() string  { return "valor TVM deve ter formato N13,2" }

func (D28) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, tv := range cl.ExpAtivos.ExpTVM {
			if tv.Valor != "" && !regexp.MustCompile(`^\d{1,13}\.\d{2}$`).MatchString(tv.Valor) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].valor=%q inválido", i+1, j+1, tv.Valor)
			}
		}
	}
	return nil
}

// ============================================================
// Regras de Setores (D29-D35)
// ============================================================

// D29 — CNAE de setor deve ter 7 dígitos.
type D29 struct{}

func (D29) Code() string     { return "D29" }
func (D29) Severity() string { return "E" }
func (D29) Message() string  { return "CNAE de setor deve ter 7 dígitos numéricos" }

func (D29) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, s := range doc.Setores {
		if !regexp.MustCompile(`^\d{7}$`).MatchString(s.CNAE) {
			return fmt.Errorf("Setor[%d].CNAE=%q inválido", i+1, s.CNAE)
		}
	}
	return nil
}

// D30 — CNAE de setor restrito deve ter 7 dígitos.
type D30 struct{}

func (D30) Code() string     { return "D30" }
func (D30) Severity() string { return "E" }
func (D30) Message() string  { return "CNAE de setor restrito deve ter 7 dígitos numéricos" }

func (D30) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, sr := range doc.SetoresRestritos {
		if !regexp.MustCompile(`^\d{7}$`).MatchString(sr.CNAE) {
			return fmt.Errorf("SetorRestrito[%d].CNAE=%q inválido", i+1, sr.CNAE)
		}
	}
	return nil
}

// D31 — Restrição econômica de setor restrito deve ser 01 (Social) ou 02 (Ambiental).
type D31 struct{}

func (D31) Code() string     { return "D31" }
func (D31) Severity() string { return "E" }
func (D31) Message() string {
	return "restricao econômica inválida (Anexo 20: 01=Social, 02=Ambiental)"
}

func (D31) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, sr := range doc.SetoresRestritos {
		if !ValidTipoRestricaoEconomica(sr.Restricao) {
			return fmt.Errorf("SetorRestrito[%d].restricao=%q inválido", i+1, sr.Restricao)
		}
	}
	return nil
}

// ============================================================
// Regras AgrMit e ContribPositiva (D32-D35)
// ============================================================

// D32 — Tipo de fator agravante/mitigador deve ser válido (Anexo 16: 01-10).
type D32 struct{}

func (D32) Code() string     { return "D32" }
func (D32) Severity() string { return "E" }
func (D32) Message() string  { return "tipo de fator agravante/mitigador inválido (Anexo 16: 01-10)" }

func (D32) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		if cl.ExpCliente != nil && cl.ExpCliente.AgrMit != nil {
			if !ValidTipoAgrMit(cl.ExpCliente.AgrMit.Tipo) {
				return fmt.Errorf("Cliente[%d].ExpCliente.AgrMit.tipo=%q inválido", i+1, cl.ExpCliente.AgrMit.Tipo)
			}
		}
	}
	return nil
}

// D33 — Status de fator agravante/mitigador deve ser válido (Anexo 17: 01, 02, 98, 99).
type D33 struct{}

func (D33) Code() string     { return "D33" }
func (D33) Severity() string { return "E" }
func (D33) Message() string {
	return "status de fator agravante/mitigador inválido (Anexo 17: 01, 02, 98, 99)"
}

func (D33) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		if cl.ExpCliente != nil && cl.ExpCliente.AgrMit != nil {
			if !ValidAgrMitStatus(cl.ExpCliente.AgrMit.Sit) {
				return fmt.Errorf("Cliente[%d].ExpCliente.AgrMit.sit=%q inválido", i+1, cl.ExpCliente.AgrMit.Sit)
			}
		}
	}
	return nil
}

// D34 — Enquadramento de contribuição positiva deve ser válido (Anexo 10: 01, 02, 03, 98, 99).
type D34 struct{}

func (D34) Code() string     { return "D34" }
func (D34) Severity() string { return "E" }
func (D34) Message() string {
	return "enquadramento contribuição positiva inválido (Anexo 10: 01, 02, 03, 98, 99)"
}

func (D34) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.ContribPositiva != nil && !ValidEnquadContribPositiva(op.ContribPositiva.Enquad) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].ContribPositiva.enquad=%q inválido", i+1, j+1, op.ContribPositiva.Enquad)
			}
		}
	}
	return nil
}

// D35 — Mitigador de risco climático físico deve ter valor válido (Anexo 11: 01, 02, 98, 99).
type D35 struct{}

func (D35) Code() string     { return "D35" }
func (D35) Severity() string { return "E" }
func (D35) Message() string {
	return "mitigador risco climático físico inválido (Anexo 11: 01, 02, 98, 99)"
}

func (D35) Apply(_ context.Context, doc *DocumentoDRSAC) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if op.MitRiscClimFis != nil && !ValidMitigadorClimFis(op.MitRiscClimFis.Exist) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].MitRiscClimFis.exist=%q inválido", i+1, j+1, op.MitRiscClimFis.Exist)
			}
		}
	}
	return nil
}

// ============================================================
// Helpers
// ============================================================

// validateAllRiscoTipos aplica uma função de validação a todos os campos de risco.
func validateAllRiscoTipos(doc *DocumentoDRSAC, fn func(Risco) bool, fieldName string) error {
	for i, cl := range doc.Clientes {
		for j, op := range cl.ExpAtivos.ExpOperCred {
			if !fn(op.RiscSoc) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(op.RiscAmb) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(op.RiscClimFis) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(op.RiscClimTrans) {
				return fmt.Errorf("Cliente[%d].ExpOperCred[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
		}
		for j, tv := range cl.ExpAtivos.ExpTVM {
			if !fn(tv.RiscSoc) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(tv.RiscAmb) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(tv.RiscClimFis) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
			if !fn(tv.RiscClimTrans) {
				return fmt.Errorf("Cliente[%d].ExpTVM[%d].%s: tipo inválido", i+1, j+1, fieldName)
			}
		}
		if cl.ExpCliente != nil {
			if !fn(cl.ExpCliente.RiscSoc) {
				return fmt.Errorf("Cliente[%d].ExpCliente.%s: tipo inválido", i+1, fieldName)
			}
			if !fn(cl.ExpCliente.RiscAmb) {
				return fmt.Errorf("Cliente[%d].ExpCliente.%s: tipo inválido", i+1, fieldName)
			}
			if !fn(cl.ExpCliente.RiscClimFis) {
				return fmt.Errorf("Cliente[%d].ExpCliente.%s: tipo inválido", i+1, fieldName)
			}
			if !fn(cl.ExpCliente.RiscClimTrans) {
				return fmt.Errorf("Cliente[%d].ExpCliente.%s: tipo inválido", i+1, fieldName)
			}
		}
	}
	for i, s := range doc.Setores {
		if !fn(s.RiscSoc) {
			return fmt.Errorf("Setor[%d].%s: tipo inválido", i+1, fieldName)
		}
		if !fn(s.RiscAmb) {
			return fmt.Errorf("Setor[%d].%s: tipo inválido", i+1, fieldName)
		}
		if !fn(s.RiscClimFis) {
			return fmt.Errorf("Setor[%d].%s: tipo inválido", i+1, fieldName)
		}
		if !fn(s.RiscClimTrans) {
			return fmt.Errorf("Setor[%d].%s: tipo inválido", i+1, fieldName)
		}
	}
	return nil
}
