# Radiant Norma — Architecture Review

**Versão:** 1.0
**Data:** 2026-07-16
**Classificação:** Confidencial — Clientes enterprise e auditors

> Este documento é uma revisão arquitetural do Radiant Norma, preparado para
> clientes enterprise e auditores de segurança. Destina-se a fornecer uma visão
> técnica abrangente da arquitetura, decisões de design, e postura de segurança
> para avaliação de risco e due diligence.

---

## 1. Visão Geral do Sistema

### 1.1 O que é o Radiant Norma

O Radiant Norma é uma **plataforma B2B de inteligência regulatória** que
automatiza o ciclo de vida completo de documentos regulatórios (CADOCs) junto
ao Banco Central do Brasil (BACEN):

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│   DADOS DE ENTRADA           RADIANT NORMA         STA/BACEN    │
│                                                                  │
│   Planilhas, APIs,           GERAÇÃO                            │
│   PDFs, Bancos ──────────►   VALIDAÇÃO (L1→L4)                  │
│   Dados de Crédito           AUDIT TRAIL                        │
│                              TRANSMISSÃO ──────────────────────► │
│                              TRACKING + DLQ                      │
│                              REAL-TIME EVENTS ─────────────────► │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 Escopo Funcional

| Funcionalidade | Descrição |
|---|---|
| **Geração de CADOCs** | Gera XML válido para 10 tipos de CADOC (3040-SCR, 3050, 2060-DRM, etc.) |
| **Validação L1→L4** | XSD (L1), Semântica (L2), Cross-document (L3), Histórico (L4) |
| **Transmissão STA** | Integração direta com Sistema de Transferência de Arquivos do BACEN |
| **Webhooks** | Notificações push para eventos de submissão e status |
| **AI Insights** | Recomendação de ações regulatórias via LLM (opt-in) |
| **Schema Registry** | Leiautes versionados por data-base; histórico de mudanças BACEN |
| **Real-time SSE** | Stream de eventos para dashboards ao vivo |
| **Multi-tenant** | Isolamento completo por IF com Postgres RLS |

---

## 2. Arquitetura Técnica

### 2.1 Stack Tecnológico

| Camada | Tecnologia | Justificativa |
|---|---|---|
| **API** | Go 1.25+ / chi router | Performance; goroutines nativas para I/O; tipagem estática |
| **Database** | PostgreSQL 16 (production); SQLite (dev) | RLS nativas; WAL para PITR; maturidade |
| **Cache/Rate Limit** | Redis (sliding window) | Atomic counters; distribuição multi-replica |
| **Queue** | PostgreSQL-backed (via `scheduler`) | DLQ com retry; mesma infraestrutura do DB |
| **Real-time** | Server-Sent Events (SSE) | Bidirecional não necessário; simples; proxy-friendly |
| **Observabilidade** | OTel (traces) + Sentry (errors) + Prometheus (metrics) | Vendor-neutral; native Go support |
| **Cloud** | AWS (EC2/RDS/S3/CloudWatch) | Compliance-ready; SOC 2 / ISO 27001 |
| **CDN/WAF** | Cloudflare | DDoS protection; global edge; API shield |
| **SDK** | Go + Python | Linguagens mais comuns em IFs brasileiras |

### 2.2 Arquitetura de Alto Nível

```
                         ┌─────────────────────────────────────────┐
                         │              Cloudflare                  │
                         │    (DDoS protection, WAF, CDN, DNS)     │
                         └──────────────────┬──────────────────────┘
                                            │ HTTPS
                         ┌──────────────────▼──────────────────────┐
                         │           AWS ALB / API Gateway         │
                         │       (Load balancing, SSL termination) │
                         └──────────────────┬──────────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
          ┌─────────▼─────────┐  ┌─────────▼─────────┐  ┌─────────▼─────────┐
          │   API Replica 1  │  │   API Replica 2  │  │   API Replica N  │
          │   (Go / chi)      │  │   (Go / chi)      │  │   (Go / chi)      │
          └─────────┬─────────┘  └─────────┬─────────┘  └─────────┬─────────┘
                    │                       │                       │
                    │         ┌─────────────┼─────────────┐         │
                    │         │             │             │         │
                    │   ┌─────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐   │
                    │   │ PostgreSQL│ │   Redis   │ │    S3     │   │
                    │   │  (RLS)    │ │ (rate lim)│ │ (payloads)│   │
                    │   └───────────┘ └───────────┘ └───────────┘   │
                    │                        │                       │
                    └────────────────────────┼───────────────────────┘
                                             │
                                   ┌─────────▼─────────┐
                                   │    STA / BACEN     │
                                   │   (SCA + WS)       │
                                   └───────────────────┘
```

### 2.3 Fluxo de Dados — Submissão de CADOC

```
CLIENTE (SDK Go / Python / Browser)
         │
         │ POST /v1/validate (validação L1-L4)
         │ POST /v1/generate (geração XML)
         │ POST /v1/sta/submit (submissão)
         ▼
┌─────────────────────────────────────────────────────┐
│  1. VALIDAÇÃO (L1-L4)                              │
│     - L1: XSD schema validation                    │
│     - L2: Regras semânticas (Go, 1.099 regras)     │
│     - L3: Cross-document (ex.: SCR vs. DRM)        │
│     - L4: Histórico (evolução de saldos)           │
│                                                      │
│  2. IDEMPOTENCY CHECK                               │
│     - idempotency-key (1o nível)                    │
│     - xml_hash deduplication (2o nível)             │
│     - Se duplicado: retorna 200 + "already processed"│
│                                                      │
│  3. PERSISTÊNCIA (same transaction)                │
│     - INSERT envios                                 │
│     - INSERT audit_log (hash chain)                 │
│     - INSERT audit_events (legível)                 │
│     - Pub/Sub SSE                                   │
│                                                      │
│  4. STA SUBMIT                                       │
│     - POST to BACEN STA (HTTPS)                     │
│     - Retry com backoff se 429/5xx                  │
│     - DLQ se todas as retries falham                │
│                                                      │
│  5. WEBHOOK DISPATCH (async)                        │
│     - INSERT webhook_deliveries (pending)            │
│     - Enqueue job                                   │
│     - Deliver com HMAC-SHA256                       │
│     - Retry [1,5,15,30,60] min                     │
└─────────────────────────────────────────────────────┘
         │
         ▼
  WEBHOOK → CLIENTE'S ENDPOINT
  SSE → CLIENTE'S DASHBOARD (real-time)
```

---

## 3. Modelo de Segurança

### 3.1 Authentication & Authorization

| Camada | Mecanismo | Implementação |
|---|---|---|
| **API authentication** | JWT RS256 | `backend/internal/auth/`; key rotation via `kid` header |
| **Multi-tenancy isolation** | Postgres RLS (FORCE mode) | `SET LOCAL app.if_id` por transaction |
| **Admin operations** | Separate admin token | `RADIANT_NORMA_ADMIN_TOKEN`; fail-closed em prod |
| **Dev mode** | `RADIANT_DEV_AUTH=1` | Warning em logs; blocked em `RADIANT_ENV=production` |

**JWT Claims:**
```json
{
  "sub": "user-id",
  "if_id": "if-12345",
  "role": "admin | if | readonly",
  "iss": "radiant-norma",
  "aud": "radiant-norma-api",
  "exp": 1753123200
}
```

### 3.2 Rede e Infraestrutura

| Camada | Proteção |
|---|---|
| **DDoS** | Cloudflare (rate limiting + challenge) + AWS Shield |
| **WAF** | Cloudflare WAF rules; OWASP Top 10 protection |
| **TLS** | TLS 1.3 enforced; HTTP → HTTPS redirect; HSTS |
| **API Gateway** | AWS ALB com SSL termination; access logs |
| **Database** | Private subnet (VPC); não exposta à internet |
| **Secrets** | AWS Secrets Manager; nunca em código ou env vars compartilhadas |

### 3.3 Proteções Aplicativas

| Proteção | Detalhe |
|---|---|
| **Rate limiting** | Per-tenant; per-endpoint; Redis sliding window (atomic Lua) |
| **Slowloris** | ReadHeaderTimeout=10s; ReadTimeout=30s |
| **CSRF** | Double-submit cookie pattern; `X-CSRF-Token` header em mutações |
| **Webhook replay** | HMAC-SHA256 signature verification; idempotency key |
| **SQL injection** | Parameterized queries (sqlx); no string concatenation |
| **XSS** | API only (returns JSON); XSS mitigado por contexto |
| **Mass assignment** | Allowlist de campos permitidos (a reforçar — ver PENTEST_REPORT.md) |
| **Error leakage** | `loggerutil.SafeError` — DSN e stack traces nunca vaza |

### 3.4 Audit e Compliance

| Controle | Implementação |
|---|---|
| **Audit log** | Hash chain SHA-256; dual-write (audit_log + audit_events) |
| **Tamper evidence** | `Verify()` recomputa hashes e compara; detecção de manipulação |
| **Data retention** | 7 anos (BACEN requirement); BACKUP_DR_POLICY.md |
| **Encryption at rest** | AES-256 (PostgreSQL + S3) |
| **Encryption in transit** | TLS 1.3 |

---

## 4. Modelo de Dados

### 4.1 Schema Principal

```
┌─────────────────────────────────────────────────────────────┐
│  ifs                                                         │
│  (instituições financeiras — tenants)                        │
│  id, cnpj, nome, tipo, segmento, plano                      │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  envios          │  │  webhook_configs │  │  radar_alerts   │
│  (submissions)   │  │  (webhook        │  │  (BACEN alerts) │
│  id, if_id, ... │  │  per-tenant)     │  │  id, if_id,...  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
              │               │               │
              ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────┐
│  audit_log / audit_events                                  │
│  (tamper-evident hash chain)                                │
│  id, if_id, actor, action, target, payload_hash,           │
│  prev_hash, entry_hash, metadata, created_at                │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Multi-Tenancy com Postgres RLS

```sql
-- Migration 014: RLS com FORCE
ALTER TABLE envios ENABLE ROW LEVEL SECURITY;
ALTER TABLE envios FORCE ROW LEVEL SECURITY;

-- Cada transação seta o app.if_id via SET LOCAL
SET LOCAL app.if_id = 'if-12345';

-- Queries filtram automaticamente por if_id
SELECT * FROM envios WHERE if_id = current_setting('app.if_id');
-- RLS filtra: WHERE if_id = current_setting('app.if_id')
```

---

## 5. Integrações

### 5.1 BACEN STA (Sistema de Transferência de Arquivos)

| Aspecto | Detalhe |
|---|---|
| **Protocolo** | HTTPS REST (SOAP deprecated); WebSocket para async |
| **Autenticação** | Basic Auth (user/pass) + Bearer token |
| **Submission flow** | Idempotency key → XML hash dedup → STA → status polling |
| **Retry** | Exponential backoff: [1s, 5s, 15s, 30s, 60s] |
| **DLQ** | Submissions que falham após 5 retries vão para DLQ |
| **Admin visibility** | `/v1/admin/dlq` — only admin role |

### 5.2 Webhooks

| Aspecto | Detalhe |
|---|---|
| **Signature** | HMAC-SHA256; `X-Radiant-Signature: sha256=<hex>` |
| **Delivery guarantee** | At-least-once (INSERT first, then enqueue) |
| **Retry** | [1, 5, 15, 30, 60] min; max 5 attempts |
| **Idempotency** | Clients should dedupe by event ID |
| **Timeout** | 30s por delivery attempt |

### 5.3 SDKs

| SDK | Repositório | Status |
|---|---|---|
| **Go** | `github.com/fortvna/radiant-norma-go` | ✅ Sprint 77 |
| **Python** | `github.com/quant-risk/radiant-norma` (sdk/py) | ✅ Sprint 78 |
| **OpenAPI** | `backend/docs/api/openapi.yaml` (v3.0.0, 38 paths) | ✅ Sprint 78 |

---

## 6. Desempenho e Escalabilidade

### 6.1 Benchmarks (internos)

| Cenário | Throughput | Latência p99 |
|---|---|---|
| `/v1/validate` (CADOC 3040) | ~500 req/s por réplica | < 200ms |
| `/v1/sta/submit` | ~50 req/s por réplica | < 2s (inclui STA network) |
| `/v1/generate/batch` (10 CADOCs) | ~20 req/s por réplica | < 5s |
| Rate limiter (Redis, sliding) | ~10.000 req/s por instância Redis | < 1ms overhead |

### 6.2 Escala Horizontal

- **API replicas:** Stateless; adicionar réplicas atrás do ALB conforme demanda.
- **PostgreSQL:** Read replicas para queries de leitura (`/v1/schemas`);
  principal para writes. RTO para failover: ~60s.
- **Redis:** Redis Cluster ou ElastiCache para multi-AZ. Failover: ~30s.
- **S3:** Serverless; não requer scale planning.

### 6.3 Limites por Tier

| Recurso | Starter | Professional | Enterprise |
|---|---|---|---|
| **Submissions/mês** | 100 | 1.000 | Ilimitado |
| **IFs por contrato** | 1 | 5 | Ilimitado |
| **Concurrent connections SSE** | 10/IF | 50/IF | 100/IF |
| **Rate limit (heavy)** | 10/min | 10/min | 50/min |
| **Webhook endpoints** | 1 | 5 | 20 |
| **CADOC types** | 3 | 8 | Todos (10) |

---

## 7. Disaster Recovery e Business Continuity

| Métrica | Valor | Detalhes |
|---|---|---|
| **RPO** | ≤ 1 hora | WAL + full backup; PITR |
| **RTO** | ≤ 4 horas | DR region; DNS failover; restore validation |
| **Backup frequency** | A cada 5 min (WAL) + diário (full) | BACKUP_DR_POLICY.md |
| **DR drill** | Trimestral | Simulação de outage completo |
| **Uptime SLA** | 99,9% (Enterprise) | SLA.md |

---

## 8. Compliance e Certificações

| Certificação/Framework | Status | Evidência |
|---|---|---|
| **LGPD** | ✅ Readiness package | `LGPD_COMPLIANCE.md` |
| **SOC 2 Type I** | ⚠️ Readiness (pendente audit externo) | `SOC2_READINESS.md` |
| **SOC 2 Type II** | 🔲 Planejado (Q2-Q3 2027) | Roadmap |
| **BACEN Homologation** | 🔲 Cliente (não Radiant) | OFAC; responsabilidadedo cliente |
| **ISO 27001** | 🔲 Planejado (Q4 2026) | — |
| **PCI-DSS** | N/A | Não applicable (não处理dados de cartão) |

---

## 9. Decisões Arquiteturais (ADRs)

| ADR | Decisão | Justificativa |
|---|---|---|
| ADR-001 | PostgreSQL com RLS para multi-tenancy | Isolamento em DB level; menos código; mais seguro |
| ADR-002 | Go para API | Performance + goroutines para I/O bound; tipagem estática |
| ADR-003 | SSE em vez de WebSocket | Unidirecional sufficient; simpler; proxy-friendly; less complexity |
| ADR-004 | Hash chain para audit log | Tamper-evident; compliance-ready; efficient storage |
| ADR-005 | Redis sliding window para rate limit | Atomic; preciso; não há burstiness |
| ADR-006 | HMAC-SHA256 para webhooks | Padrão industry; simpler que mTLS; sufficient |
| ADR-007 | DLQ em Postgres (scheduler) | Evita infraestrutura extra; mesma familiaridade de ops |

---

## 10. Equipe e Governança

| Papel | Responsável | Contato |
|---|---|---|
| **CTO** | [Nome] | cto@radiant.digital |
| **CISO / DPO** | [Nome] | dpo@radiant.digital |
| **Lead Architect** | [Nome] | [Email] |
| **SRE / DevOps** | [Nome] | [Email] |
| **Security Lead** | [Nome] | security@radiant.digital |

---

## 11. Contato e Suporte

| Canal | Disponibilidade |
|---|---|
| **Email comercial** | comercial@radiant.digital |
| **Suporte técnico** | suporte@radiant.digital |
| **Incidentes de segurança** | security@radiant.digital |
| **Documentação** | docs.radiant.digital |

---

*Este documento é confidencial e coberto pelo NDA entre a Radiant e o destinatário.*
*© 2026 Radiant Tecnologia e Sistemas LTDA. Todos os direitos reservados.*
