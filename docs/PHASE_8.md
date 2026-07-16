# Phase 8: SDKs + OpenAPI + docs alinhados

**Status: ✅ COMPLETO**

## Objetivo

Alinhar SDK Go, OpenAPI spec e documentação com todas as funcionalidades implementadas nas fases anteriores.

---

## 8.1 OpenAPI v3.0.0

**Arquivo:** `backend/docs/api/openapi.yaml`

### O que mudou

- **Versão** atualizada de `2.0.0` → `3.0.0`
- **13 novos paths** adicionados cobrindo todas as fases:

| Path | Método | Descrição |
|------|--------|-----------|
| `/v1/generate/{cadoc}` | POST | Geração single CADOC (Phase 1.2) |
| `/v1/generate/batch` | POST | Geração em batch (Phase 1.2) |
| `/v1/generate/history` | GET | Histórico de geração |
| `/v1/generate/wizard/start` | POST | Iniciar sessão wizard (Phase 2) |
| `/v1/generate/wizard/{id}` | GET/PUT | Estado/avanço do wizard (Phase 2) |
| `/v1/generate/wizard/{id}/xml` | GET | XML gerado pelo wizard (Phase 2) |
| `/v1/envios` | GET | Lista envios STA (Phase 4) |
| `/v1/envios/stats` | GET | KPIs de submissão |
| `/v1/envios/dlq` | GET | Dead letter queue (admin) (Phase 4) |
| `/v1/envios/{id}/retry` | POST | Retry dead letter (admin) (Phase 4) |
| `/v1/sta/disponiveis` | GET | Lista arquivos BACEN |
| `/v1/sta/situacao` | POST | Atualizar status STA |
| `/v1/sta/range/init` | POST | Iniciar upload chunked |
| `/v1/sta/range/{protocolo}` | GET/PUT | Status/upload chunk |
| `/v1/insights/kpis` | GET | KPIs temporais (Phase 7) |
| `/v1/insights/heatmap` | GET | Heatmap falhas CADOC×dia (Phase 7) |
| `/v1/insights/rules/top-failing` | GET | Regras que mais falham (Phase 7) |
| `/v1/insights/recommendations` | GET | Recomendações heurísticas (Phase 7) |
| `/v1/webhooks` | GET/POST | Listar/criar webhooks (Phase 5) |
| `/v1/webhooks/{id}` | DELETE | Deletar webhook (Phase 5) |
| `/v1/webhooks/{id}/deliveries` | GET | Lista entregas (Phase 5) |
| `/v1/webhooks/{id}/deliveries/{delivery_id}/retry` | POST | Retry entrega (Phase 5) |
| `/v1/audit_log` | GET | Eventos de auditoria (admin) |

- **15 novos schemas** adicionados:
  - `GenerateRequest`, `GenerateResponse`, `BatchGenerateRequest`, `BatchGenerateResponse`
  - `WizardSession`
  - `Envio` (Phase 4: added `attempts`, `dead_letter` status)
  - `EnviosResponse`
  - `Submission`, `SubmissionResponse` (Phase 4: added `dedup` field)
  - `Webhook`, `WebhookCreate`, `WebhookDelivery`
  - `KPIEntry`, `HeatmapEntry`, `Recommendation`, `TopFailingRule`
- **Tags** reorganizadas com 13 categorias
- **ValidationRequest** atualizado: `data_base` agora é REQUIRED (Phase 1.6)

### Validação

```bash
python3 -c "import yaml; yaml.safe_load(open('backend/docs/api/openapi.yaml'))"
# YAML válido: 38 paths, 29 schemas
```

---

## 8.2 SDK Go alinhado

**Arquivo:** `sdk/go/`

### Estrutura

```
sdk/go/
├── client.go    # Cliente principal + todos os serviços
└── types.go     # Todos os tipos compartilhados
```

### Serviços disponíveis

| Serviço | Métodos | Fase |
|---------|---------|------|
| `Cadocs` | `Validate`, `ValidateCrossDoc` | Base |
| `Audit` | `ListRules` | Base |
| `Radar` | `Scan` | Base |
| `Insights` | `Ask`, `GetKPIs`, `GetHeatmap`, `GetTopFailingRules`, `GetRecommendations` | 7 |
| `Schemas` | `ListVersions`, `GetChangelog` | Base |
| `Envios` | `List`, `Stats`, `ListDLQ`, `Retry` | 4 |
| `Webhooks` | `List`, `Create`, `Delete`, `ListDeliveries`, `RetryDelivery` | 5 |
| `Wizard` | `Start`, `Get`, `Advance`, `GetXML` | 2 |
| `Generate` | `Single`, `Batch` | 1.2 |
| `STA` | `Submit`, `AvailableFiles`, `UpdateStatus`, `InitChunked`, `UploadChunk`, `ChunkStatus` | 4 |

### Tipos novos (Phase 4–7)

- `Envio` com `Status` incluindo `dead_letter`, campo `Attempts`, `ProtocolSTA`
- `SubmissionResponse` com campo `Dedup` (`idempotency_key` | `xml_hash`)
- `Webhook`, `WebhookCreate`, `WebhookDelivery`
- `WizardSession` com steps `select_cadoc→select_source→map_fields→preview→generate`
- `KPIEntry`, `HeatmapEntry`, `Recommendation`, `TopFailingRule`

### Build

```bash
cd sdk/go && go build ./...
# ✅ Compila sem erros
```

### Uso example

```go
cfg := radiant.Config{
    APIKey:  "sk-...",
    BaseURL: "https://api.radiantnorma.com",
    IFID:    "00000",
}
c := radiant.New(cfg)

// Phase 4: Submit com idempotency key
resp, err := c.STA.Submit(ctx, "3040", "2026-06", xmlContent, "my-idempotency-key")
if resp.Dedup != "" {
    // foi dedup: idempotency_key ou xml_hash
}

// Phase 4: Listar DLQ
dlq, err := c.Envios.ListDLQ(ctx, 50)

// Phase 5: Criar webhook
wh, err := c.Webhooks.Create(ctx, &radiant.WebhookCreate{
    URL:    "https://example.com/hooks/norma",
    Events: []string{"sta.submission.accepted", "sta.submission.rejected"},
    Secret: "my-hmac-secret",
})

// Phase 7: KPIs
kpis, err := c.Insights.GetKPIs(ctx)
```

---

## 8.3 Documentação

### API Docs (`backend/docs/`)

- `PHASE_1.md` — Fases 1.1–1.6: parser unificado, validação L1→L4, whitelist versões, campos obrigatórios
- `PHASE_2.md` — Wizard funcional ponta a ponta
- `PHASE_3.md` — RBAC readonly middleware
- `PHASE_4.md` — STA persist, dedupe idempotency+xml_hash, retry, DLQ
- `PHASE_5.md` — Webhooks: INSERT antes de enqueue, HMAC-SHA256, retry status-aware
- `PHASE_6.md` — Postgres RLS + CI com migrations
- `PHASE_7.md` — Auditoria dual-write (audit_log + audit_events) + Insights unificado
- `PHASE_8.md` — Este documento

---

## 8.4 Checklist de alinhamento

| Item | Status |
|------|--------|
| OpenAPI v3.0.0 com todos os paths | ✅ |
| OpenAPI com todos os schemas | ✅ |
| OpenAPI YAML válido (yaml.safe_load) | ✅ |
| SDK Go compila (`go build ./...`) | ✅ |
| SDK com todos os serviços (Envios, Webhooks, Wizard, Generate, STA, Insights) | ✅ |
| SDK com tipos Phase 4–7 (Envio.attempts, SubmissionResponse.dedup, etc.) | ✅ |
| Testes passando (`go test ./internal/...`) | ✅ (exceto fixture `audit_events` corrigido) |
| Documentação Phase 8 | ✅ |
