# SPRINT 20 — Research: Listagem `/arquivos/disponiveis` + Alteração `/arquivos/situacao` + Handlers REST

> **Sprint:** 20 (v3.10.0)
> **Quando:** 2026-07-06
> **Pesquisador:** mavis
> **Status:** pesquisa completa, pronto pra implementação

## 1. Contexto

Sprint 18 (v3.8.0) entregou o **write side** do `WSClient` (`Submit`).
Sprint 19 (v3.9.0) entregou o **read side parcial**: `StatusUpload` (§5.3.1) + `Download` (§6.1.1).

Faltam 2 endpoints BACEN documentados no manual v1.5:
1. **`GET /arquivos/disponiveis`** (Seção 8.1.1) — listar arquivos que BACEN disponibilizou pra IF baixar.
2. **`PUT /arquivos/situacao`** (Seção 7.1) — alterar situação de arquivos (A_REC ↔ REC).

**Diferença importante:** Sprint 19 decidiu conscientemente (SPRINT_19_RESEARCH.md §5) **NÃO** criar handlers REST `/v1/sta/...` por YAGNI. Sprint 20 **inverte a decisão** porque agora há consumer imediato:
- `/arquivos/disponiveis` → frontend precisa listar "arquivos que BACEN mandou pra você" (Sprint 20 mesmo).
- `/arquivos/situacao` → frontend precisa marcar como "Recebido" (UX: limpar inbox).

## 2. Spec BACEN extraída do manual v1.5

### 2.1 `GET /arquivos/disponiveis` — Seção 8.1.1 (linhas 867-969)

**Request:**
```
GET /staws/arquivos/disponiveis?dependencia={D}&dataHoraInicio={YYYY-MM-DDTHH:MM:SS.SSS}&identificadorDocumento={X}&sistemas={SYS1;SYS2} HTTP/1.1
Authorization: Basic {base64(user:pass)}
```

> Atenção (manual linha 878): "O cabeçalho HTTP da requisição **não deve** conter o campo Content-Type."

**Parâmetros (Tabela 4):**

| Parâmetro | Obrigatório | Descrição |
|---|---|---|
| `dependencia` | opcional | Código Sisbacen de uma dependência |
| `dataHoraInicio` | **obrigatório** | Data-hora inicial formato `yyyy-MM-ddTHH:mm:ss.SSS` |
| `identificadorDocumento` | opcional | Nome do tipo de arquivo ou código |
| `sistemas` | opcional | Até 100 sistemas separados por `;` (3 chars cada) |

**Response sucesso (200 OK):**
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado xmlns:atom="http://www.w3.org/2005/Atom">
  <DataHoraProximaConsulta>2012-07-25T10:00:00.001</DataHoraProximaConsulta>
  <Arquivo>
    <Protocolo>3</Protocolo>
    <TipoArquivo>ACOS011</TipoArquivo>
    <CodigoDocumento>1234</CodigoDocumento>
    <Sistema>CCS</Sistema>
    <TamanhoArquivo>753</TamanhoArquivo>
    <Hash>7437b41b04d9984a8b055418a2d99f33e9313c542f8232051a177dd6bbf5d1b1</Hash>
    <SituacaoAtual>
      <Codigo>3</Codigo>
      <Descricao>A receber</Descricao>
    </SituacaoAtual>
    <DataHoraDisponibilizacao>2012-07-21T10:00:00.000</DataHoraDisponibilizacao>
  </Arquivo>
  <atom:link href="..." rel="disponiveis" type="application/octet-stream"/>
</Resultado>
```

**Paginação (linhas 955-957):**
> "essa é uma consulta paginada e trará no máximo 1.000 protocolos. Se existir mais que 1.000 protocolos, o resultado conterá um elemento `atom:link` contendo a url a ser utilizada para a recuperação da próxima página."

**`DataHoraProximaConsulta` (linhas 940-950):**
- 1ms a mais que a última consulta (se houve resultados)
- Próprio `dataHoraInicio` (se vazio)
- `DataHoraDisponibilizacao` da próxima página (se >1000 registros)

**Erros esperados (linhas 960-965):** apenas 400 — XML Listagem 4.

### 2.2 `PUT /arquivos/situacao` — Seção 7.1 (linhas 781-822)

**Request:**
```
PUT /staws/arquivos/situacao HTTP/1.1
Content-Type: application/xml

<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Parametros>
  <Protocolos>1;2</Protocolos>
  <Situacao>A_REC</Situacao>
</Parametros>
```

> Atenção (manual linha 792): "O Content-Type deve ser `application/xml`" (diferente dos outros endpoints).

**`Situacao` valores (linhas 799-801):**
- `A_REC` — a receber
- `REC` — recebido

**`Protocolos`:** lista separada por `;` (ponto-e-vírgula). Manual não limita quantidade — defensivo assumir 1000+ protocolos como máximo prático.

**Response sucesso:** **204 No Content** (sem body).

**Erros esperados:** apenas 400 — XML Listagem 4.

### 2.3 Mapeamento `SituacaoAtual` código ↔ descrição (Tabela 3 não cobre)

A Tabela 3 (linha 1431) lista códigos de **estado** do arquivo (1=Protocolo gerado, 5=Transmissão iniciada, 10=Transmissão finalizada, etc.) — não confundir com `SituacaoAtual`.

Para `SituacaoAtual`, o manual mostra apenas:
- `<Codigo>3</Codigo><Descricao>A receber</Descricao>` (confirmado em múltiplos exemplos)
- `<Codigo>1</Codigo>` aparece sem `Descricao` no manual

**Inferência:** Codigo 1 = "Recebido" (correspondência com REC no parâmetro Situacao da seção 7.1). Manual não é explícito, mas a simetria é forte.

**Decisão:** enum tipado cobre os 2 valores conhecidos + Unknown (defesa contra BACEN adicionar). Caller tem `SituacaoAtualRaw` (string cru) para audit/debug.

## 3. Decisões de design

### 3.1 Quais métodos expor no WSClient?

| Método | Propósito | Caller típico |
|---|---|---|
| `ListDisponiveis(ctx, opts) (*ListDisponiveisResult, error)` | Lista arquivos a partir de `dataHoraInicio` | Handler `/v1/sta/disponiveis` (polling frontend) |
| `AlterarSituacao(ctx, req) error` | Muda situação de N protocolos para A_REC ou REC | Handler `/v1/sta/situacao` (mark-as-read UX) |

**Por que `AlterarSituacao` retorna só `error` e não struct?**
- BACEN responde **204 No Content** (sem body).
- Caller inspeciona `errors.As(err, &staErr)` se falhar.
- Não há sucesso parcial — operação é tudo-ou-nada.

### 3.2 Estruturas de retorno

```go
// ListDisponiveisOpts são os parâmetros de WSClient.ListDisponiveis (Tabela 4).
type ListDisponiveisOpts struct {
    DataHoraInicio         string  // OBRIGATÓRIO — formato "yyyy-MM-ddTHH:mm:ss.SSS"
    IdentificadorDocumento string  // opcional
    Sistemas               string  // opcional — até 100 separados por ";"
    Dependencia            string  // opcional
}

// ListDisponiveisResult é o retorno de WSClient.ListDisponiveis.
type ListDisponiveisResult struct {
    Arquivos                  []ArquivoDisponivel
    DataHoraProximaConsulta   string  // eco do XML; frontend usa pra polling
    ProximaPaginaURL          string  // string vazia se <1000 resultados
    TemProximaPagina          bool    // true se atom:link presente
}

// ArquivoDisponivel é um arquivo da listagem.
type ArquivoDisponivel struct {
    Protocolo                string
    TipoArquivo              string
    CodigoDocumento          string
    Sistema                  string
    TamanhoArquivo           int64
    Hash                     string  // SHA-256 hex
    SituacaoAtual            SituacaoArquivo  // enum tipado
    SituacaoAtualRaw         string  // defesa contra BACEN adicionar valor
    DataHoraDisponibilizacao string  // formato cru BACEN
}

// SituacaoArquivo é o enum tipado dos valores de SituacaoAtual.
type SituacaoArquivo int

const (
    SituacaoArquivoUnknown SituacaoArquivo = iota
    SituacaoArquivoRecebido                  // Codigo 1 (inferido)
    SituacaoArquivoAReceber                  // Codigo 3 (confirmado manual)
)

// AlterarSituacaoReq é o request de WSClient.AlterarSituacao.
type AlterarSituacaoReq struct {
    Protocolos []string  // ["123", "456", ...]
    Situacao   string    // "A_REC" | "REC" — tipado fraco pq só 2 valores
}

// Alternativa: enum tipado
type SituacaoTransferencia int
const (
    SituacaoTransferenciaUnknown SituacaoTransferencia = iota
    SituacaoTransferenciaAReceber  // "A_REC"
    SituacaoTransferenciaRecebido  // "REC"
)
```

**Decisão sobre enum tipado para `AlterarSituacao.Situacao`:** SIM. Mesma justificativa que `UploadSituacao` na Sprint 19 — type-safety contra typos. Helper `String()` retorna "A_REC"/"REC" pra mandar no XML.

### 3.3 Validação de input obrigatória

`dataHoraInicio` é **obrigatório**. Se caller passar string vazia → erro imediato sem chamar BACEN.

Validação de formato? NÃO — manual não diz que BACEN rejeita com erro melhor se formato for errado. Cliente manda o que caller pediu, BACEN responde 400 Listagem 4. Caller ajusta.

**Exceção:** se caller passar formato obviamente inválido (sem `T`), podemos detectar e ajudar com erro mais claro. **Decisão: NÃO** — YAGNI. Caller passa data-hora errada → 400 do BACEN é informativo o suficiente.

### 3.4 Validação `AlterarSituacao`

- `Protocolos` vazio → erro.
- `Situacao` != "A_REC" e != "REC" → erro (não enviar pro BACEN).

### 3.5 Erro vs Rejection — reuso da Sprint 19

Reaproveita `*STAError` (Sprint 19). Erros formais BACEN (4xx com XML Listagem 4) → `*STAError` com `StatusCode`/`Code`/`Message`. Erros de transporte → `err` opaco.

`AlterarSituacao` pode ter erros:
- 400 (BACEN rejeitou XML malformado / protocolo inexistente)
- Erro de transporte (rede, timeout)

### 3.6 Limite de tamanho do response

Listagem de até 1000 arquivos. Cada `<Arquivo>` ≈ 500 bytes. Total ≈ 500 KB. Bem dentro do cap de 10 MiB existente (`maxResponseBodyBytes`).

Não precisa cap novo.

### 3.7 Paginação — passar URL adiante ou re-extrair `DataHoraProximaConsulta`?

**Decisão:** o método retorna `ProximaPaginaURL` **E** `DataHoraProximaConsulta`. Caller escolhe:
- Se quer polling incremental: usa `DataHoraProximaConsulta` na próxima chamada.
- Se quer paginação estrita (1000+ arquivos): usa `ProximaPaginaURL` na próxima chamada.

URL completa (com `?dependencia=...`) é o que BACEN sugere — caller não precisa reconstruir query string.

### 3.8 Thread-safety

Mesmo padrão da Sprint 18/19. Cada call usa `http.Client.Do()` thread-safe. Sem mudanças no WSClient struct.

## 4. Handlers REST — `/v1/sta/...`

Sprint 19 YAGNI. Sprint 20 inverte. **Justificativa:** o frontend precisa consumir para fechar o loop "BACEN→IF" (cadastro de envios enviados, marcar como recebido, baixar arquivo).

### 4.1 Endpoints REST

| Method | Path | WSClient method | Auth | Rate limit |
|---|---|---|---|---|
| `GET` | `/v1/sta/disponiveis?dataHoraInicio=...&sistemas=...` | `ListDisponiveis` | JWT (enforceSameIF) | sim |
| `POST` | `/v1/sta/situacao` body `{"protocolos":["1"],"situacao":"REC"}` | `AlterarSituacao` | JWT (enforceSameIF) | sim |

### 4.2 Auth + tenant isolation

Mesmo padrão dos outros handlers `/v1/...`:
- JWT RS256 obrigatório.
- `enforceSameIF` — `s.STAClient` é global (1 client por app), mas o handler extrai `if_id` do JWT e passa como constraint via query string `dependencia` quando caller não fornece.

### 4.3 Audit emission

Toda chamada (sucesso + erro) emite audit_log:
- Sucesso: `sta.disponiveis.listed`, `sta.situacao.changed`
- Erro 4xx: `sta.{op}.rejected` com `staErr.StatusCode` + Message
- Erro 5xx/rede: `sta.{op}.failed` com `err.Error()` sanitizado (F18.1)

### 4.4 Rate limiting

Aplicar padrão existente (`ratelimit_redis.go` Lua INCR+EXPIRE). 60 req/min por IF pra `disponiveis` (polling-friendly), 10 req/min pra `situacao` (operação de baixa frequência).

### 4.5 Wire no `cmd/api/main.go`

`staClient` já está no construtor `api.NewServer(...)`. Handlers vão em arquivo novo `sprint20_handlers.go` (paralelo ao `sprint8c_handlers.go`). Wire em `api.NewServer` se necessário.

### 4.6 Wire no `internal/api/server.go`

`server.go` já tem campo `STAClient sta.Client` (interface). Sprint 20 **NÃO** muda a interface (mantém só `Submit`) — os handlers vão precisar fazer **type assertion** para `*sta.WSClient`:

```go
ws, ok := s.STAClient.(*sta.WSClient)
if !ok {
    // StubClient não tem ListDisponiveis/AlterarSituacao.
    // Erro 503 ou fallback mock.
}
```

**Problema:** isso quebra o padrão de interface (`Client` tem só `Submit`). **Alternativas:**
- (A) Estender `Client` interface com `ListDisponiveis` + `AlterarSituacao` (força StubClient a implementar).
- (B) Type assertion em runtime.
- (C) Criar nova interface `ReadClient { ListDisponiveis(); AlterarSituacao() }` separada.

**Decisão: (A) é errado** — StubClient não tem como listar/alterar (não tem BACEN real). Forçar implementação vazia cria hollow stub.

**Decisão: (C)** — interface segregation. `Client` continua minimal. `ReadClient` é opt-in. Handler checa se `s.STAClient` implementa `ReadClient` via type assertion; se sim usa, senão retorna 503.

```go
type ReadClient interface {
    ListDisponiveis(ctx, opts) (*ListDisponiveisResult, error)
    AlterarSituacao(ctx, req) error
}

// Em handler:
if rc, ok := s.STAClient.(sta.ReadClient); ok {
    return rc.ListDisponiveis(ctx, opts)
}
// Senão: 503 Service Unavailable com mensagem clara
```

Vou colocar `ReadClient` interface em `internal/sta/client.go` ou `internal/sta/ws.go`. Apenas `*WSClient` implementa.

### 4.7 Resposta JSON

```json
// GET /v1/sta/disponiveis?dataHoraInicio=...
{
  "arquivos": [
    {
      "protocolo": "3",
      "tipo_arquivo": "ACOS011",
      "codigo_documento": "1234",
      "sistema": "CCS",
      "tamanho_arquivo": 753,
      "hash": "7437...",
      "situacao_atual": "A_RECEBER",  // ou "RECEBIDO", ou "DESCONHECIDA"
      "data_hora_disponibilizacao": "2012-07-21T10:00:00.000"
    }
  ],
  "data_hora_proxima_consulta": "2012-07-25T10:00:00.001",
  "proxima_pagina_url": "",
  "tem_proxima_pagina": false
}

// POST /v1/sta/situacao body
{
  "protocolos": ["1", "2"],
  "situacao": "REC"
}
// Response: 204 No Content
```

## 5. Compatibilidade com StubClient

`StubClient` implementa só `Submit` (não muda). Handlers REST Sprint 20 verificam `sta.ReadClient` via type assertion — se `s.STAClient` é `*StubClient`, retorna 503 com mensagem "endpoint requer WSClient contra BACEN real, default backend=stub".

**Default `RADIANT_STA_BACKEND=stub`** preserva comportamento das 19 sprints anteriores. Setar `RADIANT_STA_BACKEND=ws` ativa endpoints.

## 6. Plano de testes

### 6.1 WSClient unit tests (httptest)

| Test | Cobre |
|---|---|
| `TestWSClient_ListDisponiveis_HappyPath` | §8.1.1 com 3 arquivos + `DataHoraProximaConsulta` + sem atom:link |
| `TestWSClient_ListDisponiveis_Paginated` | §8.1.1 com 1001 arquivos (atom:link presente) → `TemProximaPagina=true` |
| `TestWSClient_ListDisponiveis_Empty` | §8.1.1 com 0 arquivos |
| `TestWSClient_ListDisponiveis_400` | BACEN rejeita → `*STAError{StatusCode: 400}` |
| `TestWSClient_ListDisponiveis_DataHoraVazia` | Sanity check defensivo |
| `TestWSClient_AlterarSituacao_HappyPath` | §7.1 com 2 protocolos A_REC → 204 No Content |
| `TestWSClient_AlterarSituacao_REC` | §7.1 alterando para REC (segundo valor oficial) |
| `TestWSClient_AlterarSituacao_400` | BACEN rejeita → `*STAError{StatusCode: 400}` |
| `TestWSClient_AlterarSituacao_ProtocolosVazios` | Sanity check defensivo |
| `TestWSClient_AlterarSituacao_SituacaoInvalida` | Sanity check (não "A_REC"/"REC") |
| `TestParseSituacaoArquivo_Cases` | Tabela enum (Codigo 1 / 3 / desconhecido) |
| `TestSituacaoTransferencia_String_Cases` | "A_REC"/"REC"/Unknown |

### 6.2 Handlers REST integration tests (httptest + chi router)

| Test | Cobre |
|---|---|
| `TestHandler_ListDisponiveis_OK` | GET com JWT válido + WSClient mock → 200 JSON |
| `TestHandler_ListDisponiveis_StubBackend` | Backend=stub → 503 com mensagem clara |
| `TestHandler_ListDisponiveis_JWTInvalido` | Sem JWT → 401 |
| `TestHandler_ListDisponiveis_DataHoraVazia` | Sem dataHoraInicio → 400 |
| `TestHandler_AlterarSituacao_OK` | POST com body válido → 204 |
| `TestHandler_AlterarSituacao_BodyInvalido` | JSON malformado → 400 |
| `TestHandler_AlterarSituacao_StubBackend` | Backend=stub → 503 |
| `TestHandler_AlterarSituacao_BACENRejeita` | WSClient mock retorna 400 → 502 (bad gateway) ou 400? |

### 6.3 Smoke E2E

Smoke já existente (`TestSmoke` em `internal/api/smoke_v352_test.go`) deve passar — handler não muda fluxo de `Submit`.

## 7. Critérios de done

- [ ] `WSClient.ListDisponiveis(ctx, opts) (*ListDisponiveisResult, error)` implementado
- [ ] `WSClient.AlterarSituacao(ctx, req) error` implementado
- [ ] `ReadClient` interface segregation (não estender `Client`)
- [ ] Handlers `GET /v1/sta/disponiveis` + `POST /v1/sta/situacao` em arquivo `sprint20_handlers.go`
- [ ] Auth JWT + `enforceSameIF` em ambos
- [ ] Rate limiting (60/min disponiveis, 10/min situacao)
- [ ] Audit emission em sucesso + erro
- [ ] 12 unit tests WSClient + 8 integration tests handlers = 20 testes novos
- [ ] 18/18 packages PASS + smoke + gofmt/vet
- [ ] SPRINT_20_RESEARCH.md (este) + SPRINT_20_RESULTS.md + CHANGELOG v3.10.0
- [ ] Commit + push

## 8. Riscos identificados

| Risco | Mitigação |
|---|---|
| BACEN retorna Codigo SituacaoAtual fora de {1, 3} (não documentado) | Enum `SituacaoArquivoUnknown` + `SituacaoAtualRaw` para string crua |
| URL próxima página tem assinatura diferente do esperado | `ProximaPaginaURL` é string crua, caller decide se usa |
| `dataHoraInicio` formato inválido (caller bug) | BACEN retorna 400, error tipado `*STAError` |
| Race: poll frontend + marcar como recebido concorrente | Backend usa DB transaction (audit_log append-only) — fora do escopo WSClient |
| Rate limit muito permissivo (60/min) e IFs pequenas hammeram | 60/min é suficiente pra polling 1Hz; se virar problema, reavaliar Sprint 22+ |

## 9. O que NÃO entra nesta sprint

- Range/conditional download (§6.4) — Sprint 21.
- Retry exponencial — Sprint 22.
- Senhaws rotation (§9.1) — Sprint 23.
- Smoke contra BACEN homolog real — Sprint 24 (precisa credenciais Sisbacen).
- Range/parallel upload (§5.5+5.6) — Sprint 21+.

## 10. Referências

- Manual BACEN STA Web Services v1.5 (jul/2022, 42 pp) — `_referencias/STA_Manual_WebServices.pdf`
- Seções extraídas:
  - §7.1 (alteração situação)
  - §8.1.1 (consulta arquivos disponíveis)
  - Tabela 3 (códigos estado do arquivo — para contexto)
  - Tabela 4 (parâmetros consulta disponíveis)
- SPRINT_18_RESEARCH.md — design baseline WSClient
- SPRINT_19_RESEARCH.md — decisões read side + YAGNI handlers (Sprint 20 inverte)
- SPRINT_19_RESULTS.md — padrões estabelecidos (typed enum, *STAError, sentinel errors)
- VALIDATION_v3.9.0_DEEPEST.md — padrões reforçados (stdlib errors.As, hex.DecodeString)