# Plano Mestre Radiant Norma — Edição "Ouro"

> **Nome-código:** Plano Ouro
> **Data:** 2026-07-05
> **Autor:** Mavis (mavis · Radiant)
> **Owner:** Henrique Costa · Fortvna Risk Solutions
> **Status:** Draft v1.0 · pendente aprovação Henrique
> **Validade:** até 2027-06-30 (4 quarters) — reavaliação trimestral

---

## §0 — Manifesto & Princípios

### 0.1 Manifesto do produto

> **"Radiant Norma é o sistema operacional do compliance regulatório brasileiro. A IF que usa Radiant Norma dorme tranquila — sabe que vai estar na norma no 9º dia útil, sabe que mudou leiaute antes do BACEN publicar, sabe que auditar é apertar um botão."**

Não vendemos software. Vendemos **tranquilidade mensurável**:
- **Latência de adaptação**: ≤ 5 dias úteis entre BACEN publicar e IF conseguir enviar no novo leiaute.
- **Cobertura executável**: ≥ 90% das regras semânticas públicas por CADOC crítico.
- **Audit defensável**: ≤ 60s pra extrair evidência forense completa de qualquer envio.

### 0.2 Princípios de design (não negociáveis)

| # | Princípio | Tradução prática |
|---|---|---|
| **P1** | **Schema-first, código depois** | Toda mudança de leiaute entra via PR no catálogo, sem deploy de código. |
| **P2** | **Multi-tenant desde o nascimento** | Postgres RLS obrigatório em TODA tabela. Nenhum `WHERE if_id =` no código de aplicação — política no DB. |
| **P3** | **Audit tudo, sempre** | Cada mutação emite 1 entrada na hash chain. Verificável em O(n) por qualquer auditor com SELECT. |
| **P4** | **Fail-fast, fail-loud** | Erros de config → panic no boot (não 503 em runtime). Misconfiguração é bug, não feature. |
| **P5** | **Defense-in-depth em secrets** | Senha Sisbacen: nunca em log, nunca em erro serializado, nunca em flag CLI visível em `ps`. |
| **P6** | **YAGNI escrito em pedra** | Cada feature nova justifica contra o princípio. Kill switch em 30 dias se não usou. |
| **P7** | **Boring infra, exciting product** | Postgres, Redis, chi, slog — tudo mainstream. Inovação fica no domínio (cross-doc, radar), não na infra. |
| **P8** | **Documentação é código** | Toda decisão de arquitetura vira ADR no repo. Toda sprint vira SPRINT_*_RESULTS.md validado. |
| **P9** | **Spec antes de implementar** | Toda sprint começa com SPRINT_*_RESEARCH.md que cita fontes, defines contracts, lista acceptance criteria. |
| **P10** | **Operação por contrato** | SLOs publicados (latência P95, uptime, MTTR). Violação = post-mortem público. |

### 0.3 Princípios de UX (Norma Console)

| # | Princípio | Tradução |
|---|---|---|
| **U1** | **Compliance officer é o usuário primário** | Não dev, não CTO. Linguagem é de risco/regulação, não de tech. |
| **U2** | **"0 → envio em 15 min"** | Onboarding self-service. Wizard guiado. Sem precisar ler manual. |
| **U3** | **Erros são didáticos** | Cada erro aponta pro campo XML, mostra exemplo correto, link pra cartilha BACEN. |
| **U4** | **Estado sempre visível** | "Meu último 3040 foi aceito em 03/07 às 14h" — sem scroll, sem filtro. |
| **U5** | **Drill-down sem perder contexto** | Drill num erro mostra a regra, mostra o cadoc, mostra o impacto regulatório, mostra como corrigir. |
| **U6** | **Real-time por padrão** | SSE/WebSocket nativo. Nada de F5. Radar alert aparece em < 3s. |
| **U7** | **Mobile-first no detalhe** | Analista júnior consulta do celular no corredor. Tabelas responsivas. |
| **U8** | **Acessibilidade AA** | Contraste WCAG 2.1 AA. Keyboard navigation. Screen reader. |
| **U9** | **i18n-ready pt-BR → en → es** | Strings em `.po`, datas em `Intl.DateTimeFormat`. Launch pt-BR, en roadmap. |
| **U10** | **Empty states honestos** | Sem "demo data" mockado em produção. Empty = vazio real, com CTA pra popular. |

### 0.4 Market positioning

| Segmento | Concorrentes diretos | Nossa vantagem |
|---|---|---|
| **SCD R$ 50M-1B** | Matera, Mitra (LUZ), cadoc.ai | Lite R$ 1,5k + onboarding 15min + 3040 + 3050 fechado |
| **IP médio R$ 1-10B** | Dattos, TOTVS | 3044 eventos em JSON + DLO/DLI + audit chain pronto SOC 2 |
| **Banco S3-S4** | Matera, Orbi, TIVIT | Cross-doc L3 + ESG first-mover + SLA 99.95% |
| **Fintech BaaS** | (comodity) | White-label + API-first + pricing per-environment |

**Movimento de mercado alvo (4Q):**
1. **Q3 2026:** "O único com 3040 + 3050 production-grade em Go puro, audit chain pronto SOC 2, smoke BACEN homolog validado."
2. **Q4 2026:** "Primeiro SaaS a fechar DLO + DDR + DRL + DLP em código aberto auditável (LGPD compliant)."
3. **Q1 2027:** "First-mover DRSAC ESG 2030 com janela IN BCB 694/2025 — concorrentes ainda vendem PowerPoint."
4. **Q2 2027:** "Plataforma completa 3040/3044/3050/2060/2061/2070/2160/2170/4111/DRSAC + cross-doc L3 — única cobertura 100% do ecossistema SCR."

---

## §1 — Roadmap macro (4 quarters)

### 1.1 Q3 2026 (jul–set) — "Fechar o CORE"

**Tema:** SCD-viável + smoke BACEN real.

| Sprint | Codinome | Entregas |
|---|---|---|
| **28** | VaultIntegration | AWS Secrets Manager / Vault para rotação Sisbacen (Sprint 23+27 encadeiam) |
| **29** | BacenHomologSmoke | Smoke real contra sta-h.bcb.gov.br/staws + www9.bcb.gov.br/senhaws. Credenciais Sisbacen reais. |
| **30** | PostgresRLS | Ativar migration `012_rls_policies.sql` (em `internal/db/migrations/`) + criar migration `014_rls_enforce.sql` com FORCE ROW LEVEL SECURITY. Defense-in-depth multi-tenant. Auditoria SOC 2. |
| **31** | RangeUploadAPI | Handlers REST `/v1/sta/range-*` — fechar YAGNI da Sprint 21. Frontend pode chunked-upload. |
| **32** | Audit3040_v2 | Portar 80+ regras restantes 3040 (B/F/C/S + Agreg + Indiv). Coverage 16% → 60%. |
| **33** | Audit3050 | Portar 170 regras 3050 TXB_V11. XSD já tem no BACEN, parser XML + 170 regras. |
| **34** | FrontendNext | Migrar Console para Next.js 15 App Router + RSC + Server Actions. Design system v2. |
| **35** | CI-Gate | GitHub Actions com pre-commit hook + go test -race + coverage gate + lint. Block merge se falhar. |
| **36** | Observability | OpenTelemetry tracing distribuído. Sentry error tracking. Better Stack/Datadog logs estruturados. |
| **37** | Pilot | Onboarding real 1 SCD-piloto (30-90 dias). Feedback loop. Ajustes. |

**Saída do Q3:** "Radiant Norma Lite" vendável pra SCD pequena.

### 1.2 Q4 2026 (out–dez) — "Multi-CADOC"

**Tema:** DLO + DDR + DRL + DLP + DRSAC research.

| Sprint | Entregas |
|---|---|
| **38** | AuditDLO | Portar 200+ regras 2061 (Limites Operacionais). XSD gerado. |
| **39** | AuditDDR | Portar 11+ regras 2070 (Requerimento Capital Diário). |
| **40** | AuditDRL | Minerar 2160 + portar regras BACEN modelos II. |
| **41** | AuditDLP | Minerar 2170 + portar regras NSFR oficial. |
| **42** | Audit3044 | Engine de regras JSON para eventos (3044). 17 regras T01-T19 portadas. |
| **43** | CrossDoc_v2 | 5+ regras cross-doc: 3040↔4111↔3050↔2160↔2170. |
| **44** | Radar_v2 | Radar detecta mudança de XSD com diff semântico, não só hash. Auto-PR pra atualizar catálogo. |
| **45** | StripeBilling | Billing via Stripe + planos Lite/Pro/Scale/Enterprise + self-service onboarding. |
| **46** | WhiteLabel | Tema customizável (logo, cores, domínio) pra Fintechs BaaS oferecerem Radiant Norma pros clientes. |
| **47** | DRSACResearch | Solicitação formal BACEN pra acessar material 2030. Se negado, plano B (consultoria especializada). |
| **48** | Pilot2 | Segundo cliente-piloto (IP médio). Validar DLO + DDR. |

**Saída do Q4:** "Radiant Norma Pro" vendável pra IP média. 4 CADOCs production-grade.

### 1.3 Q1 2027 (jan–mar) — "ESG first-mover"

**Tema:** DRSAC + 4111 + cross-doc completo.

| Sprint | Entregas |
|---|---|
| **49** | DRSAC_v1 | Catálogo DRSAC 2030 + XSD + 50+ regras iniciais (extraídas via parceria / consultoria). |
| **50** | DRSAC_v2 | Regras semânticas ESG: IPOC × saldo devedor × setor × risco climático. |
| **51** | Audit4111 | Minerar 4111 + XSD + 30+ regras iniciais. |
| **52** | CrossDoc_DRSAC | Regras cross-doc envolvendo DRSAC: 3040 ↔ DRSAC ↔ 4111. |
| **53** | AIInsights_v1 | LLM (Claude / GPT) interpreta audit_log + sugere melhorias em texto natural. Opt-in. |
| **54** | SchemaRegistry_v2 | Versionamento automático por data-base com changelog público. IF recebe notificação. |
| **55** | Pilot3 | Cliente-piloto ESG-first (Banco S3 ou Fintech com reporting ESG). |
| **56** | SOC2_Type1 | Auditoria SOC 2 Type I (readiness). Logs imutáveis, RBAC formal, change management. |

**Saída do Q1:** "Radiant Norma ESG" vendável. Diferencial competitivo massivo.

### 1.4 Q2 2027 (abr–jun) — "Plataforma"

**Tema:** Escala + marketplace + SDK.

| Sprint | Entregas |
|---|---|
| **57** | AuditDRM_Completo | Fechar 2060 (DRM Risco Mercado) — todas as 22+ regras + regras VaR se públicas. |
| **58** | AuditDLI | Fechar 2062 (DLI Limites Individuais). |
| **59** | SDK_GO | Go SDK público (github.com/fortvna/radiant-norma-go). Geração automática via OpenAPI. |
| **60** | SDK_Python | Python SDK pra data science + audit analytics. |
| **61** | Webhooks | Outbound webhooks: evento de envio aceito, evento de erro, evento de mudança de leiaute. |
| **62** | Marketplace | Catálogo público de regras customizadas por IF (compartilhável, versionado, auditado). |
| **63** | MultiRegion | Replicação multi-região (BR-SP1 primário, BR-SP2 disaster recovery). |
| **64** | Pilot4 | Banco S3-S4 piloto (venda enterprise). |
| **65** | SOC2_Type2 | SOC 2 Type II (período auditado). Penetration test anual. |
| **66** | SeriesA_Raise | (opcional) Documentação pra Series A — métricas, MRR, NRR, NPS. |

**Saída do Q2:** "Radiant Norma Enterprise" — plataforma regulatória end-to-end.

### 1.5 Milestones e kill criteria

| Marco | Data | Métrica de sucesso | Kill switch |
|---|---|---|---|
| **M1 — Piloto pagante** | 2026-09-30 | 1 SCD pagou R$ 1,5k/mês por 3 meses | Cancelamento = pivotar GTM |
| **M2 — 10 clientes** | 2026-12-31 | 10 IFs ativas, NPS ≥ 40, churn < 5%/mês | < 5 clientes = pivotar pricing/segmento |
| **M3 — ESG vendido** | 2027-03-31 | 1 cliente DRSAC ativo (mesmo beta) | 0 clientes = DRSAC ainda é vapor |
| **M4 — Series A ready** | 2027-06-30 | R$ 100k MRR, NRR > 110%, payback < 12 meses | < R$ 50k MRR = bootstrap mode |

---

## §2 — Arquitetura-alvo (SaaS production-grade)

### 2.1 Stack definitiva

| Camada | Tecnologia | Justificativa |
|---|---|---|
| **Linguagem backend** | Go 1.25+ | Padronização Fortvna, stdlib-first, GC OK pra latência BACEN |
| **HTTP router** | `go-chi/chi/v5` | Stdlib-compatible, sem magia |
| **DB transacional** | **PostgreSQL 16** (não SQLite) | Multi-tenant RLS nativo, JSONB, FTS, partitioning |
| **DB cache/sessão** | **Redis 7** | Rate limiter plugável, SSE pub/sub multi-replica, session cache |
| **Queue/worker** | **PostgreSQL SKIP LOCKED** (não RabbitMQ) | Zero infra extra, at-least-once delivery, auditável |
| **HTTP framework (front)** | **Next.js 15 (App Router + RSC)** | Server Components reduzem bundle, Server Actions simplificam CRUD |
| **UI** | **shadcn/ui + Radix + Tailwind 4** | Componentes acessíveis, customizáveis, sem vendor lock |
| **State (front)** | **TanStack Query v5 + Zustand** | Server state + UI state separados |
| **Forms** | **react-hook-form + zod** | Type-safe validation, perf |
| **Charts** | **Recharts 2** (substituível por Tremor/victory se necessário) | SVG nativo, acessível |
| **Auth** | **Keycloak** (managed) ou **Clerk** | SSO/SAML/OIDC pra IFs enterprise |
| **Billing** | **Stripe** | Planos + webhooks + customer portal |
| **Observability** | **OpenTelemetry → Grafana Cloud** | Tracing + metrics + logs unificados |
| **Errors** | **Sentry** | Stack traces + release tracking |
| **Email** | **Resend** | Transacional (alertas Radar) |
| **Secrets** | **AWS Secrets Manager** (gerenciado) | Rotação Sisbacen automática |
| **Hosting** | **AWS São Paulo (sa-east-1)** | LGPD compliance, latência BACEN |
| **CDN** | **CloudFront** | Assets + API caching |
| **CI/CD** | **GitHub Actions** | Self-hosted runner em AWS SP |
| **IaC** | **Terraform** | Multi-conta, multi-região |
| **Containers** | **Docker multi-stage + distroless** | Imagem < 50MB, sem shell |
| **Orchestration** | **ECS Fargate** (não k8s) | Simpler, AWS-native, menor ops overhead |

### 2.2 Topologia

```
                         ┌─────────────────────────────┐
                         │  Cloudflare (DNS + WAF +   │
                         │  DDoS protection)           │
                         └──────────────┬──────────────┘
                                        │
                         ┌──────────────▼──────────────┐
                         │  CloudFront (CDN + TLS)    │
                         └──────────────┬──────────────┘
                                        │
              ┌─────────────────────────┴─────────────────────────┐
              │                                                    │
   ┌──────────▼──────────┐                              ┌──────────▼──────────┐
   │  Next.js (Console)  │                              │  Radiant Norma API  │
   │  ECS Fargate SP1a   │                              │  ECS Fargate SP1a   │
   │  RSC + Server Acts  │                              │  Go + chi + OTel    │
   └──────────┬──────────┘                              └──────────┬──────────┘
              │                                                    │
              │ HTTPS · JWT (httpOnly, SameSite=Strict)            │
              └────────────────────────────┬───────────────────────┘
                                           │
              ┌────────────────────────────┼────────────────────────────┐
              │                            │                            │
   ┌──────────▼──────────┐         ┌───────▼────────┐         ┌────────▼─────────┐
   │  PostgreSQL 16      │         │  Redis 7       │         │  S3              │
   │  Multi-AZ (RDS)     │         │  ElastiCache   │         │  - CADOCs ZIP    │
   │  + RLS              │         │  - rate limit  │         │  - audit log     │
   │  + pg_cron          │         │  - SSE pubsub  │         │    (WORM)        │
   │  + partitioning     │         │  - cache       │         │  - exports       │
   │  by tenant + date   │         │                │         │                  │
   └─────────────────────┘         └────────────────┘         └──────────────────┘
                                           │
              ┌────────────────────────────┴────────────────────────────┐
              │                                                         │
   ┌──────────▼──────────┐                                   ┌──────────▼──────────┐
   │  Keycloak (IdP)     │                                   │  AWS Secrets Mgr   │
   │  SSO + SAML + OIDC  │                                   │  Sisbacen rotation  │
   └─────────────────────┘                                   └──────────┬─────────┘
                                                                          │
                                                                 ┌────────▼─────────┐
                                                                 │  BACEN STA +     │
                                                                 │  Senhaws         │
                                                                 │  sta-h.bcb.gov.br│
                                                                 └──────────────────┘
```

### 2.3 Multi-tenancy model

| Aspecto | Decisão |
|---|---|
| **Identificador** | `if_id` (alfanumérico + dash, max 64 chars). CNPJ raiz + suffix. |
| **Isolamento** | **Postgres RLS** — policies em TODA tabela tenant-scoped. |
| **Schema** | Single schema, RLS policies. (Não schema-per-tenant — overhead de migrations). |
| **Routing** | JWT claim `if_id` populado pelo middleware. Toda query roda `SET LOCAL app.current_if_id = '<if_id>'`. |
| **Cross-tenant** | Proibido por padrão. Exceções explícitas (Radar alerts, admin cross-tenant). |
| **Billing** | Stripe customer_id → tenant_id mapping. |

### 2.4 Observability stack

| Sinal | Ferramenta | Cardinalidade |
|---|---|---|
| **Métricas** | Prometheus + Grafana | 5k séries |
| **Logs** | Loki + Grafana | 100 GB/dia (estruturados, redacted) |
| **Traces** | Tempo + Grafana | 10k spans/s |
| **Errors** | Sentry | 1k events/dia |
| **Uptime** | Better Stack | 30 endpoints |
| **Status page** | status.radiant.com.br | público |

### 2.5 SLOs publicados

| SLO | Target | Error budget |
|---|---|---|
| **API availability** | 99.9% (43 min downtime/mês) | 43 min |
| **API latency P95** | < 500ms (validate), < 2s (submit) | < 5% requests |
| **STA round-trip P95** | < 5s | < 5% |
| **SSE delivery** | < 3s end-to-end | < 5% |
| **Radar detection latency** | < 24h após BACEN publicar | < 1% |
| **Audit chain integrity** | 100% | 0 |

---## §3 — Épicos (specs detalhadas)

### 3.1 Épico A — Norma Audit Core

**Objetivo:** Fechar 3040 (até 90% das regras públicas), portar 3050/2060/2061/2070/3044, e transformar o engine em plataforma multi-CADOC.

**Owner:** Eng Lead + 2 Eng Sênior.
**Critério de done:** Cada CADOC production-grade tem ≥ 80% das regras semânticas portadas em Go, XSD gerado, parser tipado, testes com ≥ 80% de cobertura por regra.

#### A.1 — Arquitetura do engine

Hoje o engine tem acoplamento forte com 3040 (`Doc3040`, `Agregado`, `Vencimentos`). Precisamos generalizar:

```go
// Domain entity (target shape)
type Document interface {
    CadocCode() string        // "3040", "3050", etc
    DataBase() string         // YYYY-MM
    CNPJ() string
    // Parse/Serialize específicos por CADOC
}

// Rule interface (generalizada)
type Rule interface {
    Code() string                 // "F02"
    Sheet() string                // "Formato"
    Severity() string             // "E" | "A" | "I"
    CadocCode() string            // qual CADOC aplica
    Applicable(Document) bool     // pré-filtro
    Apply(ctx context.Context, doc Document) error
}

// Registry (multi-CADOC)
type Registry struct {
    rules map[string]map[string]Rule  // cadoc -> code -> rule
}

func (r *Registry) Register(cadoc string, rule Rule) { ... }
func (r *Registry) For(cadoc string) []Rule { ... }
```

#### A.2 — Schema Registry v2 (versionado por data-base)

```go
type Schema struct {
    ID         string    // sha256(cadoc + version + content)
    Cadoc      string
    Version    string    // "2024-01" data-base
    XSD        []byte
    ReleasedAt time.Time
    EOLAt      *time.Time  // null = current
    Source     string      // URL BACEN, sha256
    Changelog  string      // human-readable diff vs previous
}

type SchemaRegistry interface {
    Effective(cadoc string, at time.Time) (*Schema, error)
    List(cadoc string) ([]Schema, error)
    Publish(s Schema) error            // tenant-isolated: only platform admin
    NotifyChange(cadoc string) error   // emits SSE + webhook
}
```

**Publicação:** PR no repo `radiant-norma-schemas/` (catálogo público). CI valida XSD contra XML exemplo do BACEN. Merge → publicação automática via webhook.

#### A.3 — Catálogo de regras (ruleprefs + cadoc_registry)

```go
type Regra struct {
    ID             string
    CadocCode      string
    Sheet          string      // "Básicas" | "Formato" | "Campos" | "Semantica" | "Cross-doc"
    Codigo         string      // "B06", "F02"
    Regra          string      // título curto
    Descricao      string      // markdown
    Gravidade      string      // "E" | "A" | "I"
    DataBaseInicio time.Time
    DataBaseFim    *time.Time
    MensagemErro   string      // template Go text/template
    DocumentacaoURL string
    Implementada   bool        // false = declarada no BACEN mas sem código Go
    TestesPassando bool        // atualized por CI
}

// Customização por IF
type RegraPreferencias struct {
    IfID    string
    RegraID string
    Enabled bool
    Motivo  string
}
```

#### A.4 — Cobertura por CADOC (alvo)

| CADOC | Sprint alvo | Regras no catálogo | Alvo portado | % alvo |
|---|---|---|---|---|
| **3040** | Sprint 32 | 361 | 320 | 88.6% |
| **3050** | Sprint 33 | 170 | 150 | 88.2% |
| **2060** | Sprint 38 | 22 | 20 | 90.9% |
| **2061** | Sprint 38 | 518 | 400 | 77.2% |
| **2070** | Sprint 39 | 11 | 10 | 90.9% |
| **3044** | Sprint 42 | 17 | 17 | 100% |
| **2160** | Sprint 40 | (minerar) | ~30 | n/a |
| **2170** | Sprint 41 | (minerar) | ~30 | n/a |
| **4111** | Sprint 51 | (minerar) | ~30 | n/a |
| **2030** | Sprint 49-50 | (BACEN gate) | ~50 | n/a |

#### A.5 — Acceptance criteria por sprint

- **Sprint 32 (Audit3040_v2):**
  - [ ] 80+ regras portadas com testes unitários (parser XML + cenário positivo + cenário negativo)
  - [ ] Coverage do pacote `audit/rules` ≥ 85%
  - [ ] BCValidador oficial rodado contra mesmos XMLs — diff zero nas regras Básicas/Formato
  - [ ] Documentation: tabela de regras com % cobertura e link pra cartilha BACEN

- **Sprint 33 (Audit3050):**
  - [ ] XSD 3050_Schema_TXB_V4 importado e validado contra XML exemplo
  - [ ] Parser `Doc3050` tipado
  - [ ] 150+ regras implementadas e testadas
  - [ ] Benchmarks: validate < 100ms para 3050 típico (1MB XML)

---

### 3.2 Épico B — Norma Connect (STA + Senhaws)

**Objetivo:** STA WS production-grade contra BACEN homolog E produção. Retry inteligente. DLQ. Observability.

**Owner:** 1 Eng Sênior (backend specialist).
**Critério de done:** 99.5% de envios bem-sucedidos em homolog. Zero perda de protocolo. MTTR < 15min.

#### B.1 — STA client refactor

```go
type STAClient interface {
    Submit(ctx context.Context, sub *Submission) (*Result, error)
    StatusUpload(ctx context.Context, protocolo string) (*UploadStatus, error)
    Download(ctx context.Context, protocolo string) (*DownloadResult, error)
    DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, ...) (*DownloadResult, error)
    ListDisponiveis(ctx context.Context, opts ListOpts) (*ListResult, error)
    AlterarSituacao(ctx context.Context, req AlterarReq) error
    SubmitRange(ctx context.Context, protocolo string, chunk Chunk) error
}

// ReadClient / ChunkedClient segregation mantida (interface segregation)
type RetryConfig struct {
    MaxAttempts     int           // default 3
    InitialBackoff  time.Duration // 500ms
    MaxBackoff      time.Duration // 30s
    Multiplier      float64       // 2.0
    Jitter          float64       // 0.2 (±20%)
    RetryableErrors []error       // *STAError{StatusCode: 5xx}, ErrNetwork
    NonRetryableErrors []error    // *STAError{StatusCode: 4xx}, ErrContentHashMismatch
}

// DLQ pattern
type DeadLetterQueue struct {
    db *sql.DB
}
func (q *DeadLetterQueue) Enqueue(ctx context.Context, op Operation, err error, lastAttempt Attempt) error
func (q *DeadLetterQueue) List(ctx context.Context, ifID string, limit int) ([]FailedOp, error)
func (q *DeadLetterQueue) Retry(ctx context.Context, id string) error
```

#### B.2 — Circuit breaker pra BACEN

```go
type CircuitBreaker struct {
    failures    atomic.Int64
    lastFailure atomic.Int64  // unix nanos
    state       atomic.Int32  // 0=closed, 1=half-open, 2=open
    threshold   int64         // 5
    resetAfter  time.Duration // 60s
}

func (cb *CircuitBreaker) Allow() bool { ... }
func (cb *CircuitBreaker) RecordSuccess() { ... }
func (cb *CircuitBreaker) RecordFailure() { ... }
```

#### B.3 — Senhaws rotation automática

```go
type RotationScheduler struct {
    db            *sql.DB
    secretsMgr    SecretsManager
    senhawsClient *senhaws.Client
    interval      time.Duration  // default 24h
    warningDays   int            // 7 — rotaciona quando vence em ≤ 7 dias
}

func (s *RotationScheduler) Run(ctx context.Context) error {
    for _, tenant := range s.tenantsToCheck() {
        days, err := s.senhawsClient.ConsultarVencimento(ctx, tenant)
        if err != nil { continue }
        if days <= s.warningDays {
            newPass := generateStrongPassword(32)
            if err := s.senhawsClient.AlterarSenha(ctx, newPass); err == nil {
                s.secretsMgr.Update(ctx, tenant.SecretRef, newPass)
                s.auditLog.Log(ctx, "senhaws.rotate.auto", tenant.IFID, ...)
            }
        }
    }
}
```

#### B.4 — Acceptance criteria

- [ ] Smoke contra sta-h.bcb.gov.br/staws validado — todos os endpoints (Submit, Status, Download, List, Range)
- [ ] Retry com backoff exponencial testado com 5xx mock
- [ ] DLQ para falhas permanentes — admin pode ver e reprocessar
- [ ] Circuit breaker — após 5 falhas, abre por 60s
- [ ] Senhaws rotation roda automaticamente a cada 24h
- [ ] Audit log emite `sta.submit.success`, `sta.submit.retry`, `sta.submit.dlq`, `senhaws.rotate.auto`

---

### 3.3 Épico C — Norma Radar (inteligente)

**Objetivo:** Detectar mudanças de leiaute **antes** do BACEN publicar aviso formal. Diff semântico (não só hash).

#### C.1 — Radar v2 (semantic diff)

```go
type RadarSource struct {
    Cadoc      string
    URL        string
    LastFetch  time.Time
    LastSHA    string
    Schema     *Schema
}

type Diff struct {
    Cadoc          string
    Type           string  // "added" | "removed" | "modified" | "deprecated"
    Path           string  // XPath pro campo afetado
    OldValue       any
    NewValue       any
    Severity       string  // "E" | "A" | "I"
    Description    string  // human-readable
    AffectedRules  []string  // regras que referenciam o campo
    RecommendedAction string
}

type RadarService struct {
    sources  []RadarSource
    differ   SemanticDiffer
    notifier Notifier
    fetcher  Fetcher  // httptest-friendly
}

func (s *RadarService) ScanOnce(ctx context.Context) ([]Diff, error) { ... }
func (s *RadarService) AutoPR(ctx context.Context, diffs []Diff) error { ... }
```

#### C.2 — Semantic differ

- Compara XSDs (xsd:annotation, xsd:element, xsd:restriction)
- Compara planilhas de críticas (SCR3040_Criticas.xls)
- Detecta: campo novo, campo removido, tipo mudou, validação ficou mais restritiva

#### C.3 — Acceptance criteria

- [ ] Diff mostra exatamente o que mudou em linguagem de negócio ("Campo IPOC tornou-se obrigatório")
- [ ] Auto-PR aberto no repo `radiant-norma-schemas/` quando detecta mudança
- [ ] Webhook notifica IFs impacted
- [ ] Latência P95 < 24h entre BACEN publicar e Radiant detectar

---

### 3.4 Épico D — Cross-Doc Engine v2 (L3 proprietário)

**Objetivo:** 10+ regras cross-doc cobrindo 3040 ↔ 4111 ↔ 3050 ↔ 2160 ↔ 2170 ↔ DRSAC.

#### D.1 — Rule pattern

```go
type CrossDocRule interface {
    Code() string
    Description() string
    Severity() string
    RequiredDocs() []string  // cadoc_codes necessários
    Apply(ctx context.Context, docs *DocSet) error
}

type DocSet struct {
    docs map[string]Document  // cadoc_code -> Document
}

func (d *DocSet) Get(cadoc string) (Document, bool) { ... }
func (d *DocSet) Has(cadoc string) bool { ... }
```

#### D.2 — Regras cross-doc alvo

| Code | Cadocs | Descrição |
|---|---|---|
| **XD01** | 3040 ↔ 4111 | Saldo 3040 = Saldo 4111 (mesma data-base) |
| **XD02** | 3040 ↔ 3050 | Total 3040 = Somatório 3050 (modalidade × UF) |
| **XD03** | 2160 ↔ 2170 | LCR ≥ 100% (consistência LCR/NSFR) |
| **XD04** | 3040 ↔ DRSAC | IPOC × saldo_3040 × setor_DRSAC coerente |
| **XD05** | 4111 ↔ DRSAC | Operações ESG-classificadas coerentes |
| **XD06** | 3050 ↔ 2160 | Ativos ponderados por risco (APR) usados no LCR |
| **XD07** | 3040 ↔ 4111 ↔ 3050 | Triangulação completa (mesma data-base) |
| **XD08** | 2061 ↔ 2070 | Limites operacionais vs requerimento capital |
| **XD09** | 2160 ↔ 3040 | Liquidez × risco de crédito (cenário estresse) |
| **XD10** | DRSAC ↔ 3040 | Score ESG × taxa de inadimplência |
| **XD11** | DRSAC ↔ 4111 | Operações classificadas ESG × contrapartes 4111 |
| **XD12** | Todos | Data-base consistente entre todos CADOCs do envio |

#### D.3 — Acceptance criteria

- [ ] Engine paraleliza execução (goroutine pool, max NumCPU)
- [ ] Panic recovery em cada rule (validação 12)
- [ ] Audit log emite `crossdoc.run.completed` com `rules_run`, `rules_skipped`, `errors_count`
- [ ] Latência P95 < 5s para tripla (3040+4111+DRSAC)

---

### 3.5 Épico E — Multi-tenant SaaS

**Objetivo:** Postgres RLS ativo, tenant onboarding self-service, billing Stripe.

#### E.1 — Postgres RLS (migration 012 → ativa)

```sql
-- Migration 014_rls_enforce.sql
ALTER TABLE envios ENABLE ROW LEVEL SECURITY;
ALTER TABLE envios FORCE ROW LEVEL SECURITY;  -- superuser também respeita

CREATE POLICY envios_tenant_isolation ON envios
    USING (if_id = current_setting('app.current_if_id', true));

CREATE POLICY envios_tenant_insert ON envios
    FOR INSERT
    WITH CHECK (if_id = current_setting('app.current_if_id', true));

-- Repetir para: audit_log, criticas_disabled, recommendations_ack, envios_stats, rule_prefs
```

**App-side wiring (Go):**

```go
func (s *Server) withTenantContext(ifID string, fn func(tx *sql.Tx) error) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    
    // Defense-in-depth: SET LOCAL valida if_id formato
    if !isValidIfID(ifID) {
        return ErrInvalidIfID
    }
    
    // SET LOCAL aplica só na transação
    if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_if_id = $1", ifID); err != nil {
        return err
    }
    
    if err := fn(tx); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

#### E.2 — Tenant onboarding

```go
type TenantService interface {
    Create(ctx context.Context, in CreateTenantInput) (*Tenant, error)
    Suspend(ctx context.Context, id string, reason string) error
    Resume(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error  // soft delete + LGPD anonymization
}

type Tenant struct {
    ID            string  // if_id
    CNPJ          string
    RazaoSocial   string
    TipoIF        string  // "SCD" | "IP" | "Banco" | "Cooperativa" | "DTVM"
    Plano         string  // "lite" | "pro" | "scale" | "enterprise"
    StripeID      string
    Status        string  // "active" | "suspended" | "deleted"
    CreatedAt     time.Time
    SisbacenUser  string  // encrypted at rest
    SisbacenSecretRef string  // AWS Secrets Manager ARN
    BACENHomolog  bool    // true = sta-h, false = sta prod
}

type CreateTenantInput struct {
    CNPJ         string  // validated against BACEN registry
    RazaoSocial  string
    TipoIF       string
    Plano        string
    SisbacenUser string
    SisbacenPassword string  // transit + at rest encrypted
    AdminEmail   string
}
```

**Onboarding flow (frontend wizard):**
1. CNPJ + razão social (validação automática contra base BACEN pública)
2. Tipo IF + plano (calculadora de preço)
3. Sisbacen credentials (testados in-process antes de salvar)
4. Admin email + 2FA setup
5. **Smoke BACEN real roda automaticamente** — só ativa se passar
6. Welcome dashboard

#### E.3 — Billing (Stripe)

- Planos: Lite R$ 1,5k, Pro R$ 4,5k, Scale R$ 12k, Enterprise custom
- Billing mensal, cobrança por IF ativa
- Stripe Customer Portal pra self-service
- Webhooks: `customer.subscription.updated` → atualiza plano no DB
- MRR + ARR dashboard no Grafana

#### E.4 — Acceptance criteria

- [ ] RLS ativo em TODAS as tabelas tenant-scoped (migration 014 força)
- [ ] Auditoria externa (consultoria) valida isolamento
- [ ] Onboarding completo < 15min (mediano)
- [ ] Billing cycle testado E2E (Stripe test mode → production)

---

### 3.6 Épico F — Norma ESG (DRSAC first-mover)

**Objetivo:** Capturar janela IN BCB 694/2025 (vigência dez/2026). Primeiro SaaS com DRSAC production-grade.

#### F.1 — Material necessário

- Acesso ao sistema restrito BACEN (credencial Santorini / Sisbacen)
- XSD oficial 2030 DRSAC V1.0
- Planilha de críticas (privada)
- Tabela de setores (CNAE × risco climático)
- Tabela de IPOC × operações × saldo devedor

#### F.2 — Roadmap

- **Sprint 47:** Solicitação formal BACEN. Se negado → parceria com consultoria especializada (e.g., B3Bee, Deloitte).
- **Sprint 49:** Catálogo inicial 50+ regras (S, A, AA, AAA scoring).
- **Sprint 50:** Regras cross-doc com 3040 e 4111.
- **Sprint 55:** Piloto cliente ESG-first.

#### F.3 — Aceitação

- [ ] DRSAC validado contra BCValidador (byte-a-byte)
- [ ] Regras IPOC × saldo 3040 × setor executando em < 2s
- [ ] Dashboard ESG com score agregado, drill-down por operação

---

### 3.7 Épico G — Frontend Console v2

**Objetivo:** Next.js 15 App Router + RSC + Server Actions. Design system v2. Performance.

#### G.1 — Information Architecture

```
/                           Dashboard executivo (KPIs)
/envios                     Lista de envios (filtros + drill-down)
/envios/[id]                Detalhe de envio (XML, protocolo STA, audit trail)
/envios/new                 Wizard de novo envio (upload XML/ZIP + validate + submit)
/regras                     Catálogo de regras (habilitar/desabilitar por IF)
/regras/[cadoc]             Detalhes por CADOC
/regras/[cadoc]/[code]      Detalhe de regra (descrição, exemplos, teste)
/radar                      Alertas de mudança de leiaute
/radar/[id]                 Detalhe de alerta (diff, ação recomendada)
/auditoria                  Audit log (filtros avançados, export)
/insights                   KPIs + heatmap + top-failing rules
/insights/recommendations   Recomendações IA (opt-in)
/configuracao               Tenant settings (Sisbacen, plano, equipe)
/configuracao/equipe        RBAC (admin, operator, auditor, read-only)
/configuracao/billing       Stripe customer portal redirect
/docs                       Documentação pública (Markdown + Algolia search)
/api-docs                   OpenAPI explorer (Swagger UI custom)
/login                      SSO redirect (Keycloak)
/onboarding                 Wizard inicial (CNPJ → Sisbacen → smoke → dashboard)
```

#### G.2 — Design system v2

```typescript
// Design tokens (Tailwind config)
const tokens = {
  color: {
    // Brand
    'radiant-50':  '#f0f4ff',
    'radiant-500': '#3b6ef5',
    'radiant-900': '#1a2a5e',
    // Semantic
    'success':    '#10b981',
    'warning':    '#f59e0b',
    'danger':     '#ef4444',
    'info':       '#3b82f6',
    // Neutral
    'gray-50':    '#f9fafb',
    'gray-900':   '#111827',
  },
  font: {
    sans: 'Inter, system-ui, sans-serif',
    mono: 'JetBrains Mono, ui-monospace',
  },
  radius: {
    sm: '4px', md: '8px', lg: '12px', xl: '16px',
  },
  shadow: {
    sm: '0 1px 2px rgba(0,0,0,0.05)',
    md: '0 4px 6px rgba(0,0,0,0.07)',
    lg: '0 10px 15px rgba(0,0,0,0.10)',
  },
  motion: {
    fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
    base: '250ms cubic-bezier(0.4, 0, 0.2, 1)',
    slow: '400ms cubic-bezier(0.4, 0, 0.2, 1)',
  },
};

// Componentes core (shadcn/ui customizado)
- Button, Input, Select, Combobox, DatePicker, Form
- Card, Modal, Drawer, Toast, Tooltip, Popover
- Table (com sorting + filtering + pagination server-driven)
- Tabs, Accordion, Collapsible
- DataGrid (alta densidade, exportável)
- Chart wrappers (LineChart, BarChart, Heatmap, Sankey)
- CodeBlock (syntax highlight XML/JSON)
- DiffViewer (side-by-side pra Radar)
- FileUpload (drag-drop + progress)
- StatusBadge (severity + estado)
- EmptyState (com CTA)
- LoadingSkeleton
- ErrorBoundary
```

#### G.3 — Páginas-chave (wireframes conceituais)

**Dashboard executivo (`/`):**

```
┌────────────────────────────────────────────────────────────────────┐
│  Radiant Norma   [Search]   [Tenant ▾]  [Alerts 3]  [Avatar]    │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Bom dia, Carlos.                                                 │
│                                                                    │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐                 │
│  │ 1,247   │ │ 89%     │ │ 12      │ │ R$ 0    │                 │
│  │ Envios  │ │ Aprov.  │ │ Alertas │ │ Multas  │                 │
│  │ +5%     │ │ +2pp    │ │ -3      │ │   0     │                 │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘                 │
│                                                                    │
│  ┌─ Últimos envios ──────────────┐ ┌─ Próximos prazos ───────────┐ │
│  │ ✓ 3040  Jun/2026   03/07 14h │ │ ⚠ 3050  Jun/2026  09/07   │ │
│  │ ✓ 3050  Jun/2026   03/07 14h │ │ ⚠ 3040  Jul/2026  09/08   │ │
│  │ ✗ 4111  Jun/2026   02/07 09h │ │ ○ DRSAC Q3/2026  31/10    │ │
│  │   └─ 2 erros: S05, C03       │ │                            │ │
│  └───────────────────────────────┘ └────────────────────────────┘ │
│                                                                    │
│  ┌─ Atividade em tempo real ────────────────────────────────────┐ │
│  │ 14:23  ✓ 3040 validado (R$ 12,3M, 1.234 ops)               │ │
│  │ 14:18  ⚠ Radar: 4111 leiaute V12 publicado                 │ │
│  │ 14:12  ✓ 4111 submetido (protocolo 12345)                  │ │
│  │ 14:05  ℹ Senhaws rotacionada automaticamente                │ │
│  └─────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

**Wizard de novo envio (`/envios/new`):**

```
Step 1: Upload           Step 2: Validate         Step 3: Submit         Step 4: Confirmação
┌──────────────┐         ┌──────────────┐         ┌──────────────┐       ┌──────────────┐
│              │         │              │         │              │       │              │
│  [drag XML]  │   →     │  ✓ 312 ok    │   →     │  → BACEN     │   →   │  Protocolo   │
│  ou ZIP aqui │         │  ✗ 2 errors  │         │              │       │  12345       │
│              │         │  ⚠ 5 warnings│         │  [Submit]    │       │  ✓ Aceito    │
│  CADOC: 3040 │         │              │         │              │       │              │
│  Data: 06/26 │         │  [Ver erros] │         │              │       │  [Ver envio] │
│              │         │              │         │              │       │              │
└──────────────┘         └──────────────┘         └──────────────┘       └──────────────┘
```

#### G.4 — Acceptance criteria

- [ ] LCP < 2.5s, FID < 100ms, CLS < 0.1 (Core Web Vitals)
- [ ] Bundle < 200KB (gzipped) por rota
- [ ] Acessibilidade WCAG 2.1 AA
- [ ] Lighthouse score ≥ 90 (todas categorias)
- [ ] Mobile-first (testado em iPhone SE + Pixel 5)
- [ ] i18n pt-BR (default) + en (roadmap)

---

### 3.8 Épico H — Operações (SRE + SOC 2)

**Objetivo:** Operar 24/7 com MTTR < 15min. SOC 2 Type I em Q1 2027, Type II em Q2.

#### H.1 — SLOs e error budgets

Ver §2.5.

#### H.2 — Runbooks

```
runbooks/
├── api-down.md              — API não responde, troubleshooting
├── db-pool-exhausted.md     — Conexão esgotada, leak?
├── bacen-sta-fail.md        — BACEN retornando 5xx, fallback
├── audit-chain-broken.md    — Hash chain inválido, forensic
├── rate-limit-spike.md      — Pico de tráfego, mitigação
├── tenant-isolated.md       — IF isolada por incidente, restore
├── secrets-rotation.md      — Rotação Sisbacen falhou, recovery
└── disaster-recovery.md     — Region down, failover
```

#### H.3 — SOC 2 readiness

- **Change management:** PR + review + CI gates + signed commits
- **Access control:** RBAC + IdP + audit log de mudanças de role
- **Data encryption:** TLS 1.3 in transit, AES-256 at rest
- **Backup:** Postgres PITR (7d) + S3 versioning
- **Monitoring:** uptime, latency, error rate, audit chain integrity
- **Incident response:** on-call rotation (PagerDuty), post-mortem template
- **Vendor management:** AWS, Stripe, Sentry, Keycloak — DUE DILIGENCE documentada

#### H.4 — Acceptance criteria

- [ ] Runbooks testados em game day trimestral
- [ ] SOC 2 Type I audit completo (Q1 2027)
- [ ] Penetration test anual (Q2 2027)
- [ ] MTTR < 15min (mediano últimos 6 meses)
- [ ] Uptime ≥ 99.9% (média móvel 3 meses)

---## §4 — Contracts

> **Convenção:** Todo contrato é público (em `backend/docs/api/openapi.yaml`) e versionado (prefixo `/v1/`). Mudanças breaking → `/v2/`.

### 4.1 REST API contracts

#### 4.1.1 — `/v1/validate` (POST)

**Request:**

```json
{
  "cadoc_code": "3040",
  "data_base": "2024-06",
  "xml": "<Doc3040>...</Doc3040>",
  "content_type": "application/xml"  // opcional, default application/xml
}
```

**Response 200:**

```json
{
  "request_id": "req_abc123",
  "cadoc_code": "3040",
  "data_base": "2024-06",
  "passed": false,
  "errors": [
    {
      "code": "F02",
      "sheet": "Formato",
      "severity": "E",
      "message": "data_base inválida: 2024-13",
      "xml_line": 12,
      "expected": "YYYY-MM (mês 01-12)",
      "documentation_url": "https://docs.radiant.com.br/regras/F02"
    }
  ],
  "warnings": [
    {
      "code": "S05",
      "sheet": "Semantica",
      "severity": "A",
      "message": "limite de crédito R$ 50.000 sem aprovação documentada",
      "xml_line": 234
    }
  ],
  "rules_run": ["B01", "B02", "F01", "F02", "C01", "S05"],
  "rules_skipped": [],
  "duration_ms": 87,
  "schema_version": "2024-01"
}
```

**Errors:**

- `400 Bad Request` — payload malformado, cadoc_code ausente
- `401 Unauthorized` — JWT inválido/ausente
- `403 Forbidden` — IF não autorizada a validar este CADOC
- `413 Payload Too Large` — XML > 10MB
- `422 Unprocessable Entity` — XML sintaticamente inválido
- `429 Too Many Requests` — rate limit
- `500 Internal Server Error` — bug
- `503 Service Unavailable` — engine indisponível

#### 4.1.2 — `/v1/sta/submit` (POST)

**Request:**

```json
{
  "cadoc_code": "3040",
  "data_base": "2024-06",
  "xml": "<Doc3040>...</Doc3040>",
  "cnpj": "12345678",
  "validate_first": true  // opcional, default true
}
```

**Response 200 (aceito):**

```json
{
  "request_id": "req_xyz789",
  "envio_id": "env_202407051423_a1b2",
  "protocolo_sta": "12345",
  "accepted": true,
  "validation": { /* resultado do validate (se validate_first=true) */ },
  "submitted_at": "2024-07-05T14:23:18Z"
}
```

**Response 200 (rejeitado pelo BACEN):**

```json
{
  "request_id": "req_xyz789",
  "envio_id": "env_202407051423_a1b2",
  "protocolo_sta": "12345",
  "accepted": false,
  "rejection": {
    "code": "ERR_412",
    "message": "Documento fora do período de admissão",
    "bacen_url": "https://www.bcb.gov.br/.../critica.html#412"
  },
  "validation": { /* ... */ }
}
```

**Errors:** Mesmos de validate, + `502 Bad Gateway` se BACEN inacessível, `504 Gateway Timeout` se BACEN timeout.

#### 4.1.3 — `/v1/sta/disponiveis` (GET)

**Query params:**
- `data_hora_inicio` (required, RFC3339)
- `identificador_documento` (optional)
- `sistemas` (optional, comma-separated)
- `dependencia` (optional)
- `cursor` (optional, pagination)

**Response 200:**

```json
{
  "arquivos": [
    {
      "protocolo": "67890",
      "tipo_arquivo": "Resposta_Solicitacao",
      "codigo_documento": "3040",
      "sistema": "SCR",
      "tamanho_bytes": 12345,
      "hash_sha256": "abc...",
      "situacao": "A_REC",
      "data_hora_disponibilizacao": "2024-07-05T10:00:00Z"
    }
  ],
  "data_hora_proxima_consulta": "2024-07-05T11:00:00Z",
  "proxima_pagina": null
}
```

#### 4.1.4 — `/v1/crossdoc/validate` (POST)

**Request:**

```json
{
  "cadocs": {
    "3040": "<Doc3040>...</Doc3040>",
    "4111": "<Doc4111>...</Doc4111>",
    "DRSAC": "<Doc2030>...</Doc2030>"
  },
  "data_base": "2024-06"
}
```

**Response 200:**

```json
{
  "request_id": "req_xd_001",
  "passed": false,
  "errors": [
    {
      "code": "XD01",
      "description": "Saldo 3040 ≠ Saldo 4111",
      "severity": "E",
      "details": {
        "saldo_3040": 12345678.90,
        "saldo_4111": 12345000.00,
        "diferenca": 678.90,
        "operacoes_divergentes": ["OP001", "OP045"]
      }
    }
  ],
  "warnings": [],
  "rules_run": ["XD01", "XD02", "XD03"],
  "rules_skipped": ["XD04"],
  "duration_ms": 234
}
```

#### 4.1.5 — `/v1/radar/alerts` (GET)

**Query params:** `unresolved=true`, `cadoc=3040`, `since=2024-07-01`

**Response 200:**

```json
{
  "alerts": [
    {
      "id": "alert_123",
      "cadoc": "3040",
      "type": "schema_change",
      "severity": "E",
      "title": "Campo IPOC tornou-se obrigatório",
      "description": "A partir de 2024-07, campo IPOC passou de opcional para obrigatório conforme IN BCB 530/2024.",
      "diff": {
        "old_value": "optional",
        "new_value": "required"
      },
      "affected_rules": ["F08", "C04"],
      "recommended_action": "Atualize seus XMLs para incluir o campo IPOC. Radiant Norma fará auto-PR no seu catálogo de schemas.",
      "detected_at": "2024-07-04T03:12:00Z",
      "resolved_at": null
    }
  ],
  "total": 1
}
```

#### 4.1.6 — `/v1/audit_log` (GET)

**Query params:** `actor`, `action`, `target`, `since`, `until`, `cursor`, `limit`

**Response 200:**

```json
{
  "entries": [
    {
      "id": 12345,
      "if_id": "12345678",
      "actor": "192.0.2.1",
      "action": "sta.submit",
      "target": "3040",
      "payload_hash": "abc...",
      "prev_hash": "def...",
      "entry_hash": "ghi...",
      "metadata": {
        "envio_id": "env_...",
        "protocolo": "12345",
        "accepted": true
      },
      "created_at": "2024-07-05T14:23:18Z"
    }
  ],
  "next_cursor": "eyJpZCI6MTIzNDZ9",
  "chain_verified": true
}
```

#### 4.1.7 — `/v1/insights/kpis` (GET)

**Response 200:**

```json
{
  "periodo": {
    "inicio": "2024-06-01",
    "fim": "2024-06-30"
  },
  "kpis": {
    "envios_total": 45,
    "envios_aceitos": 42,
    "envios_rejeitados": 3,
    "taxa_sucesso": 0.933,
    "regras_mais_falhadas": [
      {"code": "F02", "count": 8, "ultima_ocorrencia": "2024-06-28"}
    ],
    "tempo_medio_validacao_ms": 87,
    "cadocs_ativos": ["3040", "3050", "4111"]
  },
  "comparacao_periodo_anterior": {
    "envios_total_delta": 5,
    "taxa_sucesso_delta": 0.022
  }
}
```

#### 4.1.8 — `/v1/events/stream` (GET, SSE)

```
event: envio.submitted
data: {"envio_id":"env_...","protocolo":"12345","accepted":true}

event: radar.alert
data: {"alert_id":"alert_123","cadoc":"3040","severity":"E"}

event: audit.emitted
data: {"id":12346,"action":"sta.submit"}
```

Heartbeat a cada 30s. Reconnect automático com `Last-Event-ID`.

#### 4.1.9 — `/v1/webhooks` (POST/DELETE/GET) — out

**Configuração:**

```json
{
  "url": "https://if.com.br/webhooks/radiant",
  "events": ["envio.submitted", "envio.rejected", "radar.alert"],
  "secret": "whsec_..."  // HMAC signing
}
```

**Payload out (HMAC-SHA256 assinado):**

```json
{
  "event": "envio.submitted",
  "timestamp": "2024-07-05T14:23:18Z",
  "data": { /* ... */ }
}
```

Header `X-Radiant-Signature: t=<timestamp>,v1=<hmac>`.

Retry: exponential backoff (1min, 5min, 30min, 2h, 12h). Após 5 falhas → desabilita webhook + notifica admin.

---

### 4.2 Domain contracts

#### 4.2.1 — Entities principais

```go
// Document (genérico, parseado por CADOC)
type Document interface {
    CadocCode() string
    DataBase() time.Time
    CNPJ() string
    Raw() []byte  // XML/JSON original
    Validate() error  // XSD (L1)
}

// Crítica (regra semântica)
type Critica struct {
    ID              string
    CadocCode       string
    Sheet           string  // "Básicas" | "Formato" | "Campos" | "Semantica" | "Cross-doc"
    Codigo          string  // "B06", "F02", "XD01"
    Regra           string  // título
    Descricao       string  // markdown
    Gravidade       string  // "E" | "A" | "I"
    MensagemErro    string  // template Go text/template
    DocumentacaoURL string
    Implementada    bool    // código Go existe?
    TestesPassando  bool
    DataBaseInicio  time.Time
    DataBaseFim     *time.Time
}

// Tenant (IF)
type Tenant struct {
    ID              string  // if_id
    CNPJ            string
    RazaoSocial     string
    TipoIF          TipoIF  // enum
    Plano           Plano   // enum
    Status          Status  // enum
    SisbacenUser    string  // encrypted
    SisbacenSecret  SecretRef
    BACENHomolog    bool
    StripeCustomerID string
    CreatedAt       time.Time
}

// Envio (CADOC submission)
type Envio struct {
    ID            string
    TenantID      string
    CadocCode     string
    DataBase      string
    XMLHash       string  // sha256
    ZIPHash       string
    ProtocoloSTA  *string  // null = pending
    Status        StatusEnvio  // "pending" | "validating" | "rejected_local" | "submitted" | "accepted" | "rejected_bacen" | "failed"
    Validacao     *ValidationResult
    SubmetidoEm   *time.Time
    AceitoEm      *time.Time
    RejeitadoEm   *time.Time
    Erro          *string
    RetryCount    int
    DLQ           bool
}

// AuditEntry (imutável, hash-chained)
type AuditEntry struct {
    ID          int64
    TenantID    string
    Actor       string
    Action      string  // "sta.submit", "validate.run", "rule.toggle"
    Target      string
    PayloadHash string
    PrevHash    string
    EntryHash   string
    Metadata    json.RawMessage
    CreatedAt   time.Time
}

// CrossDocRule (regra L3)
type CrossDocRule interface {
    Code() string
    Description() string
    Severity() string
    RequiredDocs() []string
    Apply(ctx context.Context, docs *DocSet) ([]CrossDocError, error)
}

type CrossDocError struct {
    Code        string
    Description string
    Severity    string
    Details     map[string]any
}
```

#### 4.2.2 — Value objects

```go
type CNPJ string  // 8 dígitos (raiz) — validado
func (c CNPJ) Valid() bool { /* regex + checksum */ }

type DataBase string  // "YYYY-MM"
func (d DataBase) Valid() bool { /* ano + mês */ }
func (d DataBase) ToTime() time.Time { /* primeiro dia do mês */ }
func (d DataBase) Next() DataBase { /* próximo mês */ }

type ProtocoloSTA string  // número BACEN

type Severity string  // "E" | "A" | "I"
const (
    SeverityError   Severity = "E"
    SeverityWarning Severity = "A"
    SeverityInfo    Severity = "I"
)

type StatusEnvio string  // enum

type TipoIF string  // enum

type Plano string  // enum

type Status string  // enum
```

---

### 4.3 Database schema (Postgres)

#### 4.3.1 — Princípios

- **Multi-tenant via RLS**, não schema-per-tenant
- **Audit log imutável** (trigger bloqueia UPDATE/DELETE)
- **JSONB** para metadata flexível
- **Partitioning** por tenant + data_base em tabelas grandes
- **Soft delete** em tenants (LGPD)
- **Timescaledb** opcional para séries temporais (audit_log volume)

#### 4.3.2 — Tabelas principais

```sql
-- Schema: plataforma (cross-tenant)
CREATE SCHEMA plataforma;

-- Tenants
CREATE TABLE plataforma.tenants (
    id                  TEXT PRIMARY KEY,  -- if_id
    cnpj                TEXT NOT NULL UNIQUE,
    razao_social        TEXT NOT NULL,
    tipo_if             TEXT NOT NULL,
    plano               TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'active',
    sisbacen_user       TEXT NOT NULL,
    sisbacen_secret_arn TEXT NOT NULL,  -- AWS Secrets Manager ARN
    bacen_homolog       BOOLEAN NOT NULL DEFAULT true,
    stripe_customer_id  TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT chk_if_id CHECK (id ~ '^[a-zA-Z0-9_-]{1,64}$'),
    CONSTRAINT chk_tipo_if CHECK (tipo_if IN ('SCD', 'SEP', 'IP', 'BANCO', 'COOPERATIVA', 'DTVM', 'FINBaaS')),
    CONSTRAINT chk_plano CHECK (plano IN ('lite', 'pro', 'scale', 'enterprise')),
    CONSTRAINT chk_status CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE INDEX idx_tenants_cnpj ON plataforma.tenants(cnpj);
CREATE INDEX idx_tenants_status ON plataforma.tenants(status) WHERE deleted_at IS NULL;

-- Schemas (catálogo público versionado)
CREATE TABLE plataforma.schemas (
    id              TEXT PRIMARY KEY,  -- sha256(cadoc + version + content)
    cadoc           TEXT NOT NULL,
    version         TEXT NOT NULL,  -- data-base
    xsd             BYTEA NOT NULL,
    changelog       TEXT,
    source_url      TEXT,
    source_sha256   TEXT,
    released_at     TIMESTAMPTZ NOT NULL,
    eol_at          TIMESTAMPTZ,
    UNIQUE(cadoc, version)
);

CREATE INDEX idx_schemas_cadoc_effective ON plataforma.schemas(cadoc, released_at DESC) WHERE eol_at IS NULL;

-- Críticas (regras declaradas — fonte BACEN)
CREATE TABLE plataforma.criticas (
    id                  TEXT PRIMARY KEY,
    cadoc_code          TEXT NOT NULL,
    sheet               TEXT NOT NULL,
    codigo              TEXT NOT NULL,
    regra               TEXT NOT NULL,
    descricao           TEXT NOT NULL,
    gravidade           TEXT NOT NULL CHECK (gravidade IN ('E', 'A', 'I')),
    mensagem_erro       TEXT,
    documentacao_url    TEXT,
    implementada        BOOLEAN NOT NULL DEFAULT false,
    testes_passando     BOOLEAN NOT NULL DEFAULT false,
    data_base_inicio    TIMESTAMPTZ NOT NULL,
    data_base_fim       TIMESTAMPTZ,
    UNIQUE(cadoc_code, codigo, data_base_inicio)
);

CREATE INDEX idx_criticas_cadoc ON plataforma.criticas(cadoc_code);
CREATE INDEX idx_criticas_implementadas ON plataforma.criticas(cadoc_code, implementada, testes_passando);

-- Schema: tenant_data (tenant-scoped, RLS enforced)
CREATE SCHEMA tenant_data;

-- Envios (submissões BACEN)
CREATE TABLE tenant_data.envios (
    id              TEXT PRIMARY KEY,
    if_id           TEXT NOT NULL REFERENCES plataforma.tenants(id),
    cadoc_code      TEXT NOT NULL,
    data_base       TEXT NOT NULL,
    xml_hash        TEXT NOT NULL,
    zip_hash        TEXT NOT NULL,
    protocolo_sta   TEXT,
    status          TEXT NOT NULL,
    validacao       JSONB,
    submetido_em    TIMESTAMPTZ,
    aceito_em       TIMESTAMPTZ,
    rejeitado_em    TIMESTAMPTZ,
    erro            TEXT,
    retry_count     INT NOT NULL DEFAULT 0,
    dlq             BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_status_envio CHECK (status IN ('pending', 'validating', 'rejected_local', 'submitted', 'accepted', 'rejected_bacen', 'failed'))
);

CREATE INDEX idx_envios_tenant_cadoc ON tenant_data.envios(if_id, cadoc_code, data_base DESC);
CREATE INDEX idx_envios_pending ON tenant_data.envios(status) WHERE status IN ('pending', 'submitted');

-- RLS
ALTER TABLE tenant_data.envios ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.envios FORCE ROW LEVEL SECURITY;

CREATE POLICY envios_tenant_select ON tenant_data.envios
    FOR SELECT USING (if_id = current_setting('app.current_if_id', true));

CREATE POLICY envios_tenant_insert ON tenant_data.envios
    FOR INSERT WITH CHECK (if_id = current_setting('app.current_if_id', true));

CREATE POLICY envios_tenant_update ON tenant_data.envios
    FOR UPDATE USING (if_id = current_setting('app.current_if_id', true));

-- Audit log (imutável, hash-chained)
CREATE TABLE tenant_data.audit_log (
    id              BIGSERIAL PRIMARY KEY,
    if_id           TEXT NOT NULL,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    target          TEXT,
    payload_hash    TEXT NOT NULL,
    prev_hash       TEXT NOT NULL,
    entry_hash      TEXT NOT NULL,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL
);

-- Imutabilidade: trigger bloqueia UPDATE/DELETE
CREATE OR REPLACE FUNCTION audit_log_immutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log é imutável';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update
    BEFORE UPDATE OR DELETE ON tenant_data.audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_immutable();

-- RLS
ALTER TABLE tenant_data.audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.audit_log FORCE ROW LEVEL SECURITY;

CREATE POLICY audit_log_tenant_select ON tenant_data.audit_log
    FOR SELECT USING (if_id = current_setting('app.current_if_id', true));

-- Insert SEM RLS (cross-tenant audit allowed, mas if_id é validado na app)
CREATE POLICY audit_log_insert ON tenant_data.audit_log
    FOR INSERT WITH CHECK (true);

-- Radar alerts (cross-tenant — regulatory changes)
CREATE TABLE plataforma.radar_alerts (
    id              BIGSERIAL PRIMARY KEY,
    cadoc           TEXT NOT NULL,
    type            TEXT NOT NULL,
    severity        TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    diff            JSONB,
    affected_rules  TEXT[],
    recommended_action TEXT,
    detected_at     TIMESTAMPTZ NOT NULL,
    resolved_at     TIMESTAMPTZ,
    resolved_by     TEXT
);

-- Regra preferências (per-IF override)
CREATE TABLE tenant_data.rule_prefs (
    if_id           TEXT NOT NULL REFERENCES plataforma.tenants(id),
    critica_id      TEXT NOT NULL REFERENCES plataforma.criticas(id),
    enabled         BOOLEAN NOT NULL DEFAULT true,
    motivo          TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (if_id, critica_id)
);

ALTER TABLE tenant_data.rule_prefs ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.rule_prefs FORCE ROW LEVEL SECURITY;

CREATE POLICY rule_prefs_tenant ON tenant_data.rule_prefs
    USING (if_id = current_setting('app.current_if_id', true));

-- Webhooks
CREATE TABLE tenant_data.webhooks (
    id              TEXT PRIMARY KEY,
    if_id           TEXT NOT NULL REFERENCES plataforma.tenants(id),
    url             TEXT NOT NULL,
    events          TEXT[] NOT NULL,
    secret_hash     TEXT NOT NULL,  -- sha256(secret)
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_delivery_at TIMESTAMPTZ,
    consecutive_failures INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tenant_data.webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.webhooks FORCE ROW LEVEL SECURITY;

CREATE POLICY webhooks_tenant ON tenant_data.webhooks
    USING (if_id = current_setting('app.current_if_id', true));

-- Dead letter queue
CREATE TABLE tenant_data.dlq (
    id              BIGSERIAL PRIMARY KEY,
    if_id           TEXT NOT NULL,
    operation       TEXT NOT NULL,  -- "sta.submit" | "sta.download" | etc
    payload         JSONB NOT NULL,
    last_error      TEXT NOT NULL,
    attempts        INT NOT NULL,
    first_attempt_at TIMESTAMPTZ NOT NULL,
    last_attempt_at TIMESTAMPTZ NOT NULL,
    resolved_at     TIMESTAMPTZ,
    resolved_by     TEXT
);

ALTER TABLE tenant_data.dlq ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.dlq FORCE ROW LEVEL SECURITY;

CREATE POLICY dlq_tenant ON tenant_data.dlq
    USING (if_id = current_setting('app.current_if_id', true));

-- Migrations versionadas
-- Cada migration em arquivo .sql separado, embed.FS, aplicada na ordem
-- Migration 014: RLS enforce (atual)
-- Migration 015+: features pós-Q3
```

#### 4.3.3 — Índices críticos

```sql
-- Audit log: query por tenant + período + action (comum em relatórios)
CREATE INDEX idx_audit_tenant_action_time ON tenant_data.audit_log(if_id, action, created_at DESC);

-- Envios: query por tenant + cadoc + data_base
CREATE INDEX idx_envios_tenant_cadoc_db ON tenant_data.envios(if_id, cadoc_code, data_base);

-- Webhooks: dispatch
CREATE INDEX idx_webhooks_enabled ON tenant_data.webhooks(if_id) WHERE enabled = true;

-- DLQ: pending reprocess
CREATE INDEX idx_dlq_pending ON tenant_data.dlq(if_id, last_attempt_at) WHERE resolved_at IS NULL;
```

---

### 4.4 Event contracts

#### 4.4.1 — Audit log (interno, app-level)

Já descrito. **Crítico:** hash chain + RLS + imutabilidade via trigger.

#### 4.4.2 — SSE (interno, front-level)

```typescript
// Frontend consumption
type RadiantEvent = 
  | { event: 'envio.submitted'; data: { envio_id: string; protocolo: string; accepted: boolean } }
  | { event: 'envio.rejected'; data: { envio_id: string; rejection_code: string; message: string } }
  | { event: 'radar.alert'; data: AlertPayload }
  | { event: 'audit.emitted'; data: { id: number; action: string } }
  | { event: 'validate.completed'; data: { passed: boolean; errors_count: number; duration_ms: number } }
  | { event: 'senhaws.rotated'; data: { if_id: string; days_until_expiry: number } };

// Protocol
- Content-Type: text/event-stream
- Cache-Control: no-cache, no-transform
- Connection: keep-alive
- X-Accel-Buffering: no
- Heartbeat: ": ping\n\n" a cada 30s
- Reconnect: client envia Last-Event-ID header
```

#### 4.4.3 — Webhooks (externo, out)

```typescript
// Webhook delivery
type WebhookPayload = {
  event: string;          // 'envio.submitted'
  timestamp: string;      // ISO 8601
  data: object;           // event-specific
};

// Headers
{
  'Content-Type': 'application/json',
  'X-Radiant-Signature': 't=1625496000,v1=5a7e9c...',  // HMAC-SHA256(secret, timestamp + '.' + body)
  'X-Radiant-Event-ID': 'evt_abc123',
  'X-Radiant-Delivery-ID': 'del_xyz789',
  'User-Agent': 'RadiantNorma-Webhooks/1.0',
}

// Retry policy
1min, 5min, 30min, 2h, 12h — 5 attempts. Após 5 falhas → disable + notify admin.

// Verificação (lado receptor)
const expectedSig = 'sha256=' + crypto.createHmac('sha256', secret).update(`${timestamp}.${body}`).digest('hex');
const valid = crypto.timingSafeEqual(Buffer.from(expectedSig), Buffer.from(signature));
```

---

### 4.5 Service contracts

#### 4.5.1 — STAClient (BACEN STA WS)

```go
package sta

type Submission struct {
    CadocCode string
    DataBase  string
    CNPJ      string
    XML       string
    Zip       []byte  // optional, preferred
    Hash      string  // pre-computed SHA-256 (optional, computed if empty)
}

type Result struct {
    ProtocolSTA string
    Accepted    bool
    Rejection   *Rejection  // null if Accepted=true
}

type Rejection struct {
    Code       string
    Message    string
    BACENURL   string
}

type Client interface {
    Submit(ctx context.Context, sub *Submission) (*Result, error)
    StatusUpload(ctx context.Context, protocolo string) (*UploadStatus, error)
    Download(ctx context.Context, protocolo string) (*DownloadResult, error)
    DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, opts ...RangeOpt) (*DownloadResult, error)
    SubmitRange(ctx context.Context, protocolo string, chunk Chunk) error
    ListDisponiveis(ctx context.Context, opts ListOpts) (*ListResult, error)
    AlterarSituacao(ctx context.Context, req AlterarReq) error
}

// Implementações
type StubClient struct{}  // dev/test — protocolo fake, sempre aceita
type WSClient struct{...}  // production — BACEN STA WS v1.5
type RetryingClient struct { inner Client; cfg RetryConfig }  // wrap com retry
type CircuitBreakingClient struct { inner Client; cb *CircuitBreaker }
type MetricsClient struct { inner Client; metrics Metrics }

// Factory
func NewClientFromEnv(logger *slog.Logger) (Client, error) {
    backend := os.Getenv("RADIANT_STA_BACKEND")
    if backend == "" {
        backend = "stub"
    }
    switch backend {
    case "stub":  return NewStubClient(), nil
    case "ws":    return NewWSClientFromEnv(logger), nil
    default:
        return nil, fmt.Errorf("invalid backend: %s", backend)
    }
}
```

#### 4.5.2 — SenhawsClient

```go
package senhaws

type Config struct {
    BaseURL         string  // https://www9.bcb.gov.br/senhaws (homolog) | https://www3.bcb.gov.br/senhaws (prod)
    User            string  // Sisbacen UUUUUDDDD.operador
    Password        string  // current
    Timeout         time.Duration
    AllowInsecureHTTP bool  // dev only
}

type Client struct{...}

func NewClient(cfg Config) (*Client, error)
func (c *Client) AlterarSenha(ctx context.Context, novaSenha string) error
func (c *Client) ConsultarVencimento(ctx context.Context) (int, error)  // dias até vencer
```

#### 4.5.3 — AuditLogger

```go
package auditlog

type Logger struct {
    db *sql.DB
}

func New(db *sql.DB) *Logger

func (l *Logger) Log(ctx context.Context, ifID, actor, action, target string, payload []byte, metadata any) (*Entry, error)
func (l *Logger) Verify(ctx context.Context) (bool, int, error)  // (chain_valid, count, err)
func (l *Logger) VerifyFrom(ctx context.Context, fromID int64) (bool, int, error)  // incremental

type Entry struct {
    ID          int64
    IFID        string
    Actor       string
    Action      string
    Target      string
    PayloadHash string
    PrevHash    string
    EntryHash   string
    Metadata    json.RawMessage
    CreatedAt   time.Time
}
```

#### 4.5.4 — InsightsService

```go
package insights

type Service struct {
    db *sql.DB
}

type KPIs struct {
    Periodo            Periodo
    EnviosTotal        int
    EnviosAceitos      int
    EnviosRejeitados   int
    TaxaSucesso        float64
    RegrasMaisFalhadas []RuleFailureCount
    TempoMedioValidacaoMs int
    CadocsAtivos       []string
}

func (s *Service) KPIs(ctx context.Context, periodo Periodo) (*KPIs, error)

type Heatmap struct {
    Rows    []string  // cadocs
    Cols    []string  // severities
    Values  [][]int   // counts
}

func (s *Service) Heatmap(ctx context.Context, periodo Periodo) (*Heatmap, error)

type Recommendation struct {
    ID          string
    Type        string  // "rule_tuning" | "process_improvement" | "data_quality"
    Priority    string  // "high" | "medium" | "low"
    Title       string
    Description string
    Impact      string
    ActionURL   string
    AcknowledgedAt *time.Time
    AcknowledgedBy string
}

func (s *Service) Recommendations(ctx context.Context, ifID string) ([]Recommendation, error)
func (s *Service) Acknowledge(ctx context.Context, id string, by string) error
func (s *Service) Unacknowledge(ctx context.Context, id string) error
```

#### 4.5.5 — RadarService

```go
package radar

type Service struct {
    db       *sql.DB
    differ   Differ
    fetcher  Fetcher
    notifier Notifier
}

type Source struct {
    Cadoc    string
    URL      string
    Schedule time.Duration  // poll interval
}

type Alert struct {
    ID                string
    Cadoc             string
    Type              string  // "schema_change" | "critica_change" | "leiaute_change"
    Severity          string
    Title             string
    Description       string
    Diff              json.RawMessage
    AffectedRules     []string
    RecommendedAction string
    DetectedAt        time.Time
    ResolvedAt        *time.Time
}

type Differ interface {
    Diff(old, new []byte) ([]Change, error)
}

func (s *Service) ScanOnce(ctx context.Context, ifID string) ([]Alert, error)
func (s *Service) ResolveAlert(ctx context.Context, alertID int64) error
func (s *Service) ListAlerts(ctx context.Context, ifID string, unresolvedOnly bool, limit int) ([]Alert, error)
```

---

## §5 — Quality gates

### 5.1 Cobertura de testes (mínima por pacote)

| Pacote | Cobertura mínima | Cobertura atual |
|---|---|---|
| `audit/rules` | 85% | 62.8% |
| `audit` | 80% | 77.0% |
| `auditlog` | 95% | 90.8% |
| `auth` | 85% | 71.8% |
| `crossdoc` | 80% | 85.0% |
| `crossdoc/rules` | 70% | 28.3% |
| `db` | 75% | 68.4% |
| `insights` | 80% | 82.9% |
| `loggerutil` | 95% | 96.2% |
| `radar` | 80% | 81.2% |
| `realtime` | 80% | 79.1% |
| `ruleprefs` | 75% | 60.5% |
| `schema` | 80% | 81.6% |
| `senhaws` | 90% | 95.6% |
| `sta` | 85% | 80.0% |
| `worker` | 80% | 86.1% |
| `api` | 80% | 71.6% |
| `cmd/*` (entrypoints) | 60% | varia |

**Regressão de cobertura:** Se um PR diminui coverage de qualquer pacote abaixo do mínimo, CI falha.

### 5.2 Latência (P95)

| Endpoint | Target P95 |
|---|---|
| `POST /v1/validate` (XML < 1MB) | < 500ms |
| `POST /v1/validate` (XML < 10MB) | < 2s |
| `POST /v1/sta/submit` | < 5s (depende BACEN) |
| `GET /v1/schemas` | < 100ms |
| `GET /v1/rules` | < 200ms |
| `GET /v1/audit_log` | < 500ms |
| `GET /v1/insights/kpis` | < 1s |
| `GET /v1/events/stream` (SSE) | < 3s end-to-end |

### 5.3 Segurança

| Item | Requisito | Validação |
|---|---|---|
| **TLS** | 1.3 only | `testssl.sh` em CI |
| **Headers** | HSTS, CSP, X-Frame-Options, etc. | Mozilla Observatory A+ |
| **Secrets** | Zero em logs / errors / responses | Regex scanner em CI |
| **JWT** | RS256, key rotation, kid header | unit + integration tests |
| **Rate limit** | 100 req/min por IF, 10/min em side-effects | load test em CI |
| **CSRF** | Origin check em POST/PUT/DELETE | integration tests |
| **Audit chain** | Verificável em O(n) | CLI `_verify` em CI |
| **Dependencies** | Sem CVEs high/critical | `govulncheck` em CI |
| **SAST** | Sem findings high | `gosec` em CI |
| **Pen test** | Anual (Q2) | relatório armazenado |

### 5.4 Performance budgets

| Item | Budget |
|---|---|
| **API binary size** | < 50MB |
| **API memory (RSS)** | < 500MB por instância |
| **API cold start** | < 2s |
| **DB connection pool** | 20 idle, 100 max |
| **SSE concurrent connections** | 10k por instância |
| **Webhook delivery P95** | < 30s end-to-end |
| **Frontend bundle** | < 200KB por rota (gzipped) |
| **Frontend LCP** | < 2.5s |
| **Frontend FID** | < 100ms |

---

## §6 — Frontend design system (sprint 34)

### 6.1 Tokens (Tailwind config)

```typescript
// tailwind.config.ts
export default {
  theme: {
    extend: {
      colors: {
        radiant: {
          50:  '#f0f4ff',
          100: '#dce6fe',
          200: '#bccffd',
          300: '#8eaffc',
          400: '#5b86f7',
          500: '#3b6ef5',
          600: '#2a55db',
          700: '#2343b0',
          800: '#1f3889',
          900: '#1a2a5e',
          950: '#0f1633',
        },
        success: { 50: '#ecfdf5', 500: '#10b981', 900: '#064e3b' },
        warning: { 50: '#fffbeb', 500: '#f59e0b', 900: '#78350f' },
        danger:  { 50: '#fef2f2', 500: '#ef4444', 900: '#7f1d1d' },
        info:    { 50: '#eff6ff', 500: '#3b82f6', 900: '#1e3a8a' },
        gray: {
          50:  '#f9fafb',
          100: '#f3f4f6',
          200: '#e5e7eb',
          300: '#d1d5db',
          400: '#9ca3af',
          500: '#6b7280',
          600: '#4b5563',
          700: '#374151',
          800: '#1f2937',
          900: '#111827',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      borderRadius: {
        sm: '4px', md: '8px', lg: '12px', xl: '16px', '2xl': '24px',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
        DEFAULT: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
        md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
        lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
        xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
      },
      transitionTimingFunction: {
        'out-expo': 'cubic-bezier(0.19, 1, 0.22, 1)',
      },
      animation: {
        'fade-in': 'fadeIn 200ms ease-out',
        'slide-up': 'slideUp 250ms cubic-bezier(0.19, 1, 0.22, 1)',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
};
```

### 6.2 Componentes core (shadcn/ui customizado)

```typescript
// Design system packages
src/components/ui/
├── button.tsx           // 7 variants (primary, secondary, ghost, danger, success, link, outline)
├── input.tsx            // text, number, currency, cnpj
├── select.tsx           // single, multi, async (com search)
├── combobox.tsx         // command + popover (cmdk)
├── date-picker.tsx      // range, single, presets (último mês, último ano)
├── form.tsx             // react-hook-form + zod resolver
├── card.tsx             // variants: default, elevated, outlined
├── modal.tsx            // sizes: sm, md, lg, xl, full
├── drawer.tsx           // sides: left, right, bottom
├── toast.tsx            // success, error, warning, info + action
├── tooltip.tsx          // delay configurável
├── popover.tsx          // usado por combobox
├── table.tsx            // server-driven (sort, filter, pagination)
├── tabs.tsx             // horizontal, vertical
├── accordion.tsx
├── collapsible.tsx
├── data-grid.tsx        // alta densidade, exportável, virtualizado
├── chart/               // wrappers Recharts com theming
│   ├── line-chart.tsx
│   ├── bar-chart.tsx
│   ├── heatmap.tsx
│   └── sankey.tsx
├── code-block.tsx       // syntax highlight XML/JSON
├── diff-viewer.tsx      // side-by-side Radar diffs
├── file-upload.tsx      // drag-drop + progress + validate
├── status-badge.tsx     // severity + estado (envio, regra, etc)
├── empty-state.tsx      // ícone + título + descrição + CTA
├── loading-skeleton.tsx
├── error-boundary.tsx
├── pagination.tsx
├── breadcrumbs.tsx
└── nav/                 // sidebar, topbar, breadcrumbs
```

### 6.3 Páginas-chave (resumo)

| Rota | Componente principal | Features |
|---|---|---|
| `/` | Dashboard | KPIs + últimos envios + prazos + atividade tempo real |
| `/envios` | DataGrid | Filtros avançados (período, cadoc, status), export CSV/JSON, bulk retry |
| `/envios/[id]` | DetailView | XML viewer, protocolo BACEN, audit trail, retry button |
| `/envios/new` | Wizard | 4 steps, validação inline, submit + tracking |
| `/regras` | SearchableList | Filtro por cadoc, severity, status; enable/disable toggle |
| `/regras/[cadoc]/[code]` | RuleDetail | Descrição, exemplos XML, "testar agora", doc BACEN link |
| `/radar` | AlertList | Cards com diff + ação recomendada |
| `/radar/[id]` | DiffDetail | Side-by-side XSD, affected rules, PR link |
| `/auditoria` | AuditExplorer | Filtros por actor/action/target/período, export, chain verification |
| `/insights` | Dashboard | KPIs + heatmap + top-failing + recomendações IA |
| `/configuracao` | TenantSettings | Sisbacen, plano, equipe, billing, webhooks |
| `/docs` | MDXRender | Documentação pública + Algolia search |
| `/onboarding` | Wizard | CNPJ → Sisbacen → smoke → dashboard |

### 6.4 Critérios de qualidade frontend

- [ ] Lighthouse Performance ≥ 90
- [ ] Lighthouse Accessibility ≥ 95
- [ ] Lighthouse Best Practices = 100
- [ ] Lighthouse SEO ≥ 90
- [ ] LCP < 2.5s em 4G simulado
- [ ] CLS < 0.1
- [ ] INP < 200ms
- [ ] Bundle < 200KB gzipped por rota
- [ ] TypeScript strict mode
- [ ] ESLint + Prettier sem warnings
- [ ] Storybook pra componentes (visual regression testing)

---## §7 — Harness de execução (processo)

> **"Harness"** = o processo + ferramentas + rituais que sustentam a entrega. Não é burocracia, é a **esteira** que permite velocidade sustentável sem perder qualidade.

### 7.1 Ciclo de sprint (2 semanas)

```
Semana 1                                  Semana 2
─────────────────────────────────         ─────────────────────────────────
Seg  Kick-off (1h)                       Seg  Code review intensivo
       RESEARCH.md review                      Validação profunda (prep)
Ter  Pair programming                    Ter  VALIDAÇÃO_*.md (3-4h)
Qua  Implementação                       Qua  Fixes + CHANGELOG
Qui  Implementação + testes              Qui  Smoke E2E + push + tag
Sex  Mid-sprint sync (30min)             Sex  Demo + retrospectiva (1h)
Sáb  (off)                               Sáb  (off)
Dom  (off)                               Dom  (off)
```

### 7.2 Rituais

| Ritual | Quando | Duração | Quem | Output |
|---|---|---|---|---|
| **Sprint kick-off** | Seg S1 | 1h | Squad | SPRINT_*_RESEARCH.md draft |
| **Daily async** | Diário | 5min (Slack) | Dev | Status update |
| **Mid-sprint sync** | Sex S1 | 30min | Squad | Risk assessment |
| **Validação profunda** | Ter S2 | 3-4h | Eng Lead | VALIDAÇÃO_*_DEEPEST.md |
| **Pre-demo smoke** | Qui S2 | 30min | Squad | All green |
| **Demo + retro** | Sex S2 | 1h | Squad + Stakeholders | Video + retro notes |

### 7.3 Definition of Done (DoD) — checklist por sprint

```
PR review
─────────
[ ] PR aberto com template preenchido (contexto, mudanças, testes, breaking changes)
[ ] Code review por ≥ 1 pessoa (≥ 2 para arquivos críticos: auth/, auditlog/, sta/)
[ ] Comentários endereçados
[ ] CI green: lint, build, test, race, coverage, secrets scan, govulncheck
[ ] Signed commit (DCO)
[ ] Squash merge para main

Code quality
────────────
[ ] Sem warnings de linter
[ ] Sem `TODO` sem issue linkada
[ ] Sem código comentado (morto)
[ ] Sem logs com secrets
[ ] Funções públicas documentadas (godoc)
[ ] Interfaces com compile-time assert: var _ Interface = (*Type)(nil)

Tests
─────
[ ] Coverage ≥ mínimo por pacote (ver §5.1)
[ ] Testes positivos + negativos para cada caminho novo
[ ] Testes de race (go test -race)
[ ] Testes de integração para fluxos críticos
[ ] Smoke E2E (11 cenários) passa

Documentation
─────────────
[ ] SPRINT_*_RESEARCH.md (research inicial)
[ ] SPRINT_*_RESULTS.md (deliverables + métricas)
[ ] VALIDAÇÃO_*_DEEPEST.md (validação profunda)
[ ] CHANGELOG.md atualizado
[ ] ADR se decisão arquitetural (em docs/adr/)
[ ] OpenAPI atualizado (se API mudou)

Deploy
──────
[ ] Tag git criada (v3.X.0)
[ ] Imagem Docker construída e pushed
[ ] Staging deploy automático
[ ] Smoke contra staging OK
[ ] Production deploy (manual approval)
[ ] Release notes publicados
```

### 7.4 Templates

#### 7.4.1 — `SPRINT_*_RESEARCH.md`

```markdown
# SPRINT {N} — RESEARCH: {título}

> **Sprint:** {N} ({tag-alvo})
> **Quando:** YYYY-MM-DD
> **Status:** 🔄 Draft → ✅ Aprovado

## TL;DR
(2-3 frases com objetivo + decisão arquitetural)

## Problema
(O que está faltando, com evidência)

## Pesquisa
(Fontes consultadas, links BACEN, refs internas)

## Decisão arquitetural
(Diagrama + tabela de trade-offs + decisão tomada)

## Decisões YAGNI
(Lista explícita do que NÃO vai entrar)

## Decisões de design não-óbvias
(Padrões específicos do codebase que precisam ser respeitados)

## Entregas
(Lista de artefatos: arquivos, endpoints, migrations)

## Acceptance criteria
(Checklist do §7.3)

## Riscos
(Lista + mitigações)

## Próximos passos (Sprint N+1+)
```

#### 7.4.2 — `SPRINT_*_RESULTS.md`

```markdown
# SPRINT {N} — RESULTS: {título}

> **Sprint:** {N} ({tag-alvo})
> **Quando:** YYYY-MM-DD
> **Status:** ✅ Shipped

## TL;DR
(2-3 frases com entrega + métricas)

## Entregas
(Lista de artefatos finalizados)

## Decisões que pagaram
(Padrões que se provaram certos)

## Estatísticas
(Tabela de métricas — coverage, testes, LOC, packages PASS)

## Compatibilidade
(Backward compat, breaking changes)

## Lições aprendidas (carry forward)
(Insights pra próximos sprints)

## Próximos passos (Sprint N+1+)
```

#### 7.4.3 — `VALIDATION_*_DEEPEST.md`

```markdown
# Validação {N} DEEPEST — v{tag}

> **Validador:** {nome}
> **Data:** YYYY-MM-DD
> **Trigger:** (sprint N entregue)
> **Escopo:** (arquivos modificados)
> **Método:** (leitura completa + grep + coverage + re-run tests)

## TL;DR
(Resumo + findings abertos)

## Findings encontrados + fechados
(Por finding: sintoma, risco, fix, verificação)

## Findings NÃO fechados (com justificativa)
(YAGNI / carry-over / trade-off aceito)

## Estatísticas
(Tabela pré vs pós)

## Lições aprendidas
```

#### 7.4.4 — ADR (`docs/adr/NNNN-titulo.md`)

```markdown
# ADR-NNNN: {título}

> **Status:** Proposto | Aceito | Deprecado | Substituído por ADR-XXXX
> **Data:** YYYY-MM-DD
> **Decisor(es):** {nomes}

## Contexto
(Problema + restrições)

## Decisão
(O que foi decidido)

## Consequências
(Positivas, negativas, neutras)

## Alternativas consideradas
(Outras opções + por que não)
```

### 7.5 CI/CD pipeline

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
  push:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true
      
      - name: Lint
        run: |
          bash scripts/lint-no-placeholder.sh
          gofmt -l backend/
          go vet ./...
      
      - name: Build
        run: |
          cd backend
          go build ./...
          for cmd in cmd/*/; do
            go build -o /tmp/bin/$(basename $cmd) ./$cmd
          done
      
      - name: Test
        run: |
          cd backend
          go test -race -timeout 5m ./...
      
      - name: Coverage
        run: |
          cd backend
          go test -coverprofile=coverage.out ./...
          # Falha se coverage < mínimo por pacote
      
      - name: Security
        run: |
          # govulncheck
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
          # gosec
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec ./...
          # secrets scan
          # (gitleaks ou trufflehog)
      
      - name: Smoke E2E
        run: |
          cd backend
          bash scripts/smoke-e2e.sh  # 11 cenários curl
  
  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
      
      - name: Install
        working-directory: frontend
        run: npm ci
      
      - name: Type-check
        working-directory: frontend
        run: npx tsc --noEmit
      
      - name: Lint
        working-directory: frontend
        run: npm run lint
      
      - name: Build
        working-directory: frontend
        run: npm run build
      
      - name: Test
        working-directory: frontend
        run: npm test
  
  docker:
    needs: [backend, frontend]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build & push
        run: |
          docker build -t radiant-norma:${{ github.sha }} -f backend/Dockerfile .
          docker push radiant-norma:${{ github.sha }}
  
  deploy-staging:
    needs: [docker]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy
        run: ./scripts/deploy-staging.sh
  
  # Manual approval para prod (environment protection rule)
```

### 7.6 Rituais mensais

| Rítual | Quando | Output |
|---|---|---|
| **Post-mortem de incidente** | Após qualquer SEV1/SEV2 | docs/postmortems/YYYY-MM-DD-titulo.md |
| **Quarterly business review** | Último dia do quarter | Métricas: MRR, NPS, churn, uptime |
| **Security review** | Mensal | Revisão de deps, secrets, RBAC |
| **Capacity planning** | Mensal | Projeção de uso, scaling needs |
| **Customer advisory board** | Quarterly (4 IFs) | Feedback diretiva de roadmap |

### 7.7 Onboarding de novo dev (1 semana)

```
Dia 1: Setup (Go, Postgres, Redis, repo, IDE)
       Pair com mentor: roda seed, sobe API, abre frontend
       Lê: README + CHANGELOG (últimas 5 versions)
       
Dia 2: Read codebase (audit/, sta/, api/)
       Resolve 1 issue "good first issue"
       PR review por 1 arquivo de outro dev
       
Dia 3: Implementa 1 regra de validação nova (Y)
       Escreve testes (positivos + negativos)
       PR + review
       
Dia 4: Cross-doc engine
       Adiciona 1 regra cross-doc
       Validação roda em staging
       
Dia 5: Demo de 30min do que aprendeu
       Retrospectiva de onboarding
       Assign sprint backlog
```

---

## §8 — Differentials competitivos (ser o melhor do mercado)

### 8.1 Os 10 moats

#### Moat 1 — **Cross-Doc Engine (L3)**

**O quê:** Validação cruzada entre CADOCs. 3040 diz X mas 4111 diz Y? Radiant Norma pega.

**Por que moat:**
- BCValidador oficial: zero. Só valida 1 CADOC por vez.
- Matera, Mitra, cadoc.ai: não têm.
- Construir isso exige entender regras inter-CADOC e modelar dependências (research de meses).

**Métrica:** "12 regras cross-doc em produção, 0 concorrentes têm."

#### Moat 2 — **Schema Registry versionado por data-base**

**O quê:** Cada release BACEN = nova versão no registry. IFs não mexe em código.

**Por que moat:**
- Concorrentes: hardcoded XSD em releases.
- Radiant: schema + crítica + leiaute tudo versionado, com changelog público.
- Custo de migração pra Radiant: zero quando BACEN muda leiaute.

**Métrica:** "Detecção de mudança ≤ 24h, deploy em ≤ 5 dias úteis."

#### Moat 3 — **Audit hash chain (LGPD/SOC 2 ready)**

**O quê:** Cada entrada do audit log tem SHA-256(prev || payload || meta). Imutável por trigger DB.

**Por que moat:**
- Auditoria externa (PwC, Deloitte) reconhece isso como "tamper-evident log".
- SOC 2 Type II requer.
- Concorrentes: têm audit log, mas não-hash-chained (editável).

**Métrica:** "Verificável em O(n), 100% tamper-evident, trigger DB impede UPDATE/DELETE."

#### Moat 4 — **DRSAC ESG first-mover**

**O quê:** Janela IN BCB 694/2025, vigência dez/2026. Nenhum SaaS cobre hoje.

**Por que moat:**
- 12 meses de janela antes de concorrentes acordarem.
- Material BACEN protegido (credencial necessária) → entrada com atrito.
- 1 cliente-âncora = 10 follow-ons (referência).

**Métrica:** "First-mover em janela regulatória de R$ X bi."

#### Moat 5 — **Self-service onboarding 15min**

**O quê:** CNPJ → Sisbacen creds → smoke BACEN real → dashboard. Tudo wizard.

**Por que moat:**
- Matera: 12 semanas de deployment.
- cadoc.ai: 2-4 semanas.
- Radiant: 15 minutos.
- Network effects: cada onboarded = case study = mais onboardings.

**Métrica:** "Time-to-value: 15min (mediano)."

#### Moat 6 — **Open data + open schemas**

**O quê:** Repositório público `radiant-norma-schemas` no GitHub com XSDs + críticas extraídas. CC-BY.

**Por que moat:**
- Community contributions (regras, schemas).
- Marketing orgânico (devs BACEN).
- "Open core" posiciona como player sério vs Matera (closed).
- Trust signal: código aberto = auditável.

**Métrica:** "50+ stars, 10+ contribuidores externos, 1k downloads/mês."

#### Moat 7 — **Modern stack (Next.js 15 + Go + Postgres)**

**O quê:** Stack moderna, type-safe, performática. Stack "chata" (proven).

**Por que moat:**
- Contrata dev mais fácil (vs stack legado Matera).
- Velocidade de iteração: RSC + Server Actions simplificam CRUD.
- Cost: menor que competitors em infra.

**Métrica:** "Bundle < 200KB, LCP < 2.5s, API P95 < 500ms."

#### Moat 8 — **Compliance officer UX**

**O quê:** UI feita pra compliance, não dev. Linguagem de risco, não de tech. Erros didáticos.

**Por que moat:**
- Matera: feito pra dev (precisa de IT).
- cadoc.ai: genérico.
- Radiant: "compliance officer primeiro."

**Métrica:** "NPS ≥ 50, time-to-resolution < 5min."

#### Moat 9 — **Insights preditivos (AI)**

**O quê:** LLM (opt-in) interpreta audit_log + sugere melhorias em texto natural. "Sua regra F02 falhou 12 vezes esse mês, principal causa é data-base inválida em sistema X."

**Por que moat:**
- Único no mercado.
- Alto valor pra IF (economiza horas de análise).
- Requer volume de audit_log pra funcionar (data network effect).

**Métrica:** "5+ recomendações/tenant/mês, 80% acknowledged."

#### Moat 10 — **Multi-CADOC ecosystem**

**O quê:** 1 contrato, 10 CADOCs. Não é "SaaS de 3040", é "SaaS de compliance regulatório".

**Por que moat:**
- Concorrentes: 1-3 CADOCs.
- Switching cost alto: trocar 10 CADOCs = trocar stack inteira.
- Cross-doc só faz sentido com ecossistema completo.

**Métrica:** "10 CADOCs production-grade, 12 regras cross-doc."

### 8.2 Mensagem de positioning

> **Pra SCD pequena:** "Radiant Norma — faça seu primeiro CADOC em 15 minutos, sem precisar de IT."
>
> **Pra IP média:** "Radiant Norma — 10 CADOCs, 1 plataforma, audit trail pronto SOC 2."
>
> **Pra Banco S3-S4:** "Radiant Norma — único SaaS com cross-doc L3 + ESG first-monitor + audit chain verificável."
>
> **Pra Fintech BaaS:** "Radiant Norma — white-label + API-first + pricing per-environment. Ofereça compliance pros seus clientes."

### 8.3 Anti-features (o que NÃO fazer)

- ❌ **Não construir editor de XML WYSIWYG** (compliance officer sabe XML, dev prefere código).
- ❌ **Não construir dashboard executivo C-Level** (C-Level não usa Radiant direto, vê via reports).
- ❌ **Não suportar 50 CADOCs** (foco nos 10 críticos; long tail = parceria com consultoria).
- ❌ **Não fazer processamento batch assíncrono** (BACEN prefere submission instant; batch só pra casos específicos).
- ❌ **Não suportar mobile nativo** (web responsiva é suficiente; native = overhead).
- ❌ **Não virar "super-app"** (Risco/Compliance não quer 1 app pra tudo; quer foco regulatório).

---

## §9 — Riscos + Mitigações

### 9.1 Riscos técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| **BACEN muda API STA WS sem aviso** | Média | Alto | Retry wrapper + versionamento + monitoramento de schema |
| **Schema 2030 DRSAC não-publicável** | Alta | Médio | Solicitação formal + parceria consultoria (B3Bee) |
| **Postgres RLS bypass** | Baixa | Crítico | Auditoria externa + penetration test + runtime check (`SET LOCAL` em toda query) |
| **Secret Sisbacen vaza em log** | Baixa | Crítico | `loggerutil.SafeError` + regex scanner em CI + audit log emission |
| **SSE não escala** | Média | Médio | Redis pub/sub + multi-replica + backpressure |
| **Frontend bundle cresce** | Alta | Médio | Code splitting + lazy loading + bundle analyzer em CI |
| **Audit chain quebra (bug em entry hash)** | Baixa | Alto | Tests + Verify command + CI gate |
| **Dependency com CVE** | Média | Médio | `govulncheck` em CI + Dependabot + update mensal |

### 9.2 Riscos de negócio

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| **Materia lança DRSAC antes** | Média | Alto | Velocidade de execução + first-mover em janela |
| **Cliente-piloto cancela** | Alta | Médio | 2-3 clientes em paralelo (não bet tudo em 1) |
| **Pricing muito alto/baixo** | Média | Médio | A/B test + feedback loop trimestral |
| **Churn alto (>10%/mês)** | Média | Alto | NPS tracking + exit interview + product-market fit |
| **Investidor / aquisição hostil** | Baixa | Alto | Term sheet review + founder control |

### 9.3 Riscos regulatórios

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| **BACEN exige certificação específica** | Média | Alto | SOC 2 Type II + ISO 27001 roadmap |
| **LGPD enforcement muda** | Média | Médio | DPO + DPIA + privacy-by-design |
| **Lei 14.155 (fraude eletrônica)** | Baixa | Médio | Compliance review anual |

### 9.4 Kill criteria (pivotar se...)

- **Após 6 meses de piloto:** 0 cliente pagou → GTM errado.
- **Após Q3 2026:** MRR < R$ 5k → pivotar pra vertical única (SCD ESG-only).
- **Após Q1 2027:** sem DRSAC vendido → DRSAC não é viável comercialmente.
- **Após Q2 2027:** MRR < R$ 100k → bootstrap mode, sem Series A.

---

## §10 — Anexo: Decisões arquiteturais consolidadas (ADRs)

### ADR-001: Stack — Go + Postgres + Redis + Next.js

**Status:** Aceito
**Data:** 2026-07-05
**Decisor:** Henrique + Mavis

**Contexto:** Escolha de stack pra SaaS regulatório multi-tenant.

**Decisão:** Go 1.25+ no backend, Postgres 16 como DB transacional, Redis 7 pra cache/sessão/queue, Next.js 15 (App Router + RSC) no frontend.

**Consequências:**
- ✅ Padronização Fortvna (já opera em Go).
- ✅ Postgres RLS nativo (multi-tenant sem código extra).
- ✅ Hiring mais fácil (stack mainstream).
- ❌ Sem type-safety end-to-end (Go ↔ TS) — mitigável com OpenAPI gen.

**Alternativas consideradas:**
- Rust: latência melhor, mas hiring difícil.
- Node.js: mesma linguagem front/back, mas ecosystem financeiro BR fraco.
- Elixir/Phoenix: live-view interessante, mas niche.

### ADR-002: Multi-tenancy — Postgres RLS, não schema-per-tenant

**Status:** Aceito
**Data:** 2026-07-05

**Contexto:** Modelo de isolamento de dados entre IFs.

**Decisão:** Single schema, RLS policies em TODA tabela tenant-scoped. App seta `app.current_if_id` via `SET LOCAL` em cada transação.

**Consequências:**
- ✅ Migrations simples (single schema).
- ✅ Defense-in-depth (mesmo bug no app = RLS protege).
- ✅ Connection pool compartilhado.
- ❌ Toda query precisa estar em transação com SET LOCAL.
- ❌ Cross-tenant queries (admin) precisam de bypass explícito.

**Alternativas consideradas:**
- Schema-per-tenant: migrations nightmare em 100+ tenants.
- App-level `WHERE if_id =`: vetor de bug.

### ADR-003: Audit log — hash chain + imutável via trigger

**Status:** Aceito
**Data:** 2026-07-05

**Contexto:** LGPD + SOC 2 requerem log tamper-evident.

**Decisão:** Cada entry = SHA-256(prev || payload || metadata || timestamp). Trigger Postgres bloqueia UPDATE/DELETE.

**Consequências:**
- ✅ Tamper-evident verificável.
- ✅ Imutável via DB (não só convenção).
- ❌ Storage cresce rápido (mitigável: retention policy + archive S3).
- ❌ Performance: precisa index por created_at + if_id.

**Alternativas:**
- Blockchain: overkill.
- WORM storage: complexo.

### ADR-004: Schema Registry — versionado por data-base, GitHub source-of-truth

**Status:** Aceito
**Data:** 2026-07-05

**Contexto:** BACEN muda leiaute 3-5x/ano. IF precisa se adaptar rápido.

**Decisão:** Cada release BACEN = PR no repo público `radiant-norma-schemas`. CI valida XSD, merge → publicação automática via webhook. App consulta `plataforma.schemas` (cache 5min).

**Consequências:**
- ✅ Schema-first (sem deploy de código).
- ✅ Histórico auditável.
- ✅ Community contributions.
- ❌ Dependência de GitHub (mitigável: GitLab self-hosted).

### ADR-005: STA Client — interface segregation (Client / ReadClient / ChunkedClient)

**Status:** Aceito (existente desde Sprint 25)
**Data:** 2026-06-15

**Contexto:** STA WS tem 3 subsets de operações (write, read, chunked). StubClient não implementa read/chunked.

**Decisão:** Interface segregation. `Client` (write), `ReadClient` (read), `ChunkedClient` (chunked). StubClient implementa só `Client`. WSClient implementa todas.

**Consequências:**
- ✅ Compile-time assert garante implementação.
- ✅ Handler type-asserts pra capability check.
- ✅ Hollow stub evitado (forçar Stub a implementar com zero-values seria mentir).

---

## §11 — Próximos passos imediatos

### Hoje (essa semana)
- [ ] Revisar este MASTER_PLAN.md com time
- [ ] Aprovar manifesto + roadmap macro
- [ ] Criar 1ª sprint do novo ciclo (Sprint 28 — Vault Integration)

### Sprint 28 (próxima)
- [ ] Escrever SPRINT_28_RESEARCH.md
- [ ] Implementar AWS Secrets Manager integration
- [ ] Testes + smoke
- [ ] VALIDAÇÃO_49
- [ ] CHANGELOG v3.22.0
- [ ] Tag + push

### Quarterly review (set/2026)
- [ ] Métricas vs plano (MRR, NPS, churn)
- [ ] Re-ajuste roadmap
- [ ] Decisão: Series A ou bootstrap?

---

**FIM DO PLANO MESTRE — EDIÇÃO "OURO"**

> Próximo passo natural: começar a executar Sprint 28. Me diga por onde quer começar — posso detalhar a SPRINT_28_RESEARCH.md imediatamente, ou ajustar este plano antes.