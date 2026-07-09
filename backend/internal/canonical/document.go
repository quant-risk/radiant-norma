// Package canonical define o modelo canônico — representação intermediária
// IF-agnóstica usada entre conectores (SourceAdapter) e geradores (CADOCGenerator).
//
// Este modelo é o contrato central da arquitetura de geração:
//   - LLM nunca escreve XML direto: escreve CanonicalDocument
//   - Engine de geração (Generator) serializa Canonical → XML
//   - Validador (L1-L4) consome o XML gerado para audit trail
//
// A separação permite que cada gerador CADOC seja independente da fonte
// de dados (Manual, File, API, DB, MCP).
package canonical

import (
	"time"

	"github.com/shopspring/decimal"
)

// CadocType representa o código CADOC (ex: "3040", "3050", "4111").
type CadocType string

// DataBase é o ponto no tempo (date-only) usado como referência para
// todos os campos do documento. BACEN opera em bases mensais.
type DataBase time.Time

// CanonicalDocument é a representação intermediária de um documento CADOC.
// É IF-agnástica: o mesmo modelo serve para qualquer instituição,
// independenteme da fonte de dados.
//
// O modelo é JSON-tipado, com campos universais (Header, Participantes,
// Operacoes) e um mapa Extra para campos específicos de cada CADOC.
type CanonicalDocument struct {
	// IFID identifica a instituição financeira dona deste documento.
	IFID string `json:"if_id"`

	// DataBase é a data-base de referência do documento (mensal).
	DataBase DataBase `json:"data_base"`

	// CadocCode é o código CADOC (ex: "3040", "3050", "4111").
	CadocCode CadocType `json:"cadoc_code"`

	// VersaoLayout é a versão do leiaute BACEN usado.
	// Ex: "3.2", "4.0". Deve corresponder a uma versão no Schema Registry.
	VersaoLayout string `json:"versao_layout"`

	// Natureza indica o tipo de operação: "N" (Nova), "R" (Renovação),
	// "A" (Adição), "C" (Cancelamento).
	Natureza string `json:"natureza"`

	// Header contém metadados universais do documento.
	Header DocumentHeader `json:"header"`

	// Participantes lista todos os participantes do documento.
	// Inclui a própria IF e contrapartes (clientes, cooperativas, etc).
	Participantes []Participante `json:"participantes"`

	// Operacoes é a lista de operações/negócios do documento.
	// Campos universais; campos específicos vão em Extra.
	Operacoes []Operacao `json:"operacoes"`

	// Extra carrega campos específicos de cada CADOC que não são
	// universais. A chave é o nome do campo COSIF ou BACEN.
	// Ex para 3040: {"indice": "SCR", "modalidade": "PF"}
	Extra map[string]any `json:"extra,omitempty"`

	// Metadata contém informação de auditoria (não vai para o XML).
	Metadata DocMetadata `json:"metadata,omitempty"`
}

// DocumentHeader são metadados universais de qualquer documento CADOC.
type DocumentHeader struct {
	// DataHoraGeracao é quando o documento foi gerado pelo sistema.
	DataHoraGeracao time.Time `json:"data_hora_geracao"`

	// DataBaseInformacao é a data-base da informação (pode diferir de
	// DataBase em casos de retificação).
	DataBaseInformacao DataBase `json:"data_base_informacao"`

	// CNPJ é o CNPJ da IF geradora.
	CNPJ string `json:"cnpj"`

	// NomeIF é a razão social da IF.
	NomeIF string `json:"nome_if"`

	// NumeroVersao é o número sequencial de versões do documento.
	NumeroVersao int `json:"numero_versao,omitempty"`

	// CodigoIdentificacao é o código único do documento no sistema BACEN.
	CodigoIdentificacao string `json:"codigo_identificacao,omitempty"`
}

// Participante representa um ator em uma operação CADOC.
// Pode ser a própria IF, um cliente, cooperativa, ou outra contraparte.
type Participante struct {
	// Tipo define o papel do participante.
	// Valores comuns: "IF", "CLIENTE", "COOPERATIVA", "CORRESPONDENTE",
	// "FILIAL", "SEDE".
	Tipo string `json:"tipo"`

	// CNPJ ou CPF do participante (sem máscara).
	CNPJ string `json:"cnpj"`

	// Nome é a razão social ou nome fantasia.
	Nome string `json:"nome"`

	// Modalidade é o código de modalidade BACEN (ex: "1.2.3.4.5").
	// Varia por CADOC; usar Extra no CanonicalDocument para campos
	// específicos.
	Modalidade string `json:"modalidade,omitempty"`

	// UF é a unidade federativa do participante (ex: "SP", "RJ").
	UF string `json:"uf,omitempty"`

	// Classe é a classificação do participante (ex: "PJ", "PF").
	Classe string `json:"classe,omitempty"`

	// NumeroInscricao é o número de inscrição (ex: CNPJ, CPF, NIS).
	NumeroInscricao string `json:"numero_inscricao,omitempty"`

	// Ratings lista os ratings de crédito do participante.
	Ratings []Rating `json:"ratings,omitempty"`

	// Extra campos específicos do participante que vão para o XML.
	Extra map[string]any `json:"extra,omitempty"`
}

// Rating representa uma classificação de risco de crédito.
type Rating struct {
	// NomeAgencia é o nome da agência de rating (ex: "S&P", "Fitch", "Moody's").
	NomeAgencia string `json:"nome_agencia"`

	// Nota é a classificação (ex: "AAA", "BB+", "B3").
	Nota string `json:"nota"`

	// DataRating é a data em que a nota foi atribuída.
	DataRating time.Time `json:"data_rating,omitempty"`

	// Escala indica se é escala nacional ou internacional.
	// Valores: "NACIONAL", "INTERNACIONAL".
	Escala string `json:"escala,omitempty"`
}

// Operacao representa uma operação de crédito ou negócio no documento.
type Operacao struct {
	// ID é o identificador único da operação no sistema de origem.
	ID string `json:"id"`

	// TipoOperacao classifica a operação (ex: "C", "CFL", "D".
	// C = Crédito, CFL = Crédito com Flight, D = Depósito.
	TipoOperacao string `json:"tipo_operacao"`

	// Modalidade é o código COSIF da modalidade (ex: "1.2.1.4.02").
	Modalidade string `json:"modalidade"`

	// NumeroContrato é o número do contrato de crédito.
	NumeroContrato string `json:"numero_contrato,omitempty"`

	// TipoPessoa é o tipo de pessoa do devedor: "PF" ou "PJ".
	// Usado no 3040 para o campo tpCli (1=PF, 2=PJ).
	TipoPessoa string `json:"tipo_pessoa,omitempty"`

	// DataConstituicao é a data de constituição da operação.
	DataConstituicao time.Time `json:"data_constituicao,omitempty"`

	// DataVencimento é a data de vencimento da operação.
	DataVencimento time.Time `json:"data_vencimento,omitempty"`

	// DataBaseCalculo é a data-base para cálculo (pode ser diferente da
	// data-base do documento em casos de restructure).
	DataBaseCalculo DataBase `json:"data_base_calculo,omitempty"`

	// ValorPrincipal é o valor principal (sem encargos).
	ValorPrincipal Money `json:"valor_principal"`

	// EncargosTotais soma de juros + multa + mora + outros encargos.
	EncargosTotais Money `json:"encargos_totais,omitempty"`

	// IOF é o valor do IOF cobrado.
	IOF Money `json:"iof,omitempty"`

	// ValorAtualizado é o valor total atualizado (principal + encargos + IOF).
	ValorAtualizado Money `json:"valor_atualizado,omitempty"`

	// TaxaJuros é a taxa de juros efetiva ao mês (em decimal, ex: 0.015 = 1.5%).
	TaxaJuros decimal.Decimal `json:"taxa_juros,omitempty"`

	// TaxaSpread é o spread sobre o referencial (em decimal).
	TaxaSpread decimal.Decimal `json:"taxa_spread,omitempty"`

	// Indexador é o indexador da operação (ex: "CDI", "IPCA", "TJLP", "PRE").
	Indexador string `json:"indexador,omitempty"`

	// PercentualIndexador é o percentual do indexador (ex: 0.9 = 90% do CDI).
	PercentualIndexador decimal.Decimal `json:"percentual_indexador,omitempty"`

	// FaixaVencimento indica a faixa de vencimento (BACEN define faixas).
	// Valores: "V0" (até 3m), "V1" (3-6m), "V2" (6-12m), "V3" (1-3a),
	// "V4" (3-5a), "V5" (mais 5a).
	FaixaVencimento string `json:"faixa_vencimento,omitempty"`

	// NivelRisco é o nível de risco da operação (ex: "AA", "A", "B", "C").
	NivelRisco string `json:"nivel_risco,omitempty"`

	// PercentualRisco é o percentual de provisãoRequired (ex: 0.5 = 50%).
	PercentualProvisao decimal.Decimal `json:"percentual_provisao,omitempty"`

	// ClassificacaoIF é a classificação interna da IF (ex: "ES", "GH", "F").
	ClassificacaoIF string `json:"clasificacao_if,omitempty"`

	// UF is the state of the operation. May differ from participant UF.
	UF string `json:"uf,omitempty"`

	// Pais is the country code (ISO 3166-1 alpha-2).
	Pais string `json:"pais,omitempty"`

	// Ratings da operação (not just participant).
	Ratings []Rating `json:"ratings,omitempty"`

	// Garantias lista as garantias da operação.
	Garantias []Garantia `json:"garantias,omitempty"`

	// Extra campos específicos que vão para o XML.
	Extra map[string]any `json:"extra,omitempty"`
}

// Garantia representa uma garantia associada a uma operação de crédito.
type Garantia struct {
	// Tipo de garantia: "REAL", "PESSOAL", "FIDEJUSSORIA", "SEGURO".
	Tipo string `json:"tipo"`

	// Modalidade da garantia (ex: "HIPOTECA", "PENHOR", "AVAL").
	Modalidade string `json:"modalidade"`

	// ValorGarantisado é o valor coberto pela garantia.
	ValorGarantido Money `json:"valor_garantido,omitempty"`

	// DataAvaliacao é quando a garantia foi avaliada.
	DataAvaliacao time.Time `json:"data_avaliacao,omitempty"`

	// Descricao é uma descrição livre da garantia.
	Descricao string `json:"descricao,omitempty"`
}

// Money representa um valor monetário em reais (BRL).
type Money struct {
	// Valor é o valor numérico.
	Valor decimal.Decimal `json:"valor"`

	// Moeda é sempre "BRL" para documentos BACEN.
	Moeda string `json:"moeda"`

	// Divergente indica se o valor foi ajustado pelo validador.
	Divergente bool `json:"divergente,omitempty"`

	// ValorOriginal é o valor antes de ajuste (para auditoria).
	ValorOriginal decimal.Decimal `json:"valor_original,omitempty"`
}

// DocMetadata é usada para auditoria e tracing. Não é serializada no XML.
type DocMetadata struct {
	// SourceAdapter identifica qual conector gerou o documento.
	SourceAdapter string `json:"source_adapter,omitempty"`

	// SourceRef é a referência no sistema de origem (ex: ID da planilha).
	SourceRef string `json:"source_ref,omitempty"`

	// GeneratedAt é quando o CanonicalDocument foi criado.
	GeneratedAt time.Time `json:"generated_at,omitempty"`

	// GeneratedBy identifica o agente (usuário ou sistema) que gerou.
	GeneratedBy string `json:"generated_by,omitempty"`

	// Version é a versão do CanonicalDocument (semântica).
	Version string `json:"version,omitempty"`
}

// NewCanonical cria um CanonicalDocument com valores padrão.
func NewCanonical(ifID string, dataBase time.Time, cadocCode CadocType) *CanonicalDocument {
	return &CanonicalDocument{
		IFID:      ifID,
		DataBase:  DataBase(dataBase),
		CadocCode: cadocCode,
		Extra:     make(map[string]any),
		Metadata: DocMetadata{
			GeneratedAt: time.Now(),
		},
	}
}

// Validate faz validação básica do CanonicalDocument.
// Retorna erros de campos obrigatórios ausentes.
// Não substitui a validação L1-L4 (que opera no XML gerado).
func (c *CanonicalDocument) Validate() []string {
	var errs []string

	if c.IFID == "" {
		errs = append(errs, "IFID é obrigatório")
	}
	if c.CadocCode == "" {
		errs = append(errs, "CadocCode é obrigatório")
	}
	if time.Time(c.DataBase).IsZero() {
		errs = append(errs, "DataBase é obrigatória")
	}
	if c.Header.CNPJ == "" {
		errs = append(errs, "Header.CNPJ é obrigatório")
	}
	if c.Header.NomeIF == "" {
		errs = append(errs, "Header.NomeIF é obrigatório")
	}

	return errs
}
