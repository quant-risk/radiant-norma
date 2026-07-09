# ADR-0008: Adapter Pattern para Conectores de Input

> **Status:** Proposto (implementação: Sprint 57 — não iniciado)
> **Data:** 2026-07-08
> **Decisor(es):** Henrique Costa

## Contexto

IFs heterogêneas disponibilizam dados de formas completamente diferentes:
- SCD pequena: planilha Excel preenchida manualmente
- Banco médio: API REST do core banking
- IP de pagamento: conexão direta ao banco de dados Oracle
- Fintech BaaS: agente AI (Claude/Cursor) consome MCP do Radiant
- Cooperativa: PDF de relatório mensal + upload manual

O motor de geração precisa ser **agnóstico da fonte de dados**.

## Problema

Se o motor de geração depender de uma fonte específica (ex: só aceita API REST), perdemos segmentos inteiros de mercado. Se aceitarmos tudo em um único módulo, ele vira uma bagunça intestável.

## Decisão

**Adapter Pattern plugável.** Cada fonte de dados é um adapter separado atrás de uma interface comum:

```go
type SourceAdapter interface {
    Name() string                                    // "manual" | "file" | "api" | "db" | "mcp"
    Fetch(ctx context.Context, cfg SourceConfig) (*CanonicalDocument, error)
    ValidateConfig(cfg SourceConfig) error
    HealthCheck(ctx context.Context, cfg SourceConfig) error
}

type SourceConfig struct {
    ID     string
    IFID   string
    Type   string  // "manual" | "file" | "api" | "db" | "mcp"
    Config json.RawMessage  // specifics por tipo
}
```

**5 conectores (Sprint 57):**

| Adapter | Input | Output |
|---|---|---|
| `ManualAdapter` | Form UI Next.js | `CanonicalDocument` |
| `FileAdapter` | PDF/XLSX/DOCX | `CanonicalDocument` (extração via LLM) |
| `APIAdapter` | REST API do cliente | `CanonicalDocument` |
| `DBAdapter` | PostgreSQL/Oracle/MySQL (read-only) | `CanonicalDocument` |
| `MCPAdapter` | MCP server (tools generate/validate/submit) | `CanonicalDocument` |

**Fluxo:**
1. IF configura fonte em `/console/sources` (CRUD de `SourceConfig`)
2. Wizard de geração consome `SourceAdapter` via interface — zero conhecimento do tipo
3. Adapter retorna `CanonicalDocument` tipado
4. Generator consome `CanonicalDocument` — desconhece a fonte

## Consequências

**Positivas:**
- ✅ IF escolhe o conector que faz sentido pra ela
- ✅ Adicionar novo conector = novo package, zero impacto nos existentes
- ✅ Teste isolado por conector
- ✅ CanonicalDocument é o contrato estável entre adapters e generators

**Negativas:**
- ❌ Mais abstrações — tradeoff aceito pela flexibilidade
- ❌ Cada adapter precisa de manejo de erros específico

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| Unificar tudo num único módulo de "ingestão" | God module — impossível de manter/testar |
| Só API REST (mais simples) | Exclui SCDs e cooperativas que não têm API |
| Tudo via LLM (single orchestrator) | Caro em escala, impossível auditar pra compliance |
| Connector marketplace (plugins externos) | Over-engineering na Sprint 1 |
