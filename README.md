<div align="center">

<img src="assets/logo.svg" alt="Radiant Norma — inteligência regulatória para IFs brasileiras" width="480"/>

# Radiant Norma

### *Inteligência regulatória para Instituições Financeiras brasileiras.*

**Gera, valida e envia CADOCs BACEN** — com orquestração ponta-a-ponta,
auditoria tamper-evident e camadas que o BCValidador não tem.

<br>

![Status](https://img.shields.io/badge/status-v1.3.0_✅-10b981?style=for-the-badge)
![Sprint](https://img.shields.io/badge/sprint-4%2F4-6366f1?style=for-the-badge)
![Stack](https://img.shields.io/badge/stack-Go_1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-proprietary-1e293b?style=for-the-badge)
![Coverage](https://img.shields.io/badge/cadocs-10_cobertos-8b5cf6?style=for-the-badge)

<br>

*"Não basta empacotar XML. A gente gerencia o pipeline regulatório inteiro."*

<br>

[**Quickstart ↓**](#-quickstart) · [**Arquitetura**](#-arquitetura) · [**Roadmap**](#-roadmap) · [**Sprints**](#-sprints)

</div>

---

## ✦ A tese

O BACEN publica **10 CADOCs** (SCR, DRSAC, DRM, DLO, DLI, DDR, DRL, DLP, DLO,
eventos) com **centenas de leiautes versionados** e **milhares de regras de
validação semântica**. O `BCValidador` oficial valida **um documento por vez,
local, em Java**.

Quem precisa reportar — SCDs, IPs, bancos médios — acaba escrevendo o pipeline
na mão: extrai dados, gera XML, valida, comprime, sobe no STA, reconcilia o
protocolo, refaz no mês seguinte quando o leiaute muda.

**Radiant Norma** faz esse pipeline inteiro como serviço: Schema Registry
versionado por data-base, Norma Audit em **4 camadas**, STA client real,
audit log com **hash chain** (LGPD + SOC 2), multi-tenant isolado por RLS.

> **Diferencial proprietário:** L3 (cross-documento) e L4 (histórico) — o
> BCValidador não vai além de L1+L2.

---

## ✦ Cobertura atual

| CADOC | Sigla | Periodicidade | Status |
|---|---|---|---|
| **3040** | SCR — Risco de Crédito | Mensal | ✅ 25 regras portadas em Go (B/F/C/S), 320 no DB |
| **3044** | Eventos de Crédito (JSON) | Por evento | ✅ Schema + 17 regras T01-T19 |
| **3050** | Estatísticas Agregadas | Mensal/diária | ✅ V11 — 170 regras |
| **2030** | DRSAC — Risco ESG | Semestral | ⚠️ Críticas não-públicas (gap conhecido) |
| **2060** | DRM — Risco de Mercado | Mensal | ✅ 22 regras extraídas do PDF |
| **2061** | DLO — Limites Operacionais | Mensal | ✅ 518 regras |
| **2062** | DLI — Limites Individuais | Mensal | ✅ Leiaute + instruções |
| **2070** *(cód 2011)* | DDR — Requerimento Capital | Diário | ✅ 11 regras |
| **2160** | DRL — Liquidez (LCR) | Diário | ✅ Modelos II BACEN |
| **2170** | DLP — Liquidez LP (NSFR) | Mensal | ✅ Modelo de cálculo oficial |

**1.099 regras de validação semântica** extraídas; **25 regras 3040** portadas em Go com parser XML tipado.

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
```

Stack completa em [`backend/README.md`](backend/README.md).

---

## ✦ Roadmap

| Sprint | Tema | Status |
|---|---|---|
| **1** | Base documental + catálogo JSON estruturado | ✅ v1.0.0 |
| **2** | Norma Audit spike (Go CLI + XSD gerado) | ✅ v1.1.0 |
| **3** | Backend Go + API REST + Audit hash chain + STA stub | ✅ v1.2.0 |
| **4** | Honesty Patch + 25 regras 3040 + Radar Regulatório | ✅ v1.3.0 |
| **5** | Norma Console (Next.js) + Auth JWT + Cross-doc L3 | ✅ v1.5.0 |
| **6** | DOS-via-API hardening + Cadoc list cache | ✅ v1.5.0 |
| **7** | JWT RS256 + 60 regras 3040 + Frontend Next.js | ✅ v2.0.0 |
| **8** | JWT bridge real + endpoints complementares + tenant isolation + CI/CD | ✅ v2.1.0 |
| **9–12** | Frontend redesign + Insights + Drill-down server actions | ✅ v3.5.0 |
| **13** | CSRF + Cross-Tenant + DB Integrity + Rate Limit | ✅ v3.5.2 |
| **14–16** | Drill-down insights + Design system + Redis rate limiter | ✅ v3.6.0 |
| **17** | Observability + Production Hardening + lint automation | ✅ v3.7.0 |
| **18** | STA WS nativo (substituir Playwright) + cert A1/A3 | ✅ v3.8.0 |
| **19** | STA WS read side (Download + StatusUpload + X-Content-Hash) | ✅ v3.9.0 |
| **20** | STA WS listagem / disponiveis + alteração / situacao + handlers REST | ✅ v3.10.0 |
| **21** | STA WS chunked transfer (range upload §5.6 + range download §6.4) | ✅ v3.11.0 |

Última release estável: **v3.11.0** (Sprint 21 STA WS chunked transfer — `SubmitRange` §5.6 +
`DownloadRange` §6.4 com `ChunkedClient` interface segregation; 18/18 packages PASS, 63 testes
STA, smoke 11/11).

**Sub-produtos anunciados** (marca Radiant):

- 🟢 **Norma ESG** — first-mover DRSAC 2030 (janela IN BCB 694/2025, ninguém cobre)
- 🟡 **Norma Radar** — worker que detecta mudanças de leiaute em tempo real ✅ Sprint 4
- 🔵 **Norma Connect** — STA client Web/WS com retry + protocolo tracking
- 🟣 **Norma Audit** — o que está nesse repo (o produto raiz)

---

## ✦ Métricas atuais

```
Backend Go          14 arquivos · ~2.908 linhas
Catálogo crítico    1.099 regras · 6 CADOCs com críticas executáveis
Catálogo leiaute    4.244 linhas · 8 CADOCs
Regras portadas Go  25 (Básicas B06-B15 + Formato F01-F05 + Campos C01-C05 + Semantica S01-S05)
Audit log entries   N (chain validado · tamper-evident)
Endpoints REST      13 funcionais · curl-testados (validate, rules, schemas, sta, radar)
Material BACEN      137 arquivos · 50 MB capturados
Concorrentes mapeados 12 (Mitra/Matera/cadoc.ai/LUZ/Dattos/BIBlue/…)
PDFs profissionais  3 (README · ENG_REVERSA · PRODUTO_TESE_ROADMAP)
Workers CLI         3 (api · worker · radar)
```

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
├── ENG_REVERSA.md               ★ análise profunda de Mitra/Matera/cadoc.ai
├── PRODUTO_TESE_ROADMAP.md      ★ tese, personas, GTM, planos R$1,5k-12k
├── CHANGELOG.md                 histórico de sprints (v1.0 → v1.3)
├── SPRINT_2.md  SPRINT_3.md  SPRINT_4.md  retrospectivas
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

- **2030 DRSAC críticas** — não publicamente disponível; FAQ do BACEN aponta
  pra página protegida por login. 5 URLs tentadas, todas retornaram erro.
  Aguardando solicitação formal.
- **Cobertura regras 3040** — 25/320 regras portadas (7.8%). Sprint 5 cobre
  Agregadas + Individualizadas pra chegar a 50%+.
- **BCValidador oficial vs Go** — implementação Go reescreve as regras públicas
  das planilhas `SCR3040_Criticas.xls`. Comparação byte-a-byte fica pra Sprint 5
  via Docker (BCValidador é Java-only).
- **STA real** — stub gera protocolo fake. Cliente real Web/WS (Playwright +
  Sisbacen + PSTA300) entra na Sprint 5.
- **Auth** — `X-IF-ID` simples na Sprint 4. JWT + OAuth2 + refresh tokens na 5.
- **Postgres RLS** — multi-tenant só identifica hoje. Isolamento por linha via
  RLS policies na Sprint 5.
- **Frontend Norma Console** — backend-only até Sprint 4. Next.js dashboard na 5.
- **Radar URLs** — algumas URLs BACEN retornam 404 (BACEN muda paths). Sistema
  é resiliente: baseline gravado mesmo quando fetch falha.
- **Dedup 3040** — 14 warnings de UNIQUE constraint no seed (duplicatas reais
  no JSON de origem). Pendente: dedup no `extract.py`.

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