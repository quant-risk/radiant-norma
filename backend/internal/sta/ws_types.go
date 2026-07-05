// Package sta — XML structs (request/response) para STA Web Services do BACEN.
//
// Sprint 18 (v3.8.0): tipos extraídos do Manual de utilização dos Web Services do STA
// (versão 1.5, julho/2022, BACEN). Cada struct tem doc-comment referenciando
// a seção do manual.
//
// Sprint 19 (v3.9.0): read side — UploadStatus, Range, UploadSituacao, DownloadResult.
//
// Sprint 20 (v3.10.0): listagem + alteração de situação — ListDisponiveisOpts,
// ListDisponiveisResult, ArquivoDisponivel, SituacaoArquivo, AlterarSituacaoReq,
// SituacaoTransferencia.
//
// Formato: UTF-8, XML body, namespace atom só em respostas paginadas.
//
// Referência: _referencias/STA_Manual_WebServices.pdf (42 páginas).
package sta

import (
	"strconv"
	"strings"
)

// requestProtocolParams é o corpo do POST /arquivos (Seção 5.1.1 do manual).
//
// Onde:
//   - IdentificadorDocumento: nome ou código de documento (ex.: "3040" para CADOC SCR)
//   - Hash: SHA-256 hexadecimal 64 chars sobre conteúdo compactado
//   - Tamanho: tamanho em bytes do conteúdo compactado
//   - NomeArquivo: nome original do arquivo
//   - Observacao: opcional
//   - Destinatarios: opcional (ver Seção 5.1.2 — lista de Unidade/Dependencia/Operador)
type requestProtocolParams struct {
	XMLName                struct{} `xml:"Parametros"`
	IdentificadorDocumento string   `xml:"IdentificadorDocumento"`
	Hash                   string   `xml:"Hash"`
	Tamanho                int64    `xml:"Tamanho"`
	NomeArquivo            string   `xml:"NomeArquivo"`
	Observacao             string   `xml:"Observacao,omitempty"`
	Destinatarios          string   `xml:"Destinatarios,omitempty"`
}

// responseProtocol é a resposta do POST /arquivos 201 Created (Seção 5.1.1).
//
// Onde:
//   - Protocolo: número do protocolo gerado para a transmissão
//   - atom:link href: URL para PUT do conteúdo
type responseProtocol struct {
	XMLName   struct{} `xml:"Resultado"`
	Protocolo string   `xml:"Protocolo"`
	Link      struct {
		HRef string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
}

// xmlError é a resposta de erro do BACEN em formato XML (Listagem 4 do manual).
//
// Format: <Resultado><Erro><Codigo>STATUS</Codigo><Descricao>MSG</Descricao></Erro></Resultado>
type xmlError struct {
	XMLName struct{} `xml:"Resultado"`
	Erro    struct {
		Codigo    int    `xml:"Codigo"`
		Descricao string `xml:"Descricao"`
	} `xml:"Erro"`
}

// posicaoUploadResponse é a resposta do GET /arquivos/{protocolo}/posicaoupload
// (Seção 5.3.1).
//
// Onde:
//   - Protocolo: número do protocolo
//   - RangesRecebidos: lista separada por ";" com pares inicio-fim (e.g., "0-3;5-8")
//   - Situacao: "Transmissão não iniciada" | "Transmissão finalizada" | "Transmissão pendente"
type posicaoUploadResponse struct {
	XMLName         struct{} `xml:"Resultado"`
	Protocolo       string   `xml:"Protocolo"`
	RangesRecebidos string   `xml:"RangesRecebidos"`
	Situacao        string   `xml:"Situacao"`
}

// situacaoParams é o corpo do PUT /arquivos/situacao (Seção 7.1).
//
// Onde:
//   - Protocolos: lista de protocolos separados por ";"
//   - Situacao: "A_REC" (a receber) ou "REC" (recebido)
type situacaoParams struct {
	XMLName    struct{} `xml:"Parametros"`
	Protocolos string   `xml:"Protocolos"`
	Situacao   string   `xml:"Situacao"`
}

// arquivosDisponiveisResponse — resposta do GET /arquivos/disponiveis
// (Seção 8.1.1). Resposta paginada: até 1000 protocolos por chamada.
// Se >1000, retorna <atom:link href="..." rel="disponiveis"/> com URL da próxima página.
//
// Onde (linhas 887-932 do manual):
//   - DataHoraProximaConsulta: data-hora para próxima polling (formato "yyyy-MM-ddTHH:mm:ss.SSS")
//   - Arquivo[]: lista de arquivos com metadados completos
//   - atom:link: URL para próxima página (string vazia se <1000 resultados)
type arquivosDisponiveisResponse struct {
	XMLName                 struct{} `xml:"Resultado"`
	DataHoraProximaConsulta string   `xml:"DataHoraProximaConsulta"`
	Arquivos                []struct {
		Protocolo       string `xml:"Protocolo"`
		TipoArquivo     string `xml:"TipoArquivo"`
		CodigoDocumento string `xml:"CodigoDocumento"`
		Sistema         string `xml:"Sistema"`
		TamanhoArquivo  int64  `xml:"TamanhoArquivo"`
		Hash            string `xml:"Hash"`
		SituacaoAtual   struct {
			Codigo    int    `xml:"Codigo"`
			Descricao string `xml:"Descricao"`
		} `xml:"SituacaoAtual"`
		DataHoraDisponibilizacao string `xml:"DataHoraDisponibilizacao"`
	} `xml:"Arquivo"`
	Link struct {
		HRef string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
}

// ============================================================
// Sprint 19 (v3.9.0) — read side: Download + StatusUpload
// ============================================================

// UploadStatus é o resultado tipado de WSClient.StatusUpload (Seção 5.3.1).
//
// RangesRecebidos é uma lista de pares [start,end] interpretados do campo
// `RangesRecebidos` do XML (formato cru: "0-3;5-8"). Lista vazia significa
// "transmissão não iniciada".
//
// Situacao é o enum tipado dos 3 valores documentados (linha 470-475 do manual).
// SituacaoRaw guarda o string original do XML para audit/debug — defesa contra
// BACEN adicionar valor novo sem atualizar a IF.
type UploadStatus struct {
	Protocolo       string
	RangesRecebidos []Range
	Situacao        UploadSituacao
	SituacaoRaw     string
}

// Range é um par [start, end] interpretado de RangesRecebidos.
// Inclusivo em ambos os extremos — "0-3" significa bytes 0,1,2,3 (4 bytes).
type Range struct {
	Start int64
	End   int64
}

// UploadSituacao é o enum tipado dos 3 valores oficiais do manual (5.3.1,
// linha 470-475). Usar constant em vez de string comparison evita typos
// silenciosos e dá type-safety pro compilador.
type UploadSituacao int

const (
	// UploadSituacaoUnknown é default zero-value — só acontece se BACEN mandar
	// valor não mapeado. Caller deve checar != Unknown antes de usar.
	UploadSituacaoUnknown UploadSituacao = iota
	// UploadSituacaoNaoIniciada — "Transmissão não iniciada".
	UploadSituacaoNaoIniciada
	// UploadUploadPendente — "Transmissão pendente".
	UploadUploadPendente
	// UploadSituacaoFinalizada — "Transmissão finalizada".
	UploadSituacaoFinalizada
)

// String retorna a string canônica do manual (pt-BR) para logs e audit.
func (u UploadSituacao) String() string {
	switch u {
	case UploadSituacaoNaoIniciada:
		return "Transmissão não iniciada"
	case UploadUploadPendente:
		return "Transmissão pendente"
	case UploadSituacaoFinalizada:
		return "Transmissão finalizada"
	default:
		return "Desconhecida"
	}
}

// parseUploadSituacao mapeia string cru do XML → enum tipado.
// Valores fora dos 3 documentados viram Unknown (não erro — caller decide).
func parseUploadSituacao(s string) UploadSituacao {
	switch s {
	case "Transmissão não iniciada":
		return UploadSituacaoNaoIniciada
	case "Transmissão pendente":
		return UploadUploadPendente
	case "Transmissão finalizada":
		return UploadSituacaoFinalizada
	default:
		return UploadSituacaoUnknown
	}
}

// parseRanges interpreta "0-3;5-8" → []Range{{0,3},{5,8}}.
// Ranges malformados são silenciosamente descartados (defense — BACEN mandar
// lixo não pode crashar cliente). Logs/debug ficam em SituacaoRaw.
func parseRanges(s string) []Range {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	out := make([]Range, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		se := strings.SplitN(p, "-", 2)
		if len(se) != 2 {
			continue
		}
		start, err1 := strconv.ParseInt(strings.TrimSpace(se[0]), 10, 64)
		end, err2 := strconv.ParseInt(strings.TrimSpace(se[1]), 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		if start < 0 || end < start {
			continue
		}
		out = append(out, Range{Start: start, End: end})
	}
	return out
}

// DownloadResult é o resultado de WSClient.Download (Seção 6.1.1).
//
// Conteúdo é o body binário (ZIP típico). ContentHash é o SHA-256 hex dos
// próprios bytes — cross-check com header X-Content-Hash do BACEN (validação
// obrigatória de integridade conforme manual linha 641-643).
//
// ETag + LastModified + ContentHashHeader são os headers crus do BACEN, úteis
// para audit trail e (futuramente) range/conditional download.
//
// LastModified fica como string crua porque BACEN não documenta formato exato
// (provavelmente RFC 1123 mas não é garantido).
type DownloadResult struct {
	Conteudo          []byte
	ContentHash       string // SHA-256 hex computado pelo cliente (defesa contra BACEN bug)
	ETag              string
	LastModified      string
	ContentHashHeader string // valor cru de X-Content-Hash (debug/audit)
}

// ============================================================
// Sprint 20 (v3.10.0) — listagem / disponiveis + alteracao /situacao
// ============================================================

// ListDisponiveisOpts são os parâmetros de WSClient.ListDisponiveis (Tabela 4).
//
// DataHoraInicio é OBRIGATÓRIO (Tabela 4 linha 1472). Formato esperado pelo
// BACEN: "yyyy-MM-ddTHH:mm:ss.SSS". Cliente não valida formato — BACEN
// responde 400 Listagem 4 se caller passar string malformada.
//
// IdentificadorDocumento, Sistemas, Dependencia são opcionais. Sistemas pode
// ter até 100 entries separados por ";".
type ListDisponiveisOpts struct {
	DataHoraInicio         string
	IdentificadorDocumento string
	Sistemas               string
	Dependencia            string
}

// ListDisponiveisResult é o retorno de WSClient.ListDisponiveis.
//
// DataHoraProximaConsulta é o eco do XML; frontend usa pra polling incremental.
// ProximaPaginaURL é a URL completa (?dependencia=...&dataHoraInicio=...) que
// BACEN sugere pra próxima página — presente apenas se >1000 registros.
// TemProximaPagina é true se atom:link estiver presente no XML.
type ListDisponiveisResult struct {
	Arquivos                []ArquivoDisponivel
	DataHoraProximaConsulta string
	ProximaPaginaURL        string
	TemProximaPagina        bool
}

// ArquivoDisponivel é um arquivo da listagem (Seção 8.1.1 XML resposta).
type ArquivoDisponivel struct {
	Protocolo                string
	TipoArquivo              string
	CodigoDocumento          string
	Sistema                  string
	TamanhoArquivo           int64
	Hash                     string // SHA-256 hex
	SituacaoAtual            SituacaoArquivo
	SituacaoAtualRaw         string // defesa contra BACEN adicionar valor
	DataHoraDisponibilizacao string // formato cru BACEN (yyyy-MM-ddTHH:mm:ss.SSS)
}

// SituacaoArquivo é o enum tipado dos valores de SituacaoAtual do XML.
// Codigo 3 = "A receber" (confirmado em múltiplos exemplos manual linhas
// 897-925, 1007-1020, 1639-1655). Codigo 1 = "Recebido" (inferido — único
// outro valor consistente com §7.1 que usa "A_REC"/"REC").
//
// Manual não documenta tabela oficial mapeando Codigo ↔ Descricao para
// SituacaoAtual (Tabela 3 cobre EstadoAtual, que é diferente). Caller tem
// SituacaoAtualRaw pra audit/debug quando BACEN adicionar valor novo.
type SituacaoArquivo int

const (
	SituacaoArquivoUnknown SituacaoArquivo = iota
	// SituacaoArquivoRecebido — Codigo 1.
	SituacaoArquivoRecebido
	// SituacaoArquivoAReceber — Codigo 3 (confirmado manual).
	SituacaoArquivoAReceber
)

// String retorna "Recebido" / "A receber" para logs/JSON.
func (s SituacaoArquivo) String() string {
	switch s {
	case SituacaoArquivoRecebido:
		return "Recebido"
	case SituacaoArquivoAReceber:
		return "A receber"
	default:
		return "Desconhecida"
	}
}

// parseSituacaoArquivo mapeia codigo numérico do XML → enum tipado.
// Codigos fora de {1, 3} viram Unknown (defesa contra BACEN adicionar).
func parseSituacaoArquivo(codigo int) SituacaoArquivo {
	switch codigo {
	case 1:
		return SituacaoArquivoRecebido
	case 3:
		return SituacaoArquivoAReceber
	default:
		return SituacaoArquivoUnknown
	}
}

// AlterarSituacaoReq é o request de WSClient.AlterarSituacao (Seção 7.1).
type AlterarSituacaoReq struct {
	Protocolos []string
	Situacao   SituacaoTransferencia // enum tipado
}

// SituacaoTransferencia é o enum tipado dos valores oficiais de Situacao
// (Seção 7.1 linha 799-801).
type SituacaoTransferencia int

const (
	SituacaoTransferenciaUnknown SituacaoTransferencia = iota
	// SituacaoTransferenciaAReceber — "A_REC".
	SituacaoTransferenciaAReceber
	// SituacaoTransferenciaRecebido — "REC".
	SituacaoTransferenciaRecebido
)

// String retorna "A_REC" / "REC" para mandar no XML (manual linha 789).
func (s SituacaoTransferencia) String() string {
	switch s {
	case SituacaoTransferenciaAReceber:
		return "A_REC"
	case SituacaoTransferenciaRecebido:
		return "REC"
	default:
		return ""
	}
}

// parseSituacaoTransferencia mapeia string do XML → enum. Manual só define
// "A_REC" e "REC" (linha 799-801). Valores fora viram Unknown.
func parseSituacaoTransferencia(s string) SituacaoTransferencia {
	switch s {
	case "A_REC":
		return SituacaoTransferenciaAReceber
	case "REC":
		return SituacaoTransferenciaRecebido
	default:
		return SituacaoTransferenciaUnknown
	}
}

// ParseSituacaoTransferencia é a versão pública de parseSituacaoTransferencia,
// para uso por handlers REST que precisam converter string JSON → enum tipado
// antes de chamar WSClient.AlterarSituacao (Seção 7.1).
//
// Valores fora de {"A_REC", "REC"} retornam SituacaoTransferenciaUnknown — não
// retorna erro. Caller deve validar manualmente se quiser fail-fast em input
// desconhecido.
func ParseSituacaoTransferencia(s string) SituacaoTransferencia {
	return parseSituacaoTransferencia(s)
}
