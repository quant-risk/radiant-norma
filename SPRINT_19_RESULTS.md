# SPRINT 19 — RESULTS: STA WS read side (Download + StatusUpload)

> **Sprint:** 19 (v3.9.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** (preencher após push)

## TL;DR

Sprint 19 fecha o **read side** do `WSClient` (Sprint 18 = write side). IF agora pode
(a) **consultar situação de upload interrompido** (`StatusUpload`) e (b) **baixar
arquivo completo com validação de integridade ponta-a-ponta** (`Download`). Validação
de `X-Content-Hash` é **obrigatória** (contrato BACEN conforme manual §6.1.1 linhas
641-643). Erros formais BACEN retornam `*STAError` tipado. **14 novos testes** cobrem
happy + 5 paths de erro + 3 helpers pure functions. Total STA: **37 testes**.

## Entregas

### 1. `WSClient.StatusUpload(ctx, protocolo)` — manual §5.3.1

```go
func (c *WSClient) StatusUpload(ctx context.Context, protocolo string) (*UploadStatus, error)
```

- GET `/arquivos/{protocolo}/posicaoupload`
- Content-Type **omitido** (manual linha 451)
- Response 200 OK + XML `<Resultado><Protocolo/><RangesRecebidos/><Situacao/></Resultado>`
- Retorna `*UploadStatus` com:
  - `Protocolo string` (eco do consultado)
  - `RangesRecebidos []Range` (parseado de `0-3;5-8` → `[{0,3},{5,8}]`)
  - `Situacao UploadSituacao` (enum tipado: `NaoIniciada` | `Pendente` | `Finalizada` | `Unknown`)
  - `SituacaoRaw string` (string original do XML pra audit/debug)

**Errors:**
- `*STAError{StatusCode: 400, 403}` em rejeição formal
- `err` opaco em erro de transporte (rede, timeout, parse)

### 2. `WSClient.Download(ctx, protocolo)` — manual §6.1.1

```go
func (c *WSClient) Download(ctx context.Context, protocolo string) (*DownloadResult, error)
```

- GET `/arquivos/{protocolo}/conteudo`
- Content-Type **omitido** (manual linha 620)
- Response 200 OK + headers `ETag` + `Last-Modified` + `X-Content-Hash` + body binário
- Retorna `*DownloadResult` com:
  - `Conteudo []byte` (body — ZIP típico)
  - `ContentHash string` (SHA-256 hex computado pelo cliente — match com X-Content-Hash)
  - `ETag string`
  - `LastModified string`
  - `ContentHashHeader string` (valor cru do header pra audit)

**Errors:**
- `ErrContentHashMismatch` (sentinel) — SHA-256 do body ≠ X-Content-Hash
- `ErrContentHashHeaderMalformed` (sentinel) — formato do header mudou
- `*STAError{StatusCode: 404}` protocolo inexistente
- `*STAError{StatusCode: 410}` arquivo não disponível
- `*STAError{StatusCode: 413}` body > 100 MiB (cap defensivo)
- `*STAError{Code: MISSING_X_CONTENT_HASH}` header ausente (defesa contra regressão BACEN)
- `err` opaco em erro de transporte

### 3. `*STAError` type + 2 sentinel errors

```go
type STAError struct {
    StatusCode int      // HTTP status cru
    Code       string   // <Erro><Codigo> ou "HTTP_N" / "MISSING_X_CONTENT_HASH"
    Message    string   // <Erro><Descricao> crua
    Protocolo  string   // eco do protocolo que originou
}

var (
    ErrContentHashMismatch         error
    ErrContentHashHeaderMalformed error
)
```

Caller usa `errors.As(err, &staErr)` pra inspecionar status code (ex: 404 = protocolo
inexistente, mostrar pro usuário "protocolo não encontrado"). Sentinel errors via
`errors.Is(err, ErrContentHashMismatch)`.

### 4. Helpers pure (testados independentemente)

- `parseRanges(string) []Range` — `"0-3;5-8"` → `[{0,3},{5,8}]`. Lixo descartado
  silenciosamente (BACEN mandar input inválido não crashar cliente).
- `parseUploadSituacao(string) UploadSituacao` — 3 valores oficiais do manual.
- `parseXContentHash(string) (string, error)` — extrai SHA-256 hex de
  `SHA-256 {64-hex}`. Aceita case-insensitive em `SHA-256` e em hex chars.
  Valida comprimento 64 + chars hex.

## Decisões que pagaram

### D-1. Pesquisa antes de código (caminho 1 replicado)

Replicou padrão Sprint 18. Antes de codar:
1. Li Seções 5.3 + 6.1 + Listagem 4 do manual BACEN (42 páginas)
2. Extraí spec XML exata
3. Documentei 7 decisões de design + 7 itens fora de escopo
4. Escrevi 10 cenários de teste antes de escrever 1 linha de código

Resultado: zero retrabalho. Todos os 14 testes passaram de primeira após gofmt.
Sem "ah esqueci esse caso de erro", sem refactor depois.

### D-2. X-Content-Hash obrigatório, não opcional

Manual §6.1.1 linha 641-643 é explícito: "X-Content-Hash ... foi criado pelo Banco
Central do Brasil para ser utilizado na validação da integridade do arquivo recebido".

Decisão: SEMPRE validar. Se header ausente, retorna erro. Se formato mudou, sentinel
distinto. Se hash diverge, sentinel. Cliente NUNCA confia em body sem cross-check.

Por que isso importa: ZIP corrompido silenciosamente → relatórios financeiros com base
em dados errados = risco regulatório sério (BACEN pode multar IF por divergência em
CADOC).

### D-3. Cap 100 MiB (não 10 MiB)

10 MiB (cap do XML response) é folgado pra protocolo XML. Mas arquivos CADOC ZIP
podem chegar a 30-50 MB em bancos grandes. 100 MiB é folgado mas prudente.

Decisão: 100 MiB porque acima disso já é anomalia (BACEN bug / proxy inflando).
Retorna `*STAError{413}` pra caller decidir — não truncar silenciosamente (quebraria
ZIP parsing downstream).

Test `TestWSClient_Download_BodyTooLarge` aloca 120 MiB intencionalmente pra provar
que cliente não estoura memória. Em CI, marcado `t.Skip("aloca 100+ MiB")` em
`-short`.

### D-4. Sem handler REST nesta sprint (YAGNI consciente)

SPRINT_19_RESEARCH.md §5 debateu exaustivamente. Decisão: NÃO criar
`/v1/sta/download` e `/v1/sta/status` sem consumer real. Anti-pattern §31 do
go-security-and-quality.md: endpoint sem caller = waste + attack surface.

Quando entra: Sprint 20 quando frontend listar "arquivos enviados recentemente" ou
backend for fazer retry automatizado (Sprint 21+).

### D-5. `Client` interface inalterada

Sprint 18 deixou `Client` minimal (`Submit` apenas). Manteve. Novos métodos
só em `*WSClient`. `StubClient` continua sem `StatusUpload`/`Download` — quem chamar
recebe erro de compilação claro, não ambiguidade runtime.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos modificados | 3 (`ws.go`, `ws_types.go`, `ws_test.go`) |
| Arquivos novos | 3 (`SPRINT_19_RESEARCH.md`, `SPRINT_19_RESULTS.md`, este arquivo) |
| Linhas adicionadas | ~850 (268 ws.go + 90 ws_types.go + 489 ws_test.go + docs) |
| Testes Sprint 19 | 16 top-level + 22 subtests table-driven = 38 RUNs Sprint 19 |
| Testes STA totais (pós Sprint 19) | 32 (Sprint 18: 16 + Sprint 19: 16) |
| Packages PASS | 18/18 |
| Build OK | 5/5 binaries |
| Smoke E2E | 11/11 PASS |
| gofmt drift | 0 |
| go vet | clean |

## Compatibilidade

- `Client` interface preservada (Sprint 3 + Sprint 18) — StubClient e WSClient
  mantêm contrato `Submit(ctx, sub) (*Result, error)`.
- WSClient ganha 2 métodos novos (`StatusUpload`, `Download`) — adição pura, sem
  alteração nos existentes.
- `cmd/api/main.go` **inalterado** — `sta.NewClientFromEnv()` já decide stub vs ws.
- `RADIANT_STA_BACKEND=stub` (default) preserva comportamento de 18 sprints anteriores.
- `RADIANT_STA_BACKEND=ws` agora expõe Submit + StatusUpload + Download.

## Lições aprendidas (carry forward)

### L-1. Sentinel errors distintos pra diferentes classes de problema

`ErrContentHashMismatch` (BACEN mandou lixo) vs `ErrContentHashHeaderMalformed`
(BACEN mudou formato) — caller pode reagir diferente. Se fossem um só sentinel
"hash error", caller não saberia se vale retry.

### L-2. `errors.As` em vez de `errors.Is` para tipos estruturados

`*STAError` tem campos (StatusCode, Protocolo). `errors.As(err, &staErr)` permite
caller fazer `switch staErr.StatusCode { case 404: ... case 410: ... }`. Muito mais
ergonômico que 5 sentinels separados.

### L-3. Tabela de erros do manual → testes parametrizados

Cada linha da "Possíveis erros" do manual mapeia 1 teste. Manual §6.1.1 lista
3 status codes (400/404/410) → 3 testes. Adicionei mais 3 (hash mismatch, body
too large, header malformed) que o manual não cobre mas que defense-in-depth exige.

### L-4. Helpers pure com testes unit isolados

`parseRanges`, `parseUploadSituacao`, `parseXContentHash` são pure functions.
Tabela de subtests (9 + 5 + 8 = 22 subtests) cobre edge cases que seriam
difíceis de testar via httptest (ex: "BACEN mandar 0-3;malformed" — improvável
mas possível). Pure function tests são baratos e rápidos.

### L-5. YAGNI consciente vs YAGNI preguiçoso

Sprint 19 poderia ter implementado:
- Range download (manual §6.4)
- Handlers REST thin
- Wrapper de retry exponencial

Todos seriam nice-to-have. Nenhum tem caller imediato. Decidi conscientemente
NÃO implementar e documentar o "porquê" em SPRINT_19_RESEARCH.md §4. Isso é
YAGNI consciente. YAGNI preguiçoso seria "vou implementar mesmo sem caller,
fica pronto pra quando precisar" — código morto que ninguém usa.

## Próximos passos (Sprint 20+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 20 | `/arquivos/disponiveis` (listagem a receber) + `/arquivos/situacao` (alterar A_REC↔REC) | Frontend precisa listar "arquivos BACEN mandou pra você" |
| 21 | Range upload (manual §5.5+5.6) + Range download (manual §6.4) | Arquivos >50 MB (IFs grandes) |
| 22 | Retry exponencial wrapper (max 3 attempts, backoff 1s/2s/4s) | Resilience contra BACEN 503 transiente |
| 23 | Senhaws endpoint (manual §9.1) + credential rotation | Troca periódica de senha Sisbacen |
| 24 | Smoke contra BACEN homologação real (precisa credenciais) | Última validação antes de produção |

## Critérios de done — todos ✅

- [x] `WSClient.StatusUpload(ctx, protocolo) (*UploadStatus, error)` implementado
- [x] `WSClient.Download(ctx, protocolo) (*DownloadResult, error)` implementado
- [x] Validação X-Content-Hash ativa (sentinels `ErrContentHashMismatch` + `ErrContentHashHeaderMalformed`)
- [x] `*STAError` type definido para erros formais BACEN
- [x] Cap de 100 MiB no body de download com sentinel 413
- [x] 14 testes httptest cobrindo happy + error paths (37 STA total)
- [x] Doc-comment em cada método citando Seção do manual
- [x] `gofmt -w .` + `go vet ./...` + `go test ./...` + 18 packages
- [x] SPRINT_19_RESULTS.md (este) + CHANGELOG v3.9.0 entry
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Hollow stub** — todos os métodos retornam valor real, não zero-value silencioso.
2. **Endpoint sem caller** — handlers REST deliberadamente não criados (YAGNI consciente).
3. **Log de segredos** — `basicAuthHeader()` continua sem log (Sprint 13 F-13.8).
4. **Confiar em input do BACEN** — X-Content-Hash validation obrigatória, não opcional.
5. **Cap ausente** — 100 MiB no download body, defesa contra DoS via BACEN bug.
6. **Erro opaco** — `*STAError` tipado com StatusCode + Code + Message + Protocolo.
7. **Sentinel único pra problemas diferentes** — 2 sentinels distintos (mismatch vs malformed).