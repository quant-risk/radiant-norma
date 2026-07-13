// Package ingest implementa os conectores de ingestão de dados (SourceAdapter).
//
// Arquitetura:
//
//	IF configura fonte → Wizard valida conexão (HealthCheck)
//	→ Adapter.Fetch() → CanonicalDocument → Generator → XML
//
// Cada conector é um SourceAdapter plugável. O wizard UI usa a interface,
// não a implementação.
//
// Conectores disponíveis:
//   - ManualAdapter: input manual campo-a-campo (copiloto mode)
//   - FileAdapter:   arquivo XLSX/CSV/JSON (planilha do cliente)
//   - APIAdapter:    API REST/GraphQL (sistema do cliente)
//   - DBAdapter:     conexão direta PostgreSQL/MySQL
//   - MCPAdapter:    agente IA via Model Context Protocol
package ingest

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/shopspring/decimal"
)

// ErrNotImplemented é retornado por adapters stub quando o Fetch()
// ainda não foi implementado (Sprint 57 fase 2).
var ErrNotImplemented = errors.New("adapter: fetch não implementado")

// SourceType identifica o tipo de fonte de dados.
type SourceType string

const (
	SourceManual SourceType = "MANUAL"
	SourceFile   SourceType = "FILE"
	SourceAPI    SourceType = "API"
	SourceDB     SourceType = "DB"
	SourceMCP    SourceType = "MCP"
)

// SourceConfig é a configuração de uma fonte de dados.
// Cada conector interpreta os campos relevantes e ignora os outros.
type SourceConfig struct {
	// Type é o tipo de fonte.
	Type SourceType `json:"type"`

	// Name é um nome amigável para a fonte (ex: "Planilha Treasury Q2").
	Name string `json:"name"`

	// Manual é a configuração para input manual.
	Manual *ManualConfig `json:"manual,omitempty"`

	// File é a configuração para arquivo.
	File *FileConfig `json:"file,omitempty"`

	// API é a configuração para API.
	API *APIConfig `json:"api,omitempty"`

	// DB é a configuração para banco de dados.
	DB *DBConfig `json:"db,omitempty"`

	// MCP é a configuração para MCP.
	MCP *MCPConfig `json:"mcp,omitempty"`
}

// ManualConfig é a configuração para input manual.
type ManualConfig struct {
	// Fields é o mapa de campos preenchidos manualmente.
	// A chave é o nome do campo canônico.
	Fields map[string]any `json:"fields,omitempty"`
}

// FileConfig é a configuração para arquivo.
type FileConfig struct {
	// Path é o caminho do arquivo (absoluto ou relativo ao working dir).
	Path string `json:"path"`

	// Format é o formato do arquivo: "xlsx", "csv", "json".
	Format string `json:"format"`

	// Sheet é o nome ou índice da aba (para XLSX).
	Sheet string `json:"sheet,omitempty"`

	// HasHeader se true, a primeira linha é cabeçalho.
	HasHeader bool `json:"has_header,omitempty"`

	// Mapping é o mapeamento de colunas para campos canônicos.
	// A chave é o nome da coluna no arquivo.
	Mapping map[string]string `json:"mapping,omitempty"`
}

// APIConfig é a configuração para API.
type APIConfig struct {
	// Endpoint é a URL base da API.
	Endpoint string `json:"endpoint"`

	// Method é o método HTTP (GET, POST).
	Method string `json:"method"`

	// Headers são cabeçalhos HTTP customizados.
	Headers map[string]string `json:"headers,omitempty"`

	// Auth é o tipo de autenticação: "none", "bearer", "basic", "apikey".
	Auth string `json:"auth"`

	// Token é o token de autenticação (para bearer/apikey).
	Token string `json:"token,omitempty"`

	// QueryParams são parâmetros de query.
	QueryParams map[string]string `json:"query_params,omitempty"`

	// Body é o body da requisição (para POST).
	Body map[string]any `json:"body,omitempty"`
}

// DBConfig é a configuração para banco de dados.
type DBConfig struct {
	// Driver é o driver: "postgres", "mysql".
	Driver string `json:"driver"`

	// DSN é a connection string.
	DSN string `json:"dsn"`

	// Query é a SQL query que retorna os dados.
	Query string `json:"query"`

	// Timeout é o timeout da conexão.
	Timeout time.Duration `json:"timeout,omitempty"`
}

// MCPConfig é a configuração para MCP.
type MCPConfig struct {
	// ServerName é o nome do servidor MCP.
	ServerName string `json:"server_name"`

	// ToolName é o nome da tool que retorna os dados.
	ToolName string `json:"tool_name"`

	// Prompt é o prompt enviado ao agente para extrair dados.
	Prompt string `json:"prompt,omitempty"`

	// MaxTokens é o limite de tokens da resposta.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// SourceAdapter é a interface que todo conector deve implementar.
// O wizard UI usa esta interface; a implementação é plugável.
type SourceAdapter interface {
	// Name retorna o nome do conector.
	Name() string

	// Type retorna o tipo de fonte.
	Type() SourceType

	// Fetch busca dados da fonte e retorna um CanonicalDocument.
	// O CanonicalDocument pode estar parcialmente preenchido (campos
	//Opcionais em falta); o generator trata isso.
	Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error)

	// ValidateConfig valida a configuração antes de salvar.
	ValidateConfig(cfg SourceConfig) error

	// HealthCheck verifica se a fonte está acessível.
	// Para Manual, sempre retorna nil.
	HealthCheck(ctx context.Context, cfg SourceConfig) error

	// DescribeFields retorna os campos que este conector consegue fornecer.
	// Útil para o wizard mostrar o mapeamento.
	DescribeFields(cadocCode string) []FieldDescriptor
}

// FieldDescriptor descreve um campo disponível em um conector.
type FieldDescriptor struct {
	// Name é o nome do campo no formato canônico.
	Name string `json:"name"`

	// Description é a descrição do campo.
	Description string `json:"description"`

	// Type é o tipo do valor: "string", "number", "date", "money".
	Type string `json:"type"`

	// Required indica se o campo é obrigatório.
	Required bool `json:"required"`

	// Example é um exemplo de valor.
	Example string `json:"example,omitempty"`
}

// adapters registra todos os conectores disponíveis.
var adapters = make(map[SourceType]SourceAdapter)

func register(a SourceAdapter) {
	adapters[a.Type()] = a
}

func init() {
	register(&ManualAdapter{})
	register(&FileAdapter{})
	register(&APIAdapter{})
	register(&DBAdapter{})
	register(&MCPAdapter{})
}

// GetAdapter retorna o adapter para um tipo de fonte.
func GetAdapter(t SourceType) SourceAdapter {
	return adapters[t]
}

// ListAdapters retorna todos os adapters disponíveis.
func ListAdapters() []SourceAdapter {
	var out []SourceAdapter
	for _, a := range adapters {
		out = append(out, a)
	}
	return out
}

// --- ManualAdapter ---

// ManualAdapter é o conector para input manual campo-a-campo.
type ManualAdapter struct{}

func (a *ManualAdapter) Name() string     { return "Input Manual" }
func (a *ManualAdapter) Type() SourceType { return SourceManual }

func (a *ManualAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	if cfg.Manual == nil {
		return nil, fmt.Errorf("config manual é requerida")
	}

	doc := canonical.NewCanonical(cfg.Name, dataBase, canonical.CadocType(cadocCode))
	doc.Metadata.SourceAdapter = string(SourceManual)

	for k, v := range cfg.Manual.Fields {
		doc.Extra[k] = v
	}

	return doc, nil
}

func (a *ManualAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.Type != SourceManual {
		return fmt.Errorf("tipo de fonte deve ser MANUAL")
	}
	return nil
}

func (a *ManualAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	return nil // Manual sempre disponível.
}

func (a *ManualAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true, Example: "12345678000123"},
		{Name: "nome_if", Description: "Nome da IF", Type: "string", Required: true, Example: "Banco Exemplo S.A."},
		{Name: "natureza", Description: "Natureza (N/R/A)", Type: "string", Required: true, Example: "N"},
	}
}

// --- FileAdapter ---

// FileAdapter é o conector para arquivos XLSX/CSV/JSON.
type FileAdapter struct{}

func (a *FileAdapter) Name() string     { return "Arquivo (XLSX/CSV/JSON)" }
func (a *FileAdapter) Type() SourceType { return SourceFile }

func (a *FileAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.File == nil {
		return fmt.Errorf("file config é requerida")
	}
	if cfg.File.Path == "" {
		return fmt.Errorf("file path é requerido")
	}
	return nil
}

func (a *FileAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	if cfg.File == nil {
		return fmt.Errorf("file config é requerida")
	}
	_, err := os.Stat(cfg.File.Path)
	if err != nil {
		return fmt.Errorf("arquivo não encontrado: %s", cfg.File.Path)
	}
	return nil
}

func (a *FileAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true},
		{Name: "data_base", Description: "Data-base", Type: "date", Required: true},
		{Name: "operacoes", Description: "Lista de operações", Type: "array", Required: true},
	}
}

// --- APIAdapter ---

// APIAdapter é o conector para APIs REST/GraphQL.
type APIAdapter struct {
	// HTTPClient é injetável para testes. nil = http.DefaultClient.
	HTTPClient *http.Client
}

// newAPIAdapter cria um APIAdapter com cliente HTTP configurado.
func newAPIAdapter() *APIAdapter {
	return &APIAdapter{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *APIAdapter) Name() string     { return "API REST/GraphQL" }
func (a *APIAdapter) Type() SourceType { return SourceAPI }

func (a *APIAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	if cfg.API == nil {
		return nil, fmt.Errorf("api config é requerida")
	}
	c := a.HTTPClient
	if c == nil {
		c = http.DefaultClient
	}

	// Monta URL com query params.
	baseURL := cfg.API.Endpoint
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	reqURL, err := url.Parse(baseURL + "canonical")
	if err != nil {
		return nil, fmt.Errorf("url inválida: %w", err)
	}
	q := reqURL.Query()
	q.Set("cadoc_code", cadocCode)
	q.Set("data_base", dataBase.Format("2006-01"))
	for k, v := range cfg.API.QueryParams {
		q.Set(k, v)
	}
	reqURL.RawQuery = q.Encode()

	// Monta request.
	var bodyReader io.Reader
	if cfg.API.Method == "POST" && cfg.API.Body != nil {
		bodyBytes, _ := json.Marshal(cfg.API.Body)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, cfg.API.Method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	// Headers default.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.API.Headers {
		req.Header.Set(k, v)
	}

	// Auth.
	switch cfg.API.Auth {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+cfg.API.Token)
	case "basic":
		// Token contém "user:pass" codificado em base64.
		req.Header.Set("Authorization", "Basic "+cfg.API.Token)
	case "apikey":
		req.Header.Set("X-API-Key", cfg.API.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requisição falhou: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	// Decode JSON flexível: aceita qualquer formato de response.
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON: %w", err)
	}

	return a.parseResponse(raw, cadocCode, dataBase)
}

// parseResponse converte JSON arbitrário da API no CanonicalDocument.
// Suporta os formatos mais comuns de API de sistemas financeiros.
func (a *APIAdapter) parseResponse(raw json.RawMessage, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	// Tenta formato direto com campos canônicos.
	var direct struct {
		CNPJ          string           `json:"cnpj"`
		NomeIF        string           `json:"nome_if"`
		DataBase      string           `json:"data_base"`
		VersaoLayout  string           `json:"versao_layout"`
		Natureza      string           `json:"natureza"`
		Operacoes     []map[string]any `json:"operacoes"`
		Participantes []map[string]any `json:"participantes"`
		Extra         map[string]any   `json:"extra"`
	}

	if err := json.Unmarshal(raw, &direct); err == nil && direct.CNPJ != "" {
		doc := canonical.NewCanonical(direct.CNPJ, dataBase, canonical.CadocType(cadocCode))
		doc.Header.CNPJ = direct.CNPJ
		doc.Header.NomeIF = direct.NomeIF
		doc.Header.DataHoraGeracao = time.Now()
		if direct.VersaoLayout != "" {
			doc.VersaoLayout = direct.VersaoLayout
		}
		if direct.Natureza != "" {
			doc.Natureza = direct.Natureza
		}
		doc.Metadata.SourceAdapter = string(SourceAPI)

		for _, opRaw := range direct.Operacoes {
			op := a.mapOperacao(opRaw)
			doc.Operacoes = append(doc.Operacoes, op)
		}
		for _, pRaw := range direct.Participantes {
			p := a.mapParticipante(pRaw)
			doc.Participantes = append(doc.Participantes, p)
		}
		if direct.Extra != nil {
			for k, v := range direct.Extra {
				doc.Extra[k] = v
			}
		}
		return doc, nil
	}

	// Tenta formato "data": { ... } (envelope padrão).
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Data) > 0 {
		return a.parseResponse(env.Data, cadocCode, dataBase)
	}

	// Tenta formato "result": { ... } (JSON-RPC style).
	var rpc struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &rpc); err == nil && len(rpc.Result) > 0 {
		return a.parseResponse(rpc.Result, cadocCode, dataBase)
	}

	return nil, fmt.Errorf("formato de response não reconhecido")
}

// mapOperacao converte um mapa JSON em canonical.Operacao.
func (a *APIAdapter) mapOperacao(opRaw map[string]any) canonical.Operacao {
	op := canonical.Operacao{}

	if v, ok := opRaw["id"].(string); ok {
		op.ID = v
	}
	if v, ok := opRaw["modalidade"].(string); ok {
		op.Modalidade = v
	}
	if v, ok := opRaw["tipo_pessoa"].(string); ok {
		op.TipoPessoa = v
	}
	if v, ok := opRaw["tipo_operacao"].(string); ok {
		op.TipoOperacao = v
	}
	if v, ok := opRaw["numero_contrato"].(string); ok {
		op.NumeroContrato = v
	}
	if v, ok := opRaw["indexador"].(string); ok {
		op.Indexador = v
	}
	if v, ok := opRaw["uf"].(string); ok {
		op.UF = v
	}
	if v, ok := opRaw["nivel_risco"].(string); ok {
		op.NivelRisco = v
	}
	if v, ok := opRaw["faixa_vencimento"].(string); ok {
		op.FaixaVencimento = v
	}

	// Valor principal — tenta number ou string.
	if v, ok := opRaw["valor"].(float64); ok {
		op.ValorPrincipal = canonical.Money{Valor: decimal.NewFromFloat(v), Moeda: "BRL"}
	} else if v, ok := opRaw["valor"].(string); ok {
		if f, err := decimal.NewFromString(v); err == nil {
			op.ValorPrincipal = canonical.Money{Valor: f, Moeda: "BRL"}
		}
	}

	// Encargos, IOF, valor atualizado.
	for _, field := range []struct {
		key  string
		dest *canonical.Money
	}{
		{"encargos", &op.EncargosTotais},
		{"iof", &op.IOF},
		{"valor_atualizado", &op.ValorAtualizado},
	} {
		if v, ok := opRaw[field.key].(float64); ok {
			*field.dest = canonical.Money{Valor: decimal.NewFromFloat(v), Moeda: "BRL"}
		} else if v, ok := opRaw[field.key].(string); ok {
			if f, err := decimal.NewFromString(v); err == nil {
				*field.dest = canonical.Money{Valor: f, Moeda: "BRL"}
			}
		}
	}

	// Taxas.
	for _, field := range []struct {
		key  string
		dest *decimal.Decimal
	}{
		{"taxa_juros", &op.TaxaJuros},
		{"taxa_spread", &op.TaxaSpread},
		{"percentual_indexador", &op.PercentualIndexador},
		{"percentual_provisao", &op.PercentualProvisao},
	} {
		if v, ok := opRaw[field.key].(float64); ok {
			*field.dest = decimal.NewFromFloat(v)
		} else if v, ok := opRaw[field.key].(string); ok {
			if f, err := decimal.NewFromString(v); err == nil {
				*field.dest = f
			}
		}
	}

	// Datas.
	for _, field := range []struct {
		key  string
		dest *time.Time
	}{
		{"data_vencimento", &op.DataVencimento},
		{"data_constituicao", &op.DataConstituicao},
	} {
		if v, ok := opRaw[field.key].(string); ok {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				*field.dest = t
			}
		}
	}

	// Extra fields — tudo que sobrou vai em Extra.
	op.Extra = make(map[string]any)
	for k, v := range opRaw {
		switch k {
		case "id", "modalidade", "tipo_pessoa", "tipo_operacao",
			"numero_contrato", "indexador", "uf", "nivel_risco",
			"faixa_vencimento", "valor", "encargos", "iof",
			"valor_atualizado", "taxa_juros", "taxa_spread",
			"percentual_indexador", "percentual_provisao",
			"data_vencimento", "data_constituicao":
			// já mapeado acima
		default:
			op.Extra[k] = v
		}
	}

	return op
}

// mapParticipante converte um mapa JSON em canonical.Participante.
func (a *APIAdapter) mapParticipante(pRaw map[string]any) canonical.Participante {
	p := canonical.Participante{}

	if v, ok := pRaw["tipo"].(string); ok {
		p.Tipo = v
	}
	if v, ok := pRaw["cnpj"].(string); ok {
		p.CNPJ = v
	}
	if v, ok := pRaw["cpf"].(string); ok {
		p.CNPJ = v
	}
	if v, ok := pRaw["nome"].(string); ok {
		p.Nome = v
	}
	if v, ok := pRaw["cnae"].(string); ok {
		if p.Extra == nil {
			p.Extra = make(map[string]any)
		}
		p.Extra["cnae"] = v
	}
	if v, ok := pRaw["modalidade"].(string); ok {
		p.Modalidade = v
	}
	if v, ok := pRaw["uf"].(string); ok {
		p.UF = v
	}
	if v, ok := pRaw["classe"].(string); ok {
		p.Classe = v
	}

	return p
}

func (a *APIAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.API == nil {
		return fmt.Errorf("api config é requerida")
	}
	if cfg.API.Endpoint == "" {
		return fmt.Errorf("endpoint é requerido")
	}
	validMethods := map[string]bool{"GET": true, "POST": true}
	if cfg.API.Method != "" && !validMethods[cfg.API.Method] {
		return fmt.Errorf("método HTTP %q inválido (use GET ou POST)", cfg.API.Method)
	}
	validAuth := map[string]bool{"": true, "none": true, "bearer": true, "basic": true, "apikey": true}
	if !validAuth[cfg.API.Auth] {
		return fmt.Errorf("auth %q inválido (use none, bearer, basic ou apikey)", cfg.API.Auth)
	}
	return nil
}

func (a *APIAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	if cfg.API == nil {
		return fmt.Errorf("api config é requerida")
	}
	c := a.HTTPClient
	if c == nil {
		c = http.DefaultClient
	}

	// HEAD no endpoint com timeout curto (5s).
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, cfg.API.Endpoint, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}
	if cfg.API.Auth == "bearer" {
		req.Header.Set("Authorization", "Bearer "+cfg.API.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint inacessível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("endpoint retornou erro %d", resp.StatusCode)
	}
	return nil
}

func (a *APIAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true, Example: "12345678000123"},
		{Name: "nome_if", Description: "Nome da IF", Type: "string", Required: true},
		{Name: "data_base", Description: "Data-base (YYYY-MM)", Type: "date", Required: true},
		{Name: "operacoes", Description: "Lista de operações", Type: "array", Required: true},
		{Name: "participantes", Description: "Lista de participantes", Type: "array", Required: false},
	}
}

// --- DBAdapter ---

// DBAdapter é o conector para bancos de dados PostgreSQL/MySQL.
type DBAdapter struct{}

func (a *DBAdapter) Name() string     { return "Banco de Dados (PostgreSQL/MySQL)" }
func (a *DBAdapter) Type() SourceType { return SourceDB }

func (a *DBAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("db config é requerida")
	}

	// Abre conexão com o driver especificado.
	db, err := a.openDB(cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexão: %w", err)
	}
	defer db.Close()

	// Executa a query com timeout.
	timeout := cfg.DB.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, cfg.DB.Query)
	if err != nil {
		return nil, fmt.Errorf("query falhou: %w", err)
	}
	defer rows.Close()

	// Extrai metadados do documento da primeira linha (convenção: primeira
	// linha contém cnpj, nome_if, data_base, etc. — campos não-Operacao).
	var doc canonical.CanonicalDocument
	var operacoes []canonical.Operacao

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter colunas: %w", err)
	}

	for rows.Next() {
		// Usa map[string]any para scan dinâmico.
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("falha ao escanear linha: %w", err)
		}

		row := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		// Se a linha tem campos de header (cnpj, nome_if, data_base),
		// popula o doc. Se tem campos de operação, adiciona à lista.
		if cnpj, ok := row["cnpj"].(string); ok && doc.Header.CNPJ == "" {
			doc.Header.CNPJ = cnpj
			if doc.IFID == "" {
				doc.IFID = cnpj
			}
		}
		if nomeIF, ok := row["nome_if"].(string); ok && doc.Header.NomeIF == "" {
			doc.Header.NomeIF = nomeIF
		}

		// Tenta detectar se é uma linha de operação (auto-detecção por nome de coluna).
		op := a.mapRowToOperacao(row)
		if op.ID != "" || op.Modalidade != "" {
			operacoes = append(operacoes, op)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro na iteração de linhas: %w", err)
	}

	if doc.Header.CNPJ == "" {
		return nil, fmt.Errorf("query não retornou CNPJ (certifique-se que a query tem coluna 'cnpj')")
	}

	doc.CadocCode = canonical.CadocType(cadocCode)
	doc.DataBase = canonical.DataBase(dataBase)
	doc.Header.DataHoraGeracao = time.Now()
	doc.Operacoes = operacoes
	doc.Metadata.SourceAdapter = string(SourceDB)
	doc.Metadata.SourceRef = cfg.DB.Query
	if doc.Extra == nil {
		doc.Extra = make(map[string]any)
	}

	return &doc, nil
}

// openDB abre uma conexão com o driver especificado.
func (a *DBAdapter) openDB(cfg *DBConfig) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sql.Open falhou: %w", err)
	}
	// Testa a conexão.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping falhou: %w", err)
	}
	return db, nil
}

// mapRowToOperacao detecta colunas pelo nome e constrói uma Operacao.
// Usa auto-detecção: tenta casar nome da coluna com nome do campo canônico.
func (a *DBAdapter) mapRowToOperacao(row map[string]any) canonical.Operacao {
	op := canonical.Operacao{}

	// Lista de campos canônicos para auto-detecção.
	canonicalFields := []string{
		"id", "modalidade", "tipo_pessoa", "tipo_operacao",
		"numero_contrato", "valor", "encargos", "iof",
		"valor_atualizado", "taxa_juros", "taxa_spread",
		"percentual_indexador", "percentual_provisao",
		"indexador", "faixa_vencimento", "nivel_risco",
		"uf", "pais", "data_vencimento", "data_constituicao",
		"classificacao_if",
	}
	for col, val := range row {
		colLower := strings.ToLower(col)
		for _, field := range canonicalFields {
			if colLower == field || colLower == strings.ReplaceAll(field, "_", "") {
				a.setOperacaoField(&op, field, val)
				break
			}
		}
	}

	return op
}

// setOperacaoField define um campo da Operacao pelo nome (string).
func (a *DBAdapter) setOperacaoField(op *canonical.Operacao, field string, value any) {
	switch field {
	case "id":
		if s, ok := value.(string); ok {
			op.ID = s
		}
	case "modalidade":
		if s, ok := value.(string); ok {
			op.Modalidade = s
		}
	case "tipo_pessoa":
		if s, ok := value.(string); ok {
			op.TipoPessoa = s
		}
	case "tipo_operacao":
		if s, ok := value.(string); ok {
			op.TipoOperacao = s
		}
	case "numero_contrato":
		if s, ok := value.(string); ok {
			op.NumeroContrato = s
		}
	case "valor":
		op.ValorPrincipal = a.parseMoney(value)
	case "encargos":
		op.EncargosTotais = a.parseMoney(value)
	case "iof":
		op.IOF = a.parseMoney(value)
	case "valor_atualizado":
		op.ValorAtualizado = a.parseMoney(value)
	case "taxa_juros", "taxa_spread", "percentual_indexador", "percentual_provisao":
		d := a.parseDecimal(value)
		switch field {
		case "taxa_juros":
			op.TaxaJuros = d
		case "taxa_spread":
			op.TaxaSpread = d
		case "percentual_indexador":
			op.PercentualIndexador = d
		case "percentual_provisao":
			op.PercentualProvisao = d
		}
	case "indexador":
		if s, ok := value.(string); ok {
			op.Indexador = s
		}
	case "faixa_vencimento":
		if s, ok := value.(string); ok {
			op.FaixaVencimento = s
		}
	case "nivel_risco":
		if s, ok := value.(string); ok {
			op.NivelRisco = s
		}
	case "uf":
		if s, ok := value.(string); ok {
			op.UF = s
		}
	case "pais":
		if s, ok := value.(string); ok {
			op.Pais = s
		}
	case "classificacao_if":
		if s, ok := value.(string); ok {
			op.ClassificacaoIF = s
		}
	case "data_vencimento":
		if t, ok := value.(time.Time); ok {
			op.DataVencimento = t
		} else if s, ok := value.(string); ok {
			if parsed, err := time.Parse("2006-01-02", s); err == nil {
				op.DataVencimento = parsed
			}
		}
	case "data_constituicao":
		if t, ok := value.(time.Time); ok {
			op.DataConstituicao = t
		} else if s, ok := value.(string); ok {
			if parsed, err := time.Parse("2006-01-02", s); err == nil {
				op.DataConstituicao = parsed
			}
		}
	}
}

// parseMoney converte um valor em canonical.Money.
func (a *DBAdapter) parseMoney(value any) canonical.Money {
	switch v := value.(type) {
	case float64:
		return canonical.Money{Valor: decimal.NewFromFloat(v), Moeda: "BRL"}
	case int64:
		return canonical.Money{Valor: decimal.NewFromInt(v), Moeda: "BRL"}
	case int:
		return canonical.Money{Valor: decimal.NewFromInt(int64(v)), Moeda: "BRL"}
	case string:
		if d, err := decimal.NewFromString(v); err == nil {
			return canonical.Money{Valor: d, Moeda: "BRL"}
		}
	}
	return canonical.Money{}
}

// parseDecimal converte um valor em decimal.Decimal.
func (a *DBAdapter) parseDecimal(value any) decimal.Decimal {
	switch v := value.(type) {
	case float64:
		return decimal.NewFromFloat(v)
	case int64:
		return decimal.NewFromInt(v)
	case int:
		return decimal.NewFromInt(int64(v))
	case string:
		if d, err := decimal.NewFromString(v); err == nil {
			return d
		}
	}
	return decimal.Zero
}

func (a *DBAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.DB == nil {
		return fmt.Errorf("db config é requerida")
	}
	if cfg.DB.Driver == "" {
		return fmt.Errorf("driver é requerido")
	}
	validDrivers := map[string]bool{"postgres": true, "mysql": true, "pq": true}
	if !validDrivers[cfg.DB.Driver] {
		return fmt.Errorf("driver %q inválido (use postgres ou mysql)", cfg.DB.Driver)
	}
	if cfg.DB.DSN == "" {
		return fmt.Errorf("dsn é requerido")
	}
	if cfg.DB.Query == "" {
		return fmt.Errorf("query é requerida")
	}
	return nil
}

func (a *DBAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	if cfg.DB == nil {
		return fmt.Errorf("db config é requerida")
	}
	db, err := a.openDB(cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func (a *DBAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true},
		{Name: "nome_if", Description: "Nome da IF", Type: "string", Required: true},
		{Name: "operacoes", Description: "Linhas da query", Type: "array", Required: true},
	}
}

// --- MCPAdapter ---

// MCPAdapter é o conector para agentes IA via Model Context Protocol.
// O servidor MCP é acessado via HTTP+JSON-RPC.
//
// O adapter envia uma chamada de tool JSON-RPC 2.0 e interpreta o resultado
// como CanonicalDocument (suporta formatos direto, {result:{...}} ou {data:{...}}).
type MCPAdapter struct {
	// HTTPClient é injetável para testes. nil = http.DefaultClient.
	HTTPClient *http.Client
}

func (a *MCPAdapter) Name() string     { return "Agente IA (MCP)" }
func (a *MCPAdapter) Type() SourceType { return SourceMCP }

func (a *MCPAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	if cfg.MCP == nil {
		return nil, fmt.Errorf("mcp config é requerida")
	}
	c := a.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 60 * time.Second}
	}

	// Monta JSON-RPC request.
	rpcParams := map[string]any{
		"cadoc_code": cadocCode,
		"data_base":  dataBase.Format("2006-01"),
	}
	if cfg.MCP.Prompt != "" {
		rpcParams["prompt"] = cfg.MCP.Prompt
	}

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  cfg.MCP.ToolName,
		"params":  rpcParams,
	}

	bodyBytes, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar JSON-RPC: %w", err)
	}

	// Descobre o endpoint do servidor MCP.
	// O ServerName é usado como hint para construir a URL.
	// Em produção, isso viria de uma registry de servidores MCP.
	endpoint := a.resolveEndpoint(cfg.MCP.ServerName)
	if endpoint == "" {
		return nil, fmt.Errorf("servidor MCP %q não encontrado (configure a variável MCP_%s_ENDPOINT)",
			cfg.MCP.ServerName, strings.ToUpper(strings.ReplaceAll(cfg.MCP.ServerName, "-", "_")))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MCP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("MCP server retornou status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp struct {
		ID     int              `json:"id"`
		Result json.RawMessage  `json:"result"`
		Error  *json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON-RPC response: %w", err)
	}
	if rpcResp.Error != nil {
		var rpcErr struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		json.Unmarshal(*rpcResp.Error, &rpcErr)
		return nil, fmt.Errorf("MCP tool error %d: %s", rpcErr.Code, rpcErr.Message)
	}

	// O result do MCP pode ser direto ou dentro de {data:{...}}.
	canonicalDoc, err := a.parseMCPResult(rpcResp.Result, cadocCode, dataBase)
	if err != nil {
		return nil, fmt.Errorf("parse do result MCP falhou: %w", err)
	}

	canonicalDoc.Metadata.SourceAdapter = string(SourceMCP)
	canonicalDoc.Metadata.SourceRef = cfg.MCP.ServerName + "." + cfg.MCP.ToolName

	return canonicalDoc, nil
}

// resolveEndpoint retorna a URL do servidor MCP baseado no ServerName.
// Procura na ordem: variável de ambiente > endpoint padrão.
func (a *MCPAdapter) resolveEndpoint(serverName string) string {
	// Monta nome da variável: MCP_<SERVERNAME>_ENDPOINT
	envKey := "MCP_" + strings.ToUpper(strings.ReplaceAll(serverName, "-", "_")) + "_ENDPOINT"
	if endpoint := os.Getenv(envKey); endpoint != "" {
		return endpoint
	}
	// Fallback: endpoint local padrão.
	return "http://localhost:3100/mcp"
}

// parseMCPResult converte o resultado JSON-RPC em CanonicalDocument.
// Suporta os formatos mais comuns de resposta de tool MCP.
func (a *MCPAdapter) parseMCPResult(result json.RawMessage, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	// Tenta unwrap {data:{...}}.
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &env); err == nil && len(env.Data) > 0 {
		return a.parseMCPResult(env.Data, cadocCode, dataBase)
	}

	// Tenta unwrap {result:{...}}.
	var rpcResult struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(result, &rpcResult); err == nil && len(rpcResult.Result) > 0 {
		return a.parseMCPResult(rpcResult.Result, cadocCode, dataBase)
	}

	// Tenta parsear como CanonicalDocument direto.
	var doc struct {
		CNPJ          string           `json:"cnpj"`
		NomeIF        string           `json:"nome_if"`
		DataBase      string           `json:"data_base"`
		VersaoLayout  string           `json:"versao_layout"`
		Natureza      string           `json:"natureza"`
		Operacoes     []map[string]any `json:"operacoes"`
		Participantes []map[string]any `json:"participantes"`
		Extra         map[string]any   `json:"extra"`
	}
	if err := json.Unmarshal(result, &doc); err != nil {
		return nil, fmt.Errorf("result MCP não é JSON válido: %w", err)
	}
	if doc.CNPJ == "" {
		return nil, fmt.Errorf("result MCP não contém campo 'cnpj'")
	}

	canonicalDoc := canonical.NewCanonical(doc.CNPJ, dataBase, canonical.CadocType(cadocCode))
	canonicalDoc.Header.CNPJ = doc.CNPJ
	canonicalDoc.Header.NomeIF = doc.NomeIF
	canonicalDoc.Header.DataHoraGeracao = time.Now()
	if doc.VersaoLayout != "" {
		canonicalDoc.VersaoLayout = doc.VersaoLayout
	}
	if doc.Natureza != "" {
		canonicalDoc.Natureza = doc.Natureza
	}

	for _, opRaw := range doc.Operacoes {
		op := a.mapMCPOperacao(opRaw)
		canonicalDoc.Operacoes = append(canonicalDoc.Operacoes, op)
	}
	for _, pRaw := range doc.Participantes {
		p := a.mapMCPParticipante(pRaw)
		canonicalDoc.Participantes = append(canonicalDoc.Participantes, p)
	}
	if doc.Extra != nil {
		for k, v := range doc.Extra {
			canonicalDoc.Extra[k] = v
		}
	}

	return canonicalDoc, nil
}

// mapMCPOperacao converte um mapa JSON em Operacao.
func (a *MCPAdapter) mapMCPOperacao(opRaw map[string]any) canonical.Operacao {
	op := canonical.Operacao{}

	if v, ok := opRaw["id"].(string); ok {
		op.ID = v
	}
	if v, ok := opRaw["modalidade"].(string); ok {
		op.Modalidade = v
	}
	if v, ok := opRaw["tipo_pessoa"].(string); ok {
		op.TipoPessoa = v
	}
	if v, ok := opRaw["tipo_operacao"].(string); ok {
		op.TipoOperacao = v
	}
	if v, ok := opRaw["numero_contrato"].(string); ok {
		op.NumeroContrato = v
	}
	if v, ok := opRaw["indexador"].(string); ok {
		op.Indexador = v
	}
	if v, ok := opRaw["uf"].(string); ok {
		op.UF = v
	}
	if v, ok := opRaw["nivel_risco"].(string); ok {
		op.NivelRisco = v
	}
	if v, ok := opRaw["faixa_vencimento"].(string); ok {
		op.FaixaVencimento = v
	}

	// Valores monetários.
	for _, field := range []struct {
		key  string
		dest *canonical.Money
	}{
		{"valor", &op.ValorPrincipal},
		{"encargos", &op.EncargosTotais},
		{"iof", &op.IOF},
		{"valor_atualizado", &op.ValorAtualizado},
	} {
		if v, ok := opRaw[field.key].(float64); ok {
			*field.dest = canonical.Money{Valor: decimal.NewFromFloat(v), Moeda: "BRL"}
		} else if v, ok := opRaw[field.key].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				*field.dest = canonical.Money{Valor: d, Moeda: "BRL"}
			}
		}
	}

	// Decimals.
	for _, field := range []struct {
		key  string
		dest *decimal.Decimal
	}{
		{"taxa_juros", &op.TaxaJuros},
		{"taxa_spread", &op.TaxaSpread},
		{"percentual_indexador", &op.PercentualIndexador},
		{"percentual_provisao", &op.PercentualProvisao},
	} {
		if v, ok := opRaw[field.key].(float64); ok {
			*field.dest = decimal.NewFromFloat(v)
		} else if v, ok := opRaw[field.key].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				*field.dest = d
			}
		}
	}

	return op
}

// mapMCPParticipante converte um mapa JSON em Participante.
func (a *MCPAdapter) mapMCPParticipante(pRaw map[string]any) canonical.Participante {
	p := canonical.Participante{}
	if v, ok := pRaw["tipo"].(string); ok {
		p.Tipo = v
	}
	if v, ok := pRaw["cnpj"].(string); ok {
		p.CNPJ = v
	}
	if v, ok := pRaw["cpf"].(string); ok {
		p.CNPJ = v
	}
	if v, ok := pRaw["nome"].(string); ok {
		p.Nome = v
	}
	if v, ok := pRaw["cnae"].(string); ok {
		if p.Extra == nil {
			p.Extra = make(map[string]any)
		}
		p.Extra["cnae"] = v
	}
	if v, ok := pRaw["modalidade"].(string); ok {
		p.Modalidade = v
	}
	if v, ok := pRaw["uf"].(string); ok {
		p.UF = v
	}
	if v, ok := pRaw["classe"].(string); ok {
		p.Classe = v
	}
	return p
}

func (a *MCPAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.MCP == nil {
		return fmt.Errorf("mcp config é requerida")
	}
	if cfg.MCP.ServerName == "" {
		return fmt.Errorf("server_name é requerido")
	}
	if cfg.MCP.ToolName == "" {
		return fmt.Errorf("tool_name é requerido")
	}
	return nil
}

func (a *MCPAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	if cfg.MCP == nil {
		return fmt.Errorf("mcp config é requerida")
	}
	c := a.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}

	endpoint := a.resolveEndpoint(cfg.MCP.ServerName)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("servidor MCP inacessível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("servidor MCP retornou %d", resp.StatusCode)
	}
	return nil
}

func (a *MCPAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF (extraído pelo agente IA)", Type: "string", Required: true},
		{Name: "nome_if", Description: "Nome da IF", Type: "string", Required: true},
		{Name: "operacoes", Description: "Operações extraídas pelo agente", Type: "array", Required: true},
	}
}
