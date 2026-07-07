# Sprint 53 Research — AIInsights_v1

## Objetivo

Permitir que o usuário pergunte em linguagem natural sobre o estado do seu ambiente CADOC/SCR/RADAR e receba respostas fundamentadas nos dados reais do tenant.

> **Nota: opt-in.** Feature requer consentimento explícito do tenant (tabela `ifs.llm_insights_enabled`).

## Arquitetura

```
Tenant → POST /v1/insights/ask
         body: { "question": "..." }
         ↓
    InsightsService.Ask(ctx, ifID, question)
         ↓
    1. Busca últimos 50 eventos de audit_events (últimos 30 dias)
    2. Busca últimos envios (regras passadas/falhadas)
    3. Compila contexto → prompt structurado
    4. Envia para LLM (MiniMax/GPT-4o-mini/Claude)
    5. Retorna { answer, sources, model }
```

## Fontes de Dados

| Tabela | Campos relevantes |
|---|---|
| `audit_events` | if_id, actor, action, target, description, created_at |
| `audit_log` | if_id, action, metadata (JSON), created_at |
| `envios` | if_id, period, rules_passed, rules_failed, status, duration_ms, created_at |
| `insights_conversations` | id, if_id, role, content, created_at (nova) |

## API Design

```
POST /v1/insights/ask
Auth: JWT (mesma auth dos outros endpoints)
Body:  { "question": "string", "conversation_id": "optional uuid" }
200:   { "answer": "string", "conversation_id": "uuid", "sources": [...], "model": "string" }
429:   rate limit (5 req/min por tenant)
403:   llm_insights_enabled = false
```

## Conversation History

Nova tabela `insights_conversations` para manter contexto entre perguntas:

```sql
CREATE TABLE insights_conversations (
    id           TEXT PRIMARY KEY,  -- uuid
    if_id        TEXT NOT NULL REFERENCES ifs(id),
    role         TEXT NOT NULL,     -- 'user' | 'assistant'
    content      TEXT NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_conv_if ON insights_conversations(if_id, created_at);
```

O prompt envia os últimos 5 pares da conversa como contexto (máx. ~2000 tokens).

## Contexto enviado ao LLM (structurado)

```
## Dados do Tenant {if_id} — últimos 30 dias

### Últimos Envios
- 2026-06: 12 envios, 94% regras passadas, 6 falhas (D01, D04, D07)
- 2026-05: 11 envios, 91% regras passadas

### Eventos Recentes
- 2026-07-05: envio 2030 aprovado
- 2026-07-03: cross-doc XD-DR01 falhou (1 erro)
- 2026-07-01: novo usuário admin@cliente.com adicionado

## Pergunta do Usuário
{pergunta}

## Instruções
Responda em português. Use os dados acima. Se não houver informação
suficiente, diga que não sabe. Não invente dados.
```

## Rate Limiting

- 5 requisições por minuto por tenant
- Cache de resposta idêntica por 60s (key = hash pergunta+if_id)

## Dependências Externas

- **MiniMax API** (`mmx` package) ou **OpenAI SDK**
- Config: `LLM_PROVIDER` (minimax|openai), `LLM_API_KEY`, `LLM_MODEL`

## Entregas

1. Nova migration `018_insights_conversations.sql`
2. `backend/internal/insights/llm.go` — LLM service
3. `backend/internal/insights/conversation.go` — conversation storage
4. `POST /v1/insights/ask` handler
5. Rate limiting (5 req/min/tenant)
6. Feature flag `ifs.llm_insights_enabled`
