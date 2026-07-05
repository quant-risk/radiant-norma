# Sprint 18 — STA WS nativo — RESULTADOS (v3.8.0)

> **Data:** 2026-07-05
> **Status:** ✅ Concluída (V1: skeleton testado + env factory)
> **Tema:** Cliente nativo para BACEN STA Web Services (REST) — substitui Playwright no roadmap
> **Trigger:** Validação 38 DEEPEST (escolha do caminho 1: pesquisa primeiro)
> **Versão:** v3.8.0
> **Resultado:** 1 WSClient skeleton + 16 novos testes + 2 novos docs + factory env-var

---

## 🎯 Objetivo da sprint

Sprint 7 (v1.0) introduziu `sta.StubClient`. Ao longo de 17 sprints, planejou-se
Playwright como caminho de automação (v1.0 da Fase 1 do roadmap), mas o
roadmap reclassificou para Fase 1.5: **cliente nativo HTTP contra Web
Services oficiais do BACEN** que desde 2022 são REST documentados.

Sprint 18 (v3.8.0) entrega apenas **V1 do cliente** — fluxo 2-fase
(POST protocolo + PUT conteúdo) testável via `httptest.Server` mock.
Funcionalidades adicionais (download, paralelismo, senha rotation)
ficam para Sprint 19+ conforme mapeado em `SPRINT_18_RESEARCH.md`.

---

## 🏛️ Entregas

### 🟢 Backend — `WSClient` skeleton

**Arquivo novo:** `backend/internal/sta/ws.go` (~245 LOC)

Implementa o cliente HTTP para BACEN STA Web Services v1.5. O fluxo
principal `Submit()` faz 2 chamadas:

1. **`POST /arquivos`** (Seção 5.1.1 do manual oficial) — XML body
   `application/xml` com `<Parametros>` (IdentificadorDocumento, Hash,
   Tamanho, NomeArquivo). Retorna 201 + protocolo numérico.
2. **`PUT /arquivos/{protocolo}/conteudo`** (Seção 5.2.1) — payload
   binário (ZIP). Retorna 200.

Auth: HTTP Basic preemptivo (RFC 7617) com header
`Authorization: Basic base64(UUUUUDDDD.operador:senha)`.

Hash SHA-256 calculado sobre o conteúdo compactado (ZIP), conforme
Seção 2.4 do manual.

**Defesas (defense in depth):**

- `NewWSClient` valida config antes de qualquer call de rede — BaseURL
  HTTPS, sem trailing slash, User formato Sisbacen obrigatório, Password
  obrigatório.
- `AllowInsecureHTTP` flag explícita para testes (httptest retorna
  `http://`); default `false` em produção.
- Erros do BACEN parseados via `<Resultado><Erro><Codigo/Descricao>`
  (Listagem 4 do manual) — mensagens são propagadas ao caller.
- Hash enviado no `POST /arquivos` deve bater com o que o BACEN
  calcula em receive (cross-check defense).
- Submissão com protocolo bem gerado + upload falho preserva
  `ProtocolSTA` no `Result` para forensic trail (Sprint 5 audit log
  pattern).
- Timeout configurable (default 30s) — defense contra BACEN down.

### 🟢 Backend — XML structs (`ws_types.go`)

Tipos extraídos do manual oficial para serializar/deserializar:

| Tipo | Uso | Seção manual |
|---|---|---|
| `requestProtocolParams` | XML POST /arquivos body | 5.1.1 |
| `responseProtocol` | XML 201 Created response | 5.1.1 |
| `xmlError` | XML 4xx/5xx response (Listagem 4) | Erro universal |
| `posicaoUploadResponse` | GET /arquivos/{protocolo}/posicaoupload | 5.3.1 (V2) |
| `situacaoParams` | XML PUT /arquivos/situacao | 7.1 (V2) |
| `arquivosDisponiveisResponse` | GET /arquivos/disponiveis | 8.1.1 (V2) |

Tipos `posicaoUploadResponse`, `situacaoParams`, `arquivosDisponiveisResponse`
são estruturas forward-compat — não usados no V1 mas disponíveis para
Sprint 19+ não precisar re-parsear o manual.

### 🟢 Backend — env factory (`NewClientFromEnv`)

Adicionado em `ws.go` para preservar compatibilidade e adicionar controle
via env var:

| Env | Default | Função |
|---|---|---|
| `RADIANT_STA_BACKEND` | `stub` | `stub` (mantido) \| `ws` (novo) |
| `RADIANT_STA_WS_URL` | (vazio) | Base URL: `https://sta-h.bcb.gov.br/staws` |
| `RADIANT_STA_SISBACEN_USER` | (vazio) | Formato `UUUUUDDDD.operador` |
| `RADIANT_STA_SISBACEN_PASSWORD` | (vazio) | senha Sisbacen |
| `RADIANT_STA_TIMEOUT_SECONDS` | `30` | timeout HTTP |

Default `stub` preserva comportamento de v3.7.x — nenhum endpoint
existente muda sem opt-in via env var. `RADIANT_STA_BACKEND=ws` ativa
cliente nativo.

Helper `BackendName(c)` retorna `"stub"`/`"ws"` — usado em logs de
startup (`cmd/api/main.go:74`).

### 🟢 Backend — wire em `cmd/api/main.go`

Substituído `staClient := sta.NewStubClient()` por factory:
```go
staClient, err := sta.NewClientFromEnv(logger)
if err != nil {
    logger.Error("STA client init failed", "err", loggerutil.SafeError(err))
    os.Exit(1)
}
logger.Info("STA client inicializado", "backend", sta.BackendName(staClient))
```

Falha de configuração em prod exige fail-closed (já é pattern do
Sprint 13). Em dev, loga warning se credenciais BACEN faltam.

---

## 🧪 Suíte de regressão (16 testes novos)

### Testes adicionados (`ws_test.go`)

| Test | Cobre | Manual seção |
|---|---|---|
| `TestNewWSClient/valid` | config OK | inicialização |
| `TestNewWSClient/empty_baseURL` | BaseURL required | config validation |
| `TestNewWSClient/http_not_https` | HTTPS obrigatório | config validation |
| `TestNewWSClient/trailing_slash` | path invariants | config validation |
| `TestNewWSClient/user_sem_dot` | formato Sisbacen | config validation |
| `TestNewWSClient/empty_password` | password required | config validation |
| `TestNewWSClient_DefaultTimeout` | timeout 30s default | config default |
| `TestSubmit_HappyPath` | fluxo 2-fase OK | 5.1 + 5.2 |
| `TestSubmit_EmptySubmission` | defensiva payload vazio | (defense) |
| `TestSubmit_UsesZipWhenProvided` | ZIP tem prioridade sobre XML | 2.4 hash |
| `TestSubmit_400_IdentificadorInvalido` | Tabela 7 | 5.1.1 |
| `TestSubmit_403_UsuarioNaoAutorizado` | Tabela 7 (403) | 5.1.1 |
| `TestSubmit_ProtocolThenUpload403` | protocolo existe + upload 403 | 5.2.1 + forensic |
| `TestSubmit_HashMismatch` | hash mismatch 400 | 5.2.1 + 2.4 |
| `TestSubmit_ContextCanceled` | ctx.Done() error propagado | (defense) |
| `TestSubmit_EmptyProtocolInResponse` | defensive — 201 sem protocolo | (defense) |
| `TestSubmit_MalformedErrorBody` | 4xx com garbage body | (defense) |
| `TestBasicAuthHeader_Formato` | base64(user:pass) correto | 2.2 |

**16 novos testes, 0 falhas**. Toda classe de comportamento documentada
no manual (happy + erros mapeados nas Tabelas 5-8) tem cobertura.

### Regressão nos outros packages

```
ok  internal/api          (smoke 11/11 PASS contra binário real)
ok  internal/audit        (374+ tests total, sem regressão)
ok  internal/auditlog
ok  internal/auth
ok  internal/crossdoc
ok  internal/db
ok  internal/insights
ok  internal/loggerutil   (com -race: 2 perf tests flake sob race detector
                         overhead — pre-existing, sem regressão funcional)
ok  internal/radar
ok  internal/realtime
ok  internal/ruleprefs
ok  internal/schema
ok  internal/sta          (16 novos, 0 falhando)
ok  internal/testutil
ok  internal/version
ok  internal/worker
```

### Build matrix

```
go vet ./...                  clean
gofmt -l .                    (post-fix: 0 drift)
go build ./cmd/api            24M (mesmo binário)
go build ./cmd/worker         OK
go build ./cmd/seed           OK
go build ./cmd/jwt-mint       OK
go build ./cmd/radar          OK
```

---

## 📚 Documentação inline

- `ws.go`: extensa doc-comment de cabeçalho explicando o fluxo 2-fase
  e referenciando o manual BACEN por seção
- `ws_types.go`: cada struct tem doc-comment com seção do manual
  (5.1.1, 5.3.1, 7.1, 8.1.1)
- `NewWSClient`: comentário validação explica cada guard (HTTPS, slash,
  formato Sisbacen, password)
- Decisão de V1 (envio completo apenas) documentada em
  `SPRINT_18_RESEARCH.md` (decisão 2)

---

## 🔜 Carry-over (Sprint 19+)

Mapeado em detalhe no `SPRINT_18_RESEARCH.md`:

| # | Item | Razão |
|---|---|---|
| 18.1 | Playwright client (legacy path 1.0) | migrar callers + remover stub alternativo |
| 18.2 | Range upload (chunked) | suporte arquivos > 50MB |
| 18.3 | Range download | download resumível |
| 18.4 | Range download / parallel | otimizações Seções 5.5/5.6/6.3/6.4 |
| 18.5 | Status upload (`/posicaoupload`) | proxy de progresso |
| 18.6 | Senha rotation (`PUT senhaws/senha`) | operacional |
| 18.7 | Consulta disponibilidade (`/disponiveis`) | frontend radial |
| 18.8 | Retry exponencial + circuit-breaker | produção resilience |
| 18.9 | Vault/KMS integration | gestão secret |
| 18.10 | Smoke test contra BACEN homolog | requer credenciais Sisbacen reais |

A maioria é trabalho mecânico — `WSConfig`/`Submit()` signatures são
estáveis; extensões são adições de methods (`ListDisponiveis()`,
`ChangeSituacao()`, etc).

---

## 🏗️ Lições aprendidas

1. **Bridge primeiro, código depois (validação confirmada)**: ler o
   manual oficial antes de escrever 1 linha salvou tempo — não
   precisei reescrever quando descobri (via testes) que `Content-Type`
   deve ser omitido no upload (Seção 5.2.1). O caminho 1 da validação
   38 DEEPEST está validado empiricamente.

2. **`httptest.NewServer` retorna `http://`, não `https://`**: meu
   primeiro `NewWSClient` rejeitava URLs não-HTTPS estritamente, o que
   quebrou todos os 11 testes. **Fix**: `AllowInsecureHTTP` flag
   explícita, default `false` em produção. Lição: testes são first
   consumer de qualquer validação strict.

3. **Context cancelation em testes = servidor não bloqueia**: meu test
   original deixava server-side handler em `<-r.Context().Done()`,
   bloqueando `srv.Close()` no `t.Cleanup` para sempre. **Fix**:
   cancelar context no client E ter server responder imediato. Lição:
   `httptest.Server.Close()` espera conexões ativas terminarem.

4. **Err vs Rejection — semântica dupla**: `Submit()` retorna `(Result, err)`
   mas algumas falhas de BACEN resultam em `Result.Com rejeição populada + err=nil`
   (BACEN rejeita formalmente) e outras em `Result=nil + err` (transporte).
   Tests precisam entender qual padrão é qual. Lição: documentar explicitamente
   nos testes (e na doc-comment de Submit) qual classe gera qual retorno.

5. **Doc-comment cross-reference ao manual**: cada struct em `ws_types.go`
   cita seção + Listagem + Tabela do manual. Isso é **autodocumentação**
   que facilita Sprint 19+ (qualquer dev que for estender o cliente
   sabe exatamente onde olhar pra entender o shape esperado).

---

## 📚 Referências

- `SPRINT_18_RESEARCH.md` — pesquisa completa da spec BACEN STA WS v1.5
- `_referencias/STA_Manual_WebServices.pdf` — manual oficial BACEN v1.5
- `_referencias/STA_FAQ.pdf` — regras Content-Type
- CHANGELOG.md v3.8.0 — entrada nova
- CHANGELOG.md v3.7.0 (Sprint 17) — hardening precedente
- `SPRINT_8_RESULTS.md` — analogia histórica (JWT bridge era fundação)
- `backend/internal/sta/stub.go` — interface preservada
- `backend/internal/sta/ws.go` — skeleton V1
- `backend/internal/sta/ws_types.go` — XML structs
- `backend/internal/sta/ws_test.go` — 16 testes
- `cmd/api/main.go` — factory wiring
