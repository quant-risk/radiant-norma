// Package sta — XML structs (request/response) para STA Web Services do BACEN.
//
// Sprint 18 (v3.8.0): tipos extraídos do Manual de utilização dos Web Services do STA
// (versão 1.5, julho/2022, BACEN). Cada struct tem doc-comment referenciando
// a seção do manual.
//
// Formato: UTF-8, XML body, namespace atom só em respostas paginadas.
//
// Referência: _referencias/STA_Manual_WebServices.pdf (42 páginas).
package sta

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
// (Seção 8.1.1). Usada em Sprint 19+ para download de arquivos BACEN→IF.
//
// Por enquanto apenas tipo vazio para satisfazer a interface — implementação
// completa fica para Sprint 19+ quando a query de arquivos a receber for
// integrada ao frontend.
type arquivosDisponiveisResponse struct {
	XMLName                 struct{} `xml:"Resultado"`
	DataHoraProximaConsulta string   `xml:"DataHoraProximaConsulta"`
}
