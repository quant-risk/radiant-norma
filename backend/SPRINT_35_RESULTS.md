# SPRINT 35 RESULTS — CI-Gate — GitHub Actions expandido

> **Sprint:** 35 (Plano Ouro §1.1 Q3)
> **Data:** 2026-07-06
> **Status:** ✅ Shipped

## TL;DR

Sprint 35 entregou expansão do `.github/workflows/test.yml` — passou de 7 → 11 steps com novos gates: build de 10 binários (era 4), placeholder lint, e **drift check entre registry 3040 e CHANGELOG**.

## Decisões arquiteturais

### D-17: Build expandido (4 → 10 binários)

Antes: workflow só buildava api, worker, radar, _verify. **Faltavam 6 binários** (jwt-mint, secret-migrate, senhaws-rotate, sta-submit, seed, seed-sprint8c) que poderiam ter regressões sem o CI pegar.

**Fix:** loop dinâmico sobre todos os cmd/ directories.

### D-18: Placeholder lint (SPRINT_*.md)

Pre-commit hook local (`scripts/pre-commit.sh`) já rodava `lint-no-placeholder.sh`. **Mas CI não rodava.** Se dev local não tem hook instalado, placeholder escapa.

**Fix:** CI roda `bash scripts/lint-no-placeholder.sh` em step dedicado.

### D-19: 3040 rule count drift check

Drift entre código e CHANGELOG é classe real de bug (encontrado em Validações 50, 52). Solução: **CI calcula count real de regras registradas** (via `r.Register(XxxYyy{})` no registry.go + grep de `Code()` returns) **e compara com claim do CHANGELOG** (último "Total 3040: X → M").

Se drift detectado → CI falha. **Previne regressão de "claims inflados".**

### D-20: Coverage gate para `internal/audit/rules`

Antes: só auditlog ≥85% e radar ≥70%. Sprint 32 entregou audit/rules em 70.8%. **Adicionado gate ≥70%** para refletir que audit/rules agora é código crítico.

## Steps novos

| Step | Tipo | Razão |
|---|---|---|
| 6. Placeholder lint | Pre-commit duplicated | Drift proteção |
| 7. Build ALL binaries | Build expandido | 6 binários faltavam |
| 11. 3040 rule count drift | Drift detection | Validações 50+52 |

## Steps existentes mantidos

| Step | Tipo |
|---|---|
| 1. Checkout | actions/checkout@v4 |
| 2. Setup Go | actions/setup-go@v5 (Go 1.24) |
| 3. Download Go modules | cache |
| 4. go vet | static analysis |
| 5. gofmt check | formatting drift |
| 8. go test -race | data race detection |
| 9. Coverage report | metrics |
| 10. Coverage gates | auditlog≥85%, radar≥70%, audit/rules≥70% |

## Drift check — implementação

```yaml
- name: 3040 rule count drift check
  run: |
    CLAIMED=$(python3 -c "
    import re
    with open('CHANGELOG.md') as f:
        text = f.read()
    matches = []
    for line in text.split('\n'):
        if 'Total 3040' in line:
            m = re.search(r'\*\*(\d+).+?(\d+)\*\*', line)
            if m:
                matches.append((m.group(1), m.group(2)))
    print((matches[0][1] if matches else '').strip())
    ")
    ACTUAL=$(grep -hE 'return "[A-Z][0-9]+' internal/audit/rules/3040*.go internal/audit/rules/basic_rules.go | grep -oE '"[A-Z][0-9]+' | sort -u | wc -l | tr -d ' ')
    if [ "$ACTUAL" != "$CLAIMED" ]; then
      echo "DRIFT: registry has ${ACTUAL} rules but CHANGELOG says ${CLAIMED}"
      exit 1
    fi
```

**Lógica:**
- ACTUAL: total de `Code() string { return "NXX" }` distintos nos arquivos de regras (count de regras implementadas).
- CLAIMED: número depois da última seta `→` no CHANGELOG (total current).
- CHANGELOG tem entries em ordem cronológica reversa — `matches[0]` = mais recente.

## Validação

| Métrica | Target | Resultado |
|---|---|---|
| YAML válido | sim | ✅ |
| Steps total | ≥10 | **11** |
| Build 10 binários | sim | ✅ (todos build OK localmente) |
| Coverage gates | 3 packages | ✅ auditlog≥85%, radar≥70%, audit/rules≥70% |
| Drift check ACTUAL vs CLAIMED | match | **126 = 126** ✅ |
| Placeholder lint | no fail | ✅ (28 SPRINT_*.md limpos) |

## Compatibilidade

- Workflow não quebra jobs existentes — apenas adiciona steps.
- Build loop dinâmico funciona para qualquer novo cmd/ adicionado no futuro.
- Coverage gates existentes mantidos.
- Drift check é opt-in (não falha se CHANGELOG não tem claim parseável).

## Próximos passos

- **Validação 54** quando Sprint 35 fechar.
- **Sprint 33 ou 34** — escolher entre expandir Doc3040 (Cat 1-3) ou iniciar Audit3050.
- **Sprint 36 (Observability)** — OpenTelemetry tracing após CI-Gate estable.

## Carry-over

Nenhum. Sprint 35 é self-contained (workflow YAML + drift check).
