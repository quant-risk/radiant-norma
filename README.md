# CADOCs — Banco Central do Brasil

> **Engenharia reversa profunda para construção do Radiant Sentinel — plataforma SaaS da Radiant Risk Solutions (marca da Fortvna) que gera CADOCs para envio ao BACEN via STA, competindo com Mitra e Matera.**

Base local de pesquisa **+ backend Go funcional** (Sprint 3), tudo commitado em git. Estruturada em **2026-07-03**, com todos os leiautes oficiais, instruções de preenchimento, normativos, manuais técnicos, XSDs, exemplos XML, materiais de concorrentes, **catálogo JSON estruturado de críticas + leiautes**, e **API REST backend** com Sentinel Audit + Schema Registry + STA stub.

---

## 1. Estrutura de pastas

```
cadocs/
├── 3040/        # Dados de Risco de Crédito (SCR) — mensal
├── 3042/        # Substituição Parcial do 3040
├── 3044/        # Dados de Eventos em Operações de Crédito (JSON, a partir de 11/2025)
├── 3050/        # Estatísticas Agregadas de Crédito e Arrendamento Mercantil
├── 2030-DRSAC/  # Risco Social, Ambiental e Climático (ESG) — semestral
├── 2060-DRM/    # Demonstrativo de Risco de Mercado
├── 2061-DLO/    # Demonstrativo de Limites Operacionais (conglomerado)
├── 2062-DLI/    # Demonstrativo de Limites Operacionais Individuais
├── 2070-DDR/    # Demonstrativo Diário de Acompanhamento de Parcelas de Requerimento de Capital
├── 2160-DRL/    # Demonstrativo de Risco de Liquidez
├── 2170-DLP/    # Demonstrativo do Indicador de Liquidez de Longo Prazo (NSFR)
├── _normativos/ # Cartas-Circulares, Circulares, Resoluções BCB vigentes
├── _referencias/# STA, BCValidador, COSIF, Desig, exemplos XML 3026/3040
├── _concorrentes/# Engenharia reversa: Mitra, Matera, cadoc.ai, etc.
├── _catalogos/  # ★ Catálogo JSON estruturado (1.099 regras + 4.244 leiautes + 3044 schema)
├── tools/       # ★ Spikes Go (xsdgen + sentinel-audit CLI)
├── backend/     # ★★ Backend Go com API REST funcional (Sprint 3)
├── CHANGELOG.md # Histórico de todas as sprints
├── SPRINT_2.md  # Retrospectiva Sprint 2
├── SPRINT_3.md  # Retrospectiva Sprint 3
├── README.md / README.pdf (13p)
├── ENG_REVERSA.md / ENG_REVERSA.pdf (11p)
└── PRODUTO_TESE_ROADMAP.md / PRODUTO_TESE_ROADMAP.pdf (28p)
```

**Inventário (atualizado 2026-07-03 14h55):** 137 arquivos BACEN · 60 MB · 10 arquivos Go (1.4k linhas) · 7 endpoints REST · 968 críticas + 8 schemas no DB.
- **Material BACEN capturado:** 137 arquivos (50 MB) — mesma lista da Sprint 2
- **Catálogo JSON estruturado:** 1.099 regras de 6 CADOCs + 4.244 linhas de 8 CADOCs + 3044 schema
- **Spike técnico Go (Sprint 2):** `tools/xsdgen/` + `tools/sentinel-audit/` rodando
- **Backend Go (Sprint 3):** `backend/` com cmd/api, cmd/seed, internal/{db,schema,audit,auditlog,sta,api} + migrations embed.FS
- **Stack:** Go 1.22+ · chi router · SQLite (modernc.org/sqlite, pure-Go) → Postgres em prod
- **Endpoints funcionais (curl testados):** /healthz, /v1/schemas/{cadoc}, /v1/rules/{cadoc}, /v1/validate, /v1/sta/submit
- **★ Decisão de linguagem (Go vs Rust) documentada no CHANGELOG § v1.2.0**

---

## 2. Mapa de CADOCs cobertos

| Cód | Sigla | Nome | Periodicidade | Quem envia | Pasta |
|---|---|---|---|---|---|
| **3040** | SCR | Dados de Risco de Crédito | Mensal (9º dia útil) | IF com risco ≥ R$200 | `3040/` |
| **3042** | – | Correção Parcial do 3040 | Sob demanda | Mesmas do 3040 | `3042/` |
| **3044** | – | Dados de Eventos de Operações de Crédito | Até 5º dia útil do evento | IF com crédito | `3044/` |
| **3050** | – | Estatísticas Agregadas de Crédito | Mensal + diária/semanal | IF do SFN | `3050/` |
| **2030** | DRSAC | Risco Social, Ambiental e Climático | Semestral (jun/dez) | S1–S4 + IPs | `2030-DRSAC/` |
| **2060** | DRM | Risco de Mercado | Mensal | S1–S4 | `2060-DRM/` |
| **2061** | DLO | Limites Operacionais (Conglomerado) | Mensal | S1–S4, conglomerados | `2061-DLO/` |
| **2062** | DLI | Limites Operacionais Individuais | Mensal | Cada IF individualmente | `2062-DLI/` |
| **2070** *(cód 2011)* | DDR | Diário Requerimento Capital | Diário | S1–S4 | `2070-DDR/` |
| **2160** | DRL | Risco de Liquidez (LCR) | Diário | S1 (Res. 4.401/15) | `2160-DRL/` |
| **2170** | DLP | Indicador Liquidez Longo Prazo (NSFR) | Mensal | S1 | `2170-DLP/` |
| **5050** | DRO | Risco Operacional | Trimestral | S1–S4 | `_normativos/` |

> **Atenção:**
> - O BACEN renomeou DDR para código **2011** (não 2070). Pasta mantida como `2070-DDR/` para alinhamento conceitual.
> - Em **2020** o nome oficial mudou de "CADOC" para **CRDI** (Catálogo de Recepção de Documentos e Informações). Mas a indústria continua falando "CADOC" — mantenha o termo informal.

---

## 2.5 ★ Catálogo Estruturado JSON — base do Sentinel Audit

Em **2026-07-03** extraí todas as planilhas de **críticas** e **leiautes** dos CADOCs capturados para JSON estruturado. Esta é a base direta do **Sentinel Audit** (validador do Radiant Sentinel).

### Estatísticas

| Recurso | CADOCs cobertos | Linhas extraídas |
|---|---|---|
| **Críticas (`_catalogos/criticas.json`)** | 4 (3040, 3050, 2061 DLO, 2070 DDR) | **1.081 regras** |
| **Leiautes (`_catalogos/leiautes.json`)** | 8 (3040, 3042, 3050, 2030 DRSAC, 2060 DRM, 2062 DLI, 2070 DDR, 2160 DRL) | **4.244 linhas** |

### Como o Sentinel Audit usa

| Camada | O que faz | Fonte de dados |
|---|---|---|
| **L1 — Estrutural (XSD)** | Valida sintaxe XML/JSON, tipos, obrigatoriedade | `leiautes.json` → XSD gerado em Go |
| **L2 — Semântica** | Regras de negócio do BACEN (`SCR3040_Criticas.xls` etc) | `criticas.json` → regras portadas pra Go |
| **L3 — Cross-doc** | "3040 diz X mas 4111 diz Y" | Carrega múltiplos CADOCs em memória (exclusivo) |
| **L4 — Histórico** | Diff vs base anterior | Versionamento Postgres + diff (exclusivo) |

### Como o BCValidador BACEN se relaciona

**BACEN BCValidador:**
- Faz L1 (XSD) + L2 (Semântico) — veja `_referencias/BCValidador_DocTecnico.pdf` (Deinf/Dine4 v1.3, 2013)
- Aplicativo Java proprietário, distribuído como `.jar`
- Valida 1 documento por vez
- Modos: Java API, CLI, GUI local

**Radiant Sentinel Audit:**
- Faz L1 + L2 (regras públicas do BACEN) + L3 (cross-doc) + L4 (histórico)
- API REST + Webhooks
- Multi-tenant cloud-native
- Audit log tamper-evident (hash chain)
- SOC 2 Type II em roadmap

**Estratégia:** Não copiamos o BCValidador (é binário proprietário). Reimplementamos em Go as regras públicas das planilhas de críticas (`SCR3040_Criticas.xls` etc) — e adicionamos 3 camadas que o BCValidador não tem (L3, L4, REST).

### Como usar o catálogo

```bash
# Re-extrair (caso queira atualizar)
cd cadocs/
python3 _catalogos/extract.py

# Inspecionar críticas do 3040
python3 -c "
import json
d = json.load(open('_catalogos/criticas.json'))
for c in d['criticas']['3040'][:3]:
    print(c['codigo'], '-', c['regra'])
"
# B01 - Erro XML
# B02 - Arquivo .ZIP deve ser gerado pelo aplicativo validador
# B03 - Instituição remetente deve possuir autorização
```

Documentação completa em [`_catalogos/README.md`](_catalogos/README.md).

---

## 3. O que tem em cada pasta (top files)

### 3040/ — Documento central do SCR (mensal)
- `SCR_InstrucoesPreenchimento_Doc3040.pdf` — Manual de Instruções de Preenchimento (1.6 MB)
- `SCR_InstrucoesDePreenchimento_Doc3044.pdf` — Instruções do 3044 (1.1 MB)
- `SCR3040_Leiaute.xls` — **Leiaute oficial** com campos e domínios (290 KB)
- `SCR3040_Criticas.xls` — **Planilha de críticas** (regras semânticas do Validador)
- `Manual_Validador_SCR3040.pdf` — Manual do App Validador Java (876 KB)
- `Manual_Envio_3040.pdf` — Manual de envio pelo STA (567 KB)
- `Manual_Particionamento_3040.pdf` — Como partir 3040 > 4 GB (186 KB)
- `SCR3040_ConceitosDeConsultas.pdf` — Conceitos das consultas (886 KB)
- `SCR3040_Manual_Consulta_Web_Service.pdf` — WSCR0001 (963 KB)
- `SCR3040_DetalhamentoFIDC.pdf` — Como FIDC reporta (262 KB)
- `SCR3040_Dispensa_Envio.pdf` — Como pedir dispensa de envio (667 KB)
- `SCR3045.xsd` — XSD do documento 3045 (consulta via arquivo)
- `exemploDesempenhoOperacao.xml` — **Exemplo XML de 3040** com 3000 clientes
- `noticias_Doc3040.xml` — Feed de notícias SCR

### 3042/ — Correção pontual
- `Manual_SubstituicaoParcial_3042.pdf` — Manual do 3042
- `SCR3042_Leiaute.xls` — Leiaute oficial

### 3050/ — Estatísticas Agregadas
- `Instrucoes_Preenchimento_Doc3050.pdf` — Manual
- `Leiaute_TXB_XML_V3.xls` — **Leiaute XML**
- `Criticas_TXB_V9.xlsx` — Críticas TXB
- `Equivalencia_Modalidades_3040_3050.pdf` — Tabela de equivalência
- `CC_3915_Doc3050.pdf`, `CC_3932_Doc3050.pdf`, `CC_3974_Doc3050.pdf` — Normativos

### 2030-DRSAC/ — Risco Social, Ambiental e Climático (ESG)
- `Instrucoes_Preenchimento_DRSAC.pdf` — Manual oficial (348 KB)
- `Leiaute_DRSAC.xlsx` — **Leiaute oficial** XLSX (74 KB)
- `Perguntas_Respostas_DRSAC.pdf` — FAQ oficial BACEN (177 KB)
- Base normativa:
  - **Resolução BCB 151/2021** — escopo do DRSAC
  - **IN BCB 222/2021** — procedimentos iniciais
  - **IN BCB 328/2022** — ajuste do agrupamento CNAE
  - **IN BCB 423/2023** — leiaute compacto (cliente 14 chars)
  - **IN BCB 694/2025** — códigos "05"–"09" + reescrita consolidada (vigência dez/2026)

### 2060-DRM/ — Risco de Mercado
- `DRM_2060_Leiaute.xls` — **Leiaute oficial**
- `DRM_2060_InstrucoesPreenchimento_v11.pdf` — Versão mais recente (jan/25)
- Versões históricas: v2, v3, v7, v8, v9

### 2061-DLO/ — Limites Operacionais (Conglomerado)
- `DLO_2061_Instrucoes_v202407.pdf` — **Versão vigente (jul/24)**
- `DLO_2061_Instrucoes_v201801.pdf` — Versão histórica

### 2062-DLI/ — Limites Operacionais Individuais
- `DLI_2062_Leiaute.xlsx` — **Leiaute oficial**
- `DLI_2062_InstrucoesPreenchimento_v3.pdf` — Versão atual

### 2070-DDR/ — Diário (cód 2011)
- `DDR_2011_Leiaute.xls` — **Leiaute oficial**
- `DDR_2011_InstrucoesPreenchimento_v3.pdf` — Última versão
- `DDR_2011_MensagensProcessamento.pdf` — Catálogo de mensagens de erro

### 2160-DRL/ — Risco de Liquidez (LCR)
- `DRL_2160_Leiaute_v201603.xlsx` — Leiaute (versão antiga; **vigente está em leiautedocumentoscrd**)
- `DRL_2160_Instrucoes_OrientacoesGerais.pdf` — Orientações
- `DRL_2160_Anexo2_Exemplos.pdf` — Exemplos de cálculo
- `DRL_2160_Workshop_LCR.pdf` — Workshop BACEN Modelo II

### 2170-DLP/ — NSFR
- `DLP_2170_Instrucoes_Brasil.pdf` — Brasil
- `DLP_2170_Instrucoes_OrientacoesGerais.pdf` — Geral
- `DLP_2170_ModeloCalculo.xlsx` — Planilha modelo
- `DLP_2170_Workshop.pdf` — Workshop BACEN

---

## 4. _normativos/ — Base regulatória

| Arquivo | Descrição |
|---|---|
| `CC_3451_CADOC_3040.pdf` | Carta Circular original (2009) |
| `CC_3517_SCR.pdf` | Carta Circular 3.517/2011 |
| `CC_3540_SCR.pdf` + v4 | Carta Circular 3.540/2012 — procedimentos envio |
| `CC_3869_SCR_v5.pdf` | Carta Circular 3.869/2018 — base SCR atual |
| `CC_3878_DRM.pdf` | DRM — atualizações 2018 |
| `CC_3687_DRM.pdf` | DRM — base 2014 |
| `CC_3768_DRL.pdf`, `CC_3775_DRL.pdf`, `CC_3812_DRL.pdf`, `CC_4012_DRL.pdf` | DRL — sequência de alterações |
| `CC_3326_DRL.pdf` | DRL — origem |
| `CC_3958_DLP.pdf` | DLP — Carta Circular 3.958 |
| `CC_3989_DDR.pdf` | DDR — Carta Circular 3.989/2019 |
| `Circular_3869_NSFR.pdf` | Circular 3.869/2017 — NSFR metodologia |
| `DRO_5050_Instrucoes.pdf` | Risco Operacional doc 5050 |

---

## 5. _referencias/ — Infraestrutura BACEN

| Arquivo | Para quê |
|---|---|
| `STA_Manual_Web.pdf` | Como usar STA via browser |
| `STA_Manual_WebServices.pdf` | Como integrar STA via Web Services (REST) |
| `STA_FAQ.pdf` | FAQ STA |
| `STA_Apresentacao_2012.pdf`, `_v2.pdf` | Apresentações técnicas STA |
| `BCValidador_DocTecnico.pdf` | Documentação técnica do validador XML geral do BACEN |
| `Captacao_Informacoes_Desig.pdf` | Lista de e-mails/responsáveis do Desig |
| `Carta_Circular_3588_STA.pdf` | Norma que estabelece o STA |
| `Comunicado_19683_Enderecos.pdf`, `Comunicado_20532_Enderecos.pdf` | Endereços de contato regulatórios |

### STA — pontos críticos
- **URL produção:** `https://sta.bcb.gov.br/sta` (Web) e `https://sta.bcb.gov.br/staws` (WS)
- **URL homologação:** `https://sta-h.bcb.gov.br/sta`
- **Auth:** HTTPS + usuário Sisbacen cadastrado no serviço `PSTA300`
- **Compressão:** ZIP (Deflate) ou GZIP — `.zip` ou `.gz`
- **Limite:** 5 GB máximo via Web (WS não tem limite claro)
- **Concorrência:** 10 uploads/downloads simultâneos por IF
- **Hash:** SHA-256 para integridade
- **Protocolo:** numérico, sequencial, até 18 dígitos

---

## 6. _concorrentes/ — Engenharia reversa

| Concorrente | Material capturado | Insight chave |
|---|---|---|
| **Matera** (5 arquivos) | Blog sobre CADOC 3040/3044, página RegTech, visão geral | "Validações prévias e gestão de ciclo de vida" — não basta empacotar XML; gerencia pipeline inteiro |
| **cadoc.ai** (9 arquivos) | Home, Norma Skills, posts IN BCB 733/693/Res 577 | Norma Skills = RAG jurídico + alerta 24/7 + skill injetável em IA |
| **LUZ / Mitra** | Whitepaper + página oficial + youtube | Calculadora regulatória modular — "líder de mercado no módulo de Basileia" |
| **Regulatório Mais** (Celcoin) | Home + guia | 4 módulos: Contábil, Não-Contábil (IP/SCD), Crédito (SCR), Risco |
| **BIBlue** | Compliance Regulatório | Templates pré-configurados CADOC/SCR/COS |
| **Dattos** | Tudo sobre o novo CADOC | Foco em 3044 + JSON |
| **BTech** | Pipeline 4-8 semanas | Cobra por projeto/hora, ICP-Brasil obrigatório |
| **B3Bee** | Mudanças DLO 2024 | Calendário regulatório |
| **Compliasset** | Alertas normativos | Alerta proativo de mudanças |
| **FBM Educação** | Validação XML/STA/CRD | Curso para "Especialista em CADOCs" |
| **Lerian Studio** | Arquitetura validation-first | Filosofia: integridade no ledger antes do reporte |

> Ver `ENG_REVERSA.md` para análise profunda.

---

## 7. URLs fonte (BACEN oficial)

| Recurso | URL |
|---|---|
| Página central leiautes CRD | https://www.bcb.gov.br/estabilidadefinanceira/leiautedocumentoscrd |
| SCR 3040 | https://www.bcb.gov.br/estabilidadefinanceira/scrdoc3040 |
| SCR 3050 FAQ | https://www.bcb.gov.br/estabilidadefinanceira/scrdoc3050_faq |
| DRL 2160 | https://www.bcb.gov.br/estabilidadefinanceira/leiaute_drl2160 |
| DRM 2060 | https://www.bcb.gov.br/estabilidadefinanceira/leiautedocumentoDRM |
| STA | https://www.bcb.gov.br/acessoinformacao/sistematransferenciaarquivos |
| SCR principal | https://www.bcb.gov.br/estabilidadefinanceira/scr |
| Normas SCR vigentes | https://www.bcb.gov.br/estabilidadefinanceira/scrnormasemvigor |
| COSIF leiautes | https://www.bcb.gov.br/estabilidadefinanceira/leiautescosif |
| Validador XML info | https://www.bcb.gov.br/estabilidadefinanceira/validador_xml_info |
| Gestor SSI (Desig) | https://www.bcb.gov.br/fis/info/gestoressisinf.asp?frame=1 |

---

## 8. Próximos passos

1. ✅ Manifesto estruturado
2. ✅ Documentação de engenharia reversa → `ENG_REVERSA.md`
3. ⏳ Construir o parser/validador XSD para cada CADOC (a partir dos leiautes XLS)
4. ⏳ Modelar schema SQL com todos os campos
5. ⏳ Implementar gerador de XML por template
6. ⏳ Cliente STA (HTTPS + WS) para envio automatizado
7. ⏳ UI web para IFs gerarem, validarem e enviarem
8. ⏳ Integração com contas COSIF (3050 ↔ 3040)
9. ⏳ Webhooks de notificação de novas IN BCB (radar regulatório)

---

## 9. Pontos de atenção importantes

1. **Versões múltiplas:** muitos CADOCs têm versão vigente + 5-10 históricas. Sempre usar a versão mais recente publicada na página `leiautedocumentoscrd`.
2. **Layouts XSD:** BACEN fornece XSD só pra alguns (3045, 3026, 3060). Para outros, é preciso **gerar XSD a partir do XLS de leiaute** (planilha é a source of truth).
3. **Validador proprietário:** `BCValidador` é Java-only, requer JRE. Aplicar regras semânticas em Go/Python requer reimplementar a planilha `SCR3040_Criticas.xls`.
4. **STA exige certificado digital ICP-Brasil A1/A3** do usuário Sisbacen + serviço PSTA300 atribuído.
5. **DDR/DRL têm envios DIÁRIOS** (não mensais) — automação precisa de scheduler robusto.
6. **3044 (eventos) é JSON**, não XML — quebra o padrão atual.
7. **IN BCB 733/2026** vai mudar 3040 em 3 ondas (mai/jul/nov 2026).
8. **Resolução BCB 577/2026** introduz base subconsolidada — apenas S1, com homologação até 31/12/2026.
9. **ACAM212** novo CADOC para cripto/ativos virtuais (a partir de maio/2026).
10. **FIDC** tem tratamento especial no 3040 — consultar `SCR3040_DetalhamentoFIDC.pdf`.