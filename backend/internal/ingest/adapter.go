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
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
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
type APIAdapter struct{}

func (a *APIAdapter) Name() string     { return "API REST/GraphQL" }
func (a *APIAdapter) Type() SourceType { return SourceAPI }

func (a *APIAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	// Stub: implementação completa entra na Sprint 57 fase 2.
	_ = cfg
	return nil, ErrNotImplemented
}

func (a *APIAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.API == nil {
		return fmt.Errorf("api config é requerida")
	}
	if cfg.API.Endpoint == "" {
		return fmt.Errorf("endpoint é requerido")
	}
	return nil
}

func (a *APIAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	// Stub: ping no endpoint.
	_ = cfg
	return ErrNotImplemented
}

func (a *APIAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true},
		{Name: "data_base", Description: "Data-base", Type: "date", Required: true},
	}
}

// --- DBAdapter ---

// DBAdapter é o conector para bancos de dados.
type DBAdapter struct{}

func (a *DBAdapter) Name() string     { return "Banco de Dados (PostgreSQL/MySQL)" }
func (a *DBAdapter) Type() SourceType { return SourceDB }

func (a *DBAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	_ = cfg
	return nil, ErrNotImplemented
}

func (a *DBAdapter) ValidateConfig(cfg SourceConfig) error {
	if cfg.DB == nil {
		return fmt.Errorf("db config é requerida")
	}
	if cfg.DB.Driver == "" {
		return fmt.Errorf("driver é requerido")
	}
	if cfg.DB.Query == "" {
		return fmt.Errorf("query é requerida")
	}
	return nil
}

func (a *DBAdapter) HealthCheck(ctx context.Context, cfg SourceConfig) error {
	// Stub: test de conexão.
	_ = cfg
	return ErrNotImplemented
}

func (a *DBAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF", Type: "string", Required: true},
	}
}

// --- MCPAdapter ---

// MCPAdapter é o conector para agentes IA via MCP.
type MCPAdapter struct{}

func (a *MCPAdapter) Name() string     { return "Agente IA (MCP)" }
func (a *MCPAdapter) Type() SourceType { return SourceMCP }

func (a *MCPAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	_ = cfg
	return nil, ErrNotImplemented
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
	// Stub: verifica se o servidor MCP está acessível.
	_ = cfg
	return ErrNotImplemented
}

func (a *MCPAdapter) DescribeFields(cadocCode string) []FieldDescriptor {
	return []FieldDescriptor{
		{Name: "cnpj", Description: "CNPJ da IF (extraído pelo agente)", Type: "string", Required: true},
		{Name: "data_base", Description: "Data-base (extraído pelo agente)", Type: "date", Required: true},
	}
}
