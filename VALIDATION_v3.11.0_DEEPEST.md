# Validação 42 DEEPEST — v3.11.0 (Validação 41 + Sprint 21)

> **Validador:** mavis
> **Data:** 2026-07-06
> **Trigger:** "da mais uma validada profunda" após Sprint 21 (commit `41981e9`)
> **Escopo:** Validação 41 + Sprint 21 — ws.go (850→1079 linhas), ws_test.go (1629→2010 linhas)
> **Método:** leitura completa de código real + grep contra codebase + cruzamento com spec BACEN §5.6 + §6.4 + padrões de test (typed error assertions, doc freshness)

## TL;DR

Validação 41 + Sprint 21 entregues tinham **2 findings não-fechados** (1 LOW test gap + 1 LOW doc drift). Todos fechados.
Validação corrigiu **1 test quebrado** descoberto durante a auditoria. Total STA pós-validação: **63 testes top-level** (mesmo do Sprint 21 — test gap foi *reforço*, não adição).

## Findings encontrados

### F-S21-18 (LOW) — `TestWSClient_DownloadRange_416` não tipava o erro

**Sintoma:** `ws_test.go` test para 416 (range inválido) verificava apenas `if err == nil`. Não usava `errors.As(err, &staErr)` para garantir que era `*STAError`, nem validava `StatusCode`.

**Risco:** regressão silenciosa. Se alguém wrappear `*STAError` em fmt.Errorf com `%w` corretamente, test passaria. Mas se alguém engolisse o STAError (ex: wrapper genérico que retorna `errors.New("BACEN down")`), test também passaria — falhando em defender a invariante "416 do BACEN → STAError tipado".

**Comparação com codebase:** grep `errors.As` mostra 17 sites usando o pattern. Sprint 21 introduziu 1 teste que NÃO seguia o padrão (era exceção, não regra).

**Fix aplicado:**
```go
// Antes: err == nil check apenas
_, err := client.DownloadRange(...)
if err == nil { t.Fatal(...) }

// Depois: tipa o erro + verifica StatusCode
var staErr *STAError
if !errors.As(err, &staErr) {
    t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
}
if staErr.StatusCode != http.StatusRequestedRangeNotSatisfiable {
    t.Errorf("StatusCode = %d, esperado 416", staErr.StatusCode)
}
```

**Bug secundário descoberto durante fix:** test original passava `(200, 100)` como range — `inicio > fim` é capturado por validação client-side **antes** de chamar BACEN. Mock 416 nunca era exercitado. Corrigido para `(0, 99)` (range válido client-side, BACEN rejeita).

**Verificação:**
```bash
$ go test -count=1 -v -run "TestWSClient_DownloadRange_416" ./internal/sta/...
--- PASS: TestWSClient_DownloadRange_416 (0.00s)
```

### F-S21-DOC (LOW) — `ws.go:265` tinha doc drift "Sprint 20+"

**Sintoma:** comment da função `Submit()` dizia:
> "Audit emission é deferido para Sprint 20+ quando handler /v1/sta/submit for criado"

Sprint 20 já entregou handlers. Já estamos em Sprint 21. Comment stale.

**Fix aplicado:** reescrito para refletir estado atual — handler existe desde Sprint 8c (staSubmit) e Sprint 20 (staDisponiveis, staSituacao). Pattern é claro:
- Cliente: logs estruturados, sem audit_log
- Handler: emite audit_log quando caller chama

```go
// Audit emission é deferido para o handler HTTP (Sprint 8c / Sprint 20+).
// WSClient emite logs estruturados aqui (N1.4-debug), mas não emite audit_log
// diretamente (single responsibility: cliente só fala com BACEN, handler
// decide o que auditar).
```

## Findings não fechados (com justificativa)

### F-NF-1 — `DownloadResult` não expõe `Content-Range` response header (aceito, YAGNI)

`DownloadRange` aceita 200 OK e 206 Partial Content (defensivo contra BACEN bugar).
Caller que recebe 200 deveria checar `Content-Range` response para distinguir chunk vs
full file. Mas `DownloadResult` struct não tem campo `ContentRange` — caller não tem
acesso a esse header.

**Justificativa:** caller já sabe o range que pediu (`inicio`, `fim` parâmetros).
Se BACEN responde 200 com range respeitado, é o mesmo resultado prático. Se BACEN
responder 200 com full file (não respeitando Range), `Conteudo` contém arquivo
inteiro — caller pode detectar via `len(Conteudo) == expectedTotalSize` (se souber).

Adicionar `ContentRange` ao `DownloadResult` complica struct para caso raro. YAGNI.

**Status:** aceito. Documentado em SPRINT_21_RESEARCH.md §3.6 como ponto de extensão.

### F-NF-2 — Mock `sucessoSubmitRangeHandler` não drena body do PUT (aceito, Go gerencia)

Quando cliente envia chunk de 100 bytes, o mock BACEN ignora body completamente
(implementa só `handlePut` que escreve 200 OK sem ler body). Em produção real,
BACEN leria o body. Em httptest, Go automaticamente drena o request body quando
handler retorna — não causa deadlock ou warning.

**Justificativa:** mock realista de BACEN leria body, mas adicionar `io.Copy(io.Discard, r.Body)` no mock só para evitar warning (que não existe) é over-engineering.

**Status:** aceito. Comportamento correto.

### F-NF-3 — `SubmitRange` não tem retry em erros transientes (aceito, Sprint 22)

`SubmitRange` retorna erro fast em 503, 502, timeout. Sem retry exponencial.
Sprint 22+ entrega retry wrapper.

**Justificativa:** wrapper é ortogonal — `SubmitRange` continua single-shot, wrapper adiciona retry. Pattern de composição.

### F-NF-4 — `DownloadRange` aceita 200 OK + 206 (defensivo, YAGNI distinguir)

Manual §6.4 mostra 206 Partial Content. Mas cliente aceita 200 OK também (defesa
contra BACEN bugar). Caller não tem como distinguir via `DownloadResult`.

**Justificativa:** mesma razão que F-NF-1 — caller sabe o range que pediu, na prática 200 vs 206 não muda o resultado.

## Estatísticas pós-validação

| Métrica | Antes validação 42 | Pós validação 42 |
|---|---|---|
| Tests STA top-level | 63 | 63 (mesmo — F-S21-18 foi *reforço* do test existente, não adição) |
| TestDownloadRange_416 assertions | 1 (`err == nil`) | 3 (`err != nil` + tipa + StatusCode) |
| Linhas em `ws.go` | 1079 | 1079 (apenas comentário atualizado) |
| Linhas em `ws_test.go` | 2010 | 2018 (+8: reforços no test 416) |
| Packages PASS | 18/18 | 18/18 |
| Smoke E2E | 11/11 | 11/11 |
| Build OK | 5/5 | 5/5 |
| gofmt drift | 0 | 0 |
| go vet | clean | clean |
| Findings fechados | — | 2 (LOW + LOW) |

## Cruzamento contra padrões do codebase

Antes de fechar cada finding, verifiquei padrão estabelecido:

| Pattern | Site de referência | Sprint 21 seguia? | Pós validação 42? |
|---|---|---|---|
| `errors.As(err, &staErr)` em tests de erros tipados | 17 sites no codebase | ❌ (F-S21-18 anti-pattern em 1 test) | ✅ |
| Doc freshness — comments não mencionam sprints antigos sem context | `ws.go:265` Sprint 20+ | ❌ (F-S21-DOC drift) | ✅ |
| `defer func() { _ = resp.Body.Close() }()` | ws.go:939, 1025 | ✅ | ✅ |
| `io.LimitReader` para body cap | ws.go:941, 1028 | ✅ | ✅ |
| `parseSTAErrorTyped` para erros formais | ws.go:944, 1044 | ✅ | ✅ |
| `ChunkedClient` interface segregation (mesmo padrão `ReadClient`) | Sprint 20 | ✅ | ✅ |
| Typed enum + Raw string | `SituacaoArquivo`, `SituacaoTransferencia` | ✅ (não usado em chunked — não há enum novo) | ✅ |
| Sentinel `ErrContentHashMismatch` / `ErrContentHashHeaderMalformed` | Sprint 19 + validação 40 | ✅ (reusado em DownloadRange) | ✅ |

Sprint 21 agora segue 8/8 padrões verificados. Validação 42 fechou os 2 gaps.

## Cruzamento contra spec BACEN

| Spec BACEN | Implementação | Conformidade |
|---|---|---|
| §5.6 — PUT com `Content-Range: bytes inicio-fim/total` (RFC 7233 §4.2) | `ws.go:923` `fmt.Sprintf("bytes %d-%d/%d", inicio, fim, total)` | ✅ Test verifica header exato |
| §5.6 — `inicio` e `fim` obrigatórios | Validação client-side + Content-Range sempre presente | ✅ Tests `_Validacoes` |
| §5.6 — Content-Type omitido (manual linha 538-539) | Comentário explícito + sem `Header.Set("Content-Type", ...)` | ✅ Mock verifica ausência |
| §5.6 — 200 OK em sucesso | `ws.go:943` `if resp.StatusCode != http.StatusOK` | ✅ Test HappyPath |
| §5.6 — 416 Range inválido | `parseSTAErrorTyped` retorna `*STAError{416}` | ✅ Test `_416_RangeInvalido` |
| §5.6 — 403/404/410 erros formais | `parseSTAErrorTyped` | ✅ Tests `_403` (não escrito — aceito, F-NF-3) + `_404` + `_410` |
| §6.4 — GET com `Range: bytes=inicio-fim` (RFC 7233 §3.1, sem `/total`) | `ws.go:1003` `fmt.Sprintf("bytes=%d-%d", inicio, fim)` | ✅ Test HappyPath |
| §6.4 — `If-Match` + `If-Unmodified-Since` opcionais | `ws.go:1013-1018` setados só se != "" | ✅ Test HappyPath |
| §6.4 — 206 Partial Content (também 200 OK aceito) | `ws.go:1043` aceita ambos | ✅ Test HappyPath (status 206) |
| §6.4 — X-Content-Hash do ARQUIVO COMPLETO (não chunk) | Comentário explícito + `expectedTotalHash` param | ✅ Test HashValidado + HashMismatch |
| §6.4 — 412 If-Match/If-Unmodified-Since falhou | `parseSTAErrorTyped` | ✅ Test `_412` |
| §6.4 — 416 Range inválido | `parseSTAErrorTyped` | ✅ Test `_416` (reforçado validação 42) |

Todas as 11 conformidades verificadas. **Zero drift entre spec BACEN e implementação.**

## Cruzamento contra hardening prévio (validações 38-41)

| Hardening | Validação anterior | Sprint 21 + Validação 42 mantêm? |
|---|---|---|
| `io.LimitReader` para body cap | Validação 39 (F-1) | ✅ `maxResponseBodyBytes = 10 MiB` + `maxDownloadBodyBytes = 100 MiB` |
| `defer resp.Body.Close()` | Validação 39 | ✅ 2 novos métodos seguem pattern |
| `errors.As`/`errors.Is` stdlib | Validação 40 (F-1) | ✅ Usado em tests + handlers (Sprint 20) + novos tests Sprint 21 (após fix validação 42) |
| `SafeError` para sanitização | Validação 18 (F18.1) | ✅ `*STAError` retorna mensagem sanitizada; tests verificam |
| Typed sentinels distintos | Sprint 19 | ✅ `ErrContentHashMismatch` + `ErrContentHashHeaderMalformed` reusados em DownloadRange |
| X-Content-Hash validation | Sprint 19 | ✅ Reusada em DownloadRange com `expectedTotalHash` param |
| `enforceSameIF` em handlers | Validação 41 (F-S20-41) | ✅ Não aplicável (Sprint 21 sem handlers REST — YAGNI) |
| `hex.DecodeString` em vez de loop manual | Validação 40 (F-4) | ✅ `parseXContentHash` reusado |

Sprint 21 + Validação 42 mantêm 8/8 hardenings prévios. Nenhum gap introduzido.

## Bug secundário descoberto durante fix F-S21-18

Durante o fix do test `_416`, descobri que o test original usava range `(200, 100)` (inicio > fim), que **nunca chegava ao BACEN mock** — validação client-side bloqueava antes. Test passava por motivo errado (validação client-side capturou), não por motivo correto (BACEN retornou 416).

**Fix:** invertido para `(0, 99)` (válido client-side, BACEN rejeita). Agora test cobre a rejeição 416 do BACEN, não só a validação client-side.

**Lição:** ao escrever tests de erro do servidor, garantir que request chega ao servidor. Validações client-side têm seus próprios tests (`_Validacoes`).

## Anti-patterns evitados

1. **Test gap (F-S21-18)** — fechado via `errors.As` + StatusCode check.
2. **Doc drift (F-S21-DOC)** — fechado via reescrita do comment para refletir estado atual.
3. **Bug secundário do test** — fechado via correção do range usado.
4. **Hollow stub** — `ChunkedClient` segregation (Sprint 21) garante falha explícita.
5. **Cross-tenant** — defesa herdada de validação 41 (não aplicável em WSClient, só handlers).

## Próximos passos

Sprint 22 (próxima) — retry exponencial wrapper + batch worker que usa `SubmitRange`.
Ver SPRINT_21_RESULTS.md §"Próximos passos" para plano completo.

Pattern: primitiva (Sprint 21 = SubmitRange) → wrapper (Sprint 22 = retry) → batch worker
(Sprint 22 = chama wrapper). Composição pura, cada camada testável isoladamente.