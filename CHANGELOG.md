# Changelog — cadocs (Radiant Sentinel)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

## v1.1.0 — 2026-07-03 (Sprint 2: Sentinel Audit Spike + capturas restantes)

### 🎯 Objetivo da sprint
**Fechar gaps de captura + primeiro spike técnico Go** do Sentinel Audit (gerar XSD + executar regras). Validação contra BCValidador substituída por teste negativo (XML quebrado) — BCValidador é Java proprietário e não roda nativamente em ARM64 macOS.

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
- **`tools/sentinel-audit/main.go`** — executor de regras 3040 contra XML
  - Roda: `go run ./tools/sentinel-audit -xml 3040/exemploDesempenhoOperacao.xml`
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
| **XSD gerado em Go, não Python** | Pipeline consistente (xsdgen + sentinel-audit ambos Go); XSD mais confiável via stdlib Go. |
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

tools/                     ★ NOVO — Go spike do Sentinel Audit
├── go.mod
├── xsdgen/main.go         gera XSD a partir de leiautes.json
└── sentinel-audit/main.go executa regras 3040 contra XML
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
**Tema:** Infraestrutura do Radiant Sentinel (backend Go + Postgres + Docker)
- Backend Go com API REST
- Schema Registry com Postgres + JSONB
- Sentinel Audit como microserviço
- STA Client (Playwright first)
- White-label multi-tenant
- LGPD compliance: DPA template, audit log, criptografia
- BCValidador via Docker

---

## v1.0.0 — 2026-07-03 (Sprint 1: Eng. Reversa + Catálogo)

### 🎯 Objetivo da sprint
Construir a **base de conhecimento técnico** para o **Radiant Sentinel** — sentinela regulatória SaaS da Radiant Risk Solutions (marca da Fortvna) para IFs brasileiras — incluindo (a) captura completa de CADOCs BACEN + concorrentes e (b) extração estruturada em JSON pronto para o Sentinel Audit.

### ✅ Entregas

#### Documentos canônicos
- **`README.md`** (310 linhas) — manifesto + índice completo do material capturado
- **`ENG_REVERSA.md`** (382 linhas) — engenharia reversa de Mitra/LUZ, Matera, cadoc.ai, Dattos, BIBlue, Regulatório Mais (Celcoin), BTech, B3Bee + matriz BACEN × 8 concorrentes + gap DRSAC ESG
- **`PRODUTO_TESE_ROADMAP.md`** (865 linhas) — Radiant Sentinel: tese, personas, GTM, produto V0-V3, UX/UI, arquitetura Go, compliance pra IF regulada, roadmap 18 meses, identidade da marca
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
| **Nome: Radiant Sentinel** | "Sentinel" carrega sozinho, ecoa a estética Fortvna (Radiant = marca umbrella), supera conflito com concorrentes |
| **Tagline principal:** *"Radiant Sentinel — sentinela regulatória pra IF brasileira"* | Claro, em PT, diferencia Matera/Mitra |
| **PDF engine: Pandoc + Chromium** (não LibreOffice) | Suporta CSS3 grid/flexbox/gradient/cover page via HTML injection |
| **Estrutura de marca:** Fortvna → Radiant → Sentinel | Hierarquia clara, sub-produtos Sentinel ESG/Radar/Connect/Audit |
| **Planos:** Sentinel Lite (R$1,5k) / Pro (R$4,5k) / Scale (R$12k) | Entry acessível pra SCD, mid pra IF média, scale pra banco |
| **Sentinel Audit em 4 camadas** (L1 XSD + L2 Semântico + L3 Cross-doc + L4 Histórico) | L3 e L4 são exclusivos vs BCValidador (diferencial proprietário) |
| **DRSAC ESG como first-mover** | Janela IN BCB 694/2025 (vigência dez/2026), ninguém cobre |
| **Schema Registry versionado por data-base** | A cada release BACEN, IF não mexe em código |

### 📊 Estatísticas finais

```
148 arquivos · 60 MB total
├─ 14 pastas (11 CADOCs + 3 suporte)
├─ 4 markdowns principais (865 + 382 + 310 + 171 linhas)
├─ 3 PDFs (~ 2,5 MB total)
├─ 1 catálogo JSON estruturado (1,7 MB)
└─ 1.081 regras de validação extraídas (base Sentinel Audit)
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

**Autor:** Mavis · Radiant Risk Solutions (marca da Fortvna)
**Mantido por:** Time do Radiant Sentinel