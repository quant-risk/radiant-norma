# SPRINT 23 — Research: Senhaws endpoint (§9.1 + §9.2) + credential rotation

> **Sprint:** 23 (v3.13.0)
> **Quando:** 2026-07-06
> **Pesquisador:** mavis
> **Status:** pesquisa completa, pronto pra implementação

## 1. Contexto

Sprint 23 entrega **credential rotation** para o BACEN STA WS. Manual v1.5 §9.1 expõe
endpoint dedicado (`https://www9.bcb.gov.br/senhaws/senha` em homologação) que permite
**alterar senha Sisbacen programaticamente** sem passar pelo site STA Web.

**Caso de uso real:** senha Sisbacen expira a cada N dias (configurado pela instituição).
Antes da Sprint 23, admin IF tinha que:
1. Acessar `sta.bcb.gov.br/sta` no browser
2. Login com UUUUUDDDD.operador
3. Navegar até "Alterar Senha"
4. Preencher formulário (senha atual + nova + confirmação)
5. Salvar

Com Sprint 23: admin pode agendar rotação automática via cron job / cmd tool que:
1. Chama `ConsultarVencimento()` — se < 7 dias, rotaciona.
2. Chama `AlterarSenha(novaSenha)` com nova senha gerada (ex: random 32 chars).
3. Atualiza secret manager (env var / vault / AWS Secrets Manager).
4. Próxima call STA usa senha nova automaticamente.

## 2. Spec BACEN extraída do manual v1.5

### 2.1 PUT `/senha` — Seção 9.1 (linhas 1105-1140)

**Servidores (linhas 1097-1101):**
- Homologação: `https://www9.bcb.gov.br/senhaws`
- Produção: `https://www3.bcb.gov.br/senhaws`

**Request:**
```
PUT https://www9.bcb.gov.br/senhaws/senha HTTP/1.1
Authorization: Basic base64(UUUUUDDDD.operador:senha_atual)
Content-Type: application/xml

<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Parametros>
  <Senha>{senha_atual}</Senha>
  <NovaSenha>{nova_senha}</NovaSenha>
  <ConfirmacaoNovaSenha>{nova_senha}</ConfirmacaoNovaSenha>
</Parametros>
```

**Atenção (linha 1121):** "O Content-Type deve ser application/xml."

**Campos:**
- `Senha` — senha atual do usuário (Basic Auth + XML body — ambos)
- `NovaSenha` — nova senha
- `ConfirmacaoNovaSenha` — confirmação (deve ser igual a NovaSenha)

**Response sucesso:** **204 No Content**.

**Erros esperados:** 400 (XML Listagem 4). Provavelmente também 401 se senha atual errada.

### 2.2 GET `/senha/vencimento` — Seção 9.2 (linhas 1148-1178)

**Request:**
```
GET https://www9.bcb.gov.br/senhaws/senha/vencimento HTTP/1.1
Authorization: Basic base64(UUUUUDDDD.operador:senha_atual)
```

**Response sucesso (200 OK):**
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
  <DiasVencimentoSenha>{dias}</DiasVencimentoSenha>
</Resultado>
```

**Erros esperados:** 400 (XML Listagem 4).

## 3. Decisões de design

### 3.1 Pacote separado `senhaws`

**Decisão:** criar pacote `internal/senhaws` em vez de adicionar ao `internal/sta`.

**Razão:** senhaws é **serviço separado** do STA WS:
- URL diferente (www9.bcb.gov.br/senhaws vs sta-h.bcb.gov.br/staws)
- Propósito diferente (gerenciar credenciais vs enviar/receber arquivos)
- Versioning independente (BACEN pode atualizar senhaws sem mudar STA)

Misturar em `sta` quebraria single responsibility. Pacote próprio = clean separation.

### 3.2 Validações client-side antes de chamar BACEN

**Regra:** defesa em profundidade — validar formato ANTES de fazer HTTP call.

- `NovaSenha == ConfirmacaoNovaSenha` (BACEN também valida, mas cliente deve pegar antes)
- `NovaSenha` não vazia
- `Senha` (atual) não vazia

### 3.3 Não wrappear em RetryingClient

**Decisão:** `SenhawsClient` NÃO wrappea em `RetryingClient`.

**Razão:** admin tools rodam manualmente. Se falhar, admin re-executa. Retry automático
mascara bugs (ex: caller esqueceu de atualizar secret manager e fica em loop infinito
de retries). Failure fast é mais apropriado.

YAGNI — se virar problema operacional, Sprint 24+.

### 3.4 Senha em memória (não persistida)

**Decisão:** senha Sisbacen fica em `cfg.Password` em memória.

**Razão:** secret manager (env var / vault) é responsabilidade do caller. Cliente
apenas chama BACEN com senha fornecida.

**Defense:** struct `SenhawsConfig` tem doc "NÃO log em logs (F13.8)". Tests verificam
que `SenhawsClient.Error()` não vaza senha (mas isso não é nosso código — BACEN retorna).

### 3.5 Thread-safety

**Decisão:** `SenhawsClient` é thread-safe (`cfg.Password` é read-only durante uso).

Cenário: admin dispara rotação a partir de goroutine paralela a call STA ativa.
**Race condition teórica:** senha antiga é usada em chamada STA em flight enquanto
rotação acontece. **Não tratado nesta sprint** — caller tem que serializar (mutex
externo).

### 3.6 Validação de formato da nova senha

BACEN provavelmente tem regras (ex: mínimo 8 chars, alfanumérico). Manual não documenta.
**Decisão:** validar mínimo 8 chars + max 128 chars (defensivo). BACEN retorna 400 se
regras adicionais.

## 4. Estruturas propostas

```go
// SenhawsConfig configura o SenhawsClient.
type SenhawsConfig struct {
    BaseURL string  // https://www9.bcb.gov.br/senhaws (homol) ou www3 (prod)
    User string     // formato UUUUUDDDD.operador
    Password string // senha atual Sisbacen — NÃO log (F13.8)
    Timeout time.Duration  // default 30s
    HTTPClient *http.Client  // opcional, para tests
    Logger *slog.Logger
}

// SenhawsClient é o cliente para o serviço senhaws do BACEN.
type SenhawsClient struct {
    cfg SenhawsConfig
    logger *slog.Logger
}

// NewSenhawsClient valida config.
func NewSenhawsClient(cfg SenhawsConfig) (*SenhawsClient, error)

// AlterarSenha rotaciona senha Sisbacen.
//
// Após sucesso, cfg.Password está desatualizado — caller deve atualizar
// secret manager antes da próxima call STA.
//
// novaSenha: nova senha (8-128 chars).
// Retorna:
//   - nil em sucesso (204 No Content).
//   - *SenhaError em rejeição formal BACEN.
//   - err em transporte / validação client-side.
//
// Validações client-side:
//   - cfg.Password não vazio (necessário pra Basic Auth)
//   - novaSenha entre 8 e 128 chars
//   - cfg.Password != novaSenha (não muda pra mesma senha)
func (c *SenhawsClient) AlterarSenha(ctx context.Context, novaSenha string) error

// ConsultarVencimento retorna dias restantes até vencimento da senha.
//
// Retorna:
//   - dias (>= 0) em sucesso.
//   - *SenhaError em rejeição formal BACEN.
//   - err em transporte.
func (c *SenhawsClient) ConsultarVencimento(ctx context.Context) (int, error)

// SenhaError representa rejeição formal do senhaws BACEN.
type SenhaError struct {
    StatusCode int
    Code string
    Message string
}

func (e *SenhaError) Error() string { ... }
```

## 5. Compatibilidade

- Novo pacote `internal/senhaws`. Zero impacto em código existente.
- `cmd/api/main.go` inalterado (Sprint 24+ wire se virar requisito).
- Nenhum handler REST (admin tool direto, não UI).
- Nenhum wrapping em `RetryingClient` (failure fast é apropriado pra admin).

## 6. Plano de testes

| Test | Cobre |
|---|---|
| `TestNewSenhawsClient_Validacao` | BaseURL required, User formato Sisbacen, Password required |
| `TestSenhawsClient_AlterarSenha_HappyPath` | PUT 204 No Content |
| `TestSenhawsClient_AlterarSenha_400` | BACEN rejeita → `*SenhaError{400}` |
| `TestSenhawsClient_AlterarSenha_401` | Senha atual errada (BACEN retorna 401) |
| `TestSenhawsClient_AlterarSenha_SenhaCurta` | Validação client-side: < 8 chars |
| `TestSenhawsClient_AlterarSenha_SenhaLonga` | Validação client-side: > 128 chars |
| `TestSenhawsClient_AlterarSenha_MesmaSenha` | Validação: cfg.Password == novaSenha |
| `TestSenhawsClient_AlterarSenha_Vazia` | Validação client-side: novaSenha vazio |
| `TestSenhawsClient_ConsultarVencimento_HappyPath` | GET 200 + dias |
| `TestSenhawsClient_ConsultarVencimento_400` | BACEN rejeita → `*SenhaError` |
| `TestSenhawsClient_ConsultarVencimento_400_XMLError` | 200 OK mas body não parsea |
| `TestParseSenhaError` | Testa `Error()` format com/sem Protocolo |

## 7. Critérios de done

- [ ] `SenhawsClient` + `SenhawsConfig` + `NewSenhawsClient` implementados
- [ ] `AlterarSenha` + `ConsultarVencimento` implementados
- [ ] `*SenhaError` tipado para erros formais
- [ ] Validações client-side (senha length, não vazia, cfg.Password != novaSenha)
- [ ] 12 testes httptest STA
- [ ] 18/18 packages PASS + smoke + gofmt/vet
- [ ] SPRINT_23_RESEARCH.md (este) + SPRINT_23_RESULTS.md + CHANGELOG v3.13.0
- [ ] commit + push

## 8. Riscos identificados

| Risco | Mitigação |
|---|---|
| Caller rotaciona senha mas não atualiza secret manager → calls STA falham 401 | Documentação explícita: caller DEVE atualizar secret manager após sucesso |
| Nova senha gerada é fraca (ex: "12345678") | Validação client-side mínimo 8 chars + max 128; BACEN retorna 400 se regras mais fortes não atendidas |
| Race condition: rotação acontece enquanto call STA está em flight com senha antiga | Documentado. Caller usa mutex externo se necessário |
| Senha logada em logs/debug | Doc "NÃO log (F13.8)". Tests verificam que `*SenhaError.Error()` retorna só message, sem senha |
| `HTTPClient` injetado mas caller esquece `ForceAttemptHTTP2=false` | Default transport seta `ForceAttemptHTTP2=false` + TLS 1.2 min (mesmo padrão WSClient) |

## 9. O que NÃO entra nesta sprint

- **Handlers REST `/v1/senhaws/...`** — admin tool direto. UI seria Sprint 24+.
- **Senhas em vault integration** — caller decide onde armazenar.
- **Retry wrapper** — failure fast é apropriado pra admin.
- **Wire no `cmd/api/main.go`** — não tem consumer imediato.
- **Tests contra BACEN real** — Sprint 24 (precisa credenciais Sisbacen).

## 10. Referências

- Manual BACEN STA Web Services v1.5 (jul/2022, 42 pp) — `_referencias/STA_Manual_WebServices.pdf`
- Seções extraídas:
  - §9.1 (alteração senha) — linhas 1105-1140
  - §9.2 (consulta vencimento) — linhas 1148-1178
  - §2.1 (formato UUUUUDDDD.operador) — linha 212
- SPRINT_22_RESEARCH.md — padrão `Client` interface + wrapper retry (referência)
- SPRINT_22_RESULTS.md — YAGNI pattern replicável
- VALIDATION_v3.12.0_DEEPEST.md — padrões reforçados (race detector, strings.Contains, errors.As)