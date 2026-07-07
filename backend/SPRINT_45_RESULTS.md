# Sprint 45 Results — StripeBilling (Integração Stripe)

**Sprint:** 45
**Período:** 2026-07-07
**Status:** ✅ Shipped
**Versão:** v3.34.26

---

## Objetivo

Adicionar integração Stripe para billing — criar Customers, Subscriptions, processar Webhooks e gerar Customer Portal URLs. Preparar a base para planos Lite/Pro/Scale/Enterprise com trial de 14 dias.

---

## Entregáveis

### 1. `internal/billing/stripe.go` ✅

Cliente Stripe com operações de billing.

**Estruturas:**
```go
type Config struct {
    SecretKey, WebhookSecret string
    PriceLite, PricePro, PriceScale, PriceEnterprise string
    TrialDays int
    IsTestMode bool
}

type Client struct { ... }

type Customer struct { ID, Email, Name, TenantID, CreatedAt }
type Subscription struct { ID, CustomerID, Status, Plan, PriceID, TrialEndsAt, CurrentPeriodEnd, CreatedAt }
```

**Métodos:**
- `NewClient(cfg) → (*Client, error)` — valida config
- `CreateCustomer(ctx, input) → (*Customer, error)`
- `CreateSubscription(ctx, input) → (*Subscription, error)`
- `GetSubscription(ctx, id) → (*Subscription, error)`
- `GetPortalURL(ctx, customerID, returnURL) → (url, error)`
- `CancelSubscription(ctx, id) → error`
- `VerifyWebhookSignature(payload, sigHeader) → error` — HMAC-SHA256 timing-safe

**Segurança:** Bearer auth, Stripe-Version header, dry-run se key vazia.

---

### 2. `internal/billing/webhook.go` ✅

Webhook handler para eventos Stripe.

**Eventos processados:**
| Event | Ação |
|---|---|
| `customer.subscription.created` | Ativa trial |
| `customer.subscription.updated` | Atualiza plano |
| `customer.subscription.deleted` | Desativa (plano volta a lite) |
| `invoice.payment_succeeded` | Log informativo |
| `invoice.payment_failed` | Log warning (Stripe tentará novamente) |

**Características:**
- Idempotência via `stripe_event_id` em `billing_events`
- signature verification antes de processar
- Logging estruturado com `slog`

---

### 3. `internal/billing/subscription.go` ✅

Gerenciamento de subscriptions no contexto do tenant.

```go
type SubscriptionService struct { db *sql.DB, client *Client }

type TenantBillingInfo struct {
    TenantID, CNPJ, Plano, StripeCustomerID, StripeSubscriptionID,
    StripePlanID, BillingEmail, SubscriptionStatus
    TrialEndsAt *time.Time
}
```

**Métodos:**
- `GetTenantBillingInfo(ctx, tenantID)` — Busca info de billing
- `CreateCustomerAndSubscription(ctx, tenantID, email, name, plan)` — Fluxo completo
- `GetPortalURL(ctx, tenantID, returnURL)` — Gera portal URL
- `CancelSubscription(ctx, tenantID)` — Cancela e volta para lite

---

### 4. `internal/db/migrations/015_stripe_billing.sql` ✅

Migration SQLite-portátil (sem `BEGIN/COMMIT`, sem `ADD COLUMN IF NOT EXISTS`).

```sql
ALTER TABLE ifs ADD COLUMN stripe_customer_id TEXT;
ALTER TABLE ifs ADD COLUMN stripe_subscription_id TEXT;
ALTER TABLE ifs ADD COLUMN stripe_plan_id TEXT;
ALTER TABLE ifs ADD COLUMN billing_email TEXT;
ALTER TABLE ifs ADD COLUMN trial_ends_at DATETIME;

CREATE TABLE billing_events (...);
CREATE INDEX idx_ifs_stripe_customer ON ifs(stripe_customer_id);
CREATE INDEX idx_ifs_stripe_subscription ON ifs(stripe_subscription_id);
CREATE INDEX idx_billing_events_tenant ON billing_events(tenant_id);
```

---

### 5. `internal/billing/billing_test.go` ✅

15 testes cobrindo:
- `TestIsValidPlan` (lite/pro/scale/enterprise/invalid)
- `TestNewClient_EmptySecretKey` / `ValidConfig`
- `TestHMACEqual_Same/Different/DifferentLength`
- `TestVerifyWebhookSignature_EmptySecret/MalformedHeader/ValidSignature`
- `TestTenantBillingInfo_ZeroValue`
- `TestCustomer_Fields`
- `TestSubscription_WithTrialEndsAt/NoTrial`
- `TestCreateCustomerInput` / `TestCreateSubscriptionInput`

---

## Arquitetura

```
billing/
  stripe.go         — Cliente Stripe (Customer, Subscription, Portal)
  webhook.go        — WebhookHandler (events → DB)
  subscription.go   — SubscriptionService (tenant lifecycle)
  billing_test.go   — 15 testes

migrations/
  015_stripe_billing.sql — Schema Stripe (ifs columns + billing_events)
```

---

## Decisões de Design

### 1. Stripe REST API (não SDK oficial)

Usamos HTTP direto em vez do SDK `github.com/stripe/stripe-go`:
- Elimina dependência de SDK (v40+ do Stripe muda frequentemente)
- Menos código boilerplate que o SDK oficial
- Mais controlável

### 2. Migration SQLite-portátil

SQLite não suporta `ADD COLUMN IF NOT EXISTS` (< 3.35.0) nem `BEGIN/COMMIT` dentro de transação já ativa. Migration é executada em transação pelo runner, então não precisa de BEGIN/COMMIT próprio.

### 3. HMAC timing-safe

`hmacEqual` compara em tempo constante para evitar timing attacks. Timestamp do webhook verificado (< 5 min) para evitar replay.

### 4. Idempotência via stripe_event_id

Cada evento Stripe tem ID único. `billing_events.stripe_event_id UNIQUE` garante que o mesmo evento não é processado duas vezes.

---

## Testes

**Suite completo:**
```
ok  github.com/fortvna/radiant-norma/backend/internal/billing  0.454s (15 tests)
ok  github.com/fortvna/radiant-norma/backend/internal/worker    3.291s
ok  github.com/fortvna/radiant-norma/backend/internal/db        2.228s
```

**Todos os 25 packages:** ✅ PASS

---

## Limitações & TODOs

1. **Sem Stripe SDK**: HTTP direto é mais trabalho mas mais estável
2. **Sem Price IDs configurados**: Precisa configurar `STRIPE_PRICE_LITE` etc via env
3. **Sem UI de billing**: Portal redirect é self-service, mas sem página interna
4. **Trial days fixo**: Configurado no código, não configurável por tenant

---

## Validações V67-V75 Aplicadas

- **V67:** Sem `_ = context.Background()` em Apply
- **V68:** Sem loops vazios `for { _ = i }`
- **V69:** N/A (não são regras de validação)
- **V70:** Sem stubs disfarçados
- **V71:** Error wrapping com `%w`
- **V72:** Logging com `slog` (não silent fail)
- **V73:** Nil checks em todos os métodos públicos
- **V74:** N/A (billing não é rules package)
- **V75:** Sem `println`/log hardcoded

---

## arquivos_modificados

| Arquivo | Mudança |
|---|---|
| `internal/billing/stripe.go` | novo |
| `internal/billing/webhook.go` | novo |
| `internal/billing/subscription.go` | novo |
| `internal/billing/billing_test.go` | novo |
| `internal/db/migrations/015_stripe_billing.sql` | novo |
| `internal/version/version.go` | version bump 3.34.25 → 3.34.26 |
| `CHANGELOG.md` | entry v3.34.26 |
| `SPRINT_45_RESEARCH.md` | research |
