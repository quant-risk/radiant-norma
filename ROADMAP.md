# Roadmap — Radiant Norma

> **Plano detalhado:** ver [MASTER_PLAN.md](../MASTER_PLAN.md) (~85 KB, 11 seções + 5 ADRs).
> **Este arquivo:** visão macro executiva, atualizada por quarter.

---

## Q3 2026 — "FECHAR O CORE" (Sprints 28-37)

**Tema:** SCD-viável + smoke BACEN real.

| Sprint | Codinome | Entregas | Status |
|---|---|---|---|
| **28** | VaultIntegration | AWS Secrets Manager / Vault para rotação Sisbacen | ✅ |
| **29** | BacenHomologSmoke | Smoke real contra sta-h.bcb.gov.br/staws + www9.bcb.gov.br/senhaws | pendente (credenciais) |
| **30** | PostgresRLS | Ativar migration `012_rls_policies.sql` (em `internal/db/migrations/`) +
                    criar migration `014_rls_enforce.sql` com FORCE ROW LEVEL SECURITY.
                    Defense-in-depth multi-tenant. Auditoria SOC 2. | ✅ |
| **31** | RangeUploadAPI | Handlers REST `/v1/sta/range-*` — fechar YAGNI da Sprint 21 | pendente |
| **32** | Audit3040_v2 | Portar 80+ regras restantes 3040. Coverage 16% → 60% | ✅ (76% — 7 fases: 60→74→79→98→126→177→221→266; FECHADO) |
| **33** | Audit3050 | Portar 170 regras 3050 TXB_V11. XSD já tem no BACEN, parser XML + 170 regras | ✅ fechado (100% — 6 fases: 28→56→80→97→153→170; carry-over permanente 5 stubs DB) |
| **36** | Audit3040 Fase 2 | +51 regras 3040 (C21-C30, C41-C50, C56-C70, H04-H09, N01-N10) | ✅ fechado (49% — 22 reais + 3 híbridas + 29 stubs honestos I) |
| **37** | Audit3040 Fase 3 | +44 regras 3040 + 5 destravadas (I06-I15, A16-A30, S71-S90) | ✅ fechado (61% — 36 reais + 6 híbridas + 7 stubs) |
| **38** | Audit3040 Fase 4 (ÚLTIMA) | +45 regras 3040 + 9 destravadas (C71-C90, SUB01-15, X01-10) | ✅ fechado (76% — 26 reais + 28 stubs; carry-over permanente ~50 regras cross-doc) |
| **34** | FrontendNext | Migrar Console para Next.js 15 App Router + RSC + Server Actions | pendente |
| **35** | CI-Gate | GitHub Actions com pre-commit hook + go test -race + coverage gate + lint | ✅ |
| **36** | Observability | OpenTelemetry tracing + Sentry + Better Stack | pendente |
| **37** | Pilot | Onboarding real 1 SCD-piloto (30-90 dias) | pendente |

**Saída:** "Radiant Norma Lite" vendável pra SCD pequena.

---

## Q4 2026 — "MULTI-CADOC" (Sprints 38-48)

**Tema:** DLO + DDR + DRL + DLP + DRSAC research.

| Sprint | Entregas |
|---|---|
| **38** | AuditDLO — 200+ regras 2061 (Limites Operacionais) |
| **39** | AuditDDR — 11+ regras 2070 (Requerimento Capital Diário) | ✅ Fase 1+2 fechada (100% Fase 1 — 11 regras; Fase 2 — parser DRM/DLO + 7 cross-doc) |
| **40** | AuditDRL — 2160 LCR modelos II | ✅ fechada (100% catálogo LCR básico — 8 regras) |
| **41** | AuditDLP — 2170 NSFR | ✅ fechada (100% — 8 regras NSFR) |
| **42** | Audit3044 — Engine JSON eventos | ✅ fechada (17 regras T01-T19 — 15 reais + 2 carry-over) |
| **43** | CrossDoc_v2 — 5+ regras cross-doc | ✅ fechada (8 regras XD01-XD08) |
| **44** | Radar_v2 — Diff semântico + auto-PR | ✅ fechada (v3.34.25 — diff XLSX + GitHub Auto-PR) |
| **45** | StripeBilling — Lite/Pro/Scale/Enterprise + self-service | ✅ fechada (v3.34.26 — Stripe billing + webhooks) |
| **46** | WhiteLabel — Tema customizável pra Fintech BaaS | ✅ fechada (v3.34.27 — branding por tenant + 4 endpoints API) |
| **47** | DRSACResearch — Solicitação formal BACEN | ✅ fechada (v3.34.28 — parser DRSAC 2030 + 20 anexos) |
| **48** | Pilot2 — Segundo cliente (IP médio) | ✅ fechada (v3.34.29 — tenant lifecycle service) |

**Saída:** "Radiant Norma Pro" vendável pra IP média.

---

## Q1 2027 — "ESG FIRST-MOVER" (Sprints 49-56)

**Tema:** DRSAC + 4111 + cross-doc completo.

| Sprint | Entregas |
|---|---|
| **49** | DRSAC_v1 — Catálogo + XSD + 50+ regras iniciais | ✅ fechada (v3.34.30 — 35 regras D01-D35) |
| **50** | DRSAC_v2 — Regras IPOC × setor × risco climático | ✅ fechada (v3.34.31 — 8 regras cross-doc DRSAC↔SCR) |
| **51** | Audit4111 — 30+ regras iniciais | ✅ fechada (v3.34.32 — parser genérico 4111) |
| **52** | CrossDoc_DRSAC — 3040 ↔ DRSAC ↔ 4111 |
| **53** | AIInsights_v1 — LLM interpreta audit_log (opt-in) |
| **54** | SchemaRegistry_v2 — Versionamento automático + changelog público |
| **55** | Pilot3 — Cliente-piloto ESG-first | ✅ fechada (v3.34.50 — pilot service + ESG steps + REST API) |
| **56** | SOC2_Type1 — Auditoria SOC 2 Type I | ✅ fechada (v3.34.38 — readiness + evidence collector) |

**Saída:** "Radiant Norma ESG" vendável. Diferencial competitivo massivo.

---

## Q2 2027 — "PLATAFORMA + NORMA GENERATOR" (Sprints 57-67)

**Tema:** Motor de Geração de CADOCs + escala + marketplace + SDK.

| Sprint | Entregas |
|---|---|
| **57** | **NormaGeneratorFoundation** 🚨 | 📋 Decisão tomada — motor de geração CADOCs (5 conectores, 10 generators, wizard UX) — implementação backlog |
| **58** | AuditDLI | ✅ fechada (v3.34.40 — parser Documento 2062 DLI + validações estruturais) |
| **59** | SDK_GO — github.com/fortvna/radiant-norma-go | ✅ fechada (v3.34.41 — Go SDK) |
| **60** | SDK_Python | ✅ fechada (v3.34.42 — PyPI radiant-norma) |
| **61** | Webhooks outbound | ✅ fechada (v3.34.43 — registry + delivery worker + REST API) |
| **62** | Marketplace — Catálogo de regras customizadas | ✅ fechada (v3.34.44 — publish/install/rate) |
| **63** | MultiRegion — Replicação BR-SP1/SP2 | ✅ fechada (v3.34.45 — BR-SP1/SP2 replication) |
| **64** | Pilot4 — Banco S3-S4 | ✅ fechada (v3.34.46 — Banco S3-S4 onboarding) |
| **65** | SOC2_Type2 | ✅ fechada (v3.34.47 — continuous evidence collection) |
| **66** | SeriesA_Raise (opcional) | backlog (depends on M4: R$ 100k MRR) |

**Saída:** "Radiant Norma Enterprise" — plataforma regulatória end-to-end.

---

## Backlog Tooling — "DEV EXPERIENCE" (sprints opcionais entre features)

**Nota:** estas sprints são **nice-to-have** de tooling/dev-experience. Não bloqueiam milestones. Rodar quando sobrar ciclo entre features.

| Sprint | Codinome | Entregas | Status |
|---|---|---|---|
| **34-T** | **AuditForge POC** ([Autodata](https://arxiv.org/abs/2606.25996) sintético) | ✅ fechada (v3.34.51 — `cmd/synth-gen` CLI + `internal/synth/` package: Challenger LLM + Weak Solver + Judge LLM) |

### Por que Autodata-style para radiant-norma?

[Paper FAIR/Meta jun 2026](https://arxiv.org/abs/2606.25996) demonstra que loop iterativo (Challenger + Weak + Strong + Judge) gera dados sintéticos que **separam modelos fracos/fortes** — transferindo para modelo treinado outperform baseline 397B em PRBench-Legal. Análogo aqui:
- **Grounding:** XSD BACEN + tabelas ClassOp × Vencimento × Provisão (grounding context).
- **Weak solver:** `audit.Service` determinístico (regra Go).
- **Strong solver:** `audit.Service` + audit_log entry (chain-verify).
- **Challenger:** LLM gera XML de envio variando features (valor, ClassOp, vencimento, etc.).
- **Judge:** LLM avalia realismo + cobertura de edge cases.

Output: `cmd/synth-gen` CLI que produz envios sintéticos para `cmd/seed` e testes de regressão de regras.

### Quando rodar?

- **Sprint 33+ (Audit3050):** Regras 3050 (170 TXB_V11) precisam de **milhares de envios sintéticos** para validar combinações edge. POC faz isso.
- **Q4 2026 (Sprints 38-41):** AuditDLO/DDR/DRL/DLP — regras críticas, dataset sintético vira ativo de validação contínua.
- **Não bloqueia:** se ficar backlog, regras podem ser testadas manualmente com seed fixo (POC de tooling vira nice-to-have).

---

## Milestones

| Marco | Data | Métrica de sucesso | Kill switch |
|---|---|---|---|
| **M1 — Piloto pagante** | 2026-09-30 | 1 SCD pagou R$ 1,5k/mês por 3 meses | Cancelamento = pivotar GTM |
| **M2 — 10 clientes** | 2026-12-31 | 10 IFs ativas, NPS ≥ 40, churn < 5%/mês | < 5 = pivotar pricing |
| **M3 — ESG vendido** | 2027-03-31 | 1 cliente DRSAC ativo | 0 = DRSAC ainda vapor |
| **M4 — Series A ready** | 2027-06-30 | R$ 100k MRR, NRR > 110%, payback < 12 meses | < R$ 50k = bootstrap |

---

## Como contribuir

1. Leia [MASTER_PLAN.md](../MASTER_PLAN.md) §3 (Épicos) e §7 (Harness).
2. Pegue 1 issue rotulada `sprint-XX` no GitHub.
3. Siga o ciclo: RESEARCH.md → implementação → testes → RESULTS.md → validação profunda → CHANGELOG → tag.
4. PR review por ≥ 1 pessoa (≥ 2 para arquivos críticos: `auth/`, `auditlog/`, `sta/`).

---

**Última atualização:** 2026-07-08 (v3.35.6 — Pivô estratégico: decisão de arquitetura 🚨) · Sprint 57: decisão tomada, implementação backlog