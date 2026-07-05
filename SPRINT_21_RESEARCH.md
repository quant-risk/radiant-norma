# SPRINT 21 — Research: Range Upload (§5.6) + Range Download (§6.4)

> **Sprint:** 21 (v3.11.0)
> **Quando:** 2026-07-06
> **Pesquisador:** mavis
> **Status:** pesquisa completa, pronto pra implementação

## 1. Contexto

Sprint 21 fecha o **chunked transfer** do BACEN STA WS. Sprint 18-20 implementaram
single-call (envio e download de arquivo inteiro em uma chamada). Para IFs pequenas
(CADOC <10 MB), single-call é suficiente. Para IFs grandes (CADOC >50 MB), chunked
oferece:
- **Retry parcial** se conexão cai mid-upload
- **Upload paralelo** de chunks (até 10 simultâneos conforme §2.6)
- **Download paralelo** (mesma ideia, até 10 chunks simultâneos)

Manual v1.5 (jul/2022) documenta:
- §5.6 — Envio de parte de arquivo (PUT com Content-Range)
- §6.4 — Recebimento de parte de arquivo (GET com Range)

Sprint 19 introduziu `parseRanges` (parser de RangesRecebidos do posicaoupload)
— agora precisamos dos métodos reais que USAM esses ranges para fazer
resume.

## 2. Spec BACEN extraída do manual v1.5

### 2.1 PUT parcial — Seção 5.6 (linhas 520-577)

**Request:**
```
PUT /staws/arquivos/{protocolo}/conteudo HTTP/1.1
Content-Range: bytes {inicio}-{fim}/{total}

{conteúdo_arquivo}
```

**Formato Content-Range (RFC 7233 §4.2):**
```
Content-Range: bytes <inicio>-<fim>/<total>
```
- `inicio`, `fim`: byte range **inclusivo** (RFC 7233 §2.1 — diferente de HTTP/1.0)
- `total`: tamanho total do arquivo completo

**Atenção (linha 538-539):** "O cabeçalho HTTP da requisição não precisa conter o
campo Content-Type. Caso informado, não é permitido multipart/form-data."

**Response sucesso:** 200 OK (sem body — same as §5.2 single-call).

**Erros esperados:**

| Status | Significado |
|---|---|
| 400 | Erro genérico — XML Listagem 4 |
| 403 | Protocolo não pertence à instituição |
| 404 | Protocolo não encontrado |
| 410 | Protocolo foi cancelado pelo BACEN |
| 416 | Range informado é inválido |
| 501 | Range multipart não é suportado |

### 2.2 GET parcial — Seção 6.4 (linhas 684-754)

**Request:**
```
GET /staws/arquivos/{protocolo}/conteudo HTTP/1.1
Range: bytes={inicio}-{fim}
If-Match: {etag}
If-Unmodified-Since: {data_modificacao_arquivo}
```

**Atenção (linha 701):** "O cabeçalho HTTP da requisição não deve conter o campo Content-Type."

**Observação (linha 703):** "os cabeçalhos If-Match e If-Unmodified-Since são opcionais."

**`If-Match` + `If-Unmodified-Since` (RFC 7232):**
- `If-Match: <etag>` — pré-condição (412 se etag atual !=)
- `If-Unmodified-Since: <data>` — pré-condição temporal (412 se arquivo modificado depois)

**Response sucesso (206 Partial Content):**
```
HTTP/1.1 206 Partial Content
ETag: {etag}
Last-Modified: {data_modificacao_arquivo}
X-Content-Hash: SHA-256 {hash_arquivo}

{conteudo_arquivo}
```

**Detalhe crítico (Sprint 19 notou):** `X-Content-Hash` é do **arquivo completo**,
não do chunk. Cliente valida comparando contra hash conhecido (ex.: vindo de
`ListDisponiveis` ou download anterior).

**Erros esperados:**

| Status | Significado |
|---|---|
| 400 | Erro genérico — XML Listagem 4 |
| 404 | Protocolo não encontrado |
| 410 | Arquivo não disponível para download |
| 412 | Validação `If-Match`/`If-Unmodified-Since` falhou |
| 416 | Range informado é inválido |
| 501 | Range multipart não é suportado |

## 3. Decisões de design

### 3.1 Quais métodos expor no WSClient?

| Método | Propósito | Caller típico |
|---|---|---|
| `SubmitRange(ctx, protocolo, inicio, fim, total, chunk)` | Envia 1 chunk do arquivo | Batch worker (Sprint 22+) para upload paralelo |
| `DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince)` | Baixa 1 chunk + valida hash do arquivo completo | Batch worker pra retry/download paralelo |

`Submit` (single-call) + `Download` (single-call) **continuam existindo** —
não são deprecados. Caller escolhe baseado no tamanho do arquivo:
- <50 MB: single-call (já implementado).
- >50 MB: chunked com retry (Sprint 22+).

### 3.2 Estruturas — `Range` já existe

Sprint 19 introduziu:
```go
type Range struct {
    Start int64
    End   int64  // inclusivo
}
```

Reusar para os ranges de request. **Não criar struct novo** — DRY.

### 3.3 Validação X-Content-Hash no DownloadRange

**Decisão crítica:** Cliente **NÃO** pode validar hash do chunk isolado (BACEN não
disponibiliza). Cliente pode:
1. Receber chunk + X-Content-Hash do arquivo completo.
2. Calcular SHA-256 do arquivo completo esperado (já conhecido via ListDisponiveis).
3. Comparar X-Content-Hash com expectedTotalHash.

Se match → chunk veio de arquivo íntegro.
Se mismatch → BACEN bugou ou proxy corrompeu → erro.

**Por que essa validação é possível:** ListDisponiveis retorna `Hash` (SHA-256 hex)
para cada arquivo. Cliente pode passar como `expectedTotalHash` no DownloadRange.
Se cliente não tem hash prévio, passa string vazia = sem validação.

**API:**
```go
func (c *WSClient) DownloadRange(
    ctx context.Context,
    protocolo string,
    inicio, fim int64,
    expectedTotalHash string,  // "" = skip validation
    ifMatch string,            // "" = no If-Match
    ifUnmodifiedSince string,   // "" = no If-Unmodified-Since
) (*DownloadResult, error)
```

### 3.4 Validação `Content-Range` no SubmitRange

Cliente **DEVE** calcular Content-Range correto. Manual linha 534: "os parâmetros
inicio e fim são obrigatórios".

Validações client-side antes de chamar BACEN:
- `inicio >= 0`
- `fim >= inicio`
- `total > 0` e `total >= fim+1`
- `len(chunk) == fim-inicio+1` (chunks devem ter tamanho exato do range declarado)

### 3.5 Erro vs Rejection — reuso Sprint 19

`*STAError` continua sendo o tipo para rejeição formal BACEN (4xx/5xx com XML).
Mesma convenção que StatusUpload/Download/ListDisponiveis/AlterarSituacao.

`ErrContentHashMismatch` (Sprint 19) reusado para DownloadRange.

### 3.6 Limite de tamanho do chunk

Cap em `maxDownloadBodyBytes = 100 MiB` (Sprint 19). Chunk > 100 MiB → `*STAError{413}`
(mesmo que Download single-call). CACEN raramente manda chunk >50 MB, mas cap
defensivo.

### 3.7 Thread-safety

Mesmo padrão. `http.Client.Do()` thread-safe. Sem mudança em WSClient struct.

### 3.8 Interface segregation — `ChunkedClient`

**Decisão:** criar `ChunkedClient` interface segregation (mesmo padrão da `ReadClient`
da Sprint 20).

```go
type ChunkedClient interface {
    SubmitRange(ctx, protocolo, inicio, fim, total, chunk) error
    DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince) (*DownloadResult, error)
}
```

Apenas `*WSClient` implementa. `*StubClient` NÃO implementa. Caller faz
type assertion.

**Por que não estender `Client` ou `ReadClient`?** Mesma justificativa da Sprint 20:
forçar StubClient a implementar com zero-values seria hollow stub.

## 4. Handlers REST — YAGNI consciente

Sprint 21 **NÃO** cria handlers REST. Razão: sem consumer imediato.
- Range upload/download é útil pra IFs grandes (>50 MB), mas consumidor natural
  seria batch worker (Sprint 22+) que implementa retry automatizado.
- Frontend baixar CADOC >50 MB também seria caso de uso, mas UX de range download
  no browser é complexa (HTTP Range + retry transparente). Sprint 23+.

**Decisão:** métodos ficam disponíveis no WSClient. Handlers REST entram na Sprint 23+
quando batch worker chamar.

> **Anti-pattern §31 do `go-security-and-quality.md`:** endpoint sem caller = waste.
> Range upload/download é **capability**, não produto ainda.

## 5. Compatibilidade com StubClient

`StubClient` continua implementando só `Submit`. **NÃO** estende pra `SubmitRange` ou
`DownloadRange`. Quem tentar chamar recebe erro de compilação claro (interface
segregation).

`ChunkedClient` type assertion em runtime permite fallback gracioso:
```go
if cc, ok := s.STAClient.(sta.ChunkedClient); ok {
    cc.SubmitRange(...)
}
```

## 6. Plano de testes

12 cenários httptest:

| Test | Cobre |
|---|---|
| `TestWSClient_SubmitRange_HappyPath` | §5.6 com 1 chunk + Content-Range header correto |
| `TestWSClient_SubmitRange_416_RangeInvalido` | BACEN rejeita → `*STAError{416}` |
| `TestWSClient_SubmitRange_403` | Protocolo outra IF |
| `TestWSClient_SubmitRange_404` | Protocolo inexistente |
| `TestWSClient_SubmitRange_410` | Protocolo cancelado |
| `TestWSClient_SubmitRange_Validacao_InicioNegativo` | Defensiva client-side |
| `TestWSClient_SubmitRange_Validacao_TamanhoChunk` | len(chunk) != range |
| `TestWSClient_DownloadRange_HappyPath` | §6.4 com 206 Partial Content + X-Content-Hash |
| `TestWSClient_DownloadRange_412` | If-Match/If-Unmodified-Since falhou |
| `TestWSClient_DownloadRange_416` | Range inválido |
| `TestWSClient_DownloadRange_HashValidado` | expectedTotalHash matches X-Content-Hash |
| `TestWSClient_DownloadRange_HashMismatch` | expectedTotalHash != X-Content-Hash → sentinel |
| `TestChunkedClient_InterfaceSegregation` | WSClient implementa, StubClient NÃO |

## 7. Critérios de done

- [ ] `WSClient.SubmitRange(ctx, protocolo, inicio, fim, total, chunk) error` implementado
- [ ] `WSClient.DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince) (*DownloadResult, error)` implementado
- [ ] `ChunkedClient` interface segregation (não estender `Client` ou `ReadClient`)
- [ ] 13 testes httptest STA (12 + 1 interface segregation)
- [ ] 18/18 packages PASS + smoke + gofmt/vet
- [ ] SPRINT_21_RESEARCH.md (este) + SPRINT_21_RESULTS.md + CHANGELOG v3.11.0
- [ ] commit + push

## 8. Riscos identificados

| Risco | Mitigação |
|---|---|
| BACEN retorna X-Content-Hash mudando formato (não documentado) | Sentinel `ErrContentHashHeaderMalformed` da Sprint 19 detecta |
| Chunk overflow 100 MiB (BACEN bug) | Cap defensivo `maxDownloadBodyBytes` |
| Cliente calcula SHA-256 errado do arquivo completo | Documentação caller-side: `expectedTotalHash` deve vir de `ListDisponiveis.Hash`, não ser calculado pelo cliente |
| Race: 2 chunks paralelos contra mesmo protocolo | BACEN serializa por protocolo (§5.5 manual); caller não precisa se preocupar |
| Manual §5.5 menciona "até 10 conexões paralelas" — caller pode hammerar | Rate limit no Router global; caller consciente do limite BACEN |

## 9. O que NÃO entra nesta sprint

- **Handlers REST `/v1/sta/range-upload` + `/v1/sta/range-download`** — YAGNI até
  Sprint 23+ quando batch worker chamar.
- **Upload paralelo de N chunks simultâneos** — caller (Sprint 22+) decide.
- **Retry exponencial** — Sprint 22 (wrapper).
- **Smoke contra BACEN real** — Sprint 24.

## 10. Referências

- Manual BACEN STA Web Services v1.5 (jul/2022, 42 pp) — `_referencias/STA_Manual_WebServices.pdf`
- Seções extraídas:
  - §5.5 (envio paralelo)
  - §5.6 (envio de parte)
  - §6.3 (recebimento paralelo)
  - §6.4 (recebimento de parte)
  - §2.6 (limites de conexão — 10 simultâneos, 120/min)
- SPRINT_18_RESEARCH.md — WSClient skeleton
- SPRINT_19_RESEARCH.md — read side básico + X-Content-Hash validation
- SPRINT_20_RESEARCH.md — read side completo + ReadClient interface segregation
- VALIDATION_v3.10.0_DEEPEST.md — padrões reforçados (enforceSameIF, stdlib errors.As, etc)