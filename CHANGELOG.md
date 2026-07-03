# Changelog — cadocs (Radiant Sentinel)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

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

### 🚧 Limitações conhecidas

| Limitação | Sprint que resolve |
|---|---|
| 3050 crítica está em V9 (V11 mais recente) | Sprint 2 (C.1) |
| 2060 DRM críticas é PDF (não extraído) | Sprint 2 (C.2) |
| 2030 DRSAC críticas não capturado | Sprint 2 (C.3) |
| Datas-base das críticas não normalizadas | Sprint 2 (C.4) |
| 3044 (JSON) sem leiaute planilhado | Sprint 2 (extrair do manual) |
| 2170 DLP sem leiaute | Sprint 2 (procurar URL) |

### 📂 Estrutura final

```
cadocs/
├── 3040/        14 arq  6,3M   ← SCR central (mensal)
├── 3042/         2 arq  292K
├── 3044/         1 arq  1,1M
├── 3050/        11 arq  3,7M   ← com XSD oficial TXB V4
├── 2030-DRSAC/   4 arq  680K
├── 2060-DRM/     9 arq  2,7M   ← com críticas PDF
├── 2061-DLO/     3 arq  9,2M
├── 2062-DLI/     2 arq  716K
├── 2070-DDR/     5 arq  2,0M
├── 2160-DRL/     7 arq  5,9M
├── 2170-DLP/     4 arq  3,3M
├── _normativos/ 18 arq  5,5M
├── _referencias/19 arq  5,5M  ← BCValidador + STA + exemplos
├── _concorrentes/29 arq 7,9M  ← LUZ/Mitra, Matera, cadoc.ai, etc.
├── _catalogos/  10 arq  2,4M  ← ★ NOVO catálogo JSON
├── README.md / README.pdf (13p)
├── ENG_REVERSA.md / ENG_REVERSA.pdf (11p)
├── PRODUTO_TESE_ROADMAP.md / PRODUTO_TESE_ROADMAP.pdf (28p)
├── _gen_pdfs.py (10K) — gerador de PDFs
└── _pdf_style.css (6K) — CSS profissional
```

### 🎯 Próxima sprint (ver SPRINT_2.md)
Foco: **completar capturas faltantes + primeiro spike técnico Go** (gerar XSD a partir do `leiautes.json`).

---

**Autor:** Mavis · Radiant Risk Solutions (marca da Fortvna)
**Mantido por:** Time do Radiant Sentinel