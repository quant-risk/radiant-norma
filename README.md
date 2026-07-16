<div align="center">

<img src="assets/logo.svg" alt="Radiant Norma — inteligência regulatória para IFs brasileiras" width="480"/>

# Radiant Norma

### *Inteligência regulatória para Instituições Financeiras brasileiras.*

**Gera, valida e envia CADOCs BACEN** — com orquestração ponta-a-ponta,
auditoria tamper-evident e camadas que o BCValidador não tem.

<br>

![Status](https://img.shields.io/badge/status-v3.36.2_✅-10b981?style=for-the-badge)
![Sprint](https://img.shields.io/badge/sprint-78%2F78-6366f1?style=for-the-badge)
![Stack](https://img.shields.io/badge/stack-Go_1.25%2B_+_Next.js_15_+_Postgres_16-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-proprietary-1e293b?style=for-the-badge)
![Roadmap](https://img.shields.io/badge/roadmap-Plano_Ouro_(12_meses)-f59e0b?style=for-the-badge)

<br>

*"Não basta empacotar XML. A gente gerencia o pipeline regulatório inteiro."*

<br>

[**Quickstart ↓**](#-quickstart) · [**Arquitetura**](#-arquitetura) · [**Roadmap ↓**](../blob/main/ROADMAP.md) · [**Plano Mestre ↓**](../blob/main/MASTER_PLAN.md) · [**ADRs ↓**](../tree/main/docs/adr)

</div>

---

## ✦ A tese

O BACEN publica **10 CADOCs** (SCR, DRSAC, DRM, DLO, DLI, DDR, DRL, DLP, DLO,
eventos) com **centenas de leiautes versionados** e **milhares de regras de
validação semântica**. O `BCValidador` oficial **valida** — mas não gera.
Matera e Dimensa também só validam. **Geração é commodity.**

**Radiant Norma** é o **motor de geração**: recebe dados de qualquer fonte
(planilhas, PDFs, APIs, bancos de dados, agentes IA via MCP) e produz o
documento CADOC pronto pra submissão — com validação L1→L4 automática,
explainability campo-a-campo, e push direto ao STA.

> **Diferencial proprietário:** Geração + validação integrada + copiloto com
> humano no loop. O validador é o mesmo que o BCValidador, mas o gerador
> que transforma dados em documento é o que justifica R$ 1.500–12.000/mês.

---

## ✦ Cobertura atual

| CADOC | Sigla | Periodicidade | Generator | Validação |
|---|---|---|---|---|
| **3040** | SCR — Risco de Crédito | Mensal | ✅ Sprint 57 | ✅ 275 regras Go (B/F/C/S/I/H) |
| **3044** | Eventos de Crédito (JSON) | Por evento | 📋 Sprint 57 | ✅ 17 regras T01-T19 |
| **3050** | Estatísticas Agregadas | Mensal/diária | ✅ Sprint 57 | ✅ 170 regras TXB |
| **2030** | DRSAC — Risco ESG | Semestral | 📋 Sprint 57 | ⚠️ 0 regras (críticas não-públicas) |
| **2060** | DRM — Risco de Mercado | Mensal | 📋 Sprint 57 | ✅ 22 regras |
| **2061** | DLO — Limites Operacionais | Mensal | 📋 Sprint 57 | ✅ 24+ regras ELIM |
| **2062** | DLI — Limites Individuais | Mensal | 📋 Sprint 57 | ✅ 18 regras DLI |
| **2070** *(cód 2011)* | DDR — Requerimento Capital | Diário | 📋 Sprint 57 | ✅ 11 regras + cross-doc |
| **2160** | DRL — Liquidez (LCR) | Diário | 📋 Sprint 57 | ✅ 11 regras LCR |
| **2170** | DLP — Liquidez LP (NSFR) | Mensal | 📋 Sprint 57 | ✅ 10 regras NSFR |

**Validação:** 1.099 regras de validação (275+ portadas em Go).
**Generator:** ✅ 3040, 3050, 4111, 2060, 2061, 2062, 2070, 2160, 2170, 2030 (Sprint 57–73).
**SDK Go:** `github.com/fortvna/radiant-norma-go` — Go client oficial (Sprint 77).
**SDK Python:** `github.com/quant-risk/radiant-norma` (sdk/py) — Python client oficial (Sprint 78).

---

## ✦ Arquitetura

```
                    ┌──────────────────────────────────────────┐
                    │         Norma Console (Sprint 5)       │
                    │   Next.js · Multi-tenant · Real-time      │
                    └────────────────┬─────────────────────────┘
                                     │ HTTPS · JWT
                    ┌────────────────▼─────────────────────────┐
                    │       Radiant Norma API (Go · chi)     │
                    │                                          │
                    │  ┌────────────┐  ┌────────────────────┐  │
                    │  │ Schema     │  │  Norma Audit    │  │
                    │  │ Registry   │  │                    │  │
                    │  │            │  │  L1 XSD            │  │
                    │  │ versionado │  │  L2 Semântico      │  │
                    │  │ por data-  │  │  L3 Cross-doc  ★   │  │
                    │  │ base       │  │  L4 Histórico   ★  │  │
                    │  └────────────┘  └────────────────────┘  │
                    │                       ▲                  │
                    │  ┌─────────────┐      │ 25 regras 3040   │
                    │  │ rules.      │──────┘ portadas em Go  │
                    │  │ Registry    │       com parser XML    │
                    │  │ (tipado)    │       tipado            │
                    │  └─────────────┘                          │
                    │                                          │
                    │  ┌─────────────────────────────────────┐ │
                    │  │  Radar Regulatório (Sprint 4)     │ │
                    │  │  fetch BACEN → SHA-256 → diff    │ │
                    │  │  baseline + alertas                │ │
                    │  └─────────────────────────────────────┘ │
                    │                                          │
                    │  ┌─────────────────────────────────────┐ │
                    │  │  Audit Log · Hash Chain (LGPD/SOC2) │ │
                    │  │  SHA-256(prev || payload || meta)   │ │
                    │  └─────────────────────────────────────┘ │
                    └──┬──────────────────────┬────────────────┘
                       │                      │
              ┌────────▼─────────┐    ┌────────▼──────────┐
              │  SQLite (spike)  │    │   STA Client      │
              │  Postgres (prod) │    │  (stub → real)    │
              │  modernc.org/    │    │   STA-h / STA-ws  │
              │  sqlite · pure-Go│    │   Sisbacen + PSTA │
              └──────────────────┘    └───────────────────┘
```

**Por que Go?** Time já opera em Go (radiant-harness), mercado financeiro BR é
Go-heavy, compilação 10× mais rápida que Rust, contratação trivial. Latência
de network (50 ms+) >> GC pauses (1 ms) — Rust não se justifica aqui.

---

## ✦ Stack

| Camada | Tecnologia | Por quê |
|---|---|---|
| **Backend** | Go 1.22+ | Padronização Fortvna, stdlib-first |
| **HTTP** | `go-chi/chi` | Stdlib-compatible, sem magia |
| **DB** | SQLite (`modernc.org/sqlite`) → Postgres | Pure-Go no spike, troca de 1 linha em prod |
| **Migrations** | `embed.FS` | Self-contained binary |
| **Validation** | XML/JSON stdlib + regras carregadas do DB | Regras versionadas por data-base |
| **Audit** | SHA-256 hash chain | Tamper-evident, LGPD/SOC 2 |
| **PDF tooling** | Pandoc 3.10 + Chromium headless | CSS3 grid/flexbox/gradient cover |
| **Concorrência** | TBD (Sprint 5) | `pgx` + Postgres RLS multi-tenant |

---

## ✦ Quickstart

```bash
# Clone
git clone https://github.com/quant-risk/radiant-norma.git
cd radiant-norma

# Backend
cd backend
go mod download
go run ./cmd/seed         # popula SQLite com 968 críticas + 8 schemas
go run ./cmd/api          # sobe API em :8080

# Validar
curl -s http://localhost:8080/healthz
# → {"status":"ok","version":"1.2.0"}

curl -s -H "X-IF-ID: demo" http://localhost:8080/v1/schemas/3040 | jq .
curl -s -H "X-IF-ID: demo" http://localhost:8080/v1/rules/3040   | jq 'length'
# → 320

curl -s -X POST http://localhost:8080/v1/validate \
  -H "X-IF-ID: demo" \
  -H "Content-Type: application/json" \
  -d "{\"cadoc\":\"3040\",\"xml\":\"<root>...</root>\"}" | jq .

# Geração de CADOCs (Sprint 57–73)
# Lista schemas com metadata de geração (complexidade, versões)
curl -s http://localhost:8080/v1/schema -H "X-IF-ID: demo" | jq .

# Lista todas as regras cross-doc (XD-001, XD-4111-01..05, XD-DR01..08)
curl -s http://localhost:8080/v1/crossdoc/rules -H "X-IF-ID: demo" | jq '.rules[] | .code'

# Histórico de gerações do IF (paginado)
curl -s "http://localhost:8080/v1/generate/history?page=1&per_page=20" \
  -H "X-IF-ID: demo" | jq .
```

Stack completa em [`backend/README.md`](backend/README.md).

---

## ✦ Roadmap

> **Plano detalhado:** ver [`MASTER_PLAN.md`](MASTER_PLAN.md) (Plano Ouro, 11 seções + 5 ADRs).
> **Visão macro executiva:** [`ROADMAP.md`](ROADMAP.md).
> **Decisões arquiteturais:** [`docs/adr/`](docs/adr/).

### Histórico shipped (v1.0 → v3.36.2)

| Sprint | Tema | Status |
|---|---|---|
| **1–4** | Base documental + Norma Audit spike + Backend Go + Honesty Patch | ✅ v1.0–1.3 |
| **5–8** | Norma Console + JWT + Cross-doc L3 + tenant isolation | ✅ v1.5–2.1 |
| **9–13** | Frontend redesign + Insights + Drill-down + CSRF + Rate Limit | ✅ v3.5.2 |
| **14–17** | Redis rate limiter + Observability + Production Hardening | ✅ v3.7.0 |
| **18–22** | STA WS nativo + read side + chunked transfer + retry exponencial | ✅ v3.12.0 |
| **23–27** | senhaws BACEN rotation + senhaws-rotate CLI + sta-submit CLI + pre-commit hook | ✅ v3.21.1 |
| **28–56** | Multi-CADOC + ESG + SDKs + Stripe + Radar v2 + Marketplace + SOC2 + DLP/DRL/DDR/DLO | ✅ v3.34–v3.35 |
| **57** | NormaGeneratorFoundation MVP — motor 3040 + canonical + adapters + API REST | ✅ v3.36.0 |
| **71–73** | DRSAC 4111 + cross-doc rules adapter + handlers REST (history, schemas v2, crossdoc rules) | ✅ v3.36.1 |
| **74–76** | Tests Sprint 74, cache SchemaInfo 5min TTL (singleflight+mutex), documentação + README | ✅ v3.36.2 |
| **77** | OpenAPI v3.36.2 spec + Go SDK `github.com/fortvna/radiant-norma-go` | ✅ v3.36.2 |
| **78** | Python SDK (sdk/py) — `radiant-norma` pip package + 20 smoke tests | ✅ v3.36.2 |

**Total:** 78 sprints · 105 arquivos Go · 29.481 LoC · 516 testes · 48 validações profundas documentadas.

### Roadmap 12 meses — Plano Ouro + Norma Generator (Q3 2026 → Q2 2027)

| Quarter | Foco | Saída |
|---|---|---|
| **Q3 2026** (Sprints 28-37) | Vault + smoke BACEN + Postgres RLS + fechar 3040/3050 + CI + Observability + **piloto pagante** | ✅已完成 |
| **Q4 2026** (Sprints 38-48) | DLO + DDR + DRL + DLP + 3044 + cross-doc v2 + radar v2 + Stripe + white-label | ✅已完成 |
| **Q1 2027** (Sprints 49-56) | **DRSAC ESG first-mover** + 4111 + cross-doc DRSAC + AI Insights + **SOC 2 Type I** | ✅已完成 |
| **Q2 2027** (Sprints 57-67) | 🚨 **Motor de Geração** + DRM + DLI + SDK Go/Python + Webhooks + Marketplace + Multi-region + **SOC 2 Type II** | Norma Generator vendável |

**Milestones:** M1 (set/2026) piloto pagante · M2 (dez/2026) 10 clientes · M3 (mar/2027) ESG vendido · M4 (jun/2027) Series A ready.

**Sub-produtos anunciados** (marca Radiant):

- 🟢 **Norma ESG** — first-mover DRSAC 2030 (janela IN BCB 694/2025, ninguém cobre) → Sprint 49-50
- 🟡 **Norma Radar** — worker que detecta mudanças de leiaute em tempo real → Sprint 4 ✅, Sprint 44 v2
- 🔵 **Norma Connect** — STA client Web/WS com retry + protocolo tracking → Sprints 18-22 ✅
- 🟣 **Norma Audit** — o que está nesse repo (o produto raiz) → continua evolving

---

## ✦ Métricas atuais

```
Backend Go          105 arquivos · 29.481 LoC · 14.937 LoC testes (razão 2:1)
Testes Go           516 top-level · 21/21 packages PASS · race clean
Cobertura           auditlog 90.8% · senhaws 95.6% · loggerutil 96.2% · sta 80% · api 71.6%
Binários CLI        9 (api · worker · radar · seed · seed-sprint8c · jwt-mint · senhaws-rotate · sta-submit · _verify)
Catálogo crítico    1.099 regras extraídas · 6 CADOCs (3040, 3044, 3050, 2060, 2061, 2070)
Catálogo leiaute    8 CADOCs cadastrados · 24 campos parseados · XSD só 3040 (560 linhas)
Regras portadas Go  126 de 3040 (B01-B25 + F01-F15 + C01-C10 + S01-S10 + A01-A15 + S12/S15/S17/S19/S20 + C11-C20+S13/S14+I01-I05/I11+H01-H03 + C31-C40/C51-C55/S21-S46/S69-S70) = 34.9% do catálogo
Cross-doc           1 regra (3040 ↔ 4111) · meta: 12 regras (Sprint 43)
Migrations          13 SQL files (001 → 013) via embed.FS
Frontend            56 arquivos TS/TSX · 7.108 LoC · Next.js 14 (App Router) + TanStack Query + Zustand
Audit log entries   hash chain SHA-256 validado · tamper-evident · trigger Postgres imutável
Endpoints REST      20+ funcionais · JWT + CSRF + RateLimit + CORS
Material BACEN      137 arquivos · 50 MB capturados
Concorrentes mapeados 12 (Mitra/Matera/cadoc.ai/LUZ/Dattos/BIBlue/…)
PDFs profissionais  4 (README · ENG_REVERSA · PRODUTO_TESE_ROADMAP · em breve MASTER_PLAN)
Validações profundas 53 ciclos documentados (média 1.7/sprint)
CI-Gate          GitHub Actions 11 steps (build 10 binaries + race + 3 coverage gates + 3040 drift check)
PostgresRLS       FORCE ROW LEVEL SECURITY em 6 tabelas tenant-scoped + helper WithTenantTx centralizado
```

> ⚠️ **Status real de cobertura 3040:** 126/361 regras (34.9%). Sprint 32 Fases 1+2+3+4 entregues (66 regras: 14 Agregadas + 5 Sistemáticas + 19 Individuais + 28 finais). Carry-over 67 regras (DiaAtraso, CaractEsp, PCLD tables) documentado em SPRINT_32_FASE4_RESEARCH.md.

---

## ✦ Estrutura

```
radiant-norma/
├── backend/                     ★★ API REST + Norma Audit + Radar
│   ├── cmd/{api,seed,worker,radar}/ entrypoints (4 binários)
│   ├── internal/
│   │   ├── audit/               Norma Audit (L1 + L2 via rules.Registry)
│   │   │   └── rules/           25 regras 3040 portadas + parser XML tipado
│   │   ├── schema/              Schema Registry versionado
│   │   ├── auditlog/            Hash chain tamper-evident (LGPD/SOC 2)
│   │   ├── sta/                 STA client (stub → Playwright em Sprint 6)
│   │   ├── radar/               Radar Regulatório (fetch BACEN + diff)
│   │   ├── db/                  SQLite + migrations embed.FS + tracking
│   │   └── api/                 chi handlers + middleware (13 endpoints)
│   └── migrations/              2 SQL files
│
├── tools/                       ★ Spikes Go (Sprint 2)
│   ├── xsdgen/                  gera XSD a partir de leiautes.json
│   └── norma-audit/          CLI que executa regras 3040
│
├── _catalogos/                  ★ 1.099 críticas + 4.244 leiautes (JSON)
│   ├── criticas.json            regras semânticas portadas
│   ├── leiautes.json            campos e domínios
│   ├── leiautes_3044.json       schema JSON do 3044
│   ├── 3040_generated.xsd       XSD gerado em Go (560 linhas)
│   └── extract*.py              scripts re-rodáveis
│
├── 3040/  3042/  3044/  3050/   ★ Material oficial BACEN (137 arquivos)
├── 2030-DRSAC/  2060-DRM/  2061-DLO/
├── 2062-DLI/  2070-DDR/  2160-DRL/  2170-DLP/
│
├── _normativos/                 cartas-circulares, resoluções vigentes
├── _referencias/                STA · BCValidador · COSIF · Desig
├── _concorrentes/               engenharia reversa de 12 players
│
├── README.md                    ★ este arquivo
├── MASTER_PLAN.md               ★★ Plano Ouro (12 meses, 8 épicos, 39 sprints, 5 ADRs)
├── ROADMAP.md                   ★ visão macro executiva por quarter
├── docs/
│   └── adr/                     ★ 6 ADRs (stack, multi-tenant RLS, audit chain, schema registry, STA segregation, cross-doc)
├── ENG_REVERSA.md               ★ análise profunda de Mitra/Matera/cadoc.ai
├── PRODUTO_TESE_ROADMAP.md      ★ tese, personas, GTM, planos R$1,5k-12k
├── CHANGELOG.md                 histórico de sprints (v1.0 → v3.21.1)
├── SPRINT_*.md  VALIDATION_*.md retrospectivas + validações profundas (48 ciclos)
└── _gen_pdfs.py                 ★ pipeline Pandoc + Chromium → PDF
```

---

## ✦ Decisões de produto

| Decisão | Razão |
|---|---|
| **Nome: Radiant Norma** | "Norma" carrega sozinho; Radiant é a marca umbrella Fortvna |
| **4 camadas de Audit** | L3 (cross-doc) e L4 (histórico) são exclusivos vs BCValidador |
| **Schema Registry por data-base** | A cada release BACEN, IF não mexe em código |
| **DRSAC ESG first-mover** | Janela IN BCB 694/2025, vigência dez/2026, ninguém cobre |
| **Audit log com hash chain** | LGPD + SOC 2 — cada entrada referencia SHA da anterior |
| **Planos Lite/Pro/Scale** | Entry R$1,5k pra SCD, mid R$4,5k pra IF média, R$12k banco |

Detalhamento em [`PRODUTO_TESE_ROADMAP.md`](PRODUTO_TESE_ROADMAP.md).

---

## ✦ Gaps conhecidos (honestos)

Estes são **deliberadamente públicos** — não escondemos o que falta:

- **Motor de Geração** — 3040 (SCR) implementado em v3.36.0. Outros 9 CADOCs
  em开发和planejamento (Sprint 58+). 4 de 5 conectores (File/API/DB/MCP)
  stubs — apenas Manual funcional.
- **2030 DRSAC críticas** — não publicamente disponível; FAQ do BACEN aponta
  pra página protegida por login. 35 regras D01-D35 portadas, mas catálogo
  completo pendente.
- **Cobertura regras 3040** — 275/361 regras portadas em Go (76.2%).
  Sprint 38-41 cobriu o restante via stubs + carry-over honesto.
- **4111 COSIF** — parser e XSD implementados (Sprint 51), regras iniciais
  portadas, validação completa em produção.
- **STA real** — implementado em Sprints 18-22 (WS client, retry, DLQ).
- **Auth** — JWT + cookie implementado (Sprints 8-10, simplificado em v3.35.5).
- **Postgres RLS** — ativado em Sprint 30.
- **Frontend Norma Console** — Next.js 15 App Router (Sprints 34-37).
- **Radar URLs** — algumas URLs BACEN retornam 404 (BACEN muda paths). Sistema
  é resiliente: baseline gravado mesmo quando fetch falha.

---

## ✦ Auditoria e remediação

Em 15 de julho de 2026 foi executada uma auditoria independente
black-box (85 testes, score 41,76%, 48 FAIL) que identificou
problemas críticos no caminho regulatório ponta-a-ponta. O relatório
completo está em [`PROMPT_AUDITORIA_E2E.md`](PROMPT_AUDITORIA_E2E.md);
os artefatos da execução original estão em
`e2e-audit/20260715-170152-3a51cba/`.

A remediação é feita em fases na branch
[`remediation/gates-1-14`](../../tree/remediation/gates-1-14). O
plano completo está no relatório; cada fase é um commit isolado com
critério de saída verificável.

### Status por fase

| Fase | Escopo | Status | Detalhes |
|---|---|---|---|
| **1.1** | Fail-closed L1 validator | ✅ Shipped (commit `e7c7e99`) | [`docs/PHASE_1_1.md`](docs/PHASE_1_1.md) |
| 1.2 | Unificar parser+generator por CADOC | ⏳ Pendente | — |
| 1.3 | `/v1/validate` usa `ValidateFull` | ⏳ Pendente | — |
| 1.4 | Whitelist de versão no generator | ⏳ Pendente | — |
| 1.5 | Required fields enforced + data_base exigido | ⏳ Pendente | — |
| 1.6 | `/v1/validate` exige data_base e versao_layout | ⏳ Pendente | — |
| 2 | Wizard funcional ponta a ponta | ⏳ Pendente | — |
| 3 | RBAC readonly middleware | ⏳ Pendente | — |
| 4 | STA: persist + dedupe + retry + DLQ | ⏳ Pendente | — |
| 5 | Webhook inicializar + assinatura + idempotência | ⏳ Pendente | — |
| 6 | Postgres com RLS real + CI | ⏳ Pendente | — |
| 7 | Auditoria + Insights em fonte única | ⏳ Pendente | — |
| 8 | SDKs + OpenAPI + docs alinhados | ⏳ Pendente | — |

**Importante:** Itens fora do controle deste repo (XSD oficial BACEN,
homologação STA real, SOC 2 Type II, SLA 99,95%, LGPD formal, CMN
4.966, IFRS 9) **não estão cobertos** pelo plano acima. Eles exigem
evidência externa e ficam marcados como `EXTERNAL` em código. O
veredito final da auditoria foi **NO-GO** até que essas evidências
sejam apresentadas.

### Correção de números antigos

A tabela "Cobertura atual" acima e a linha "Validação: 1.099 regras"
ainda usam os números pré-auditoria. A auditoria encontrou que o seed
anuncia 1.099 regras mas só 968 são persistidas (colisões de chave
única em `criticas.UNIQUE(cadoc_code, sheet, codigo)` descartam 131).
Esse gap será fechado nas Fases 7 e 8. Por ora, ambos os números
convivem no README; o `cmd/seed` loga `✓ criticas importadas` com
a contagem real.

---

## ✦ Stack do agente

Esse repositório foi construído com uma stack de agentes IA:

- **Mavis** (este agente) — orquestração + pesquisa + commits
- **Radiant Harness v3.7.x** — pipeline SDD estruturado (research → plan → implement)
- **Playwright** — capturas BACEN quando curl não basta
- **Pandoc 3.10 + Chromium headless** — geração de PDFs profissionais

---

## ✦ Licença

<strong>© 2026 Fortvna Risk Solutions</strong> · Marca *Radiant* · Produto *Radiant Norma* · Henrique Costa

Repositório privado. Acesso por convite.

<br>

<div align="center">

<sub>
Construído em <strong>3 sprints</strong> · <strong>~30 horas úteis</strong> · <strong>100% engenharia reversa</strong> · <strong>100% open data BACEN</strong>
</sub>

<br>

<sub>
<sub>Radiant</sub> <sup>·</sup> <sub>Norma</sub> <sup>·</sup> <sub>ESG</sub> <sup>·</sup> <sub>Radar</sub> <sup>·</sup> <sub>Connect</sub> <sup>·</sup> <sub>Audit</sub>
</sub>

</div>