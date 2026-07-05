# SPRINT 20 — RESULTS: STA WS listagem / disponiveis + alteração / situacao + handlers REST

> **Sprint:** 20 (v3.10.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** `fa4dc13` (Sprint 20 — ver VALIDAÇÃO 41+ para commits subsequentes)

## TL;DR

Sprint 20 fecha o **read side completo** do BACEN STA WS e entrega os **handlers
REST** correspondentes. IF agora pode (a) **listar arquivos que BACEN disponibilizou**
(`/v1/sta/disponiveis` polling frontend), (b) **marcar como recebido**
(`/v1/sta/situacao` UX), (c) via interface segregation, o **StubClient** continua
funcionando mas retorna **503** quando caller tenta read side sem ter configurado
`RADIANT_STA_BACKEND=ws`. **24 testes novos** (16 httptest + 8 handler integration).
Total STA: **51 testes** (16 Sprint 18 + 16 Sprint 19 + 16 Sprint 20 + 3 validação 40).

## Entregas

### 1. `WSClient.ListDisponiveis(ctx, opts)` — manual §8.1.1

```go
func (c *WSClient) ListDisponiveis(ctx context.Context, opts ListDisponiveisOpts) (*ListDisponiveisResult, error)
```

- GET `/arquivos/disponiveis?dataHoraInicio=...&...`
- Content-Type omitido (manual linha 878)
- Paginação: até 1000 protocolos; >1000 retorna `<atom:link href="..." rel="disponiveis"/>` — exposto em `ProximaPaginaURL` + `TemProximaPagina`.
- `DataHoraProximaConsulta` ecoado do XML (polling incremental).
- **Validação client-side:** `DataHoraInicio` obrigatório (Tabela 4 linha 1472).

### 2. `WSClient.AlterarSituacao(ctx, req)` — manual §7.1

```go
func (c *WSClient) AlterarSituacao(ctx context.Context, req AlterarSituacaoReq) error
```

- PUT `/arquivos/situacao` com body XML `<Parametros>`
- Content-Type **OBRIGATÓRIO** `application/xml` (manual linha 792) — único endpoint com essa exigência.
- Valores válidos `A_REC` | `REC` (enum tipado `SituacaoTransferencia`).
- 204 No Content em sucesso (sem body).
- **Validações client-side:** `Protocolos` não-vazio + `Situacao` válida.

### 3. `ReadClient` interface segregation (decisão arquitetural)

```go
type ReadClient interface {
    ListDisponiveis(ctx, opts) (*ListDisponiveisResult, error)
    AlterarSituacao(ctx, req) error
}
```

- Apenas `*WSClient` implementa (compile-time check + runtime type-assertion test).
- `*StubClient` **NÃO** implementa (test `TestReadClient_InterfaceSegregation` prova).
- Handlers fazem `s.STAClient.(sta.ReadClient)` — falha graceful com 503.

**Por que não estender `Client` interface?** Forçar StubClient a implementar
ListDisponiveis/AlterarSituacao com zero-values seria hollow stub piorado — caller
acharia que funcionou mas BACEN nunca foi chamado. Falhar explícito (503) é melhor
que mentir.

### 4. Handlers REST

| Method | Path | WSClient method | Auth | Status codes |
|---|---|---|---|---|
| `GET` | `/v1/sta/disponiveis?dataHoraInicio=YYYY-MM-DDTHH:MM:SS.SSS` | `ListDisponiveis` | JWT + enforceSameIF | 200 / 400 / 401 / 503 |
| `POST` | `/v1/sta/situacao` body `{"protocolos":["1"],"situacao":"REC"}` | `AlterarSituacao` | JWT + enforceSameIF | 204 / 400 / 401 / 503 |

**`dataHoraInicio` default = if_id do tenant** quando caller não fornece — defesa
contra chamada cross-tenant sem `dependencia` explícita.

**Audit emission:**
- Sucesso: `sta.disponiveis.listed`, `sta.situacao.changed` (com metadata: qtde, paginação, etc).
- Erro formal BACEN (4xx): `sta.{op}.rejected` com `staErr.StatusCode` + Message sanitizado.
- Erro transporte: `sta.{op}.failed` com err.Error() sanitizado via `SafeError`.
- Backend stub: `sta.{op}.stub_backend` (audit informational — caller precisa mudar config).

### 5. Tipos públicos

| Tipo | Propósito |
|---|---|
| `ListDisponiveisOpts` | params (Tabela 4) |
| `ListDisponiveisResult` | retorno: arquivos + paginação |
| `ArquivoDisponivel` | 1 arquivo da listagem |
| `SituacaoArquivo` enum | Codigo 1 = Recebido / Codigo 3 = A receber |
| `AlterarSituacaoReq` | request do /situacao |
| `SituacaoTransferencia` enum | A_REC / REC |

`SituacaoArquivoRaw` e `SituacaoAtualRaw` guardam string crua do XML — defesa
contra BACEN adicionar valor novo sem atualizar IF.

## Decisões que pagaram

### D-1. Interface segregation (vs estender `Client`)

Já discutido acima. **Resultado:** test `TestReadClient_InterfaceSegregation`
compila-time + runtime prova que segregation funciona. Handlers lidam com
fallback graciosamente (503).

### D-2. `dataHoraInicio` default = if_id

Caller que não fornece `dependencia` recebe automaticamente o tenant do JWT.
**Por quê:** IFs pequenas chamam `/v1/sta/disponiveis` sem pensar em filtro —
default seguro é "mostre MEUS arquivos". Caller que quer ver de outra IF passa
explicitamente (e enforceSameIF bloqueia se != tenant).

### D-3. `SituacaoArquivo` enum + Raw (defesa contra BACEN evoluir)

Manual não documenta tabela oficial para `SituacaoAtual` (a Tabela 3 cobre
**estado** do arquivo, não situação). Códigos 1 e 3 são inferência + confirmação
do manual.

**Solução:** enum tipado cobre os 2 valores conhecidos + `Unknown`. `SituacaoAtualRaw`
guarda string cru. **Caller pode detectar "valor novo do BACEN"** via
`SituacaoAtual == Unknown && SituacaoAtualRaw != ""`.

### D-4. Cap 10 MiB do response (reusa `maxResponseBodyBytes`)

Listagem até 1000 arquivos × 500 bytes ≈ 500 KB. Bem abaixo do cap de 10 MiB já
existente. **Não precisei de cap novo** — só reuso.

### D-5. `url.Values.Encode()` para query string

Substitui concatenação manual `?dataHoraInicio=X&sistemas=Y&...`. Vantagens:
- URL encoding automático (acentos, espaços, etc).
- Ordem determinística (testável).
- Menos chance de typo.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (`internal/api/sprint20_handlers.go` + `_test.go`) |
| Arquivos modificados | 4 (`ws.go`, `ws_types.go`, `ws_test.go`, `server.go`) |
| Testes Sprint 20 | 24 (16 httptest STA + 8 handler integration) |
| Total STA | 51 testes top-level |
| Packages PASS | 18/18 |
| Build OK | 5/5 binaries |
| Smoke E2E | 11/11 PASS |
| gofmt drift | 0 |
| go vet | clean |

## Compatibilidade

- `Client` interface **inalterada** — StubClient e WSClient mantêm `Submit(ctx, sub) (*Result, error)`.
- `*WSClient` ganha 2 métodos novos + implementa nova `ReadClient` interface.
- `*StubClient` **NÃO** implementa `ReadClient` — handlers retornam 503 com mensagem clara.
- `cmd/api/main.go` **inalterado** — `sta.NewClientFromEnv()` continua decidindo stub vs ws.
- `RADIANT_STA_BACKEND=stub` (default) preserva comportamento das 19 sprints anteriores.
  - Submit funciona (legacy).
  - Disponiveis/Situacao retornam 503 com audit `sta.{op}.stub_backend`.
- `RADIANT_STA_BACKEND=ws` ativa todos os endpoints (Submit + StatusUpload + Download + ListDisponiveis + AlterarSituacao).

## Lições aprendidas (carry forward)

### L-1. Interface segregation vs estender interface

**Estender** força implementação vazia = hollow stub.
**Segregation** permite falha explícita quando capability ausente.

Pattern replicável: sempre que tiver "subset de operações que um subset de implementações
suporta", segregar interface. Caller faz type assertion ou recebe erro claro.

### L-2. `url.Values.Encode()` em vez de concatenar query string

Pattern replicável. Manual building de `?k1=v1&k2=v2` é bug-prone (encoding, ordering,
empty values). `url.Values` resolve tudo.

### L-3. Default seguro = tenant ID

Para endpoints read-by-tenant, default `dependencia = if_id do JWT`. Caller precisa
passar explicitamente pra cross-tenant (que enforceSameIF já bloqueia).

Pattern replicável pra qualquer endpoint "show me my X" — não force caller a repetir
tenant que ele já provou ser.

### L-4. Compilação de 200 arquivos: vet+vét+gofmt+test+build+smoke = ~15s

Custo de validação completa. Aceitável pra sprint pequena. Pra CI em PR futuro,
esse gate é barreira contra drift como o gofmt da validação 38.

## Próximos passos (Sprint 21+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 21 | Range upload (§5.5+5.6) + range download (§6.4) | IFs grandes com CADOC >50 MB |
| 22 | Retry exponencial wrapper (max 3 attempts, backoff 1s/2s/4s) | Resilience contra BACEN 503 transiente |
| 23 | Senhaws endpoint (§9.1) + credential rotation | Troca periódica de senha Sisbacen |
| 24 | Smoke contra BACEN homolog real (precisa credenciais) | Última validação antes de produção |

## Critérios de done — todos ✅

- [x] `WSClient.ListDisponiveis(ctx, opts)` implementado + tipado
- [x] `WSClient.AlterarSituacao(ctx, req)` implementado
- [x] `ReadClient` interface segregation (não estender `Client`)
- [x] `GET /v1/sta/disponiveis` + `POST /v1/sta/situacao` em arquivo `sprint20_handlers.go`
- [x] Auth JWT + `enforceSameIF` em ambos
- [x] Audit emission em sucesso + erro (4 classes: sucesso, rejected, failed, stub_backend)
- [x] 16 httptest STA + 8 integration handlers = 24 testes novos
- [x] 18/18 packages PASS + smoke + gofmt/vet
- [x] SPRINT_20_RESEARCH.md + SPRINT_20_RESULTS.md + CHANGELOG v3.10.0
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Hollow stub** — `ReadClient` segregation evita StubClient fingir que implementa read side.
2. **Endpoint sem caller** — handlers têm consumer imediato (frontend listar arquivos BACEN).
3. **Vazamento err.Error()** — `handleSTAReadError` retorna mensagem genérica ao cliente.
4. **Cross-tenant silencioso** — `dataHoraInicio` default = tenant, cross-tenant bloqueado por enforceSameIF.
5. **Query string concatenada** — `url.Values.Encode()` evita encoding/typo bugs.
6. **Enum sem defesa** — `SituacaoArquivoUnknown` + Raw string detectam valor novo BACEN.
7. **Derrubar backend inteiro se stub** — handlers retornam 503 específico (não 500), audit `stub_backend`.