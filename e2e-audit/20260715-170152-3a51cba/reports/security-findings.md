# Security Findings — Radiant Norma

> **Run:** 20260715-170152-3a51cba
> **Data:** 2026-07-15

Achados de segurança baseados em (1) CHANGELOG (defeitos já corrigidos), (2) inspeção estática do código, (3) ADRs, (4) MASTER_PLAN §5.3.

---

## P0 — Corrigidos em versões recentes

### DEF-SEC-001 (P0, CORRIGIDO v3.36.4) — Data race em staRangeUpload
- **Severidade:** P0 — integridade
- **Local:** `backend/internal/api/sta_range_handlers.go` (atualmente dirty)
- **Descrição:** `Session.ReceivedBytes/Ranges/Status` lidos FORA de `sessionsMu.Lock` após PUTs concorrentes.
- **Fix v3.36.4:** snapshot completo da struct sob lock; `writeJSON` e `go s.persistSession()` usam snapshot.
- **Impacto se recorrente:** corrupção de estado em sessão STA, respostas incorretas, panic intermitente.
- **Recomendação:** rodar `go test -race ./internal/api/` em CI.

### DEF-SEC-002 (P0, CORRIGIDO v3.36.4) — Information disclosure via err.Error()
- **Severidade:** P0 — confidentiality
- **Local:** `backend/internal/api/sta_range_handlers.go` (atualmente dirty)
- **Descrição:** `staRangeInit` retornava `err.Error()` no body de resposta, expondo URL/hostname/status do BACEN.
- **Fix v3.36.4:** log detalhado server-side + `s.userError()` retorna mensagem genérica.
- **Impacto se recorrente:** atacante aprende topologia interna do STA client.
- **Recomendação:** padronizar `s.userError()` em todos os handlers e proibir `err.Error()` em responses via linter custom.

### DEF-SEC-003 (P0, CORRIGIDO v3.34.52) — Pilot endpoints auth bypass
- **Severidade:** P0 — confidentiality + integrity
- **Local:** `backend/internal/pilot/`
- **Descrição:** endpoints de pilot não exigiam auth, permitindo acesso anônimo.
- **Fix v3.34.52:** auth obrigatória em todos os endpoints pilot.
- **Impacto se recorrente:** qualquer um pode criar/ler/modificar tenants piloto.
- **Recomendação:** adicionar teste de regressão que falhe se pilot endpoint responder 200 sem auth.

---

## P0 — Em aberto (não corrigidos no HEAD)

### DEF-SEC-004 (P0, ABERTO) — Cobertura `audit/rules = 61.6%` < mínimo 85%
- **Severidade:** P0 — defense-in-depth
- **Medido:** 61.6% (master/branch 3a51cba)
- **Mínimo declarado:** 85% (MASTER_PLAN §5.1)
- **Gap:** -23.4 pp
- **Impacto:** mutações em regras críticas podem passar sem detecção; cobertura insuficiente para o claim de "production-grade".
- **Hard gate disparado:** CI deveria estar falhando. Verificar se gate de cobertura está configurado.
- **Recomendação:** escrever testes para as 23.4 pp faltantes ou aceitar formalmente a redução do mínimo.

### DEF-SEC-005 (P0, ABERTO) — Cobertura `crossdoc/rules = 23.4%` < mínimo 70%
- **Severidade:** P0 — integridade cross-CADOC
- **Medido:** 23.4%
- **Mínimo declarado:** 70%
- **Gap:** -46.6 pp
- **Impacto:** regras cross-doc (XD01-XD12 + DRSAC + 4111) podem falhar silenciosamente; risco regulatório elevado (cross-doc é o **moat proprietário**).
- **Hard gate disparado:** CI deveria estar falhando.
- **Recomendação:** escrever testes para todas as 25 regras cross-doc com fixtures mínimas por CADOC.

---

## P2 — Em aberto

### DEF-SEC-006 (P2, ABERTO) — `MCP_<NAME>_ENDPOINT` env var não documentada
- **Local:** `backend/internal/ingest/adapter.go:1042`
- **Descrição:** MCPAdapter falha com mensagem "configure a variável MCP_<NAME>_ENDPOINT" sem que essa env var esteja documentada em README, ADR ou LLM Integration Guide.
- **Impacto:** setup operacional confuso; risco de configuração errada em produção.
- **Recomendação:** documentar em README e ADR-0008; ou implementar registry de servidores MCP ao invés de env var.

### DEF-SEC-007 (P2, ABERTO) — OpenAPI mistura prefixo `/v1`
- **Local:** `docs/openapi/v1.yaml` (paths `/marketplace*` e `/webhooks*`)
- **Descrição:** alguns paths têm `/v1`, outros não. Clientes SDK podem ter comportamento divergente.
- **Impacto:** inconsistência de contrato.
- **Recomendação:** regenerar spec a partir dos handler signatures (Sprint 77 já tentou isso — verificar o output).

### DEF-SEC-008 (P2, ABERTO) — `ErrNotImplemented` declarado mas não retornado
- **Local:** `backend/internal/ingest/adapter.go:45`
- **Descrição:** declaração existe mas nenhum dos 5 adapters retorna esse erro (LLM Integration Guide estava desatualizado).
- **Impacto:** engana leitores; código morto.
- **Recomendação:** remover ou usar como sentinel real para adapters faltantes.

---

## Validações que **NÃO foram executadas** (limitação desta auditoria)

Por bloqueio de ambiente (iCloud I/O + disco cheio + limite de subagente), **as seguintes validações de segurança NÃO foram executadas** e devem ser feitas em uma próxima janela:

1. **Cross-tenant isolation** com 2+ tenants simultâneos (BENCH-08)
2. **JWT signature/alg/kid inválidos** (BENCH-08)
3. **Cookie forjado / truncado** (BENCH-08)
4. **CSRF** em cookie auth (BENCH-08)
5. **IDOR** em alert/envio/webhook/delivery/wizard/pilot/branding/audit/history (BENCH-08)
6. **Postgres RLS** real com 2+ conexões (BENCH-09)
7. **SQL injection** em queries dinâmicas (BENCH-08)
8. **XXE / entity expansion / nesting** no parser XML (BENCH-08)
9. **rate limit serial e paralelo** (BENCH-08)
10. **webhook signature replay / timing-safe compare** (BENCH-08)
11. **varredura de segredos** em arquivos versionados (BENCH-08)
12. **govulncheck** (sem execução)
13. **gosec** (sem execução)
14. **testssl.sh** Mozilla Observatory A+ (BENCH-09)
15. **SSRF** controlado contra ranges privados / metadata (BENCH-08)
16. **path traversal / symlink** no file adapter (BENCH-08)
17. **Pen test anual** (não aplicável neste ambiente)
18. **MFA setup** no onboarding wizard (não exercitado)
19. **Rotação senhaws** com Vault/AWS SM (BENCH-11)
20. **Webhook com fila cheia / crash / restart** (BENCH-08)

---

## Score de segurança (parcial)

Não posso atribuir score final porque **não rodei BENCH-08 nem BENCH-11**. Mas posso afirmar:

- ✅ Arquitetura de segurança está bem desenhada (Postgres RLS, JWT, hash chain, CSRF, rate limit, defense-in-depth).
- ✅ 3 P0s recentes já foram corrigidos e entregues em produção.
- ✅ ADR-0002 (RLS) e ADR-0003 (audit chain) têm migrations reais.
- ❌ 2 P0s abertos (cobertura abaixo do mínimo declarado).
- ❌ Vários controles anunciados não foram exercitados nesta auditoria.

**Recomendação:** antes de qualquer exposição a clientes pagantes, (a) auditar BENCH-08 e BENCH-11 com profundidade; (b) aumentar cobertura de testes; (c) configurar CI gates de cobertura com `block-merge-if-below`.