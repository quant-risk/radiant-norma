# Changelog — cadocs (Radiant Norma)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

## v1.3.0 — 2026-07-03 (Sprint 4: Honesty Patch + Audit utilizável + Radar)

### 🎯 Objetivo da sprint
Fechar **5 hollow stubs** da Sprint 3 detectados em validação end-to-end,
portar **25 regras semânticas do 3040** com parser XML tipado, e implementar
o **Radar Regulatório** (worker que detecta mudanças de leiaute — diferencial
first-mover da Radiant Norma).

### ✅ Entregas

#### 🔴 P0 — Honesty Patch (5/5)
Sprint 3 fechou o ciclo dados→serviço mas, ao validar end-to-end com curl
fresh sandbox, descobrimos 5 hollow stubs que a CHANGELOG não declarava.
Sprint 4 começa consertando a casa:

| Bug | Fix |
|---|---|
| `/v1/validate` exigia `cadoc_code` no JSON mas docs/clientes enviavam `cadoc` | `audit.ValidationRequest.UnmarshalJSON` custom — aceita ambos |
| Apenas 5/320 regras 3040 implementadas | Sprint 4 P1 (25 regras) |
| `cmd/worker/` diretório vazio (anunciado mas não codado) | `cmd/worker/main.go` (180 linhas) — processa envios pending do DB |
| `/v1/sta/submit` lia params da URL, inconsistente com resto da API | `sta.Submission` com JSON tags + handler unificado (body JSON preferencial, query retrocompat) |
| Tabela `envios` rejeitava INSERT silenciosamente (FK pra `ifs` vazia) | Seed popula 2 IFs demo + migration 002 adiciona `xml_content` e `zip_content` |

#### 🟡 P1 — Audit utilizável (B1: 25 regras 3040)
- **`internal/audit/rules/`** — package novo com interface Rule + Registry
- **`registry.go`** (130 linhas) — `Rule` interface + `Builtin3040()` retorna 25 regras
- **`3040.go`** (440 linhas) — parser XML tipado (`Doc3040`, `Agregado`, `Vencimentos`) + 25 regras:
  - **Básicas**: B06-B15 (contadores, limites, partes rejeitadas, max erros/avisos)
  - **Formato**: F01-F05 (taxa efetiva, datas, código contrato, conglomerado, RefBacen Sicor)
  - **Campos Obrigatórios**: C01-C05 (PJ, não obrigatórios, garantias, cessões)
  - **Semântica**: S01-S05 (detalhamento cliente, vendor, ocultação, crédito a liberar, limite)
- **Severity** da Rule tem prioridade sobre DB (regras implementadas são fonte da verdade)

**Cobertura de regras:** 5/320 (1.5%) → **25/320 (7.8%)**

#### 🟢 P2 — Radar Regulatório (B3: first-mover)
- **`internal/radar/radar.go`** (260 linhas) — Service com:
  - `ScanOnce()` — fetch URLs BACEN → SHA-256 → compara com baseline → insere alert se mudou
  - `DefaultSources` — 3 URLs candidatas (3040, 3050, DRSAC FAQ)
  - Resiliente: 404 não quebra, baseline gravado mesmo se fetch falhar
- **`cmd/radar/main.go`** (100 linhas) — worker CLI com `--once` mode e intervalo configurável
- **4 endpoints REST novos:**
  - `GET /v1/radar/alerts` (filtro `?unresolved=true`)
  - `GET /v1/radar/alerts/{id}`
  - `POST /v1/radar/alerts/{id}/resolve`
  - `POST /v1/radar/scan` (trigger scan manual)

#### 🔧 Migration tracking via `schema_migrations`
- Migration 001 era idempotente (`CREATE TABLE IF NOT EXISTS`)
- Migration 002 (`ALTER TABLE envios ADD COLUMN xml_content`) não é idempotente
- Solução: tabela `schema_migrations` rastreia quais migrations foram aplicadas
- Migrate pula automaticamente migrations já aplicadas

### 🏗️ Decisões técnicas Sprint 4

| Decisão | Razão |
|---|---|
| **`rules.Registry` interface-based** | Cada regra = struct com `Code()`, `Sheet()`, `Severity()`, `Apply()`. Adicionar regra = 1 struct + 1 linha no registry |
| **Parser XML tipado** | encoding/xml stdlib + struct tags. Zero dependência externa. Mais simples que XPath |
| **Severity da Rule > DB gravidade** | DB tem gravidade vazia pra regras novas. Implementação é fonte da verdade |
| **cmd/worker é stub** | Processa pending, atualiza status. Sem retry exponencial/DLQ (próxima sprint com asynq) |
| **cmd/radar fetch URLs candidatas** | Resiliente a 404. URLs BACEN mudam; sistema sobrevive a path errado |
| **`radar_alerts.alert_type='_baseline_*'`** | Reaproveita tabela. Em produção, tabela dedicada `radar_baselines` |
| **Migration tracking** | Resolve ALTER TABLE não-idempotente sem hack IF NOT EXISTS (que SQLite não suporta em ALTER) |

### 📊 Estatísticas Sprint 4

```
backend/ → 14 arquivos Go (2.908 linhas) — era 10/1.400 Sprint 3
   ├─ cmd/api/main.go              100 linhas (entrypoint)
   ├─ cmd/seed/main.go             280 linhas (+ seed IFs demo)
   ├─ cmd/worker/main.go           180 linhas (queue processor) — NOVO
   ├─ cmd/radar/main.go            100 linhas (radar worker) — NOVO
   ├─ internal/api/server.go       290 linhas (10 handlers)
   ├─ internal/audit/service.go    320 linhas (+ registry integration)
   ├─ internal/audit/rules/registry.go  140 linhas (Rule interface) — NOVO
   ├─ internal/audit/rules/3040.go     440 linhas (25 regras + parser) — NOVO
   ├─ internal/auditlog/log.go     140 linhas (hash chain)
   ├─ internal/db/db.go             40 linhas
   ├─ internal/db/migrate.go        90 linhas (+ schema_migrations) — modificado
   ├─ internal/radar/radar.go      260 linhas (fetch + diff + alerts) — NOVO
   ├─ internal/schema/registry.go  140 linhas
   └─ internal/sta/stub.go          95 linhas (+ JSON tags)

   migrations/  2 arquivos
   ├─ 001_initial.sql              128 linhas
   └─ 002_envios_xml.sql             5 linhas — NOVO
```

### 🧪 Testes E2E (curl sandbox limpo)

```
✓ POST /v1/validate XML válido (4832 B) → passed=true, 0 erros, 2-4ms
✓ POST /v1/validate DtBase="20-08"     → F02 detecta, severity=E
✓ POST /v1/validate Remessa=0          → B06 detecta, severity=E
✓ POST /v1/validate Mod=0213 v110=0    → S05 detecta, severity=E
✓ POST /v1/validate XML quebrado       → L1-PARSE + 13 regras falham (correto)
✓ POST /v1/sta/submit JSON             → protocolo + envio_id + persiste
✓ POST /v1/sta/submit ?cadoc=          → retrocompat OK
✓ Worker processa envio pending        → status=accepted, protocolo gerado
✓ POST /v1/radar/scan                  → scan em 1.7s, baseline gravado
✓ POST /v1/radar/alerts/{id}/resolve   → marca resolved, 404 se inexistente
```

### 🚧 Gaps remanescentes (Sprint 5)

| Gap | Por quê | Sprint 5 |
|---|---|---|
| **Driver Postgres real** | Sem Docker local | `pgx` + config flag |
| **JWT/OAuth em vez de X-IF-ID** | Mantido header simples | `internal/auth/` + refresh tokens |
| **Postgres RLS multi-tenant** | X-IF-ID só identifica | Policies por IF |
| **STA Web/WS real** | Stub OK pra demo | `internal/sta/web.go` |
| **Mais regras 3040** | Cobertura 7.8% | Sprint 5: Agregadas + Individualizadas |
| **Cross-doc engine (L3)** | Requer multi-CADOC | Carregar 3040 + 4111 + DRSAC em paralelo |
| **Frontend Norma Console** | Backend-only | Next.js dashboard |
| **Dedup 3040 (14 warnings)** | Duplicatas reais no JSON | Deduplicar no extract.py |
| **Radar: asynq queue** | Stub usa ticker simples | Substituir por fila real |
| **Testes unitários** | Sem coverage ainda | `*_test.go` + coverage report |

### 🎯 Próxima sprint (Sprint 5 — preview)
**Tema:** Norma Console (frontend) + mais regras + cross-doc L3
- Frontend Next.js com dashboard IFs
- Portar 30+ regras Agregadas + Individualizadas do 3040
- Cross-doc engine L3 (3040 ↔ 4111 ↔ DRSAC)
- Radar com asynq queue + retries

---

## v1.2.0 — 2026-07-03 (Sprint 3: Infraestrutura backend Go + API REST)

### 🎯 Objetivo da sprint
Construir a **infraestrutura backend do Radiant Norma** em Go: API REST, Schema Registry, Norma Audit como microservice, audit log com hash chain, STA stub. Postgres-ready (mas SQLite pra spike local).

### ✅ Entregas

#### Estrutura backend
- **`backend/`** — projeto Go completo com 9 arquivos:
  - `cmd/api/main.go` — entrypoint da API REST (chi router)
  - `cmd/seed/main.go` — popular banco com criticas.json + leiautes.json
  - `internal/db/db.go` — conexão SQLite (driver modernc.org/sqlite, pure-Go, sem CGo)
  - `internal/db/migrate.go` — migrations via embed.FS, idempotente
  - `internal/db/migrations/001_initial.sql` — schema completo (5 tabelas)
  - `internal/schema/registry.go` — Schema Registry service (GetEffective, List, Insert)
  - `internal/audit/service.go` — Norma Audit (carrega críticas do DB, valida XML/JSON)
  - `internal/auditlog/log.go` — Audit log com hash chain tamper-evident
  - `internal/sta/stub.go` — STA client stub (interface + mock que gera protocolo fake)
  - `internal/api/server.go` — handlers HTTP REST com auth middleware

#### Schema Postgres-ready (SQLite no spike)
5 tabelas criadas via migrations embed:
- `ifs` — multi-tenant (CNPJ, nome, tipo, segmento, plano, STA creds)
- `schema_versions` — versionamento por data-base (effective_from + UNIQUE)
- `criticas` — 968 importadas (de 6 CADOCs: 3040, 3050, 2061 DLO, 2070 DDR, 2060 DRM, 3044)
- `envios` — histórico de submissões STA (pending/validated/sent/accepted/rejected)
- `audit_log` — hash chain (cada entrada referencia SHA da anterior)
- `radar_alerts` — mudanças de leiaute detectadas (Sprint 4)

#### Endpoints REST (7 funcionais)
| Método | Path | Função |
|---|---|---|
| GET | `/healthz` | Liveness |
| GET | `/v1/schemas` | Lista CADOCs suportados |
| GET | `/v1/schemas/{cadoc}` | Schema effective de um CADOC |
| GET | `/v1/schemas/{cadoc}/versions` | Histórico de versões |
| GET | `/v1/rules/{cadoc}` | Críticas habilitadas (320 do 3040) |
| POST | `/v1/validate` | **Norma Audit** — valida XML/JSON contra regras |
| POST | `/v1/sta/submit` | STA stub (gera protocolo fake) |

#### Multi-tenant
- Middleware `X-IF-ID` obrigatório em `/v1/*`
- Sem header → 401 Unauthorized
- Cada IF isolada por header (sem row-level security ainda — Sprint 4)

#### Norma Audit end-to-end
- ✅ XML válido (3040 exemplo, 4832 B) → **passed=true, 0 erros, 0 warnings, 2ms**
- ✅ XML quebrado (22 B) → **L1-PARSE detectado**, B04 warning
- Hash SHA-256 do payload retornado em `xml_hash`
- Duração em `duration_ms`
- Audit log gravado com hash chain

#### Audit log tamper-evident
- Cada entrada: SHA-256(prev_hash + payload_hash + metadata + actor + action + target + timestamp)
- Genesis hash = `"0"*64`
- Função `Verify()` valida integridade da cadeia inteira
- 5 entradas geradas em testes, todas com chain válido

#### STA Stub
- `StubClient.Submit()` retorna protocolo fake `2026070329287c9b3b181`
- Sempre aceita (configurável `AlwaysAccept=false` pra rejeitar)
- Calcula hash SHA-256 do ZIP/XML

### 🏗️ Decisões técnicas Sprint 3

| Decisão | Razão |
|---|---|
| **SQLite com modernc.org/sqlite** | Sem CGo, sem Postgres/Docker no ambiente local, mas com abstração `database/sql` que permite trocar pra Postgres (`lib/pq` ou `pgx`) mudando 1 linha |
| **chi router** | Stdlib-compatible, leve, sem dependência mágica |
| **embed.FS para migrations** | Self-contained binary, sem ler filesystem em runtime |
| **audit_log hash chain** | LGPD + SOC 2: tamper-evident. Cada entrada referencia SHA da anterior |
| **X-IF-ID em vez de JWT** | Spike. Sprint 4 vai substituir por JWT/OAuth |
| **STA stub antes de Playwright** | Permite testes end-to-end sem dependência externa |
| **XML como string no JSON request** | `json.RawMessage` causava double-encoding. String normal resolve |

### 📊 Estatísticas Sprint 3

```
backend/ → 10 arquivos Go (1.400 linhas)
   ├─ cmd/api/main.go          ~80 linhas (entrypoint + graceful shutdown)
   ├─ cmd/seed/main.go         ~250 linhas (seed de JSON → DB)
   ├─ internal/api/server.go    ~250 linhas (7 handlers + middleware)
   ├─ internal/audit/service.go ~200 linhas (Norma Audit)
   ├─ internal/schema/registry.go ~140 linhas (Schema Registry)
   ├─ internal/auditlog/log.go ~120 linhas (hash chain)
   ├─ internal/sta/stub.go     ~80 linhas
   ├─ internal/db/db.go + migrate.go ~80 linhas
   └─ migrations/001_initial.sql ~110 linhas

Banco SQLite populado:
   schema_versions: 8
   criticas:        968
   audit_log:       5 (geradas em testes)
```

### 🚧 Gaps remanescentes (Sprint 4)

| Gap | Por quê | Sprint 4 |
|---|---|---|
| **Driver Postgres real** | Sem Docker/Postgres local | Adicionar `pgx` + config flag |
| **JWT/OAuth em vez de X-IF-ID** | Spike | Implementar `internal/auth/` |
| **row-level security multi-tenant** | X-IF-ID só identifica, não isola | Postgres RLS policies |
| **STA Web/WS client real** | Playwright precisa setup | Implementar `internal/sta/web.go` |
| **Mais regras no Norma Audit** | Só B01-B05 implementadas | Portar 50% das críticas 3040 |
| **Cross-doc engine (L3)** | Requer múltiplos CADOCs em memória | Carregar 3040 + 4111 em paralelo |
| **Worker assíncrono** | cmd/seed é one-shot | Bull/gocron pra STA queue |
| **Frontend (Norma Console)** | Backend-only | Next.js dashboard |

### 🎯 Próxima sprint (Sprint 4 — preview)
**Tema:** Autenticação, multi-tenant isolado, STA real (Playwright)
- JWT + OAuth2 com refresh tokens
- Postgres RLS policies
- STA Web client (Playwright)
- Portar 30+ regras semânticas do 3040
- Radar regulatório (worker de detecção de mudanças)

---

## v1.1.0 — 2026-07-03 (Sprint 2: Norma Audit Spike + capturas restantes)

### 🎯 Objetivo da sprint
**Fechar gaps de captura + primeiro spike técnico Go** do Norma Audit (gerar XSD + executar regras). Validação contra BCValidador substituída por teste negativo (XML quebrado) — BCValidador é Java proprietário e não roda nativamente em ARM64 macOS.

### ✅ Entregas

#### Capturas adicionais
- **C.1** — `3050/Criticas_TXB_V11.xlsx` (V11 mais recente que V9) — 170 regras
- **C.2** — `2060-DRM` críticas extraídas do PDF `Criticas_Pos_Processamento_2060_V2_Jan25.pdf` (22 regras, JSON estruturado em `_catalogos/drm_criticas_raw.json`)
- **C.3** — 2030 DRSAC críticas: **não publicamente disponível** — documentado como gap (5 URLs tentadas, todas retornaram página de erro BACEN; FAQ confirma que está em página protegida por login)
- **C.5** — `_catalogos/leiautes_3044.json` — schema JSON completo do 3044 (4 campos raiz + 5 objetos + 17 regras de validação T01-T19)
- **C.6** — `2170-DLP/DLP_2170_Modelo_de_calculo_v201907.xlsx` — modelo de cálculo oficial (substitui versão antiga)

#### Normalização C.4 — datas ISO 8601
- 518 críticas do 2061 DLO agora têm `data-base inicio` em formato ISO 8601 (`2015-01-01` em vez de `201501`)
- Suporte para: int (AAAAMM), float (Excel serial), string (AAAA-MM, AAAA-MM-DD, AAAAMM)
- Função `normalize_date()` em `_catalogos/extract.py`

#### Spike técnico Go (T.1 + T.2)
- **`tools/xsdgen/main.go`** — gerador de XSD a partir de `leiautes.json`
  - Roda: `go run ./tools/xsdgen -in _catalogos/leiautes.json -cadoc 3040 -out _catalogos/3040_generated.xsd`
  - Resultado: **`_catalogos/3040_generated.xsd`** — 560 linhas, validado com `xmllint --noout` ✅
  - 248 linhas de campos processadas (tags + atributos + obrigatoriedade)
  - Tipos BACEN (A8, N19,2) mapeados para XSD (xs:string, xs:decimal)
  - **Limitação conhecida**: XSD não oficial (BACEN só publica 3045 e 3026); alguns tipos ainda são genéricos
- **`tools/norma-audit/main.go`** — executor de regras 3040 contra XML
  - Roda: `go run ./tools/norma-audit -xml 3040/exemploDesempenhoOperacao.xml`
  - **Implementa 5 regras** (B01-B05) carregadas dinamicamente do catálogo JSON
    - B01: arquivo XML deve parsear
    - B04: encoding deve estar declarado
    - B05: arquivo não pode estar vazio/muito pequeno
    - B02/B03: skip (requerem STA stub e BACEN API)
  - **Teste positivo**: XML de exemplo válido → todas as 3 regras executadas passam ✅
  - **Teste negativo**: XML quebrado (22 bytes) → 3 erros detectados corretamente ✅
  - **Exit code**: 0 se todas regras OK, 1 se algum erro

### 🏗️ Decisões técnicas Sprint 2

| Decisão | Razão |
|---|---|
| **DRSAC críticas como gap conhecido** | Não está publicamente disponível; FAQ do BACEN aponta pra página protegida por login. Não tentar mais — documentar e seguir. |
| **DRM críticas via PDF parse** | PDF tem 22 críticas em texto puro (`pdftotext + regex`), funcionou bem. Mais simples que esperar planilha. |
| **XSD gerado em Go, não Python** | Pipeline consistente (xsdgen + norma-audit ambos Go); XSD mais confiável via stdlib Go. |
| **Tipos BACEN genéricos** | Mapeamento A→string, N→decimal; refinar em Sprint 3 com dicionário de tipos BACEN completo. |
| **T.3 substituído por teste negativo** | BCValidador é Java proprietário, não roda em ARM64; validação cruzada fica para Sprint 3 via Docker. |

### 📊 Estatísticas finais Sprint 2

```
Total catálogo: 1.099 regras de validação (era 1.081)
├─ 3040:    361 regras
├─ 3050:    170 regras (V11 — Sprint 2 C.1)
├─ 2061-DLO: 518 regras
├─ 2070-DDR:  11 regras
├─ 2060-DRM:  22 regras (Sprint 2 C.2)
└─ 3044:      17 regras (Sprint 2 C.5)

Catálogo total: 5 CADOCs com críticas (era 4)
Schema adicional: leiautes_3044.json (3044 — JSON)
Total material BACEN: 137 arquivos (era 134)
```

### 📂 Novos artefatos

```
_catalogo/
├── criticas.json          1.099 regras (era 1.081)
├── leiautes.json          4.244 linhas (inalterado)
├── leiautes_3044.json     ★ NOVO — schema JSON do 3044
├── drm_criticas_raw.json  ★ NOVO — 22 críticas DRM extraídas do PDF
├── 3040_generated.xsd    ★ NOVO — XSD do 3040 gerado pelo xsdgen
├── extract.py             atualizado (suporta V11 + DRM + normalize_date)
├── extract_drm.py         ★ NOVO — script que extrai DRM do PDF
└── extract_3044.py        ★ NOVO — script que gera schema 3044

tools/                     ★ NOVO — Go spike do Norma Audit
├── go.mod
├── xsdgen/main.go         gera XSD a partir de leiautes.json
└── norma-audit/main.go executa regras 3040 contra XML
```

### 🚧 Gaps remanescentes (vão pra Sprint 3)

| Gap | Origem | Sprint 3 |
|---|---|---|
| 2030 DRSAC críticas | Não público | Solicitar formalmente ou extrair FAQ por inferência |
| Tipos BACEN refinados no XSD | Mapeamento genérico atual | Dicionário completo A1-A50, N1-N20 → XSD types |
| BCValidador real (vs Go) | Java proprietário | Rodar via Docker pra comparação real |
| STA Web Services client | Não implementado | Spike com STA-stub |
| Regras 3044 implementadas em Go | JSON pronto, falta Go | T.4 Sprint 3 |

### 🎯 Próxima sprint (Sprint 3 — preview)
**Tema:** Infraestrutura do Radiant Norma (backend Go + Postgres + Docker)
- Backend Go com API REST
- Schema Registry com Postgres + JSONB
- Norma Audit como microserviço
- STA Client (Playwright first)
- White-label multi-tenant
- LGPD compliance: DPA template, audit log, criptografia
- BCValidador via Docker

---

## v1.0.0 — 2026-07-03 (Sprint 1: Eng. Reversa + Catálogo)

### 🎯 Objetivo da sprint
Construir a **base de conhecimento técnico** para o **Radiant Norma** — inteligência regulatória SaaS da Radiant () para IFs brasileiras — incluindo (a) captura completa de CADOCs BACEN + concorrentes e (b) extração estruturada em JSON pronto para o Norma Audit.

### ✅ Entregas

#### Documentos canônicos
- **`README.md`** (310 linhas) — manifesto + índice completo do material capturado
- **`ENG_REVERSA.md`** (382 linhas) — engenharia reversa de Mitra/LUZ, Matera, cadoc.ai, Dattos, BIBlue, Regulatório Mais (Celcoin), BTech, B3Bee + matriz BACEN × 8 concorrentes + gap DRSAC ESG
- **`PRODUTO_TESE_ROADMAP.md`** (865 linhas) — Radiant Norma: tese, personas, GTM, produto V0-V3, UX/UI, arquitetura Go, compliance pra IF regulada, roadmap 18 meses, identidade da marca
- **`_catalogos/README.md`** (171 linhas) — documentação do catálogo JSON

#### PDFs (3 docs)
- `README.pdf` — **13 páginas**, capa gradient azul escuro com tipografia bold
- `ENG_REVERSA.pdf` — **11 páginas**, capa profissional
- `PRODUTO_TESE_ROADMAP.pdf` — **28 páginas**, capa + jornada UX detalhada
- Pipeline: **Pandoc 3.10 + Chromium headless** (`chrome-headless-shell-mac-arm64`) com CSS3 grid/flexbox + cover page via HTML injection
- Page-break-after: always, footer com marca + paginação automática
- Layout profissional com accent bar, badge, meta-grid na capa

#### Material BACEN capturado (134 arquivos, 56 MB)
| Tipo | Qtd | Detalhes |
|---|---|---|
| **Leiautes oficiais** | 10 | 3040, 3042, 3044 (PDF), 3050, 2030 DRSAC, 2060 DRM, 2061 DLO, 2062 DLI, 2070 DDR (cód 2011), 2160 DRL, 2170 DLP |
| **Críticas (planilhas)** | 4 | 3040 (361 regras), 3050 (191), 2061 DLO (518), 2070 DDR (11) — **1.081 regras no total** |
| **XSDs** | 3 | 3045, 3026, 3050 TXB V4 |
| **XMLs de exemplo** | 3 | 3040 (3000 clientes + cessão), 3026 |
| **Instruções de preenchimento** | 50+ | incluindo versões históricas DRM v2…v11, DLO 2018/2024/2025 |
| **Normativos** | 18 | cartas-circulares, circulares e DRO 5050 |
| **STA** | 6 | manuais Web + WS + FAQ + 2 apresentações |
| **BCValidador** | 1 | manual técnico oficial Deinf/Dine4 v1.3 |
| **Concorrentes** | 29 páginas | LUZ/Mitra, Matera, cadoc.ai, Dattos, BIBlue, etc. |

#### Catálogo JSON estruturado (novo!)
- **`_catalogos/criticas.json`** (534 KB) — **1.081 regras** de 4 CADOCs
- **`_catalogos/leiautes.json`** (1,1 MB) — **4.244 linhas** de campos de 8 CADOCs
- **`_catalogos/extract.py`** — script Python re-rodável (openpyxl + xlrd)
- **`_catalogos/_extracao.log`** — log de extração
- **`_catalogos/README.md`** — documentação completa

#### Scripts e tooling
- **`_gen_pdfs.py`** (10 KB) — gerador de PDFs (Pandoc + Chromium)
- **`_pdf_style.css`** (6 KB) — CSS profissional com capa gradient

### 🏗️ Decisões arquiteturais

| Decisão | Razão |
|---|---|
| **Nome: Radiant Norma** | "Norma" carrega sozinho, ecoa a estética Fortvna (Radiant = marca umbrella), supera conflito com concorrentes |
| **Tagline principal:** *"Radiant Norma — inteligência regulatória pra IF brasileira"* | Claro, em PT, diferencia Matera/Mitra |
| **PDF engine: Pandoc + Chromium** (não LibreOffice) | Suporta CSS3 grid/flexbox/gradient/cover page via HTML injection |
| **Estrutura de marca:** Fortvna → Radiant → Norma | Hierarquia clara, sub-produtos Norma ESG/Radar/Connect/Audit |
| **Planos:** Norma Lite (R$1,5k) / Pro (R$4,5k) / Scale (R$12k) | Entry acessível pra SCD, mid pra IF média, scale pra banco |
| **Norma Audit em 4 camadas** (L1 XSD + L2 Semântico + L3 Cross-doc + L4 Histórico) | L3 e L4 são exclusivos vs BCValidador (diferencial proprietário) |
| **DRSAC ESG como first-mover** | Janela IN BCB 694/2025 (vigência dez/2026), ninguém cobre |
| **Schema Registry versionado por data-base** | A cada release BACEN, IF não mexe em código |

### 📊 Estatísticas finais

```
148 arquivos · 60 MB total
├─ 14 pastas (11 CADOCs + 3 suporte)
├─ 4 markdowns principais (865 + 382 + 310 + 171 linhas)
├─ 3 PDFs (~ 2,5 MB total)
├─ 1 catálogo JSON estruturado (1,7 MB)
└─ 1.081 regras de validação extraídas (base Norma Audit)
```

### 🚧 Limitações conhecidas (Sprint 1)

| Limitação | Sprint que resolveu |
|---|---|
| 3050 crítica está em V9 | ✅ Sprint 2 C.1 (V11) |
| 2060 DRM críticas é PDF (não extraído) | ✅ Sprint 2 C.2 |
| 2030 DRSAC críticas não capturado | ⏸ Sprint 2 C.3 (gap conhecido, não público) |
| Datas-base das críticas não normalizadas | ✅ Sprint 2 C.4 (ISO 8601) |
| 3044 (JSON) sem leiaute planilhado | ✅ Sprint 2 C.5 (schema extraído do manual) |
| 2170 DLP sem leiaute | ✅ Sprint 2 C.6 (modelo de cálculo v201907) |

---

**Autor:** Mavis · Radiant ()
**Mantido por:** Time do Radiant Norma