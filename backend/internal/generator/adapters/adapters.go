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
