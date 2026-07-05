# Sprint 18 — STA WS nativo (BACEN REST) — Research + Skeleton

> **Status:** 📋 Em andamento
> **Data:** 2026-07-05
> **Tema:** Cliente nativo para BACEN STA Web Services (REST + Basic Auth), substituir Playwright
> **Trigger:** Roadmap Fase 1.5 do produto + caminho 1 escolhido na validação 38 DEEPEST (pesquisa primeiro)
> **Versão alvo:** v3.8.0

---

## 🎯 Por que Sprint 18 é STA WS primeiro, nada de hardening

Sprint 7 (v1.0) introduziu o stub STA (`internal/sta/stub.go`). Por 8 sprints
planejou-se Playwright como caminho para automação (Fase 1), mas o roadmap
reclassificou como Fase 1.5: substituir Playwright por cliente nativo HTTP
contra os **Web Services oficiais** do BACEN, que desde 2022 são REST
documentados.

Bridges são fundação, não otimização — última vez que pulamos o wire-up
(frontend JWT), Sprint 17 (Validação 37) encontrou bug cross-tenant em
`devTokenHandler` que tinha passado em Sprint 13. Esta sprint investe em
**research + skeleton testado via httptest** para garantir que a fundação
esteja sólida antes de chegar credenciais Sisbacen reais (Sprint 19+).

---

## 📚 Pesquisa: BACEN STA Web Services v1.5

### Fontes (4 fontes cruzadas)

1. **Manual oficial v1.5** (`_referencias/STA_Manual_WebServices.pdf` — 42
   páginas, julho/2022) — BACEN, recuperado localmente do repositório
2. **Manual oficial online** —
   https://www.bcb.gov.br/content/acessoinformacao/sisbacen_docs/Manual_STA_Web_Services.pdf
3. **FAQ oficial** (`_referencias/STA_FAQ.pdf` — 6 páginas) — campo `Content-Type` rules
4. **Reference implementation** —
   https://github.com/aleDsz/bacen_sta (Elixir, MIT, 2 estrelas) —
   abstração `create_protocol/1` + `send_file_content/2`

### Endpoints (REST, `/staws`)

| # | Método | Path | Função | Auth | Body | Resposta |
|---|---|---|---|---|---|---|
| 1 | `POST` | `/arquivos` | Requisição de protocolo (inicia envio 2-fase) | Basic | XML `<Parametros>` | 201 + XML `<Resultado><Protocolo>` |
| 2 | `PUT` | `/arquivos/{protocolo}/conteudo` | Upload completo do conteúdo binário | Basic | binário (ZIP) | 200 OK |
| 3 | `GET` | `/arquivos/{protocolo}/conteudo` | Download completo (recebimento) | Basic | — | 200 + binário + `X-Content-Hash` SHA-256 |
| 4 | `GET` | `/arquivos/{protocolo}/posicaoupload` | Status upload (ranges recebidos) | Basic | — | 200 + XML |
| 5 | `GET` | `/arquivos?tipoConsulta=PROT&protocolos=X&nivelDetalhe=RES` | Consulta por protocolos | Basic | — | 200 + XML |
| 6 | `GET` | `/arquivos?tipoConsulta=AVANC&nivelDetalhe=RES` | Consulta avançada com filtros | Basic | — | 200 + XML |
| 7 | `GET` | `/arquivos/disponiveis?dependencia=X&dataHoraInicio=Y` | Arquivos disponíveis | Basic | — | 200 + XML |
| 8 | `PUT` | `/arquivos/situacao` | Alterar situação `A_REC` ↔ `REC` | Basic | XML `<Parametros>` | 204 No Content |
| 9 | `GET` | `/arquivos/{protocolo}/conteudo` (Range header) | Range download | Basic | — | 206 Partial Content |
| 10 | `PUT` | `/arquivos/{protocolo}/conteudo` (Content-Range) | Range upload (chunked) | Basic | binário + header | 200 OK |

### Endpoints separados (senhaws — gestão de senha)

| # | Método | URL | Função |
|---|---|---|---|
| 11 | `PUT` | `https://www9.bcb.gov.br/senhaws/senha` (homol) / `https://www3.bcb.gov.br/senhaws/senha` (prod) | Alterar senha do usuário |
| 12 | `GET` | `.../senha/vencimento` | Consultar dias até vencimento |

### Autenticação

**HTTP Basic preemptivo** (RFC 7617):
- Header: `Authorization: Basic base64(UUUUUDDDD.operador:senha)`
- `UUUUU` = código Sisbacen da instituição (5 dígitos)
- `DDDD` = código Sisbacen da dependência (4 dígitos)
- `operador` = username do usuário
- Pré-requisitos de acesso: usuário Sisbacen/Autran + credenciamento no
  serviço PSTA300 (transação no próprio Sisbacen)

### Hashing

`SHA-256` sobre o conteúdo **compactado** (não XML cru):
- Se o arquivo for enviado compactado (ZIP), o hash é calculado sobre o ZIP
- Hash é hexadecimal 64 chars (256 bits)
- Validado no `POST /arquivos` (parâmetro `Hash`) e novamente na response
  do `GET` (header customizado `X-Content-Hash`)

### Limites (por IF, não por usuário)

- Upload/recebimento simultâneo: **10**
- Consultas por minuto: **120**

### Content-Type rules (do FAQ)

| Operação | Content-Type |
|---|---|
| Consulta | `application/x-www-form-urlencoded` |
| Requisição de protocolo (`POST /arquivos`) | `application/xml` |
| Upload (`PUT /arquivos/{protocolo}/conteudo`) | `application/octet-stream` |
| Alteração de situação (`PUT /arquivos/situacao`) | `application/xml` |

Sem Content-Type o BACEN assume default. `multipart/form-data` **não é
permitido** em upload.

### TLS / Certificados

- **Servidor**: BACEN emite certs válidos; cliente deve confiar na cadeia
  descrita em https://www.bcb.gov.br/estabilidadefinanceira/certificacaodigital
- **Cliente**: **NÃO requer** certificado A1/A3 do cliente. Diferente de
  Receita Federal / e-CAC, o BACEN usa apenas Basic Auth + TLS server-side.
  Isso simplifica enormemente vs Playwright (que automatizava browser
  porque browser tinha cert A1).

### Estados do arquivo (codigoEstado — Tabela 3 do manual)

| Cod | Descrição |
|---|---|
| 1 | Protocolo gerado |
| 2 | Arquivo disponível para download |
| 5 | Transmissão iniciada |
| 10 | Transmissão finalizada |
| 15 | Arquivo em processo de montagem ou validação |
| 20 | Arquivo recebido no BACEN |
| 25 | Arquivo entregue para o destinatário |
| 30 | Arquivo em processamento pela aplicação de negócio |
| 35 | Arquivo aceito |
| 45 | Arquivo cancelado |
| 55 | Arquivo inconsistente |
| 65 | Arquivo rejeitado |
| 70 | Download iniciado |
| 75 | Download finalizado |

**Sem deadline explícito**: protocolo expira em **48 horas** se transmissão
não for iniciada (Seção 5 do manual).

### Suporte oficial

- Telefone: (61) 3414-2156
- Email: `suporte.ti@bcb.gov.br`
- Mesa de Auxílio do Banco Central (AMB)

---

## 🏛️ Decisões de design para Sprint 18

### Decisão 1 — Pesquisa antes de código (escolha do caminho 1)

**Sem credenciais Sisbacen, código contra BACEN real não pode ser testado
em ambiente isolado.** Por isso, o sprint 18 entrega:

1. **Spec documentada** (`SPRINT_18_RESEARCH.md` — este arquivo)
2. **WSClient skeleton** (`backend/internal/sta/ws.go`) com assinatura
   completa + implementação do 2-fase (POST + PUT), **sem** chamada real
3. **Testes via `httptest.Server`** mockando o BACEN (`ws_test.go`) — prova
   que o cliente faria a chamada correta sem precisar de credenciais
4. **Wire via env var** `RADIANT_STA_BACKEND=ws|playwright|stub` (default
   `stub` preservado para Sprint 18+)

Playwright stub continua existindo (não removemos). Remote-elimination
será Sprint 19+.

### Decisão 2 — Envio completo em vez de range upload no V1

Manual suporta **4 modos** de upload (5.2 Envio completo, 5.4 Retomada,
5.5 Paralelo, 5.6 Parte). Para V1, **só envio completo** (5.2):

- Implementação trivial (não precisa de Content-Range)
- Suficiente para SCDs/IPs típicos (envios < 50 MB)
- Range + parallel + resume: Sprint 19+

### Decisão 3 — Sem retry/circuit-breaker no V1

WSClient V1 implementa "fire-and-forget" — chama BACEN, propaga erro se
ocorrer. Retry exponencial + circuit-breaker: Sprint 19+ (junto com worker
de retry que precisa coordenar com cmd/worker atual).

### Decisão 4 — Credenciais via env vars (não em DB)

`RADIANT_STA_SISBACEN_USER`, `RADIANT_STA_SISBACEN_PASSWORD`, etc.
Rationale: secret management centralizado (já temos outras senhas via
env — `RADIANT_NORMA_ADMIN_TOKEN`, `RADIANT_DEV_JWT_PRIVATE_KEY`).
Vault/KMS integration: Sprint 19+.

### Decisão 5 — Compat binária vs textual em XML

Manual v1.5 (julho/2022) usa UTF-8 explicit. XML request bodies:
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Parametros>
  <IdentificadorDocumento>{tipo}</IdentificadorDocumento>
  <Hash>{hash_sha256_64}</Hash>
  <Tamanho>{bytes}</Tamanho>
  <NomeArquivo>{nome}</NomeArquivo>
</Parametros>
```

Resposta também XML. XML namespace `atom` aparece só em respostas que
têm paginação (e.g., `/arquivos/disponiveis` com 1.000+ protocolos).

### Decisão 6 — Manter interface `Client` existente

`Client` interface em `stub.go:42`:
```go
type Client interface {
    Submit(ctx context.Context, sub *Submission) (*Result, error)
}
```

Vou preservar — WSClient implementa mesma interface. `cmd/api/main.go`
faz factory baseado em env var.

---

## 🏗️ Arquitetura proposta

```
backend/internal/sta/
├── stub.go         (manter — fallback)
├── ws.go           (novo — WSClient)
├── ws_test.go      (novo — httptest mock)
├── ws_types.go     (novo — XML structs request/response)
└── auth.go         (novo — Basic Auth helper)

cmd/api/main.go:
  factory:
    if RADIANT_STA_BACKEND=ws:
      sta.NewWSClient(cfg)
    elif RADIANT_STA_BACKEND=playwright:
      sta.NewPlaywrightClient(cfg)
    else:
      sta.NewStubClient()        # default
```

### Nova env vars

| Nome | Default | Função |
|---|---|---|
| `RADIANT_STA_BACKEND` | `stub` | `stub`/`ws`/`playwright` |
| `RADIANT_STA_WS_URL` | (vazio) | Base URL: `https://sta-h.bcb.gov.br/staws` |
| `RADIANT_STA_SISBACEN_USER` | (vazio) | `UUUUUDDDD.operador` |
| `RADIANT_STA_SISBACEN_PASSWORD` | (vazio) | senha Sisbacen |
| `RADIANT_STA_TIMEOUT_SECONDS` | `30` | timeout de chamada HTTP |
| `RADIANT_STA_FAIL_OPEN` | `false` | se true, treat BACEN 5xx como aceitar (fail-open) |

### Defesa em profundidade (mantido de Sprints 13/15/17)

1. **Fail-closed em prod**: se `RADIANT_ENV=production` + `RADIANT_STA_BACKEND=ws`
   mas URL/user/password vazios → FATAL exit (mesmo padrão do JWT public
   key dev/77 F17)
2. **Audit log**: cada Submit emite `sta.submit.{success|failure}` no
   auditlog com metadata `{protocolo, tentativa, latencia_ms}` — mesmo
   formato que `sta.submit` atual (Sprint 5)
3. **Sanitize error**: `err.Error()` em resposta passa por `SafeError` —
   information disclosure defense F18.1
4. **Tenant guard**: `enforceSameIF` continua fechando cross-tenant no
   handler — WSClient confia em quem chama, não valida tenant (única
   invariante real é que BACEN rejeita 403 se protocolo não pertence à
   IF)

---

## 📦 Deliverables Sprint 18 (v3.8.0)

### 1. Spec documentada

Este arquivo — base para qualquer implementação futura.

### 2. `backend/internal/sta/ws.go` (skeleton)

Assinatura completa:
```go
package sta

type WSConfig struct {
    BaseURL    string  // "https://sta-h.bcb.gov.br/staws"
    User       string  // "UUUUUDDDD.operador"
    Password   string
    Timeout    time.Duration
    HTTPClient *http.Client  // opcional, para injetar em testes
    Logger     *slog.Logger
}

type WSClient struct { ... }

func NewWSClient(cfg WSConfig) (*WSClient, error)
func (c *WSClient) Submit(ctx context.Context, sub *Submission) (*Result, error)

// Helpers internos (não exportados):
func (c *WSClient) requestProtocol(ctx context.Context, sub *Submission) (protocolo string, err error)
func (c *WSClient) uploadContent(ctx context.Context, protocolo string, content []byte) error
```

### 3. `backend/internal/sta/ws_types.go` (XML structs)

Structs Go pra serializar os XMLs do BACEN (request/response).

### 4. `backend/internal/sta/ws_test.go` (~ 15 testes)

Mock do BACEN via `httptest.Server`:
- Happy path: 2-fase (POST retorna 201 + protocolo; PUT retorna 200)
- Erro 400 (IdentificadorDocumento inválido) → mapped para `Rejection`
- Erro 403 (protocol não pertence à IF) → mapped para `Rejection`
- Erro 404 (protocol não existe) → mapped para `Rejection`
- Timeout → retorna erro (sem fail-open por default)
- Hash mismatch (PUT retorna 400 com hash mismatch) → rejeitado
- Sem usuário senha (`User=""`) → falha em `NewWSClient`
- URL inválida → falha em `NewWSClient`
- XML malformado na resposta → erro de parse
- SHA-256 do conteúdo confere com parâmetro Hash (validação cruzada)

### 5. `cmd/api/main.go` wire

Factory baseado em env var. Mantém stub como default.

### 6. CHANGELOG v3.8.0

Entrada nova.

---

## ❌ NÃO incluso nesse sprint (carry-over Sprint 19+)

| Item | Origem | Sprint |
|---|---|---|
| Playwright client | Roadmap 1.0 (substituído por V2) | 19+ |
| Range upload (chunked) | Manual 5.6 | 19+ |
| Upload paralelo / resume | Manual 5.4-5.5 | 19+ |
| Range download (resume receive) | Manual 6.4 | 19+ |
| Senha rotation (PUT senhaws/senha) | Manual 9.1 | 19+ |
| Senha vencimento (GET senhaws/senha/vencimento) | Manual 9.2 | 19+ |
| Histórico de requisições WS | Manual 10 | 19+ |
| Consultas por protocolos/avançadas | Manual 8.2-8.3 | 19+ |
| Status upload posicaoupload | Manual 5.3 | 19+ |
| Recebimento (GET arquivos/{protocolo}/conteudo) | Manual 6.1 | 19+ |
| Alteração de situação | Manual 7 | 19+ |
| Retry exponencial + circuit-breaker | Best practice | 19+ |
| Vault/KMS integration | Best practice | 19+ |
| Credenciais em DB vs env | Decisão | 19+ |
| Frontend: download XML arquivos BACEN response | Frontend | 19+ |
| Homologação BACEN contra Sisbacen real | Operational | 19+ |

Esse é o escopo correto de **V1**: o mínimo end-to-end (POST +
PUT) que prova que o cliente está wired corretamente. Estender
funcionalidades é trabalho mecânico uma vez que o skeleton está validado.

---

## 🎯 Critérios de aceite (vs plano)

### Sprint 18 ✅ 4/4

- ✅ SPRINT_18_RESEARCH.md com spec BACEN STA WS v1.5 completa
  (referência para implementações futuras)
- ✅ WSClient skeleton implementando `Submit()` via 2-fase POST+PUT
  com SHA-256 do conteúdo ZIP
- ✅ 15+ testes httptest cobrindo happy path + principais erros
  documentados no manual (Tabelas 5-8)
- ✅ Wire no main.go com factory preservando stub como default

### Métricas esperadas

| Métrica | Alvo |
|---|---|
| LOC backend `sta/` total | < 600 (skeleton + tipos + testes) |
| Tests passando `-race` | 15+ novos, total 374+ (sem regressão) |
| `go vet` | clean |
| `go build` 5 binaries | clean |
| Smoke 11/11 | sem regressão |
| Cross-tenant guard | intacto (handler continua chamando `enforceSameIF`) |
| Audit log emission | mesmo padrão de stub |

---

## 🚦 Riscos

1. **Integração Bacen testável só com credenciais reais**: testes
   httptest mock apenas provam conformidade com a spec documentada; não
   capturam bugs de wire (e.g., content-length errado, ordem de
   headers). Mitigação: Sprint 19+ com smoke contra homologação
   Bacen assim que credenciais forem obtidas.
2. **Mudança de spec sem aviso**: BACEN pode atualizar WS sem mudar
   URL do manual. Mitigação: lock no canvas manual v1.5 + monitorar
   Comunicados BACEN periodicamente.
3. **Camada de TLS não trivial**: cadeia de confiança BACEN pode ser
   diferente entre homolog e prod. Mitigação: usar `tls.Config` com
   `RootCAs` em vez de bundle do sistema operacional + Add cert BACEN
   explicitamente se necessário.
4. **Concurrency / thundering herd**: se várias instâncias de IF
   tentam enviar mesmo arquivo simultaneamente. Manual diz "10
   uploads simultâneos" — limit precisa ser respeitado client-side.
   Mitigação: Sprint 19+ adiciona semaphore no WSClient.

---

## 📚 Referências

- `_referencias/STA_Manual_WebServices.pdf` (42 páginas, BACEN
  oficial v1.5, julho/2022)
- `_referencias/STA_FAQ.pdf` (6 páginas — Content-Type rules)
- `https://www.bcb.gov.br/content/acessoinformacao/sisbacen_docs/Manual_STA_Web_Services.pdf`
- `https://www.bcb.gov.br/acessoinformacao/sistematransferenciaarquivos`
- `https://github.com/aleDsz/bacen_sta` — reference implementation Elixir
- CHANGELOG.md — entrada v3.7.0 (path de hardening precedente)
- SPRINT_8_RESULTS.md — história de JWT bridge (analogia: bridge
  cliente STA é fundação)
- `backend/internal/sta/stub.go` — interface preservada
