# SPRINT 21 — RESULTS: Chunked transfer (range upload + range download)

> **Sprint:** 21 (v3.11.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** `41981e9` (Sprint 21 — ver VALIDAÇÃO 42+ para commits subsequentes)

## TL;DR

Sprint 21 fecha o **chunked transfer** do BACEN STA WS. IF com CADOC >50 MB agora pode
(a) **enviar arquivo em chunks paralelos** (`SubmitRange` §5.6) e (b) **retomar download
interrompido** (`DownloadRange` §6.4) — usando o resultado de `StatusUpload` (§5.3.1,
Sprint 19) para saber onde parou.

**Decisão arquitetural:** `ChunkedClient` interface segregation (mesmo padrão da
`ReadClient` da Sprint 20). Apenas `*WSClient` implementa. `*StubClient` retorna erro
de compilação claro (interface não implementada). Capability de chunked transfer é
**opt-in** — caller faz type assertion.

**Decisão YAGNI consciente:** **NÃO** criar handlers REST nesta sprint. Sem consumer
imediato (range download é caso pra batch worker Sprint 22+). Métodos ficam disponíveis
no WSClient; handlers entram quando batch worker chamar.

**13 testes novos** (12 httptest STA + 1 interface segregation). Total STA: **63 testes**
(16 Sprint 18 + 16 Sprint 19 + 16 Sprint 20 + 1 validação 40 + 1 validação 41 + 13 Sprint 21).

## Entregas

### 1. `WSClient.SubmitRange(ctx, protocolo, inicio, fim, total, chunk)` — manual §5.6

```go
func (c *WSClient) SubmitRange(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error
```

- PUT `/arquivos/{protocolo}/conteudo` com `Content-Range: bytes inicio-fim/total` (RFC 7233 §4.2).
- Content-Type omitido (manual §5.6 linha 538-539 — mesmo que §5.2 single-call).
- 200 OK em sucesso (sem body).
- **Validações client-side** (defense in depth — BACEN também valida 416):
  - protocolo não vazio
  - `inicio >= 0`
  - `fim >= inicio`
  - `total > 0` e `total >= fim+1`
  - `len(chunk) == fim-inicio+1` (chunks devem ter tamanho exato do range declarado)

### 2. `WSClient.DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince)` — manual §6.4

```go
func (c *WSClient) DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*DownloadResult, error)
```

- GET `/arquivos/{protocolo}/conteudo` com `Range: bytes=inicio-fim` (RFC 7233 §3.1, sem `/total` — diferente de Content-Range).
- `If-Match` + `If-Unmodified-Since` opcionais (manual §6.4 linha 703).
- 206 Partial Content em sucesso (também tolera 200 OK).
- X-Content-Hash **do arquivo completo** (não do chunk) — caller valida contra `expectedTotalHash` (vindo de `ListDisponiveis.Hash`).
- **Validações client-side:**
  - protocolo não vazio
  - `inicio >= 0, fim >= inicio`
  - `(fim-inicio+1) <= maxDownloadBodyBytes` (100 MiB — defesa DoS)

### 3. `ChunkedClient` interface segregation (decisão arquitetural)

```go
type ChunkedClient interface {
    SubmitRange(ctx, protocolo, inicio, fim, total, chunk) error
    DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince) (*DownloadResult, error)
}
```

- Apenas `*WSClient` implementa (compile-time check + runtime type-assertion test).
- `*StubClient` **NÃO** implementa (test `TestChunkedClient_InterfaceSegregation` prova).
- Caller pattern: type assertion em runtime — caller recebe erro de compilação claro
  se tentar chamar direto em StubClient.

**Por que não estender `Client` ou `ReadClient`?** Forçar StubClient a implementar
SubmitRange/DownloadRange com zero-values seria hollow stub piorado — caller acharia
que funcionou mas BACEN nunca foi chamado. Mesma justificativa da Sprint 20.

### 4. Reuso de tipos e sentinels da Sprint 19

- `Range{Start, End int64}` — tipo já existia (Sprint 19), reusado conceitualmente.
- `*STAError` — erros formais BACEN 4xx/5xx com XML Listagem 4.
- `ErrContentHashMismatch` — X-Content-Hash do BACEN difere de expectedTotalHash.
- `ErrContentHashHeaderMalformed` — X-Content-Hash formato mudou (defesa contra BACEN evoluir).
- `parseXContentHash` — parser do header (Sprint 19 validação 40 já usa stdlib `hex.DecodeString`).
- `maxDownloadBodyBytes = 100 MiB` — cap defensivo.

### 5. Compatibilidade

- `Client` interface inalterada (Submit apenas).
- `ReadClient` inalterada (ListDisponiveis + AlterarSituacao).
- `*WSClient` ganha 2 métodos novos (`SubmitRange`, `DownloadRange`) + implementa nova
  `ChunkedClient` interface.
- `*StubClient` inalterado — não implementa `ChunkedClient`.
- `cmd/api/main.go` inalterado.
- Handlers REST Sprint 20 inalterados (não chamam métodos chunked).

## Decisões que pagaram

### D-1. Interface segregation (mesmo padrão Sprint 20)

`ChunkedClient` separada de `ReadClient`/`Client`. **Por que 3 interfaces?** Cada
uma representa uma capability distinta:
- `Client` — submit (write-side, low-level)
- `ReadClient` — listagem + alteração situação (read-side BACEN→IF)
- `ChunkedClient` — transfer chunked (range upload + download)

Extender uma única interface cresceria ela. Segregar mantém minimal + testável.

### D-2. YAGNI handlers REST

Sprint 19 fez a mesma decisão (sem handlers). Sprint 20 inverteu (handlers criados
quando frontend consumiu). Sprint 21 volta a YAGNI porque:
- Batch worker (Sprint 22+) é o caller natural de SubmitRange (retry).
- Frontend consumiria DownloadRange? Talvez, mas UX de range download no browser é
  complexa (HTTP Range + transparent retry) — Sprint 23+.

Pattern replicável: handler só entra quando há caller imediato. Capability no client
fica disponível desde cedo (low cost) mas vira "exposed capability" sem uso imediato
(= latente, não hollow).

### D-3. `expectedTotalHash` parameter vs calcular internamente

Cliente **passa** o hash esperado; cliente **não calcula** SHA-256 do arquivo completo
durante download (porque cliente não tem o arquivo completo, está baixando).

Caller tem 2 opções:
1. Passar hash de `ListDisponiveis.Hash` (caso comum — cliente sabe antecipadamente).
2. Passar "" para skip validation (cliente confia cegamente no BACEN).

Documentação explicita no godoc: "expectedTotalHash deve vir de ListDisponiveis.Hash,
não ser calculado pelo cliente".

### D-4. 200 OK + 206 Partial Content ambos aceitos

Manual §6.4 mostra 206 Partial Content, mas BACEN poderia (teoricamente) retornar
200 OK com range respeitado. Cliente aceita ambos sem distinção — caller pode
checar `Content-Range` no response se precisar saber exatamente.

**Defensivo:** se BACEN bugar e mandar 200 sem range, caller ainda recebe body + hash.
Sem rejeição artificial.

### D-5. Sanitização de headers sensíveis

`If-Match` + `If-Unmodified-Since` são passados verbatim para BACEN. Caller é
responsável por não colocar PII nesses headers (improvável — são version stamps).

OK, sem defesa adicional necessária.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos modificados | 1 (`internal/sta/ws.go`) |
| Arquivos com testes | 1 (`internal/sta/ws_test.go`) |
| Testes Sprint 21 | 13 (12 httptest + 1 interface segregation) |
| Total STA | 63 testes top-level |
| Packages PASS | 18/18 |
| Build OK | 5/5 binaries |
| Smoke E2E | 11/11 PASS |
| gofmt drift | 0 |
| go vet | clean |
| Linhas adicionadas (ws.go) | +234 |
| Linhas adicionadas (ws_test.go) | +370 |

## Lições aprendidas (carry forward)

### L-1. Content-Range (upload) vs Range (download) são diferentes

| Aspecto | Content-Range (PUT) | Range (GET) |
|---|---|---|
| Sintaxe | `bytes inicio-fim/total` | `bytes=inicio-fim` |
| Inclui total | Sim | Não |
| RFC | 7233 §4.2 | 7233 §3.1 |

Erro fácil: trocar `=` por espaço ou esquecer `/total`. Tests capturam esse erro.

### L-2. X-Content-Hash é do arquivo COMPLETO, não do chunk

Manual §6.4 linha 716: `X-Content-Hash: SHA-256 {hash_arquivo}` (singular, sem "do chunk").
Caller tem que ter hash antecipadamente. Source natural: `ListDisponiveis.Hash`.

### L-3. Chunked + Retry = futuro do upload/download de CADOC

Sprint 21 entrega as **primitivas**. Sprint 22+ (retry wrapper) usa essas primitivas
para implementar retry transparente. Pattern: primitiva → wrapper → handler (à la
io.Reader + bufio.Reader).

### L-4. YAGNI consciente vs preguiçoso (mesma lição Sprint 19)

YAGNI consciente: "sem caller imediato, não crio handler" — documentado em
SPRINT_21_RESEARCH.md §4. Capability fica disponível (low cost).
YAGNI preguiçoso: "vou implementar mesmo sem caller" — código morto que ninguém usa.

## Próximos passos (Sprint 22+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 22 | Retry exponencial wrapper + batch worker que usa SubmitRange | Resilience + retry automatizado |
| 23 | Senhaws endpoint (§9.1) + credential rotation | Troca periódica de senha Sisbacen |
| 24 | Smoke contra BACEN homolog real (precisa credenciais) | Última validação antes de produção |
| 25 | Handlers REST `/v1/sta/range-*` (quando batch worker chamar) | Frontend ou batch trigger UI |

## Critérios de done — todos ✅

- [x] `WSClient.SubmitRange(ctx, protocolo, inicio, fim, total, chunk) error` implementado
- [x] `WSClient.DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince) (*DownloadResult, error)` implementado
- [x] `ChunkedClient` interface segregation (não estender `Client` ou `ReadClient`)
- [x] 13 testes httptest STA (12 cenários + 1 interface segregation)
- [x] 18/18 packages PASS + smoke + gofmt/vet
- [x] SPRINT_21_RESEARCH.md (10 seções) + SPRINT_21_RESULTS.md (este) + CHANGELOG v3.11.0
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Cross-tenant data exfiltration** — defesa herdada da validação 41 (`enforceSameIF`).
2. **Hollow stub** — `ChunkedClient` segregation garante falha explícita.
3. **Vazamento err.Error()** — `parseSTAErrorTyped` retorna `STAError.Message` sanitizado.
4. **Confiar em body sem cross-check** — `expectedTotalHash` validado contra X-Content-Hash.
5. **Cap ausente** — `maxDownloadBodyBytes = 100 MiB` reusado da Sprint 19.
6. **Sentinel único** — reusa `ErrContentHashMismatch` + `ErrContentHashHeaderMalformed`
   da Sprint 19 (não inventou novos).
7. **Endpoint sem caller** — handlers REST deliberadamente não criados (YAGNI consciente).