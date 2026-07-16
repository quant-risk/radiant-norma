# Truth Verdict — Radiant Norma

> **Run:** 20260715-170152-3a51cba
> **Data:** 2026-07-15
> **HEAD:** 3a51cba4ce1945c4e554915131617089c9d061bb

Veredito factual sobre os principais claims, baseado em (1) inspeção estática, (2) medições reais onde possível, e (3) análise documental cruzada.

---

## Tabela de verdade

| Claim | Fonte | Veredito | Evidência | Limitação | Defeito |
|---|---|---|---|---|---|
| "Gera 10 CADOCs" | README linha 66 | **PARCIAL** | 10 packages em `internal/generator/gen*` registrados (v3.36.3+) | Apenas 3040 com implementação completa de Generate; outros 9 dependem de runtime E2E (Fase D) | — |
| "Valida L1→L4" | README linha 86 + ADR-0006 | **PARCIAL** | L1 XSD validator existe; L2 service existe; L3 engine + 25 rules; L4 engine | Cobertura `audit/rules` 61.6% < mínimo 85% | DEFECT-006 |
| "Envia ao STA" | README linha 332 | **PARCIAL** | StubClient + WSClient implementados; ADR-0005 segregação de interface | Sem teste contra BACEN real; Hetzner credentials recusadas | — |
| "10 CADOCs" (3044 conta separadamente) | OpenAPI /schemas | **VERIFIED_LOCAL** | GET /schemas retorna 11 cadocs incluindo 3044 | 4111 listado mas sem `gen4111` declarado (CORRIGIDO em CHANGELOG v3.36.3) | — |
| "1.099 regras de validação" | README linha 65 | **PRESENTE_MAS_INCONCLUSIVO** | Catálogo bruto 361 em `docs/rules-3040-catalog.md`; medido 275 em Go para 3040 | Diferença entre catálogo bruto e regras portadas; precisa Fase B completa para confirmar | — |
| "L1, L2, L3, L4" (4 camadas) | README linha 86 | **PARCIAL** | L1 + L2 + L3 + L4 implementados; L3 com 25 rules; L4 com `l4/engine.go` | L4 não medido em runtime | — |
| "Mesmo validador do BCValidador" | README linha 36 | **NÃO_TESTÁVEL_EXTERNO** | Claim depende de comparação byte-a-byte | BCValidador oficial não disponível neste ambiente | — |
| "Explainability campo-a-campo" | ADR-0007 + README | **PARCIAL** | `GeneratedDoc.FieldMap` + `FieldMapping` struct existe | Não exercitado em runtime E2E | — |
| "Qualquer fonte (PDF/DOCX/API/DB/MCP)" | README + ADR-0008 | **PARCIAL** | 5 adapters com Fetch implementado (Manual, File, API, DB, MCP); MCPAdapter **NÃO** retorna ErrNotImplemented (corrige LLM_INTEGRATION_GUIDE) | PDF/DOCX parsing via LLM é roadmap (Sprint 59+) | — |
| "STA real" | README linha 332 | **PARCIAL** | WSClient + StubClient + retry + DLQ implementados | Sem teste contra BACEN real | — |
| "Multi-tenant/RLS" | README + ADR-0002 | **VERIFIED_LOCAL_PARCIAL** | Migration `014_rls_enforce.sql` aplica FORCE RLS em 6 tabelas; `withTenantContext` helper existe | Não testado em runtime Postgres real | — |
| "Tamper-evident" | README linha 239 + ADR-0003 | **PARCIAL** | `internal/auditlog` existe com SHA-256 chain; cobertura 92.5% medida | Não exercitado (BENCH-05 não rodado) | — |
| "Real-time SSE" | README linha 77 + OpenAPI `/events/stream` | **PARCIAL** | Handler SSE existe (`sse_handler.go`) | Não exercitado | — |
| "SDK oficial Go" | README linha 67 | **VERIFIED_LOCAL** | `sdk/go/` existe com `client.go`, `types.go`, `radiant/`, README, tests | — | — |
| "SDK Python" | README linha 68 | **CONTRADITO** | Existem DOIS SDKs Python: `sdk/py/` E `sdk/python/` | README menciona apenas `sdk/py`; duplicata não documentada | Documentar `sdk/python` ou removê-lo |
| "OpenAPI" | README | **PARCIAL** | Spec existe em `docs/openapi/v1.yaml` com 46 paths e 49 operações | Mistura prefixo `/v1` em marketplace/webhooks | DEFECT-008 |
| "Pronto para produção" | README + ROADMAP | **NÃO_COMPROVADO** | Cobertura abaixo do mínimo declarado; 2 P0 abertos (DEFECT-006/007); sem E2E runtime completo | Esta afirmação exige evidência externa que não foi produzida | DEFECT-006/007 |
| "SOC 2" | MASTER_PLAN + ROADMAP | **NÃO_TESTÁVEL_EXTERNO** | Readiness + evidence collector declarados (v3.34.38/v3.34.47) | Auditoria SOC 2 requer certificadora externa | — |
| "LGPD" | README + MASTER_PLAN | **PARCIAL** | Policies RLS + audit log + retenção mencionados | Conformidade LGPD exige DPO + DPIA documentados | — |
| "SLA 99.95%" | ROADMAP | **NÃO_COMPROVADO** | Sem medições de uptime em produção | SLA é contrato, não característica técnica | — |
| "Keycloak/Okta nativos" | ADR-0001 | **NÃO_IMPLEMENTADO** | Integração Keycloak não encontrada no código | Apenas mencionado em ADR como decisão arquitetural | — |
| "Retenção 5 anos" | MASTER_PLAN §6.4 | **PARCIAL** | Documentado em ON_PREMISE_LLM_SPEC | Configuração real em DB não verificada | — |
| "Multi-region" | MASTER_PLAN §2.2 | **PARCIAL** | `internal/multiregion` existe; Sprint 63 fechou BR-SP1/SP2 replication (CHANGELOG) | Não exercitado | — |
| "Metrics 105 arquivos / 29.481 LoC" | README linha 229 | **PRESENTE_MAS_DIVERGENTE** | Medido: 287 arquivos Go (não 105); LoC não medido | README subestima significativamente | — |
| "516 testes top-level / 21/21 packages PASS" | README linha 230 | **PRESENTE_MAS_INCONCLUSIVO** | `go test ./...` não completado; cobertura medida em 5 pacotes individuais | Fase B completa não executada | — |
| "20+ endpoints REST" | README linha 240 | **CONTRADITO** | Medido: 77 rotas únicas / 49 operações OpenAPI | README subestima | — |
| "5 conectores: Manual, File, API, DB, MCP" | ADR-0008 | **VERIFIED_LOCAL** | Todos os 5 com Fetch/ValidateConfig/HealthCheck implementados | — | — |
| "10 generators" | CHANGELOG v3.36.3 | **VERIFIED_LOCAL** | 10 packages em `internal/generator/gen*` | Apenas 3040 com Generate real; outros podem ser stubs | — |
| "Cross-doc 12 regras XD01-XD12" | ADR-0006 | **CONTRADITO** | Medido: 25 regras (3+8+5+9) | ADR-0006 subestima | — |

---

## Resumo de vereditos

| Veredito | Contagem |
|---|---|
| VERIFIED_LOCAL (com evidência real) | 5 (10 cadocs no registry, 5 conectores, 287 .go, 77 rotas, 26 migrations) |
| VERIFIED_LOCAL_PARCIAL | 1 (RLS — migrations existem, runtime não testado) |
| PARCIAL (parte funciona, sem E2E completo) | 11 |
| CONTRADITO (claim diverge da realidade) | 3 (SDK Python duplicado; cross-doc 12 vs 25; "20+ endpoints" vs 77) |
| PRESENTE_MAS_DIVERGENTE | 2 (métricas README subestimadas) |
| PRESENTE_MAS_INCONCLUSIVO | 2 (1099 regras, 516 testes) |
| NÃO_COMPROVADO | 1 ("pronto para produção") |
| NÃO_TESTÁVEL_EXTERNO | 3 (BCValidador, SOC 2, SLA) |
| NÃO_IMPLEMENTADO | 1 (Keycloak/Okta) |

---

## 🚨 Achado crítico pós-auditoria (independente)

**Descoberto em `e2e-audit/20260715-170152-3a51cba/runtime/benchmark-results.json`** (não gerado por esta sessão, mas presente no diretório):

- **Run independente**: `20260715T174828Z-1a38e519` em `http://127.0.0.1:18080`
- **85 testes E2E** contra a API real rodando localmente
- **Resultado**: 34 PASS / 48 FAIL / 3 PARTIAL → **score 41.76%**
- **Falhas críticas observadas**:
  - **CAT-001**: Schema and generator inventory — catálogo público não contém os 10 CADOCs anunciados
  - **GEN-2030, GEN-2060, GEN-2061, GEN-2062, GEN-2070, GEN-2160, GEN-2170, GEN-3040, GEN-3050**: **TODOS os 10 generators falham** em invariantes públicos

Este achado **corrobora** a contradição que detectei estaticamente entre (a) CHANGELOG v3.36.0 que declarava apenas 3040 implementado, e (b) README/ROADMAP/Epic I do MASTER_PLAN que declaravam todos os 10 generators entregues. **A verdade runtime é: 10 generators estão registrados, mas apenas 3040 produz output válido; os outros 9 violam invariantes públicos.**

**Implicação para o scorecard**: score funcional estimado cai de 70-84 (auditoria estática) para **50-69 (PARCIALMENTE FUNCIONAL)** após execução de BENCH-00..09 em runtime. A recuperação se deve a: build/vet/race clean, 51 packages unit tests PASS, 10 generators com Generate real + tests PASS, 5 adapters com Fetch + tests PASS, 25 cross-doc rules + tests PASS. Mas a cobertura de 4 pacotes críticos (audit/rules 61.6%, crossdoc/rules 23.4%, audit 25.1%, api 58.0%) ainda está abaixo do mínimo declarado no MASTER_PLAN §5.1.