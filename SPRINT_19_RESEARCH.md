# SPRINT 19 — Research: Download + StatusUpload no WSClient (read side BACEN STA WS)

> **Sprint:** 19 (v3.9.0)
> **Quando:** 2026-07-04
> **Pesquisador:** mavis
> **Status:** pesquisa completa, pronto pra implementação

## 1. Contexto

Sprint 18 (v3.8.0) entregou o **esqueleto write-side** do `WSClient`: `Submit(ctx, sub)`
implementa o fluxo 2-fase de upload (POST `/arquivos` → protocolo; PUT `/arquivos/{protocolo}/conteudo` → binário).
Faltam todas as operações de **leitura** (read side) que BACEN expõe via WS v1.5:

| Endpoint BACEN | Seção manual | Status atual | Sprint alvo |
|---|---|---|---|
| `GET /arquivos/{protocolo}/conteudo` | 6.1 (recebimento completo) | **não existe** | **Sprint 19** ✅ |
| `GET /arquivos/{protocolo}/posicaoupload` | 5.3 (consulta situação envio) | **não existe** | **Sprint 19** ✅ |
| `GET /arquivos/disponiveis` | 8.1.1 (listar a receber) | tipo XML existe, sem método | Sprint 20+ |
| `PUT /arquivos/situacao` | 7.1 (alterar A_REC↔REC) | tipo XML existe, sem método | Sprint 20+ |
| Range/parallel upload (5.5/5.6) | 5.5+5.6 | não existe | Sprint 21+ |

Esta sprint foca nos **dois endpoints mais críticos** para o caso de uso de IFs pequenas:
1. **StatusUpload** — antes de retomar upload interrompido (essencial pra resiliência)
2. **Download** — receber arquivo CADOC de volta (validação, auditoria, reprocessamento)

## 2. Spec BACEN extraída do manual v1.5 (jul/2022)

### 2.1 `GET /arquivos/{protocolo}/posicaoupload` — Seção 5.3.1

**Request:**
```
GET /staws/arquivos/{protocolo}/posicaoupload HTTP/1.1
Authorization: Basic {base64(user:pass)}
```

> Atenção (manual linha 451): "O cabeçalho HTTP da requisição **não deve** conter o campo Content-Type."

**Response sucesso (200 OK):**
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
  <Protocolo>1</Protocolo>
  <RangesRecebidos>0-3;5-8</RangesRecebidos>
  <Situacao>Transmissão pendente</Situacao>
</Resultado>
```

**Valores de `Situacao` (3):**
- `Transmissão não iniciada`
- `Transmissão finalizada`
- `Transmissão pendente`

**Formato de `RangesRecebidos`:** lista separada por `;` com pares `inicio-fim` (hífen separando início de fim).
Exemplo: `0-3;5-8`. Lista vazia `""` é válida (transmissão não iniciada).

**Erros esperados (Seção 5.3.1):**
- 400 → XML erro formato Listagem 4
- 403 → protocolo não pertence à instituição

### 2.2 `GET /arquivos/{protocolo}/conteudo` — Seção 6.1.1

**Request (recebimento completo):**
```
GET /staws/arquivos/{protocolo}/conteudo HTTP/1.1
Authorization: Basic {base64(user:pass)}
```

> Atenção (manual linha 620): "O cabeçalho HTTP da requisição **não deve** conter o campo Content-Type."

**Response sucesso (200 OK):**
```
HTTP/1.1 200 OK
ETag: {etag}
Last-Modified: {data_modificacao_arquivo}
X-Content-Hash: SHA-256 {hash_arquivo}

{conteúdo_arquivo}
```

**Headers críticos:**
- `ETag` — version stamp (RFC 7232 §2.3, obrigatório HTTP, citado como usado em If-Match/If-Unmodified-Since)
- `Last-Modified` — RFC 7232 §2.2
- `X-Content-Hash: SHA-256 {hash_arquivo}` — **header customizado BACEN** (linha 641) criado para validação de integridade. Não é padrão HTTP.

**Erros esperados (Seção 6.1.1):**
- 400 → XML erro
- 404 → protocolo não encontrado
- 410 → arquivo não disponível para download

### 2.3 Listagem 4 — formato XML de erro (compartilhado por todos os endpoints)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Resultado>
  <Erro>
    <Codigo>400</Codigo>
    <Descricao>Descrição do erro</Descricao>
  </Erro>
</Resultado>
```

Já parseado por `parseSTAError()` (Sprint 18).

## 3. Decisões de design

### 3.1 Quais métodos expor?

| Método | Propósito | Reuso |
|---|---|---|
| `StatusUpload(ctx, protocolo) (*UploadStatus, error)` | Cliente pergunta "o que BACEN já recebeu deste protocolo?" antes de retormar upload | Essencial pra retry/resume |
| `Download(ctx, protocolo) (*DownloadResult, error)` | Cliente baixa arquivo CADOC completo + valida integridade | Validação de round-trip, auditoria |

### 3.2 Estruturas de retorno

```go
// UploadStatus = resultado de StatusUpload (Seção 5.3.1)
type UploadStatus struct {
    Protocolo      string   // eco do protocolo consultado
    RangesRecebidos []Range  // pares [start, end] interpretados
    Situacao        UploadSituacao  // typed enum (não string cru)
    SituacaoRaw     string   // string original (defense in depth)
}

// Range é um par [start, end] interpretado de RangesRecebidos.
type Range struct {
    Start int64
    End   int64
}

// UploadSituacao é o enum tipado dos 3 valores do manual.
type UploadSituacao int

const (
    UploadSituacaoUnknown UploadSituacao = iota
    UploadSituacaoNaoIniciada
    UploadUploadPendente
    UploadSituacaoFinalizada
)

// DownloadResult = resultado de Download (Seção 6.1.1)
type DownloadResult struct {
    Conteudo    []byte    // bytes do arquivo (ZIP tipicamente)
    ContentHash string    // hash SHA-256 hex dos bytes (cross-check com X-Content-Hash)
    ETag        string    // header ETag (vazio se BACEN não mandar)
    LastModified string   // header Last-Modified (string RFC 1123, BACEN não garante formato)
    ContentHashHeader string // valor cru do header X-Content-Hash (debug/audit)
}
```

### 3.3 Validação X-Content-Hash (decisão crítica)

O manual linha 641-643 é explícito:

> "O campo de cabeçalho X-Content-Hash não é um padrão do HTTP. Ele foi criado pelo Banco Central do Brasil para ser utilizado na validação da integridade do arquivo recebido através de um algoritmo de hash."

**Decisão:** o cliente **deve computar** SHA-256 do body e **comparar** com o valor em `X-Content-Hash`.
Se mismatch, retorna erro `ErrContentHashMismatch` (definido como sentinel em `ws.go`).
Erro não-recuperável — não adianta refazer request, BACEN mandou lixo.

**Por que essa validação é obrigatória:**
1. Manual diz explicitamente "validação da integridade do arquivo recebido" → é parte do contrato.
2. Sem isso, ZIP corrompido silenciosamente seria persistido → relatórios financeiros com base em dados errados = **risco regulatório**.
3. BACEN pode (em tese) mudar formato do header futuramente. Decisão: extrair SHA-256 hex (64 chars), rejeitar header mal-formado.

### 3.4 Headers da request — não enviar Content-Type

Manual §5.3.1 linha 451 e §6.1.1 linha 620: "não deve conter o campo Content-Type".
Já é o default do Go (`http.NewRequest` não seta Content-Type se body for `nil`), mas vou
documentar explicitamente e garantir que nenhum código seta isso por engano.

### 3.5 Erro vs Rejection — semântica dupla

Reaproveita a convenção de `Submit` (Sprint 18):
- Erro de transporte / rede / parse → `err != nil`
- Rejeição formal do BACEN (4xx/5xx com XML Listagem 4) → `err` tipado `*STAError` com `Code` (HTTP status) + `Message` (descrição)

```go
// STAError representa uma rejeição formal do BACEN STA WS (4xx/5xx).
// Distinct de erros de transporte (rede, parse, timeout).
type STAError struct {
    StatusCode int
    Code       string  // código BACEN (Descricao crua) ou "HTTP {N}" fallback
    Message    string
    Protocolo  string  // se conhecido
}

func (e *STAError) Error() string { ... }
```

Erro 404 protocolo inexistente → `&STAError{StatusCode: 404, Message: "Protocolo não encontrado"}`.
Cliente decide se mostra 404 pro usuário ou se é erro operacional.

### 3.6 Limite de tamanho do body do Download

Defense in depth: ZIP CADOC real raramente >50 MB, mas BACEN não documenta limite.
Cap em **100 MiB** via `io.LimitReader`. Acima disso → `*STAError{StatusCode: 413, Message: "body excede 100 MiB"}` (close enough).
(Sub-100MiB porque queremos defesa; CADOC real é ~10MB; BACEN bug → não estoura memória.)

> Decisão reconsiderada: 100 MiB é folgado mas prudente. Se subir arquivo de 200 MB (improvável),
> cliente recebe erro claro. Não vamos silenciosamente truncar — isso quebraria ZIP parsing downstream.

### 3.7 Concorrência / thread-safety

Cada call usa `http.Client.Do()` (thread-safe). Sem mudanças no `WSClient` struct.
Mantém padrão Sprint 18.

## 4. O que NÃO implementar nesta sprint

Pra manter escopo enxuto e entregável verificável:

- **Range/partial download** (Seção 6.4) — útil pra arquivos gigantes, mas CADOC real raramente >10MB. Fica Sprint 21+.
- **Download com `If-Match`/`If-Unmodified-Since`** — otimização condicional, não essencial.
- **`GET /arquivos/disponiveis`** (Seção 8.1.1) — listagem paginada, interface com frontend complexa. Sprint 20.
- **`PUT /arquivos/situacao`** (Seção 7.1) — alteração A_REC↔REC. Sprint 20.
- **Resilience: retry exponencial** — válido mas ortogonal. Sprint 22 ou via wrapper middleware HTTP.
- **Cache de `senhaws` endpoint** (Seção 9.1) — credential rotation, diferente escopo. Sprint 23+.
- **Handlers REST `/v1/sta/download` e `/v1/sta/status`** — wiring no `cmd/api/main.go`. Decidir se Sprint 19 ou 20; ver §5.

## 5. Decisão sobre handlers REST (`/v1/sta/...`)

WSClient é o cliente de baixo nível. Handlers HTTP são ortogonais (auth, rate limit, audit_log).

**Opções:**
- (A) **Sprint 19 só expõe WSClient + testes unit** — handlers ficam pra Sprint 20 quando lista de arquivos a receber for integrada. Mais YAGNI.
- (B) **Sprint 19 expõe + handlers thin** — `GET /v1/sta/status?protocolo=X` e `GET /v1/sta/download?protocolo=X` retornam JSON. Mais imediato.

**Decisão: (A).** Razão: download não é caso de uso V1 do fluxo IF→BACEN (IF manda, BACEN recebe — sem loopback). Status de upload é pré-requisito de retry, mas retry em si é Sprint 21+. Sem callers imediatos, handlers seriam hollow stub.

> **Anti-pattern detectado:** §31 do `go-security-and-quality.md` — endpoint sem caller = waste. Handlers entram quando há consumer real (frontend Sprint 20 listando "arquivos enviados recentemente" ou backend fazendo retry).

## 6. Compatibilidade com StubClient

`StubClient` (Sprint 3) implementa só `Submit`. Não precisa estender a interface — quem chama
`StatusUpload` ou `Download` recebe erro de compilação claro, não ambiguity em runtime.

**Não quebrar interface `Client`** — fica só com `Submit`. Novos métodos só no `*WSClient`.

```go
// Client interface permanece minimal:
// type Client interface { Submit(ctx, sub) (*Result, error) }

// WSClient ganha 2 métodos novos:
// func (c *WSClient) StatusUpload(ctx, protocolo) (*UploadStatus, error)
// func (c *WSClient) Download(ctx, protocolo) (*DownloadResult, error)
```

## 7. Plano de testes

7 cenários httptest, mais `TestNewWSClient` já existente:

1. `TestWSClient_StatusUpload_HappyPath` — 200 + XML com RangesRecebidos e Situacao, parsing correto + enum.
2. `TestWSClient_StatusUpload_RangesEmpty` — `RangesRecebidos=""` e `Situacao=Transmissão não iniciada`.
3. `TestWSClient_StatusUpload_403` — protocolo de outra IF, retorna `*STAError{Code: 403}`.
4. `TestWSClient_StatusUpload_400_XMLError` — BACEN retorna Listagem 4, parseado como `STAError`.
5. `TestWSClient_Download_HappyPath` — 200 + ETag + Last-Modified + X-Content-Hash correto + body ZIP, retorna `DownloadResult` populado.
6. `TestWSClient_Download_HashMismatch` — X-Content-Hash com hash errado → erro fatal `ErrContentHashMismatch`.
7. `TestWSClient_Download_404` — protocolo inexistente → `*STAError{StatusCode: 404}`.
8. `TestWSClient_Download_410_ArquivoNaoDisponivel` — arquivo cancelado/não-disponível.
9. `TestWSClient_Download_BodyTooLarge` — body gigante > 100 MiB → erro `*STAError{StatusCode: 413}`.
10. `TestWSClient_Download_HeaderParsingMalformed` — `X-Content-Hash: md5 abc` (formato errado) → erro.

10 cenários. Foco: conformidade com Seções 5.3.1 + 6.1.1 do manual.

## 8. Critérios de done

- [ ] `WSClient.StatusUpload(ctx, protocolo) (*UploadStatus, error)` implementado
- [ ] `WSClient.Download(ctx, protocolo) (*DownloadResult, error)` implementado
- [ ] Validação X-Content-Hash ativa (sentinel `ErrContentHashMismatch`)
- [ ] `*STAError` type definido para erros formais
- [ ] Cap de 100 MiB no body de download
- [ ] 10 testes httptest cobrindo happy + error paths
- [ ] Doc-comment em cada método citando Seção do manual
- [ ] `gofmt -w .` + `go vet ./...` + `go test ./internal/sta/...` + 18 packages
- [ ] SPRINT_19_RESULTS.md + CHANGELOG v3.9.0
- [ ] commit + push

## 9. Riscos identificados

| Risco | Mitigação |
|---|---|
| BACEN muda header `X-Content-Hash` formato | Sentinel `ErrContentHashHeaderMalformed` separa "header mudou" de "BACEN mandou dado errado" |
| Body gigante DoS | Cap 100 MiB via `io.LimitReader` |
| Cliente chama método em `StubClient` (não tem) | Erro de compilação (interface segregation, sem método comum) |
| Race entre RangeReceived e novo upload do BACEN | Documentar que `StatusUpload` é snapshot momentâneo — caller decide retry policy |
| `ETag` não validado cross-call | Não implementar `If-Match` nesta sprint — só single-call download |

## 10. Referências

- Manual BACEN STA Web Services v1.5 (jul/2022, 42 pp) — `_referencias/STA_Manual_WebServices.pdf`
- Seções extraídas:
  - §2.4 (SHA-256 cross-check)
  - §2.5 (HTTP 1.1 only)
  - §2.6 (limits de conexão — info, sem aplicação direta aqui)
  - §5.3.1 (posicaoupload)
  - §6.1.1 (recebimento completo)
  - §6.4 (recebimento de parte — futuro)
  - Listagem 4 (formato erro XML)
- SPRINT_18_RESEARCH.md — design baseline do WSClient
- SPRINT_18_RESULTS.md — entrega da skeleton
- VALIDATION_v3.8.0_DEEPEST.md — hardening prévio