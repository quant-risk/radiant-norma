// Sprint 57 v3.34.37: NormaGeneratorFoundation — source adapters.
package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SourceAdapter é a interface que todo connector deve implementar.
type SourceAdapter interface {
	Fetch(ctx context.Context, cadocCode, dataBase string) (*CanonicalDoc, error)
	AdapterName() string
}

// CanonicalDoc é o documento canônico retornado pelo adapter.
type CanonicalDoc struct {
	IFID         string         `json:"if_id"`
	CadocCode    string         `json:"cadoc_code"`
	DataBase     string         `json:"data_base"`
	VersaoLayout string         `json:"versao_layout"`
	Participantes []Participante `json:"participantes,omitempty"`
	Operacoes    []Operacao     `json:"operacoes,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
}

// Participante representa um participante no documento.
type Participante struct {
	Tipo string `json:"tipo"`
	CNPJ string `json:"cnpj,omitempty"`
	CPF  string `json:"cpf,omitempty"`
	Nome string `json:"nome"`
	CNAE string `json:"cnae,omitempty"`
	IsIF bool   `json:"is_if"`
}

// Operacao representa uma operação no documento.
type Operacao struct {
	ID         string            `json:"id"`
	TipoPessoa string            `json:"tipo_pessoa"`
	Modalidade string            `json:"modalidade"`
	Valor      float64           `json:"valor"`
	Moeda      string            `json:"moeda"`
	Indexador  string            `json:"indexador,omitempty"`
	TaxaJuros  float64           `json:"taxa_juros,omitempty"`
	Vencimento string            `json:"vencimento,omitempty"`
	UF         string            `json:"uf,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}

// ============================================================
// DBAdapter — PostgreSQL → CanonicalDocument
// ============================================================

// DBAdapter consulta o DB diretamente para construir o CanonicalDocument.
type DBAdapter struct {
	db *sql.DB
}

// NewDBAdapter cria um DBAdapter.
func NewDBAdapter(db *sql.DB) *DBAdapter {
	return &DBAdapter{db: db}
}

// AdapterName implements SourceAdapter.
func (a *DBAdapter) AdapterName() string { return "db" }

// Fetch consulta o DB para dados do CADOC mais recente.
// Se dataBase não for vazio, filtra pela data-base específica.
func (a *DBAdapter) Fetch(ctx context.Context, cadocCode, dataBase string) (*CanonicalDoc, error) {
	var rows *sql.Rows
	var err error
	if dataBase != "" {
		rows, err = a.db.QueryContext(ctx, `
			SELECT e.id, e.if_id, e.data_base, e.xml_content, e.status
			FROM envios e
			WHERE e.cadoc_code = $1 AND e.data_base = $2 AND e.deleted_at IS NULL
			ORDER BY e.created_at DESC
			LIMIT 1
		`, cadocCode, dataBase)
	} else {
		rows, err = a.db.QueryContext(ctx, `
			SELECT e.id, e.if_id, e.data_base, e.xml_content, e.status
			FROM envios e
			WHERE e.cadoc_code = $1 AND e.deleted_at IS NULL
			ORDER BY e.created_at DESC
			LIMIT 1
		`, cadocCode)
	}
	if err != nil {
		return nil, fmt.Errorf("DBAdapter.Fetch: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("DBAdapter: nenhum envio encontrado para CADOC %s", cadocCode)
	}

	var id, ifID, dbDataBase, xmlContent, status string
	if err := rows.Scan(&id, &ifID, &dbDataBase, &xmlContent, &status); err != nil {
		return nil, fmt.Errorf("DBAdapter.Scan: %w", err)
	}

	return &CanonicalDoc{
		IFID:         ifID,
		CadocCode:    cadocCode,
		DataBase:     dbDataBase,
		VersaoLayout: "1.0",
		Extra:        map[string]any{"source_envio_id": id, "status": status},
	}, nil
}

// ============================================================
// APIAdapter — REST API → CanonicalDocument
// ============================================================

// APIAdapter chama APIs internas para obter dados.
type APIAdapter struct {
	BaseURL    string
	HTTPClient HTTPClient
}

// HTTPClient interface for HTTP calls.
type HTTPClient interface {
	Do(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error)
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Body       []byte
}

// NewAPIAdapter cria um APIAdapter com cliente HTTP default.
func NewAPIAdapter(baseURL string) *APIAdapter {
	return &APIAdapter{
		BaseURL:    baseURL,
		HTTPClient: NewDefaultHTTPClient(),
	}
}

// AdapterName implements SourceAdapter.
func (a *APIAdapter) AdapterName() string { return "api" }

// Fetch chama a API interna para obter dados do CADOC.
func (a *APIAdapter) Fetch(ctx context.Context, cadocCode, dataBase string) (*CanonicalDoc, error) {
	// Stub: GET /internal/cadoc/{cadocCode}/canonical
	u, _ := url.Parse(a.BaseURL + "/internal/cadoc/" + cadocCode + "/canonical")
	q := u.Query()
	if dataBase != "" {
		q.Set("data_base", dataBase)
	}
	u.RawQuery = q.Encode()
	resp, err := a.HTTPClient.Do(ctx, http.MethodGet, u.String(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("APIAdapter.Fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("APIAdapter: %d response from %s", resp.StatusCode, u.String())
	}
	var doc CanonicalDoc
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return nil, fmt.Errorf("APIAdapter: parse error: %w", err)
	}
	return &doc, nil
}

// DefaultHTTPClient is the default HTTP client implementation.
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a default HTTP client.
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{client: &http.Client{Timeout: 30 * time.Second}}
}

// Do implements HTTPClient.
func (d *DefaultHTTPClient) Do(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return &Response{StatusCode: resp.StatusCode, Body: bodyBytes}, nil
}

// ============================================================
// MCPAdapter — MCP tool → CanonicalDocument
// ============================================================

// MCPAdapter chama MCP tools para obter dados.
type MCPAdapter struct {
	ToolName string
	CallFunc func(ctx context.Context, tool, args string) (string, error)
}

// NewMCPAdapter cria um MCPAdapter.
func NewMCPAdapter(toolName string, callFunc func(ctx context.Context, tool, args string) (string, error)) *MCPAdapter {
	return &MCPAdapter{ToolName: toolName, CallFunc: callFunc}
}

// AdapterName implements SourceAdapter.
func (a *MCPAdapter) AdapterName() string { return "mcp" }

// Fetch chama a tool MCP para obter dados do CADOC.
func (a *MCPAdapter) Fetch(ctx context.Context, cadocCode, dataBase string) (*CanonicalDoc, error) {
	if a.CallFunc == nil {
		return nil, fmt.Errorf("MCPAdapter.Fetch: no call function configured")
	}
	args := fmt.Sprintf(`{"cadoc_code": %q, "data_base": %q}`, cadocCode, dataBase)
	result, err := a.CallFunc(ctx, a.ToolName, args)
	if err != nil {
		return nil, fmt.Errorf("MCPAdapter.Fetch: %w", err)
	}
	var doc CanonicalDoc
	if err := json.Unmarshal([]byte(result), &doc); err != nil {
		return nil, fmt.Errorf("MCPAdapter: failed to parse result: %w", err)
	}
	return &doc, nil
}

// ============================================================
// ManualAdapter — UI input → CanonicalDocument (Sprint 57 v3.36.3)
// ============================================================

// ManualAdapter recebe um CanonicalDocument pré-montado pela UI do wizard.
// Não busca dados — apenas retorna o doc que foi injetado.
//
// Use case: usuário preenche formulário no wizard de geração (frontend)
// → POST com payload JSON contendo o CanonicalDocument pronto → ManualAdapter
// apenas valida e retorna para o generator processar.
type ManualAdapter struct {
	// Doc é o CanonicalDocument injetado no construtor (thread-safe: read-only).
	Doc *CanonicalDoc
}

// NewManualAdapter cria um adapter com o doc pré-montado.
func NewManualAdapter(doc *CanonicalDoc) *ManualAdapter {
	return &ManualAdapter{Doc: doc}
}

// NewManualAdapterFromJSON cria um adapter a partir de um payload JSON.
func NewManualAdapterFromJSON(payload []byte) (*ManualAdapter, error) {
	var doc CanonicalDoc
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("ManualAdapter: parse JSON: %w", err)
	}
	if doc.CadocCode == "" {
		return nil, fmt.Errorf("ManualAdapter: cadoc_code obrigatório")
	}
	return &ManualAdapter{Doc: &doc}, nil
}

// AdapterName implements SourceAdapter.
func (a *ManualAdapter) AdapterName() string { return "manual" }

// Fetch retorna o doc injetado, validando cadoc_code contra o argumento.
// dataBase é opcional — se fornecida, sobrescreve doc.DataBase.
func (a *ManualAdapter) Fetch(_ context.Context, cadocCode, dataBase string) (*CanonicalDoc, error) {
	if a.Doc == nil {
		return nil, fmt.Errorf("ManualAdapter: no doc injected")
	}
	if a.Doc.CadocCode != "" && a.Doc.CadocCode != cadocCode {
		return nil, fmt.Errorf("ManualAdapter: cadoc mismatch (want %q, got %q)", cadocCode, a.Doc.CadocCode)
	}
	doc := *a.Doc // cópia
	doc.CadocCode = cadocCode
	if dataBase != "" {
		doc.DataBase = dataBase
	}
	if doc.VersaoLayout == "" {
		doc.VersaoLayout = "1.0"
	}
	return &doc, nil
}

// ============================================================
// FileAdapter — XLSX/CSV upload → CanonicalDocument (Sprint 57 v3.36.3)
// ============================================================

// FileReader é a interface mínima que FileAdapter precisa para ler um arquivo.
// In production, callers passam http.Request.Body ou similar.
type FileReader interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// FileAdapter parseia arquivos XLSX/CSV upados pelo usuário.
//
// Implementação Sprint 57 v3.36.3: extrai apenas o cabeçalho (primeira linha
// do CSV ou primeira linha da primeira sheet do XLSX) para detectar
// mapeamento de campos. A transformação completa Operacao[] é delegada
// ao wizard frontend (que tem UI para o usuário confirmar mapeamento).
//
// O parser retorna um CanonicalDoc "stub" com:
//   - DataBase preenchida
//   - CadocCode preenchida
//   - Operacoes vazias (frontend vai popular via wizard step 3 — map_fields)
//   - Extra["file_headers"] com a lista de colunas detectadas
//
// Use case: usuário faz upload de XLSX → backend lê colunas → frontend
// mostra UI para o usuário mapear colunas → frontend re-submete com
// Operacoes populadas via ManualAdapter.
type FileAdapter struct {
	// FileName é o nome do arquivo (usado para detectar extensão).
	FileName string
	// FileSize é o tamanho em bytes (validado antes de parse).
	FileSize int64
	// Content é o conteúdo do arquivo em bytes.
	Content []byte
}

// NewFileAdapter cria adapter com conteúdo em memória.
func NewFileAdapter(fileName string, content []byte) *FileAdapter {
	return &FileAdapter{
		FileName: fileName,
		FileSize: int64(len(content)),
		Content:  content,
	}
}

// AdapterName implements SourceAdapter.
func (a *FileAdapter) AdapterName() string { return "file" }

// Fetch parseia o arquivo e retorna CanonicalDoc com cabeçalhos detectados.
// dataBase e cadocCode são fornecidos pelo caller; file content é parseado
// a partir de a.Content (lido em memória).
//
// Formatos suportados:
//   - .csv   — primeira linha = cabeçalho
//   - .xlsx  — primeira linha da primeira sheet = cabeçalho
//   - .json  — parse direto como CanonicalDoc
//
// Outros formatos: retorna erro explicativo.
func (a *FileAdapter) Fetch(_ context.Context, cadocCode, dataBase string) (*CanonicalDoc, error) {
	if len(a.Content) == 0 {
		return nil, fmt.Errorf("FileAdapter: empty content")
	}

	headers, err := a.extractHeaders()
	if err != nil {
		return nil, fmt.Errorf("FileAdapter: %w", err)
	}

	return &CanonicalDoc{
		CadocCode:    cadocCode,
		DataBase:     dataBase,
		VersaoLayout: "1.0",
		Extra: map[string]any{
			"file_name":    a.FileName,
			"file_size":    a.FileSize,
			"file_headers": headers,
			// Operacoes vazias — frontend wizard step 3 vai popular.
			"operacoes_pending_mapping": true,
		},
	}, nil
}

// extractHeaders parseia o arquivo e retorna a primeira linha de cabeçalhos.
func (a *FileAdapter) extractHeaders() ([]string, error) {
	ext := fileExt(a.FileName)
	switch ext {
	case ".csv":
		return parseCSVHeader(a.Content)
	case ".xlsx":
		return parseXLSXHeader(a.Content)
	case ".json":
		// JSON pode ser CanonicalDoc completo — extrai só headers via struct tags.
		var probe struct {
			Operacoes []map[string]any `json:"operacoes"`
		}
		if err := json.Unmarshal(a.Content, &probe); err != nil {
			return nil, fmt.Errorf("json parse: %w", err)
		}
		if len(probe.Operacoes) == 0 {
			return nil, nil
		}
		// Coletar chaves da primeira operação.
		headers := make([]string, 0, len(probe.Operacoes[0]))
		for k := range probe.Operacoes[0] {
			headers = append(headers, k)
		}
		return headers, nil
	default:
		return nil, fmt.Errorf("unsupported file extension %q (suportado: .csv, .xlsx, .json)", ext)
	}
}

// fileExt retorna a extensão do filename em lowercase (sem dot), ou "".
func fileExt(name string) string {
	for i := len(name) - 1; i >= 0 && name[i] != '/'; i-- {
		if name[i] == '.' {
			return lower(name[i:])
		}
	}
	return ""
}

// lower é um strings.ToLower manual (evita import extra nesse arquivo de stubs).
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// parseCSVHeader retorna os campos da primeira linha (cabeçalho CSV).
// Suporta RFC 4180 (vírgula, aspas duplas, escape \").
//
// Limitação: ignora CR (\r) como whitespace. Linhas com CRLF (\r\n) são
// tratadas como LF simples. Quote-only escape é suportado (\" → " dentro
// de campo quoted).
func parseCSVHeader(content []byte) ([]string, error) {
	if len(content) == 0 {
		return nil, nil
	}
	var headers []string
	var current []byte
	inQuotes := false
	for i := 0; i < len(content); i++ {
		c := content[i]
		switch {
		case c == '"':
			if inQuotes && i+1 < len(content) && content[i+1] == '"' {
				current = append(current, '"')
				i++ // skip escape
			} else {
				inQuotes = !inQuotes
			}
		case c == ',' && !inQuotes:
			headers = append(headers, string(current))
			current = current[:0]
		case (c == '\n' || c == '\r') && !inQuotes:
			// Fim da linha (ou CR antes do LF). Append e retorna.
			headers = append(headers, string(current))
			return headers, nil
		default:
			current = append(current, c)
		}
	}
	// Última linha sem newline.
	headers = append(headers, string(current))
	return headers, nil
}

// parseXLSXHeader retorna os campos da primeira linha da primeira sheet.
//
// Implementação simplificada: procura "<row " no conteúdo bruto.
// Para XLSX reais (que são ZIP+XML), use github.com/xuri/excelize/v2.
//
// Esta versão é útil apenas para testes e arquivos XLSX "pre-flattened"
// onde o XML interno foi extraído por outro processo.
func parseXLSXHeader(content []byte) ([]string, error) {
	rowStart := findSubstring(content, []byte("<row "))
	if rowStart < 0 {
		return nil, fmt.Errorf("xlsx: no <row> found (arquivo não é XLSX ou está corrompido)")
	}
	rowEnd := findSubstring(content[rowStart:], []byte("</row>"))
	if rowEnd < 0 {
		return nil, fmt.Errorf("xlsx: </row> not found")
	}
	// rowEnd é offset relativo; calcular absoluto.
	rowXML := content[rowStart : rowStart+rowEnd]

	var headers []string
	cells := splitCells(rowXML)
	for _, cell := range cells {
		val := extractCellValue(cell)
		if val != "" {
			headers = append(headers, val)
		}
	}
	return headers, nil
}

// findSubstring retorna o índice da primeira ocorrência de needle em haystack,
// ou -1 se não encontrar.
func findSubstring(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// splitCells divide o XML de uma row em cells (<c ...>...</c>).
func splitCells(rowXML []byte) [][]byte {
	var cells [][]byte
	i := 0
	for i < len(rowXML) {
		cStart := findSubstring(rowXML[i:], []byte("<c "))
		if cStart < 0 {
			break
		}
		cStart += i
		cEnd := findSubstring(rowXML[cStart:], []byte("</c>"))
		if cEnd < 0 {
			// Self-closing ou malformado — pegar até próximo <c
			cEnd = findSubstring(rowXML[cStart+3:], []byte("<c "))
			if cEnd < 0 {
				cEnd = len(rowXML) - cStart
			} else {
				cEnd += cStart + 3
			}
		} else {
			cEnd += cStart + 4
		}
		cells = append(cells, rowXML[cStart:cEnd])
		i = cEnd
	}
	return cells
}

// extractCellValue extrai o valor de uma cell XLSX.
// Suporta <v>VALUE</v> e <is><t>VALUE</t></is>.
func extractCellValue(cell []byte) string {
	// Tentar <v>VALUE</v>
	vStart := findSubstring(cell, []byte("<v>"))
	if vStart >= 0 {
		vEnd := findSubstring(cell[vStart:], []byte("</v>"))
		if vEnd > 0 {
			return string(cell[vStart+3 : vStart+vEnd])
		}
	}
	// Tentar <is><t>VALUE</t></is>
	isStart := findSubstring(cell, []byte("<is>"))
	if isStart >= 0 {
		tStart := findSubstring(cell[isStart:], []byte("<t>"))
		if tStart >= 0 {
			tStart += isStart
			tEnd := findSubstring(cell[tStart:], []byte("</t>"))
			if tEnd > 0 {
				return string(cell[tStart+3 : tStart+tEnd])
			}
		}
	}
	return ""
}
