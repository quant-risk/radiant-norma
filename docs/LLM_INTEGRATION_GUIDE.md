# Radiant Norma — LLM Integration Guide

**Data:** 2026-07-09
**Versão:** 1.0
**Status:** Guia prático de integração de IA no Radiant Norma

---

## 1. Visão Geral: Onde a IA Entra

A IA **não substitui** o pipeline de geração — ela **preenche o gap entre dado bruto e CanonicalDocument**. O resto continua determinístico em Go.

```
┌──────────────────────────────────────────────────────────────────────┐
│                          FONTE DE DADOS                              │
│                                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐     │
│  │ Manual     │  │ Planilha   │  │ Banco      │  │ PDF / DOCX │     │
│  │ (form UI)  │  │ XLSX/CSV   │  │ PostgreSQL │  │ Word, PDF  │     │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘     │
│        │               │               │               │            │
│        ▼               ▼               ▼               ▼            │
│   ManualAdapter   FileAdapter    DBAdapter       (Docling + LLM)   │
│                                                                      │
└──────────────────────────┬───────────────────────────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │  CanonicalDocument     │ ◄─── CONTRATO CENTRAL
              │  (JSON tipado)         │      "LLM nunca escreve XML"
              └────────┬───────────────┘
                       │
                       ▼
              ┌────────────────────────┐
              │  CADOCGenerator        │
              │  (gen3040, gen3050...) │ ◄─── Determinístico em Go
              └────────┬───────────────┘
                       │
                       ▼
                   XML CADOC
```

### As 3 Pontas Onde IA Pode Ajudar

| # | Ponta | Hoje | Com IA |
|---|-------|------|--------|
| 1 | **Entrada não-estruturada** (PDF/DOCX/e-mail) | Não tem adapter | Docling + LLM preenche CanonicalDocument |
| 2 | **Mapping inteligente** de campos | Manual: usuário mapeia coluna→campo | LLM sugere mapping a partir de exemplos |
| 3 | **Resumo de validações e normativos** | L1-L4 retornam erros em código | LLM traduz erros em linguagem natural + impacto |

---

## 2. Caminho 1: MCPAdapter (o que já existe no código)

O `MCPAdapter` já está definido em `backend/internal/ingest/adapter.go` (linha 378) mas retorna `ErrNotImplemented`. É o slot pronto para você plugar um agente IA.

### O que é MCP?

**Model Context Protocol** — protocolo aberto (criado pela Anthropic em 2024, hoje padrão da indústria) que permite um agente IA expor **tools** que outros sistemas podem chamar.

```
┌─────────────────┐         ┌────────────────────┐
│  Radiant Norma  │  JSON   │  Servidor MCP      │
│  (Go client)    │ ◄─────► │  (Python ou Go)    │
│                 │  RPC    │                    │
│  MCPAdapter     │         │  Tools:            │
│                 │         │  - extract_data    │
│                 │         │  - classify_doc    │
│                 │         │  - validate_layout │
└─────────────────┘         └────────────────────┘
```

### Como Implementar (passo a passo)

#### Passo 1: Criar o servidor MCP

Em `backend/cmd/mcp-server/main.go` (novo), usando SDK Python (mais maduro):

```python
# mcp_server.py — exemplo de servidor MCP para Radiant Norma
from mcp.server import Server, stdio
from mcp.types import Tool, TextContent
import ollama  # cliente para Ollama local

server = Server("radiant-norma-mcp")

@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="extract_canonical_from_xlsx",
            description="Extrai campos de um XLSX e retorna CanonicalDocument JSON",
            inputSchema={
                "type": "object",
                "properties": {
                    "file_path": {"type": "string"},
                    "cadoc_code": {"type": "string", "enum": ["3040","3050"]},
                    "data_base": {"type": "string", "format": "date"}
                },
                "required": ["file_path", "cadoc_code", "data_base"]
            }
        ),
        Tool(
            name="summarize_normativo",
            description="Resume normativo BACEN em pt-BR e classifica impacto",
            inputSchema={
                "type": "object",
                "properties": {
                    "texto": {"type": "string"},
                    "cadocs_afetados": {"type": "array", "items": {"type": "string"}}
                },
                "required": ["texto"]
            }
        ),
        Tool(
            name="explain_validation_error",
            description="Explica erro de validação L1-L4 em linguagem natural",
            inputSchema={
                "type": "object",
                "properties": {
                    "rule_code": {"type": "string"},
                    "context": {"type": "object"}
                },
                "required": ["rule_code"]
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "extract_canonical_from_xlsx":
        # 1. Docling parseia XLSX → texto estruturado
        from docling.document_converter import DocumentConverter
        converter = DocumentConverter()
        result = converter.convert(arguments["file_path"])
        texto = result.document.export_to_markdown()

        # 2. LLM extrai campos e devolve JSON
        prompt = f"""Você é um especialista em CADOCs BACEN.
Extraia os campos do documento abaixo no formato JSON CanonicalDocument.

CADOC: {arguments['cadoc_code']}
Data-base: {arguments['data_base']}

Schema do CanonicalDocument:
{open('canonical_schema.json').read()}

Documento:
{texto}

Responda APENAS o JSON válido, sem markdown, sem explicações."""

        response = ollama.chat(
            model="qwen2.5:14b-instruct-q8_0",
            messages=[{"role": "user", "content": prompt}],
            format="json"  # força saída JSON
        )
        return [TextContent(type="text", text=response['message']['content'])]

    # ... outras tools
```

#### Passo 2: Implementar o MCPAdapter em Go

Em `backend/internal/ingest/mcp_adapter.go` (substituir stub):

```go
package ingest

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/fortvna/radiant-norma/backend/internal/canonical"
)

type MCPAdapter struct {
    client *http.Client
    baseURL string // ex: "http://localhost:8081"
}

func (a *MCPAdapter) Fetch(
    ctx context.Context,
    cfg SourceConfig,
    cadocCode string,
    dataBase time.Time,
) (*canonical.CanonicalDocument, error) {

    // Monta request MCP
    req := mcpRequest{
        Jsonrpc: "2.0",
        ID:      1,
        Method:  "tools/call",
        Params: mcpParams{
            Name: cfg.MCP.ToolName,  // ex: "extract_canonical_from_xlsx"
            Arguments: map[string]any{
                "file_path":  cfg.MCP.ServerName, // reuso de campo
                "cadoc_code": cadocCode,
                "data_base":  dataBase.Format("2006-01-02"),
                "prompt":     cfg.MCP.Prompt,
            },
        },
    }

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        a.baseURL+"/mcp", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := a.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("mcp call: %w", err)
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)

    // Parse resposta MCP → CanonicalDocument
    var mcpResp mcpResponse
    if err := json.Unmarshal(raw, &mcpResp); err != nil {
        return nil, fmt.Errorf("parse mcp response: %w", err)
    }

    var doc canonical.CanonicalDocument
    if err := json.Unmarshal([]byte(mcpResp.Result.Content[0].Text), &doc); err != nil {
        return nil, fmt.Errorf("parse canonical: %w", err)
    }

    return &doc, nil
}
```

#### Passo 3: Registrar o servidor MCP no deploy

Em `docker-compose.yml`:

```yaml
services:
  # ... postgres, api, worker, radar ...

  mcp-server:
    build:
      context: ./mcp-server
      dockerfile: Dockerfile
    container_name: radiant-norma-mcp
    environment:
      OLLAMA_HOST: "http://ollama:11434"
    depends_on:
      - ollama
    ports:
      - "8081:8081"

  ollama:
    image: ollama/ollama:latest
    container_name: radiant-norma-ollama
    volumes:
      - ollama_models:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - capabilities: [gpu]  # só Setup B/C
```

#### Passo 4: Configurar uma fonte MCP no Radiant Norma

```bash
# CLI (a ser criada em cmd/radiant-cli/)
radiant-cli source create \
  --name "Planilha SCR Treasury Q2" \
  --type mcp \
  --server-name "radiant-norma-mcp" \
  --tool-name "extract_canonical_from_xlsx" \
  --prompt "Extrair apenas operações do tipo prefixado, em milhares de R$"
```

Pronto. A partir daqui, o `internal/ingest` chama o LLM, o LLM devolve um `CanonicalDocument`, e o resto do pipeline (`internal/generator` → XML → `internal/audit`) funciona **sem nenhuma alteração**.

---

## 3. Caminho 2: worker de normativos (resumo + classificação)

Não é sobre gerar CADOCs — é sobre **manter a IF atualizada**. Cria em `cmd/worker/normativos.go`:

```go
package worker

import (
    "context"
    "log/slog"

    "github.com/fortvna/radiant-norma/backend/internal/radar"
    "github.com/fortvna/radiant-norma/backend/internal/llm"
    "github.com/fortvna/radiant-norma/backend/internal/mailer"
    "github.com/fortvna/radiant-norma/backend/internal/tickets"
)

// NormativoProcessor é chamado pelo cmd/worker quando internal/radar detecta
// um novo normativo (resolução, carta circular, instrução).
type NormativoProcessor struct {
    radar     *radar.Radar
    llm       *llm.Client
    mailer    *mailer.SMTP
    tickets   *tickets.Internal
    logger    *slog.Logger
}

func (p *NormativoProcessor) HandleNewNormativo(ctx context.Context, alert radar.Alert) error {
    // 1. Baixa o documento (URL está em alert.URL)
    texto, err := download(alert.URL)
    if err != nil {
        return err
    }

    // 2. RAG: busca contexto histórico de normativos similares
    contexto, err := p.llm.RetrieveContext(ctx, texto, topK=5)
    if err != nil {
        p.logger.Warn("rag failed, continuing without context", "err", err)
    }

    // 3. Prompt estruturado para extrair impacto
    prompt := buildNormativoPrompt(texto, contexto, alert.CadocCode)
    resumo, err := p.llm.Complete(ctx, prompt, llm.WithJSONSchema(normativoSchema))
    if err != nil {
        return err
    }

    // 4. Decidir ação baseado em severidade
    switch resumo.Severidade {
    case "BAIXA", "MEDIA":
        return p.mailer.SendToClient(ctx, alert, resumo)
    case "ALTA", "CRITICA":
        if err := p.mailer.SendToClient(ctx, alert, resumo); err != nil {
            return err
        }
        return p.tickets.Create(ctx, tickets.Ticket{
            Title:       fmt.Sprintf("Normativo %s exige revisão", alert.Numero),
            Severity:    tickets.Severity(resumo.Severidade),
            Description: resumo.DescricaoMudanca,
            CadocCode:   alert.CadocCode,
        })
    case "REQUER_DESENVOLVIMENTO":
        if err := p.mailer.SendToClient(ctx, alert, resumo); err != nil {
            return err
        }
        return p.tickets.Create(ctx, tickets.Ticket{
            Title:    fmt.Sprintf("Implementar mudanças do normativo %s", alert.Numero),
            Severity: tickets.SeverityHigh,
            Type:     tickets.TypeDevelopment,
            Estimate: resumo.EstimativaEsforco, // LLM sugere esforço
        })
    }
    return nil
}
```

### O prompt estruturado (chave do sucesso)

```go
func buildNormativoPrompt(texto, contexto, cadocCode string) string {
    return fmt.Sprintf(`Você é um analista regulatório sênior especializado em CADOC %s do BACEN.

TAREFA: Analise o normativo abaixo e produza um resumo estruturado.

HISTÓRICO RELEVANTE (do RAG):
%s

NORMATIVO:
%s

INSTRUÇÕES:
1. Identifique o tipo: RESOLUCAO, CARTA_CIRCULAR, INSTRUCAO, OUTRO
2. Resuma em 3 parágrafos (máx 200 palavras), foco em IMPACTO OPERACIONAL
3. Liste os campos de CADOC %s afetados (códigos ou nomes)
4. Classifique severidade: BAIXA, MEDIA, ALTA, CRITICA
5. Indique se REQUER_DESENVOLVIMENTO (mudança de schema/layout) ou não
6. Se for desenvolvimento, estime esforço em horas-pessoa
7. Escreva em português técnico, sem juridiquês

RESPONDA EM JSON com schema:
{
  "tipo": "RESOLUCAO|CARTA_CIRCULAR|INSTRUCAO|OUTRO",
  "resumo_executivo": "...",
  "campos_cadoc_afetados": ["..."],
  "severidade": "BAIXA|MEDIA|ALTA|CRITICA",
  "requer_desenvolvimento": true|false,
  "estimativa_horas": 0,
  "data_vigencia": "YYYY-MM-DD"
}`, cadocCode, contexto, texto, cadocCode)
}
```

---

## 4. Caminho 3: RAG para Normativos Históricos

Cria em `internal/llm/rag.go`:

```go
package llm

import (
    "context"
    "github.com/fortvna/radiant-norma/backend/internal/chroma"
    "github.com/fortvna/radiant-norma/backend/internal/llm/embeddings"
)

type RAG struct {
    chroma    *chroma.Client
    embedder  *embeddings.BGEM3 // 560MB, sempre em VRAM
    generator *Generator         // Qwen2.5-14B
}

func (r *RAG) RetrieveContext(ctx context.Context, query string, topK int) (string, error) {
    // 1. Embedding da query
    vec, err := r.embedder.Embed(ctx, query)
    if err != nil {
        return "", err
    }

    // 2. Busca semântica no ChromaDB
    results, err := r.chroma.Query(ctx, chroma.QueryRequest{
        Collection: "normativos_bacen",
        Embedding:  vec,
        TopK:       topK,
        Filter:     map[string]any{"cadoc_code": "3040"}, // opcional
    })
    if err != nil {
        return "", err
    }

    // 3. Formata contexto para o prompt
    var sb strings.Builder
    for i, doc := range results.Documents {
        fmt.Fprintf(&sb, "[Doc %d — %s]\n%s\n\n", i+1, doc.Metadata["source"], doc.Content)
    }
    return sb.String(), nil
}
```

### Como popular o ChromaDB

Script one-shot em `scripts/ingest_normativos.go`:

```go
// Lê todos os PDFs/DOCXs em _normativos/, parseia com Docling,
// gera embedding com BGE-M3, e insere no ChromaDB.
// Roda uma vez na implantação + cron mensal para atualizações.
```

---

## 5. Pacote interno/llm — esqueleto

Criar em `backend/internal/llm/`:

```go
// client.go — wrapper sobre Ollama HTTP API
package llm

type Client struct {
    baseURL string // http://localhost:11434
    http    *http.Client
}

type GenerateRequest struct {
    Model   string
    Prompt  string
    Format  string // "json" para forçar JSON output
    Options map[string]any
}

func (c *Client) Complete(ctx context.Context, req GenerateRequest) (string, error) {
    // POST /api/generate
}

func (c *Client) Chat(ctx context.Context, model string, msgs []Message) (string, error) {
    // POST /api/chat
}

// embeddings.go — BGE-M3 via Ollama
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
    // POST /api/embeddings  model="bge-m3"
}
```

---

## 6. Como NÃO usar LLM (regras que ficam em Go)

| Caso | Por que não é LLM |
|------|-------------------|
| **Geração de XML** | `internal/generator/gen3040` usa templates Go. LLM erra sintaxe XML, schema, namespaces. |
| **Validação L1-L4** | Regras escritas em Go com parser XML. Determinístico, auditável, repetível. |
| **Cálculo de totais agregados** | `shopspring/decimal` com precisão monetária. LLM alucina em aritmética. |
| **Hash tamper-evident** | `internal/audit` usa SHA-256 chain. Criptográfico, não estatístico. |
| **Submissão STA** | `internal/sta` faz HTTP/TLS com BACEN. Certificados, mTLS — coisa de infra. |

**Regra de ouro:** LLM só entra onde há **ambiguidade semântica**. Onde há regra clara, código claro.

---

## 7. Passo-a-passo de implementação

### Sprint 58 — Fundação
- [ ] Criar `internal/llm/client.go` (wrapper Ollama)
- [ ] Criar `internal/llm/embeddings.go` (BGE-M3)
- [ ] Criar `cmd/mcp-server/main.py` (servidor MCP em Python)
- [ ] Substituir stub `MCPAdapter.Fetch()` por chamada real
- [ ] Adicionar `ollama` e `mcp-server` ao `docker-compose.yml`

### Sprint 59 — Docling + RAG
- [ ] Implementar Docling sidecar
- [ ] Criar `internal/chroma/client.go`
- [ ] Script `scripts/ingest_normativos.go` (popula ChromaDB)
- [ ] Criar `internal/llm/rag.go` (RetrieveContext)

### Sprint 60 — Worker de Normativos
- [ ] Criar `cmd/worker/normativos.go`
- [ ] Templates de email (cliente + interno)
- [ ] Sistema de tickets interno (CLI)

### Sprint 61 — UI/UX
- [ ] Frontend: indicador "esta fonte usa IA" no wizard
- [ ] Frontend: preview do CanonicalDocument antes de gerar XML
- [ ] Frontend: log de auditoria mostrando o que o LLM extraiu

### Sprint 62 — Validação + Benchmarks
- [ ] Golden tests: 50 normativos conhecidos, comparar LLM vs humano
- [ ] Latência alvo: < 30s para extração de XLSX de 100 linhas
- [ ] Acurácia alvo: > 90% de campos canônicos preenchidos corretamente

---

## 8. Custos de Inferência (Setup B, RTX 4090)

| Tarefa | Modelo | Latência | Tokens |
|-------|--------|----------|--------|
| Extrair XLSX 100 linhas → Canonical | qwen2.5:14b-q8 | 8-15s | ~6k out |
| Resumir normativo 30 páginas | qwen2.5:14b-q8 | 20-40s | ~3k out |
| Classificar severidade | qwen2.5:7b-q8 | 2-4s | ~200 out |
| Embedding 1 normativo | bge-m3 | 0.3s | 1024 dim |
| RAG query (top 5) | bge-m3 + chroma | 0.5s | — |

**Custo por tarefa (energia):** < R$ 0,01 (Setup B consome ~250W × 15s = ~1Wh).

---

## 9. Resumo: 3 Formas de Usar IA no Radiant Norma

```
┌──────────────────────────────────────────────────────────────┐
│  FORMA 1: MCPAdapter                                         │
│  "IA converte dado não-estruturado em CanonicalDocument"    │
│  Substitui/estende ManualAdapter, FileAdapter, DBAdapter    │
│  User-facing: aparece no wizard como nova opção de fonte    │
├──────────────────────────────────────────────────────────────┤
│  FORMA 2: Worker de Normativos                               │
│  "IA resume, classifica e roteia mudanças regulatórias"      │
│  Roda em batch, noturno, sem interação humana                │
│  Output: email + ticket automático                           │
├──────────────────────────────────────────────────────────────┤
│  FORMA 3: RAG em validações                                  │
│  "IA explica erros de validação L1-L4 em linguagem natural" │
│  Acessado on-demand via botão '?' no console                 │
│  Output: tooltip com explicação + link pro normativo        │
└──────────────────────────────────────────────────────────────┘
```

As três são **independentes** — pode implementar uma sem as outras. Recomendo começar pelo **MCPAdapter** (Forma 1) porque é o que dá ROI mais imediato e usa a infraestrutura que já está stub'd no código.

---

## 10. Changelog

| Versão | Data | Descrição |
|--------|------|----------|
| 1.0 | 2026-07-09 | Versão inicial — guia de integração LLM |
