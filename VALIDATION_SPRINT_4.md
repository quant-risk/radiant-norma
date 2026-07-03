# Validação Profunda — Sprint 4 (v1.3.0 → v1.3.1)

> **Data:** 2026-07-03
> **Escopo:** Code review completo de tudo entregue na Sprint 4 (Honesty Patch + 25 regras 3040 + Radar Regulatório)
> **Resultado:** 17 bugs identificados, **17 corrigidos**, validação E2E passa 11/11

## Resumo executivo

A validação profunda releu **14 arquivos Go + 2 SQL migrations + cmd/_verify util**, identificou **17 bugs** distribuídos entre crítico (10), médio (4) e baixo (3), e corrigiu todos. v1.3.1 é o resultado da validação.

A v1.3.0 foi marcada como "funcional" após smoke tests básicos (validate XML válido/quebrado, STA submit). A validação profunda revelou:

- **Race conditions** que podiam quebrar o audit log em produção concorrente
- **Lógica de regra errada** (S05 checava modalidade 0213 quando deveria ser 19)
- **Verify() mentiroso** que não detectava tampering real (só checava chain, não entry hash)
- **L1 não abortava L2** gerando 13+ erros redundantes quando XML quebrava
- **Worker sem claim atômico** que duplicaria envios com 2 instâncias
- **Healthz retornava version "1.2.0"** — mentira sobre a versão real

## Bugs encontrados e fixes

### 🔴 CRÍTICOS (LGPD/SOC 2, comportamento errado)

#### 1. `auditlog.Verify()` não detectava tampering real
- **Arquivo:** `internal/auditlog/log.go`
- **Sintoma:** Verify só checava encadeamento de PrevHash. Se alguém modificasse `actor`, `target`, ou `metadata`, Verify passava porque só verificava `e.PrevHash == prevHash`.
- **Fix:** Verify agora **recomputa** EntryHash a partir de todos os campos (`prev + payload + metadata + actor + action + target + ifID + timestamp`). Se qualquer campo foi modificado, hash não bate.
- **Teste:** Modify `actor` → Verify retorna `entry hash inválido em id=2 ... entry foi modificada`

#### 2. `auditlog.Log()` race condition em inserções concorrentes
- **Arquivo:** `internal/auditlog/log.go`
- **Sintoma:** `SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1` + INSERT eram não-atômicos. Dois goroutines simultâneos podiam pegar o mesmo prev_hash e gerar duas entries com mesmo PrevHash — chain quebrada.
- **Fix:** `BeginTx()` com `tx.QueryRow` + `tx.Exec` (serializado no SQLite via BEGIN IMMEDIATE).
- **Impacto:** LGPD/SOC 2 — sem fix, audit log era unreliable em produção concorrente.

#### 3. `auditlog.Log()` e `Verify()` usavam timestamps diferentes
- **Arquivo:** `internal/auditlog/log.go`
- **Sintoma (bug adicional durante o fix #1):** Log usava `time.Now()` no Go; Verify lia `created_at` do DB (que era `CURRENT_TIMESTAMP` do SQLite). Microssegundos diferentes → Verify falhava sempre, mesmo em chain íntegra.
- **Fix:** Log passa timestamp explícito no INSERT (`created_at = ?` com `timestampStr`); Verify formata o `created_at` lido do DB com mesmo formato.
- **Teste:** 3 entries normais → Verify OK; modify `actor` → Verify FAIL.

#### 4. `Validate()` não abortava L2 quando L1 falhava
- **Arquivo:** `internal/audit/service.go`
- **Sintoma:** XML completamente quebrado (`<broken/>`) gerava **L1-PARSE + 13 erros "parser 3040 falhou"** (uma por cada regra do registry). Ruído absurdo.
- **Fix:** `return resp, nil` logo após L1-PARSE (não tenta L2).
- **Teste:** XML quebrado → 1 erro (L1-PARSE), era 13+.

#### 5. `db.Migrate()` sem lock — race entre múltiplas instâncias
- **Arquivo:** `internal/db/migrate.go`
- **Sintoma:** API e worker rodando simultaneamente podiam ambos aplicar a mesma migration (`check + apply + mark` não era atômico). Corrompia o schema.
- **Fix:** `BeginTx()` com check+apply+mark dentro. Re-check dentro da tx impede race.
- **Impacto:** Multi-binário deployment sem erro de "migration already applied".

#### 6. `cmd/worker` e `cmd/radar` não chamavam `db.Migrate`
- **Arquivos:** `cmd/worker/main.go`, `cmd/radar/main.go`
- **Sintoma:** Se rodados standalone antes da API ter criado o schema, falhavam com "no such table".
- **Fix:** Ambos agora chamam `db.Migrate(d)` no boot.

#### 7. Worker race condition (dois workers pegam mesmo envio)
- **Arquivo:** `cmd/worker/main.go`
- **Sintoma:** `SELECT ... WHERE status='pending' LIMIT N` + UPDATE não é atômico. Dois workers simultâneos processariam o mesmo envio 2x (submissão duplicada ao BACEN!).
- **Fix:** Claim atômico via `UPDATE envios SET status='processing' WHERE id = (SELECT ... LIMIT 1) RETURNING ...`. Loop por envio único.
- **Impacto:** Em produção com múltiplos workers, sem fix teríamos envios duplicados na STA.

#### 8. `S05LimiteCredito` implementava regra errada
- **Arquivo:** `internal/audit/rules/3040.go`
- **Sintoma:** Implementação verificava `Mod="0213"` (cheque especial). Catálogo original BACEN diz: "A modalidade 'Limite de Crédito' (19)...". Regra inteira errada — código validava a regra errada!
- **Fix:** Agora valida `Mod="19"`, e que v150/v160/v165 são zero (v110/v120 são permitidos = limite).
- **Teste:** Mod=19 com v150=500 → S05 detecta. Mod=0213 → S05 não detecta (correto, agora é regra BACEN).

#### 9. `F02Datas` regex pré-compilado a cada chamada (perf)
- **Arquivo:** `internal/audit/rules/3040.go`
- **Sintoma:** `regexp.MustCompile(...)` dentro de `Apply` rodava 25x por validate (uma vez por regra que invoca F02).
- **Fix:** `var datePattern = regexp.MustCompile(...)` package-level.

#### 10. `getRadarAlert` lookup O(N)
- **Arquivo:** `internal/api/server.go` + `internal/radar/radar.go`
- **Sintoma:** Handler chamava `ListAlerts(ctx, false, 1000)` e iterava linearmente. Com 1000 alertas = 1000 reads.
- **Fix:** Novo `Service.GetAlertByID(ctx, id)` com query SQL direta (`WHERE id = ?`).

### 🟡 MÉDIOS

#### 11. `Healthz` retornava version "1.2.0"
- **Arquivo:** `internal/api/server.go`
- **Sintoma:** Version hardcoded não foi atualizada para v1.3.0 (e agora v1.3.1).
- **Fix:** Atualizado para `1.3.1`.

#### 12. `staSubmit` salvava `zip_content = body inteiro` (JSON, não XML)
- **Arquivo:** `internal/api/server.go`
- **Sintoma:** Quando body era JSON (`Content-Type: application/json`), o handler fazia `sub.Zip = body` — salvava o JSON inteiro no `zip_content`. DB poluído.
- **Fix:** `sub.Zip = []byte(sub.XML)` — usa XML puro (stub).

#### 13. `cmd/api/main.go` sem `ReadTimeout/WriteTimeout/IdleTimeout`
- **Arquivo:** `cmd/api/main.go`
- **Sintoma:** Slowloris attack vetor; clients mal-comportados podiam segurar conexões.
- **Fix:** `ReadTimeout: 30s`, `WriteTimeout: 60s`, `IdleTimeout: 120s`.

#### 14. `cmd/api/main.go` graceful shutdown com race
- **Arquivo:** `cmd/api/main.go`
- **Sintoma:** `os.Exit(1)` dentro do goroutine que serve. Se rodasse antes do `Shutdown`, matava abruptamente sem fechar conexões.
- **Fix:** Erros via channel `serverErr`, shutdown controlado via `select` entre `serverErr` e signal.

### 🟢 BAIXOS

#### 15. `Builtin3040()` chamava `slog.Info` em init
- **Arquivo:** `internal/audit/rules/registry.go`
- **Sintoma:** "rules registered count=25" log toda vez que `audit.New()` rodava (incluindo testes).
- **Fix:** Log movido pra fora do constructor.

#### 16. `cmd/seed/main.go` struct field `Tipo` duplicava tag JSON
- **Arquivo:** `cmd/seed/main.go`
- **Sintoma:** `Tipo string \`json:"tipo"\`` e `TipoIndicio string \`json:"tipo"\`` — mesma tag, conflito.
- **Fix:** `Tipo string \`json:"tipo_"\``.

#### 17. `sta.StubClient.Submit` protocolo com ano hardcoded
- **Arquivo:** `internal/sta/stub.go`
- **Sintoma:** `fmt.Sprintf("2026%02d%02d%05d%s", ...)` — ano fixo em "2026", em 2027 quebraria. Também `%05d` não comportava `Second()*1000 + millis%1000` (até 60000+).
- **Fix:** `fmt.Sprintf("%04d%02d%02d%05d%s", time.Now().Year(), ...)` + `int(time.Now().UnixMilli()%100000)`.

## Bônus: o que a validação NÃO encontrou (mas deveria ter sido feito antes)

- **Testes unitários**: zero `*_test.go`. Tudo validado via curl scripts. Próxima sprint.
- **BCValidador comparison**: não rodamos BCValidador real (precisa Docker). Continua pendente.
- **STA real (Playwright)**: ainda é stub. Pendente pra Sprint 6.
- **Dedup 3040 (14 UNIQUE warnings)**: ainda emite warnings. Pendente — `extract.py` precisa dedup.

## Validação E2E — 11/11 testes passando

```
✓ 1) Build limpo + vet limpo
✓ 2) Healthz version: 1.3.1
✓ 3) Validate L1 aborta (1 erro em vez de 13)
✓ 4) F02 severity=E (DtBase inválido)
✓ 5) S05 detecta Mod=19 com v150>0 (regra BACEN correta)
✓ 6) cadoc E cadoc_code aceitos
✓ 7) STA submit JSON + retrocompat + persistência XML puro
✓ 8) Audit log Verify detecta tampering
✓ 9) Worker claim atômico (3 envios → 3 accepted)
✓ 10) Radar idempotência (3 scans → 1 baseline)
✓ 11) Validate avg 0ms (cache ParseDoc3040)
```

## Decisões de arquitetura reforçadas

1. **Custom UnmarshalJSON > JSON tags conflitantes** — confirma decisão anterior; facilitou aceitar `cadoc` E `cadoc_code`.
2. **Severity da Rule > DB gravidade** — essencialmente fix #2 do auditlog provou que DB é unreliable sem fonte da verdade.
3. **Migrations com tracking** — fix #5 do migrate provou que sem tracking, multi-instância corrompe.
4. **Claim atômico em workers** — fix #7 provou que "SELECT then UPDATE" é anti-pattern.
5. **LGPD/SOC 2 = recompute hash** — fix #1+#3 do auditlog provou que só encadear PrevHash é teatro; precisa recomputar.

## Lições aprendidas (cross-project)

1. **Smoke tests não são validação**: v1.3.0 passou curl mas tinha 10 bugs críticos. Validação profunda releu código linha por linha.
2. **Implementação pode estar errada mesmo se compila e "testa OK"**: S05 modalidade 0213 vs 19 passou no smoke test porque meu XML de teste era Mod=0213 (que a regra errada "validava").
3. **Catálogo é fonte da verdade**: sempre comparar implementação com texto literal da regra. Vou automatizar isso na Sprint 5 com testes que puxam do JSON.
4. **Verify() deve recomputar**: hash chain só com PrevHash encadeado é falso seguro. Tem que recomputar EntryHash pra detectar tampering real.
5. **Race conditions precisam de BEGIN IMMEDIATE explícito**: SQLite tem BEGIN IMMEDIATE justamente pra isso. Não usar SELECT+INSERT sem lock.

---

**Autor:** Mavis · Radiant (validação profunda da Sprint 4)
**Versão:** v1.3.1
**Status:** ✅ Todos os 17 bugs corrigidos, 11/11 testes E2E passando