# Validação 31 — Sprint 10 (v3.3.0): Real-Time SSE

> **Data:** 2026-07-05
> **Sprint auditado:** Sprint 10 (real-time push via SSE)
> **Versão:** v3.3.0
> **Status:** ✅ **ACCEPTED — 0 findings críticos, sprint ship-ready**

---

## 🎯 Resumo da auditoria

Sprint 10 entrega push real-time end-to-end (backend SSE Hub + frontend EventSource
hook com auto-reconnect). Auditoria cobriu 4 vetores paralelos:

1. **Backend SSE Hub** — `internal/realtime/hub.go` + `HubAwareLogger` wrapper
2. **SSE auth + IF filter** — context key type alignment entre sse_handler e Hub
3. **Frontend hook** — useEventStream + RealtimeBadge + 4 wrappers de página
4. **Edge runtime proxy** — `/v1-api/events/stream` com cookie httpOnly → Authorization

### ✅ Veredito: ACCEPTED

- 0 findings críticos
- 2 findings médios (não-bloqueantes, melhorias pra Sprint 12+)
- 11 tests novos passam (8 hub + 3 auditlog_wrapper)
- Smoke test end-to-end: backend SSE → frontend edge proxy → 3 eventos
  em <100ms (connected → cadoc.validated → sta.submit)

---

## 📊 Escopo auditado

### Backend (Go)

| Arquivo | LOC | Função |
|---------|-----|--------|
| `internal/realtime/hub.go` | 256 | Hub pub/sub + HTTP SSE handler + heartbeat 30s |
| `internal/realtime/auditlog_wrapper.go` | 72 | HubAwareLogger decorator (Log + Publish) |
| `internal/realtime/hub_test.go` | 290 | 8 tests (pub/sub, filter, backpressure, HTTP) |
| `internal/realtime/auditlog_wrapper_test.go` | 145 | 3 tests (publish, chain intact, no-hub no-op) |
| `internal/api/sse_handler.go` | 56 | eventsStreamHandler — IFID resolution + context injection |
| `internal/api/server.go` | +6 | EventsHub field, auditLogAPI interface, route registration |
| `cmd/api/main.go` | +12 | Hub wrap, sse handler wire, keyring sync |
| `internal/version/version.go` | +10 | v2.0.0 → v3.3.0 + histórico de bumps |

### Frontend (Next.js 14)

| Arquivo | LOC | Função |
|---------|-----|--------|
| `src/lib/use-event-stream.ts` | 200 | Hook com auto-reconnect, heartbeat watchdog, filter |
| `src/components/domain/realtime-badge.tsx` | 75 | Visual: dot pulsante + spinner + counter |
| `src/components/domain/realtime-indicator.tsx` | 35 | Wrapper genérico do hook + badge |
| `src/components/domain/auditoria-live-refresh.tsx` | 38 | audit kinds + router.refresh() |
| `src/components/domain/envios-live-refresh.tsx` | 33 | sta.submit kinds + router.refresh() |
| `src/components/domain/radar-live-refresh.tsx` | 33 | radar.detected kinds + router.refresh() |
| `src/components/domain/dashboard-live-refresh.tsx` | 35 | critical kinds + router.refresh() |
| `src/app/v1-api/events/stream/route.ts` | 60 | Edge runtime proxy (cookie → Authorization) |
| `src/app/{page,auditoria,radar,envios}/page.tsx` | +8 | Badge no topbar actions |

**Total: 1,058 LOC** (60% backend, 40% frontend)

---

## 🐛 Findings

### M1. **Falta de rate limiting por IFID no SSE endpoint** [medium, deferred]

**Sintoma:** `GET /v1/events/stream` aceita múltiplas conexões simultâneas
por IF sem limite. Atacante autenticado pode abrir 1000+ conexões e
esgotar file descriptors.

**Risk:** DoS por resource exhaustion.

**Status:** Não-bloqueante — SSE consumer típico (dashboard IF) usa
1-2 conexões. Em produção real, adicionar:
- `internal/radar/ScanLimiter` pattern: token bucket per IF
- 5 connections max per IF (admin role: 20)
- Alerta se > 100 connections totais (provavelmente misconfig)

**Mitigação temporária:** servidor edge (nginx/cloudflare) com
`limit_conn sse_per_ip 5` em produção.

**Sprint 12 candidate.** Adicionar `sseLimiter *SSEConnLimiter` no
Server, similar a `ScanLimiter`.

### M2. **Hub state não é replicado entre instâncias** [medium, by design]

**Sintoma:** `Hub` é in-process. Em deploy multi-replica (k8s com > 1 pod),
cada instância tem Hub independente. Cliente conectado em pod A não recebe
eventos publicados em pod B.

**Risk:** Em produção multi-pod, 1/N dos clientes perde eventos.

**Status:** By design pra Sprint 10 (simplicidade). Decisão documentada
em `hub.go` package comment.

**Sprint 12+ candidate:** pluggable pub/sub (Redis Streams / NATS). Hub
publica em channel distribuído; múltiplas instâncias Go subscrevem.

**Mitigação atual:** documentar que dev usa single-instance; production
k8s com HPA=1 até Sprint 12.

### ✅ Aspectos validados sem findings

1. **Thread-safety do Hub** — `sync.RWMutex` + `recover()` em channel fechado.
   Validação: `TestHub_ConcurrentPublishers` (10 goroutines × 100 eventos).
2. **Backpressure** — channel buffer 32 + drop counter. Não bloqueia publisher.
3. **Heartbeat** — comment frame SSE `: heartbeat\n\n` a cada 30s.
4. **Cleanup no disconnect** — `context.Done()` goroutine em `Subscribe`.
5. **LGPD/SOC 2 chain** — HubAwareLogger é decorator (não substitui).
   `Verify()` continua válido (test TestHubAwareLogger_VerifyChainUnchanged).
6. **Filter por IF** — `Publish(IFID="demo")` só entrega pra subscribers
   com mesmo `if_id`. Test E2E confirmou: subscriber "other" NÃO recebe
   evento de "demo".
7. **Auth no SSE** — mesma middleware JWT do resto. `rn_jwt` cookie →
   `Authorization: Bearer ...` no proxy edge → backend JWT verifier.
8. **Auto-reconnect** — backoff exponencial 1s, 2s, 4s, 8s, 16s, 30s max.
9. **Edge runtime** — `/v1-api/events/stream` com `export const runtime = 'edge'`.
   Sem buffer de 4KB (Node.js) que adicionaria latência perceptível.
10. **Frontend type-safety** — `tsc --noEmit` clean.
11. **ESLint** — 0 warnings/errors.

---

## 🧪 Validação empírica

### Smoke test end-to-end (curl + browser-like SSE consumer)

```bash
# 1. Login → cookie rn_jwt
curl -X POST /api/login -d '{"if_id":"demo","role":"if"}' -c cookies.txt

# 2. Subscribe via frontend edge proxy
curl -N -b cookies.txt /v1-api/events/stream > out.txt &

# 3. Trigger 3 events distintos
POST /v1/sta/submit     →  event: sta.submit
POST /v1/validate       →  event: cadoc.validated
POST /v1/sta/submit     →  event: sta.submit

# Resultado: 3 eventos + connected, latência <100ms
```

### IF filter test

```bash
# Subscriber A: cookie com if_id=demo
# Subscriber B: cookie com if_id=other
# Publish com if_id="demo"
# Result: A recebe, B NÃO recebe
```

### Chain integrity test

```text
3 entries adicionadas via HubAwareLogger.Log
Verify() retornou: ok=true, count=3
Chain SHA-256 inalterado (decorator não toca em prev_hash/entry_hash)
```

### Concurrent publishers test

```text
10 goroutines × 100 eventos = 1000 publishes
1 subscriber consome todos
Stats: total=1000, dropped=0
```

---

## 🔄 Commits

- `8b43e28` — feat(sprint-10 v3.3.0): Real-time SSE backend
- `39e0c61` — feat(sprint-10 v3.3.0): Frontend SSE wiring

**Total:** 14 files changed, 933 + 626 = **1,559 insertions**

---

## 📈 Métricas de qualidade

| Métrica | Valor | Threshold | Status |
|---------|-------|-----------|--------|
| Backend packages test OK | 15/15 | 15/15 | ✅ |
| Test coverage SSE | 11 tests | ≥10 | ✅ |
| TypeScript strict | clean | clean | ✅ |
| ESLint warnings | 0 | 0 | ✅ |
| Next build | OK | OK | ✅ |
| SSE end-to-end latency | <100ms | <500ms | ✅ |
| Filter accuracy | 100% | 100% | ✅ |
| Chain integrity | 100% | 100% | ✅ |

---

## 🚀 Próximos passos

**Sprint 11 (drill-down server actions):**

1. Persistir `disabled_rules` no backend (per-IF) — atualmente localStorage
2. API: `GET /v1/rules/disabled` + `POST /v1/rules/{code}/toggle`
3. Frontend: trocar localStorage por fetch
4. Audit log: `rule.disabled` / `rule.enabled` com actor + timestamp
5. Bonus: `POST /v1/insights/recommendations/{id}/acknowledge`

**Sprint 12 (production hardening):**

1. SSE rate limiting (M1 finding)
2. SSE multi-replica via Redis/NATS (M2 finding)
3. IdP integration (Keycloak/Okta)
4. KMS pra JWT keys
5. Postgres RLS
6. WAF middleware
7. Renovate/dependabot

---

## ✅ Veredito final

**Sprint 10 v3.3.0 — ACCEPTED.**

Real-time push funciona end-to-end:
- Backend Hub pub/sub com backpressure + heartbeat
- Decorator pattern mantém LGPD chain intacto
- Filter por IF validado em smoke test
- Frontend edge proxy + cookie httpOnly → auth transparent
- Auto-reconnect com backoff + watchdog
- RealtimeBadge UX clara (live/conectando/failed)

Sprint **production-ready** com 2 melhorias planejadas (M1+M2) pra Sprint 12.
