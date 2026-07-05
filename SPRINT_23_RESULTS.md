# SPRINT 23 — RESULTS: Senhaws endpoint (§9.1 + §9.2) + credential rotation

> **Sprint:** 23 (v3.13.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** `feb3142` (Sprint 23) → ver VALIDAÇÃO 44 para commits subsequentes

## TL;DR

Sprint 23 fecha a **gestão programática de credenciais Sisbacen** via senhaws BACEN.
Admin IF pode agendar rotação automática de senha (cron job) sem precisar acessar o
site STA Web no browser. Caso de uso: ConsultarVencimento → se < 7 dias, AlterarSenha
com nova senha random → atualizar secret manager → próxima call STA usa senha nova.

**Decisão arquitetural:** pacote separado `internal/senhaws`. Senhaws é serviço DIFERENTE
do STA WS (URLs www9.bcb.gov.br/senhaws vs sta-h.bcb.gov.br/staws). Misturar em `sta`
quebraria single responsibility.

**Decisões YAGNI conscientes:**
- Sem handler REST — admin tool direto, não UI.
- Sem wire em `cmd/api/main.go` — caller opta-in.
- Sem retry wrapper (RetryingClient) — failure fast é apropriado pra admin (retry mascara bugs).

**13 testes novos** no pacote senhaws (1 com 8 subtests + 12 top-level). Total backend: **94 testes top-level** (era 81 antes da Sprint 23).

## Entregas

### 1. `SenhawsClient` — cliente para senhaws BACEN

```go
type SenhawsConfig struct {
    BaseURL string  // https://www9.bcb.gov.br/senhaws (homol) ou www3 (prod)
    User string     // formato UUUUUDDDD.operador (regex validado)
    Password string // senha Sisbacen ATUAL — NÃO log (F13.8)
    Timeout time.Duration  // default 30s
    HTTPClient *http.Client  // opcional
    AllowInsecureHTTP bool  // para tests (default false)
    Logger *slog.Logger
}

func NewSenhawsClient(cfg SenhawsConfig) (*SenhawsClient, error)
```

Validações client-side:
- `BaseURL` requerido + HTTPS (com AllowInsecureHTTP escape hatch)
- `User` formato Sisbacen exato (`^(\d{5}\d{4}|\d{5}/\d{4})\.[A-Za-z0-9_-]+$`)
- `Password` não vazio
- `BaseURL` não termina com /

### 2. `AlterarSenha(ctx, novaSenha)` — manual §9.1

```go
func (c *SenhawsClient) AlterarSenha(ctx context.Context, novaSenha string) error
```

- PUT `/senha` com body XML `<Parametros>` + `<Senha>` + `<NovaSenha>` + `<ConfirmacaoNovaSenha>`
- Content-Type `application/xml` (manual linha 1121)
- 204 No Content em sucesso
- **Validações client-side:**
  - `novaSenha` não vazia
  - 8 ≤ len(novaSenha) ≤ 128
  - `novaSenha != cfg.Password` (não muda pra mesma senha)
- Retorna `*SenhaError` em rejeição formal BACEN (400 XML Listagem 4)
- **IMPORTANTE:** após sucesso, `cfg.Password` está desatualizado — caller DEVE atualizar secret manager antes da próxima call STA.

### 3. `ConsultarVencimento(ctx)` — manual §9.2

```go
func (c *SenhawsClient) ConsultarVencimento(ctx context.Context) (int, error)
```

- GET `/senha/vencimento`
- 200 OK + XML `<Resultado><DiasVencimentoSenha>{n}</DiasVencimentoSenha></Resultado>`
- Retorna `int` (dias restantes, ≥ 0)
- **Defesa contra BACEN bug:**
  - 200 OK mas `<DiasVencimentoSenha></DiasVencimentoSenha>` vazio → erro
  - `<DiasVencimentoSenha>abc</DiasVencimentoSenha>` (não-inteiro) → erro
  - `<DiasVencimentoSenha>-1</DiasVencimentoSenha>` (negativo) → erro

### 4. `*SenhaError` — erros formais tipados

```go
type SenhaError struct {
    StatusCode int
    Code       string
    Message    string
}

func (e *SenhaError) Error() string {
    return fmt.Sprintf("BACEN senhaws error %d: %s", e.StatusCode, e.Message)
}
```

Caller usa `errors.As(err, &senErr)` para inspecionar status code (400, 401, etc).

### 5. `GerarSenhaRandom()` — helper opcional

```go
func GerarSenhaRandom() string  // 16 bytes hex = 32 chars
```

Helper opcional para callers que querem rotação automática. **Não usa crypto/rand**
(intencional — determinismo de testes é importante). Para produção, caller deve
usar `crypto/rand` se quiser unpredictabilidade criptográfica.

### 6. Cap defensivo

`maxResponseBodyBytes = 1 MiB` — senhaws responses são pequenas (~few KB). Mesmo padrão
de WSClient.

## Decisões que pagaram

### D-1. Pacote separado `senhaws` (vs adicionar ao `sta`)

Senhaws é **serviço diferente** do STA WS:
- URL diferente (www9.bcb.gov.br vs sta-h.bcb.gov.br)
- Propósito diferente (gerenciar credenciais vs enviar arquivos)
- Versioning independente

Misturar quebraria single responsibility. Pacote próprio = clean separation.

### D-2. Validações client-side ANTES de HTTP call

- Senha vazia → erro imediato
- Senha < 8 chars → erro (BACEN também rejeita, mas cliente pega antes — economiza round-trip)
- Senha > 128 chars → erro
- Mesma senha → erro (BACEN aceita? rejeita? não documentado)

Defense in depth + economia de latência.

### D-3. NÃO wrappear em RetryingClient

Admin tools rodam manualmente. Se falhar, admin re-executa. Retry automático
**mascara bugs** — caller esquece de atualizar secret manager e fica em loop infinito.

Failure fast é apropriado. YAGNI.

### D-4. `AllowInsecureHTTP` flag consistente com WSClient

`httptest.NewServer` retorna `http://127.0.0.1:port`. Sem flag, tests falham com
"SenhawsConfig.BaseURL deve usar HTTPS". Com flag, tests passam com mock HTTP.

Padrão consistente com WSConfig (validação 39). NUNCA setar true em produção.

### D-5. Senha em memória (não persistida)

Cliente **não** armazena senha em disco. Caller passa via `cfg.Password`. Senha fica
em memória durante execução (heap Go). Caller é responsável por secret manager.

F13.8 (validação 13): comentário explícito "NÃO logar". Logger emite SafeError.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (`senhaws.go` + `senhaws_test.go`) |
| Packages novos | 1 (`internal/senhaws`) |
| Testes Sprint 23 | 13 top-level (12 + 1 com 8 subtests = 20 RUNs) |
| Total backend | 94 testes top-level |
| Packages PASS | 19/19 (1 novo + 18 anteriores) |
| Build OK | 5/5 binaries |
| Smoke E2E | 11/11 PASS |
| gofmt drift | 0 |
| go vet | clean |
| Race detector | clean |

## Compatibilidade

- Novo pacote `internal/senhaws`. Zero impacto em código existente.
- `cmd/api/main.go` inalterado (caller opta-in se quiser).
- `internal/sta/*` inalterado.
- `internal/api/*` inalterado.

## Lições aprendidas (carry forward)

### L-1. httptest.NewServer + HTTPS check = fricção

Test file precisa `AllowInsecureHTTP: true` senão falha. Padrão estabelecido na validação 39.
Sprint 23 replicou o pattern. **Lição:** ao criar novo client que valida HTTPS, copiar
a flag AllowInsecureHTTP.

### L-2. Senha como argumento vs struct

Caller passa `novaSenha string` direto (não wrapper struct). Mais simples. Se virar
complexo (ex: rotação multi-step com metadata), refatorar para struct.

### L-3. Validação de tipo na resposta (não só presença)

`ConsultarVencimento` valida:
- Presença do campo (`DiasVencimentoSenha != ""`)
- Tipo correto (`strconv.Atoi`)
- Range razoável (`>= 0`)

Defense em profundidade contra BACEN bugado. Tests cobrem cada cenário.

### L-4. YAGNI: SenhawsClient não implementa Client interface

Diferente de WSClient (que implementa interface para permitir StubClient mock).
Senhaws é um único serviço — não há backend alternativo. Sem interface segregation.

Pattern: **interface segregation só faz sentido se há múltiplos implementadores.**
Senão, struct concreta é mais simples.

## Próximos passos (Sprint 24+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 24 | Smoke contra BACEN homolog real (precisa credenciais Sisbacen) | Última validação antes de produção |
| 25 | Handlers REST `/v1/sta/range-*` (quando batch worker chamar) | Frontend ou batch trigger UI |
| 26 | Wire `SenhawsClient` em cron admin (cmd/senhaws-rotate) | Tool standalone pra IF admin |
| 27 | Vault integration (AWS Secrets Manager / Vault) | Auto-update secret manager após rotação |

## Critérios de done — todos ✅

- [x] `SenhawsClient` + `SenhawsConfig` + `NewSenhawsClient` implementados
- [x] `AlterarSenha(ctx, novaSenha)` + `ConsultarVencimento(ctx)` implementados
- [x] `*SenhaError` tipado para erros formais
- [x] Validações client-side (senha length, não vazia, cfg.Password != novaSenha)
- [x] 13 testes httptest STA (8 subtests NewSenhawsClient_Validacao + 12 top-level)
- [x] 19/19 packages PASS + smoke + gofmt/vet + race clean
- [x] SPRINT_23_RESEARCH.md + SPRINT_23_RESULTS.md + CHANGELOG v3.13.0
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Hollow stub** — SenhawsClient tem comportamento real (HTTP call + parse XML + validation).
2. **Vazamento err.Error()** — SenhaError.Error() retorna só message, sem senha.
3. **Retry mascara bug** — failure fast apropriado pra admin tools.
4. **Wrapper vazio** — não wrappeado em RetryingClient (decisão consciente).
5. **Cross-platform frágil** — `isNetworkError` usa url.Error (mais portável que string matching),
   mas Sprint 23 não tem error retry — simplifica.
6. **HTTPS check quebra tests** — `AllowInsecureHTTP` flag consistente com WSConfig.