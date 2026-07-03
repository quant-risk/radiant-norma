# Sprint 2 — Radiant Norma

> **Duração:** 2026-07-04 a 2026-07-17 (2 semanas)
> **Objetivo:** Completar capturas faltantes, fazer primeiro spike técnico Go, validar arquitetura do Norma Audit contra BCValidador.

## Contexto

Sprint 1 entregou:
- ✅ 134 arquivos BACEN capturados
- ✅ Catálogo JSON estruturado (1.081 críticas + 4.244 leiautes)
- ✅ 3 documentos canônicos + 3 PDFs

**Gaps conhecidos (do CHANGELOG § Limitações):**
- 3050 crítica em V9 (V11 mais recente)
- 2060 DRM críticas é PDF (não extraído)
- 2030 DRSAC críticas não capturado
- Datas-base não normalizadas
- 3044 (JSON) sem leiaute planilhado
- 2170 DLP sem leiaute
- **ZERO código Go escrito** — só extração de dados

## Backlog priorizado

### 🔴 P0 — Crítico (bloqueia F0 do roadmap)

#### C.1 — Capturar 3050 críticas V11
- **Por quê:** V9 está no disco, mas V11 (mais recente) está em `aprendervalor.bcb.gov.br`
- **Tempo:** 30 min
- **Saída:** `3050/Criticas_TXB_V11.xlsx` (substitui V9 ou convive com ele)
- **Critério done:** arquivo no disco, parseado, integrado em `criticas.json`

#### C.2 — Extrair 2060 DRM críticas do PDF
- **Por quê:** única crítica que ficou de fora (PDF em vez de planilha)
- **Tempo:** 1h
- **Saída:** `2060-DRM/criticas_extraidas.json` + integração em `criticas.json`
- **Técnica:** `pdftotext + regex` ou copiar tabela à mão do PDF
- **Critério done:** ~50-100 críticas adicionadas

#### C.3 — Capturar 2030 DRSAC críticas (URL alternativa)
- **Por quê:** ESG é first-mover, sem críticas não dá pra validar
- **Tempo:** 30 min
- **Caminho:** `https://www.bcb.gov.br/content/estabilidadefinanceira/Leiaute_de_documentos/drsac2030/Criticas-Validacao-DRSAC.xlsx` (mencionado em B3Bee)
- **Critério done:** ~20-30 críticas DRSAC no catálogo

### 🟡 P1 — Importante (não bloqueia, mas reduz dívida técnica)

#### C.4 — Normalizar data-base das críticas
- **Por quê:** mistura YYYYMM (Excel serial), AAAAMM, AAAA-MM-DD
- **Tempo:** 2h
- **Saída:** campo `data_base_inicio` normalizado em ISO 8601 (`2020-06-01`)
- **Critério done:** todas as 1.081 críticas com data_base_inicio parseável

#### C.5 — Capturar 3044 leiaute JSON schema
- **Por quê:** 3044 é JSON, sem schema não dá pra validar L1
- **Tempo:** 1h
- **Caminho:** extrair do manual `SCR_InstrucoesDePreenchimento_Doc3044.pdf` (já temos)
- **Saída:** `_catalogos/leiautes_3044.json` com estrutura do JSON
- **Critério done:** schema JSON parseado, exemplos validados

#### C.6 — Capturar 2170 DLP leiaute planilhado
- **Por quê:** único Basileia que não tem leiaute capturado
- **Tempo:** 1h
- **Caminho:** `https://www.bcb.gov.br/estabilidadefinanceira/leiautesDLP`
- **Critério done:** arquivo .xls/.xlsx em `2170-DLP/`

### 🟢 P2 — Diferencial técnico (validação contra BCValidador)

#### T.1 — Spike Go: gerar XSD a partir de `leiautes.json` (3040)
- **Por quê:** sem XSD oficial (BACEN só publica 3045 e 3026), precisamos gerar
- **Tempo:** 4-6h
- **Entregável:** `tools/xsdgen/main.go` + XSD gerado em `2040/schema.xsd`
- **Critério done:** XSD válido, exemplo XML valida contra ele usando `xmllint --schema`
- **Padrão de qualidade:**
  ```go
  // xsdgen lê leiautes.json e emite XSD
  // Cada campo vira um <xs:element> ou <xs:attribute>
  // Suporta obrigatoriedade, tipos, domínios
  ```

#### T.2 — Spike Go: ler `criticas.json` e executar 5 regras do 3040
- **Por quê:** validar que a estrutura JSON é executável em Go
- **Tempo:** 4-6h
- **Entregável:** `tools/norma-audit/main.go` que carrega 3040 + executa B01-B05
- **Critério done:** valida `exemploDesempenhoOperacao.xml` (já temos), emite erros esperados

#### T.3 — Validação paralela Norma Audit vs BCValidador
- **Por quê:** garantir que reimplementação em Go bate com a oficial
- **Tempo:** 2h (depois de T.1 + T.2)
- **Como:** rodar os dois no mesmo XML de exemplo, comparar lista de erros
- **Critério done:** ≥ 90% match nas primeiras 50 críticas do 3040

### 🔵 P3 — Backlog futuro (depois da Sprint 2)

- L.1 — Domain `radiant-norma.com.br` (R$ 50)
- L.2 — INPI registro de marca "Radiant Norma" (R$ 355)
- L.3 — Briefing de logo conceitual pro designer
- L.4 — Landing page (Next.js + Vercel)
- L.5 — 5 calls discovery com SCDs da rede Fortvna
- L.6 — Spike: gerar XSD para 3050 + 2030 DRSAC (depois de T.1 validado)
- L.7 — Schema Registry Postgres + versionamento
- L.8 — STA client (Playwright primeiro, WS nativo depois)

## Estimativa de horas

| Categoria | Horas |
|---|---|
| P0 (C.1, C.2, C.3) | 2h |
| P1 (C.4, C.5, C.6) | 4h |
| P2 (T.1, T.2, T.3) | 12-14h |
| **TOTAL Sprint 2** | **18-20h** (~ 1 semana útil focada) |

## Definition of Done da Sprint 2

- [ ] `criticas.json` tem **1.150+ regras** (1.081 + V11 + DRM + DRSAC)
- [ ] `leiautes.json` tem **5.000+ linhas** (4.244 + 3044 + 2170)
- [ ] Datas-base normalizadas em todas as críticas
- [ ] XSD Go gerado para 3040 (T.1)
- [ ] Norma Audit Go executa 5 regras do 3040 (T.2)
- [ ] Validação ≥ 90% match vs BCValidador (T.3)
- [ ] CHANGELOG.md atualizado com v1.1.0
- [ ] Commit `feat(sprint-2): ...`
- [ ] SPRINT_3.md criado com próximo backlog

## Riscos da Sprint 2

| Risco | Mitigação |
|---|---|
| V11 do 3050 mudou formato | Se V11 tiver estrutura muito diferente, manter V9 e documentar |
| 2060 DRM PDF é escaneado (não texto) | Usar `ocrmypdf` ou copiar à mão do PDF |
| XSD gerado tem gaps vs oficial | Documentar limitações, gerar aviso "use com cautela" |
| BCValidador não roda em macOS ARM | Rodar via Docker ou validar com `xmllint` apenas |
| Go installation ausente | Verificar `go version` no início da sprint |

## Ferramentas que vou precisar

```bash
# Verificar Go
go version

# Instalar se necessário
brew install go

# Dependências
go get github.com/xuri/excelize/v2  # ler .xls/.xlsx
go get encoding/xml                  # parse XML
go get github.com/xeipuuv/gojsonschema  # validar JSON Schema

# xmllint (para T.1 validação)
brew install libxml2
```

## Próxima sprint (Sprint 3 — preview)

**Tema:** Infraestrutura do Radiant Norma (backend Go + Postgres + Docker)

- Backend Go com API REST
- Schema Registry com Postgres + JSONB
- Norma Audit como microserviço
- STA Client (Playwright first)
- White-label multi-tenant
- LGPD compliance: DPA template, audit log, criptografia
- Testes contra BCValidador com suite automatizada

---

**Autor:** Mavis · Radiant ()
**Stakeholder:** Henrique Costa · henrique@fortvna.com.br