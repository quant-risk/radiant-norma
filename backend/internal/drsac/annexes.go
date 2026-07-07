// Annexes define os domínios válidos para cada campo do DRSAC.
// Baseado nos Anexos 01-20 da IN_BCB_222 e instruções de preenchimento.
//
//nolint:revive // godoc
package drsac

// Anexo 01 — Tipo de Envio
const (
	TipoEnvioInclusao     = "I"
	TipoEnvioSubstituicao = "S"
)

// ValidTipoEnvio verifica se o tipo de envio é válido.
func ValidTipoEnvio(v string) bool { return v == TipoEnvioInclusao || v == TipoEnvioSubstituicao }

// Anexo 02 — Sistema de Registro de TVM
var TipoSisReg = map[string]bool{
	"B3":    true, // B3 S.A.
	"CERC":  true, // CERC-RENDA FIXA
	"CSDBR": true, // CSD Brazil
	"Outro": true, // Outro sistema
}

// ValidSisReg verifica se o sistema de registro é válido.
func ValidSisReg(v string) bool { return TipoSisReg[v] }

// Anexo 03 — Tipo de TVM
var TipoTVM = map[string]bool{
	"CPR":   true, // Certificado de Produtor Rural
	"CDCA":  true, // Certificado de Depósito de Crédito Agro
	"CRA":   true, // Certificado de Recebíveis do Agronegócio
	"DEB":   true, // Debênture
	"Outro": true,
}

// ValidTipoTVM verifica se o tipo de TVM é válido.
func ValidTipoTVM(v string) bool { return TipoTVM[v] }

// Anexo 04 — Registro Sicor
var TipoSicor = map[string]bool{"S": true, "N": true}

// ValidSicor verifica se o valor Sicor é válido.
func ValidSicor(v string) bool { return TipoSicor[v] }

// Anexo 05 — Tipo de Cliente
const (
	TipoClientePF          = "01" // Pessoa Física
	TipoClientePJ          = "02" // Pessoa Jurídica
	TipoClienteEstrangeiro = "03"
	TipoClienteNaoIdentif  = "04"
	TipoClienteNaoRat      = "05"
	TipoClienteProdRural   = "06"
)

// ValidTipoCliente verifica se o tipo de cliente é válido.
func ValidTipoCliente(v string) bool {
	switch v {
	case TipoClientePF, TipoClientePJ, TipoClienteEstrangeiro,
		TipoClienteNaoIdentif, TipoClienteNaoRat, TipoClienteProdRural:
		return true
	}
	return false
}

// Anexo 06 — Tipos de Risco Social
var TipoRiscoSocial = map[string]bool{
	"01": true, // Violação de Convenções da OIT
	"02": true, // Trabalho infantil
	"03": true, // Trabalho análogo ao escravo
	"04": true, // Discriminação
	"05": true, // Violência e assédio
	"99": true, // Fora do escopo
}

// ValidTipoRiscoSocial verifica se o tipo de risco social é válido.
func ValidTipoRiscoSocial(v string) bool { return TipoRiscoSocial[v] }

// Anexo 07 — Tipos de Risco Ambiental
var TipoRiscoAmbiental = map[string]bool{
	"01": true, // Desmatamento ilegal
	"02": true, // Exploração ilegal de espécies
	"03": true, // Poluição
	"04": true, // Crimes contra flora
	"05": true, // Crimes contra fauna
	"06": true, // Poluição marinha
	"07": true, // Danos a área de proteção
	"08": true, // Substâncias controladas
	"09": true, // Outros crimes ambientais
	"99": true, // Fora do escopo
}

// ValidTipoRiscoAmbiental verifica se o tipo de risco ambiental é válido.
func ValidTipoRiscoAmbiental(v string) bool { return TipoRiscoAmbiental[v] }

// Anexo 08 — Tipos de Risco Climático Físico
var TipoRiscoClimaticoFisico = map[string]bool{
	"01": true, // Evento climático extremo
	"02": true, // Mudança gradual do clima
	"03": true, // Eventos hidrológicos
	"99": true, // Fora do escopo
}

// ValidTipoRiscoClimaticoFisico verifica se o tipo de risco climático físico é válido.
func ValidTipoRiscoClimaticoFisico(v string) bool { return TipoRiscoClimaticoFisico[v] }

// Anexo 09 — Avaliação de Risco (usado em todos os campos "av")
const (
	AvAlto        = "01"
	AvMedio       = "02"
	AvBaixo       = "03"
	AvIrrelevante = "04"
	AvNaoAvaliado = "98"
	AvForaEscopo  = "99"
)

// ValidAvaliacaoRisco verifica se o código de avaliação é válido.
func ValidAvaliacaoRisco(v string) bool {
	switch v {
	case AvAlto, AvMedio, AvBaixo, AvIrrelevante, AvNaoAvaliado, AvForaEscopo:
		return true
	}
	return false
}

// Anexo 10 — Classificação de Contribuição Positiva
var EnquadContribPositiva = map[string]bool{
	"01": true, // Fontaine et al. (2015)
	"02": true, // EU Taxonomy
	"03": true, // Classificação própria
	"98": true, // Não aplicável
	"99": true, // Fora do escopo
}

// ValidEnquadContribPositiva verifica se o enquadramento é válido.
func ValidEnquadContribPositiva(v string) bool { return EnquadContribPositiva[v] }

// Anexo 11 — Mitigador de Risco Climático Físico
const (
	MitigadorExiste    = "01"
	MitigadorNaoExiste = "02"
)

// ValidMitigadorClimFis verifica se o valor é válido.
func ValidMitigadorClimFis(v string) bool {
	switch v {
	case MitigadorExiste, MitigadorNaoExiste, AvNaoAvaliado, AvForaEscopo:
		return true
	}
	return false
}

// Anexo 12 — Tipos de Histórico de Absorção/Emissão GEE
var TipoHistAbsorvEmissGEE = map[string]bool{
	"01": true, // Escopo 1
	"02": true, // Escopo 2
	"03": true, // Escopo 3
	"04": true, // Compensação voluntária
	"05": true, // Remoção por sumidouros
	"99": true, // Fora do escopo
}

// ValidTipoHistAbsorvEmissGEE verifica se o tipo é válido.
func ValidTipoHistAbsorvEmissGEE(v string) bool { return TipoHistAbsorvEmissGEE[v] }

// Anexo 13 — Tipos de Expectativa de Absorção/Emissão GEE
var TipoExpAbsorvEmissGEE = map[string]bool{
	"01": true, // Cenário BAU
	"02": true, // Cenário Net Zero 2050
	"03": true, // Cenário NDC
	"99": true, // Fora do escopo
}

// ValidTipoExpAbsorvEmissGEE verifica se o tipo é válido.
func ValidTipoExpAbsorvEmissGEE(v string) bool { return TipoExpAbsorvEmissGEE[v] }

// Anexo 14 — Tipos de Compensação de Emissão GEE
var TipoCompEmissaoGEE = map[string]bool{
	"01": true, // Mercado regulado (CRC)
	"02": true, // Mercado voluntário
	"03": true, // Offset de natureza
	"99": true, // Fora do escopo
}

// ValidTipoCompEmissaoGEE verifica se o tipo é válido.
func ValidTipoCompEmissaoGEE(v string) bool { return TipoCompEmissaoGEE[v] }

// Anexo 15 — Status da Informação GEE
const (
	GEESitAbsorcao   = "01"
	GEESitEmissao    = "02"
	GEESitNaoAval    = "98"
	GEESitForaEscopo = "99"
)

// ValidGEESituacao verifica se a situação GEE é válida.
func ValidGEESituacao(v string) bool {
	switch v {
	case GEESitAbsorcao, GEESitEmissao, GEESitNaoAval, GEESitForaEscopo:
		return true
	}
	return false
}

// Anexo 16 — Tipos de Fatores Agravantes/Mitigadores
var TipoAgrMit = map[string]bool{
	"01": true, // Política de sustentabilidade
	"02": true, // Certificação ambiental
	"03": true, // Certificação social
	"04": true, // Prática de due diligence
	"05": true, // Trajetória de descarbonização
	"06": true, // Alinhamento com taxonomy
	"07": true, // Engajamento com comunidades
	"08": true, // Transparência hídrica
	"09": true, // Biodiversidade
	"10": true, //outro
}

// ValidTipoAgrMit verifica se o tipo é válido.
func ValidTipoAgrMit(v string) bool { return TipoAgrMit[v] }

// Anexo 17 — Status do Fator Agravante/Mitigador
const (
	AgrMitExiste    = "01"
	AgrMitNaoExiste = "02"
)

// ValidAgrMitStatus verifica se o status é válido.
func ValidAgrMitStatus(v string) bool {
	switch v {
	case AgrMitExiste, AgrMitNaoExiste, AvNaoAvaliado, AvForaEscopo:
		return true
	}
	return false
}

// Anexo 18 — Tipos de Risco Climático de Transição
var TipoRiscoClimaticoTransicao = map[string]bool{
	"01": true, // Risco de tecnologia
	"02": true, // Risco de reputação
	"03": true, // Risco de mercado
	"04": true, // Risco regulatório
	"99": true, // Fora do escopo
}

// ValidTipoRiscoClimaticoTransicao verifica se o tipo é válido.
func ValidTipoRiscoClimaticoTransicao(v string) bool { return TipoRiscoClimaticoTransicao[v] }

// Anexo 19 — Códigos de País (ISO 3166-1 numérico, 3 dígitos)
// Mapeamento parcial dos principais países do exposures DRSAC.
// Completo: consultar tabela ISO 3166-1.
var CodigosPaisISO = map[string]bool{
	"076": true, // Brasil
	"032": true, // Argentina
	"170": true, // Colômbia
	"484": true, // México
	"152": true, // Chile
	"604": true, // Peru
	"862": true, // Venezuela
	"858": true, // Uruguai
	"600": true, // Paraguai
	"218": true, // Equador
	"840": true, // Estados Unidos
	"124": true, // Canadá
	"250": true, // França
	"276": true, // Alemanha
	"380": true, // Itália
	"724": true, // Espanha
	"826": true, // Reino Unido
	"392": true, // Japão
	"156": true, // China
	"702": true, // Singapura
	"036": true, // Austrália
}

// ValidCodigoPais verifica se o código de país é válido.
func ValidCodigoPais(v string) bool { return CodigosPaisISO[v] }

// Anexo 20 — Tipos de Restrição Econômica
var TipoRestricaoEconomica = map[string]bool{
	"01": true, // Social
	"02": true, // Ambiental
}

// ValidTipoRestricaoEconomica verifica se o tipo de restrição é válido.
func ValidTipoRestricaoEconomica(v string) bool { return TipoRestricaoEconomica[v] }
