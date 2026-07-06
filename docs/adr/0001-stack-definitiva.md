# ADR-0001: Stack definitiva — Go + Postgres + Redis + Next.js

> **Status:** Aceito
> **Data:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

Radiant Norma é um SaaS regulatório multi-tenant para Instituições Financeiras brasileiras. Precisamos escolher a stack que vai sustentar crescimento de 1 → 100 → 1000 tenants sem reescrita, mantendo:

- Latência P95 < 500ms em endpoints de validação.
- Isolamento forte de dados entre IFs (LGPD + auditoria).
- Hiring viável no Brasil (mercado limitado).
- Padrão com a marca Fortvna (que já opera em Go).

## Decisão

| Camada | Tecnologia | Versão |
|---|---|---|
| Backend | Go | 1.25+ |
| HTTP router | chi | v5 |
| DB transacional | PostgreSQL | 16+ |
| Cache/sessão/queue | Redis | 7+ |
| Frontend | Next.js (App Router + RSC) | 15+ |
| UI primitives | shadcn/ui + Radix + Tailwind | 4+ |
| Auth | Keycloak (managed) ou Clerk | latest |
| Billing | Stripe | latest |
| Observability | OpenTelemetry + Grafana Cloud | latest |
| Errors | Sentry | latest |
| Email | Resend | latest |
| Secrets | AWS Secrets Manager | — |
| Hosting | AWS São Paulo (sa-east-1) | — |
| CDN | CloudFront | — |
| CI/CD | GitHub Actions | — |
| IaC | Terraform | latest |
| Containers | Docker multi-stage + distroless | — |
| Orchestration | ECS Fargate | — |

## Consequências

**Positivas:**
- ✅ Padronização Fortvna (já opera em Go, time já conhece).
- ✅ Postgres RLS nativo (multi-tenant sem código extra).
- ✅ Hiring mainstream (stack conhecida no mercado BR).
- ✅ Latência: Go + Postgres é suficiente pra qualquer carga BACEN.
- ✅ Postgres partitioning + JSONB + FTS resolvem 90% dos casos.
- ✅ Next.js 15 RSC reduz bundle JS e simplifica mutações (Server Actions).

**Negativas:**
- ❌ Sem type-safety end-to-end (Go ↔ TS). Mitigável com `openapi-typescript` gerando tipos a partir do spec.
- ❌ Go GC pauses (~1ms) — irrelevante vs latência BACEN (50ms+).
- ❌ Two-language codebase (Go + TS) — exige disciplina de contrato via OpenAPI.

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Rust** | Latência 2-3× melhor, mas hiring no Brasil é 10× mais difícil. Para SaaS de validação XML (CPU leve), overhead de Rust não se justifica. |
| **Node.js (NestJS)** | Mesma linguagem front/back, mas ecosystem financeiro BR é fraco em Node. Times Fortvna não têm experiência. |
| **Elixir/Phoenix** | LiveView é interessante pra real-time, mas é nicho. Hiring muito difícil. |
| **Python (FastAPI)** | Excelente pra AI/ML, mas performance e concorrência limitadas pra 1000 tenants. |
| **Java (Spring)** | Matera usa (concorrente). Performance OK mas hiring de devs Java seniors é caro + lock-in cultural. |

## Notas de implementação

- Go: `embed.FS` para migrations, sem CGO (`CGO_ENABLED=0`), single static binary.
- Postgres: `pgx/v5` driver (não `database/sql` puro), prepared statements com cache.
- Redis: `go-redis/v9`, usar `miniredis` em tests.
- Next.js: Server Components por padrão, Client Components só onde precisa interactivity.
- shadcn/ui: copiar componentes para `src/components/ui/` (não instalar como dep).
- ECS Fargate: 0.5 vCPU / 1GB RAM por task inicial, autoscale por CPU 70%.