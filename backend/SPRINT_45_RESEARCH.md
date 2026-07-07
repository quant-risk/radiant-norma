# Sprint 45 — RESEARCH — StripeBilling

> **Data:** 2026-07-07
> **Sprint:** 45
> **Domínio:** Billing (Stripe)
> **Versão atual:** v3.34.25
> **Próxima:** v3.34.26

---

## 1. Contexto

### O que existe

O schema já tem `plano` na tabela `ifs`:

```sql
CREATE TABLE IF NOT EXISTS ifs (
    ...
    plano TEXT NOT NULL DEFAULT 'lite', -- lite, pro, scale, enterprise
    ...
);
```

Planos definidos no MASTER_PLAN §E.3:

| Plano | Preço | Cobertura |
|---|---|---|
| **Lite** | R$ 1.500/mês | 3040 + 3050 + Radar |
| **Pro** | R$ 4.500/mês | Lite + DLO/DDR/DRL/DLP + 3044 |
| **Scale** | R$ 12.000/mês | Pro + DRSAC + AI Insights |
| **Enterprise** | custom | Multi-tenant + SLA 99.95% + suporte dedicado |

### O que vamos construir

Integração Stripe para:
1. Criar Customer quando nova IF faz onboarding
2. Criar Subscription com base no plano selecionado
3. Webhooks para sincronizar status (ativo, cancelado, payment failed)
4. Customer Portal link para self-service
5. Billing API protegida (admin only)

---

## 2. Arquitetura

### Componentes

```
backend/
  internal/
    billing/               (NOVO — package de billing)
      stripe.go            — Cliente Stripe + operations
      webhook.go           — Handlers de webhook Stripe
      subscription.go      — Gerenciamento de subscriptions
    db/
      migrations/
        015_stripe_billing.sql  (NOVO — adiciona stripe_customer_id)
```

### Fluxo de dados

```
Onboarding novo tenant
    → Cria Stripe Customer
    → Cria Stripe Subscription (trial de 14 dias)
    → Salva stripe_customer_id + stripe_subscription_id na tabela ifs
    → Ativa tenant

Stripe Webhook (customer.subscription.updated)
    → Verifica assinatura
    → Atualiza plano no DB (ifs.plano)
    → Loga evento

Billing Portal
    → Redirect para Stripe Customer Portal
    → Tenant pode upgrade/downgrade/cancelar
```

---

## 3. Design de API

### Endpoints

| Método | Path | Descrição |
|---|---|---|
| `GET` | `/api/v1/admin/billing/:tenant_id` |Detalhes de billing do tenant |
| `POST` | `/api/v1/admin/billing/:tenant_id/create-customer` | Cria Stripe Customer |
| `POST` | `/api/v1/admin/billing/:tenant_id/create-subscription` | Cria Subscription |
| `GET` | `/api/v1/admin/billing/:tenant_id/portal-url` | Gera Stripe Portal URL |
| `POST` | `/api/v1/webhooks/stripe` | Webhook receiver (assinatura Stripe) |

### Webhook Events

| Event | Ação |
|---|---|
| `customer.subscription.created` | Ativa trial |
| `customer.subscription.updated` | Atualiza plano |
| `customer.subscription.deleted` | Desativa tenant |
| `invoice.payment_succeeded` | Confirma pagamento |
| `invoice.payment_failed` | Alerta (não bloqueia) |

---

## 4. Schema do DB

### Migration 015: Adicionar Stripe IDs

```sql
ALTER TABLE ifs
    ADD COLUMN stripe_customer_id TEXT,
    ADD COLUMN stripe_subscription_id TEXT,
    ADD COLUMN stripe_plan_id TEXT,      -- price_xxx do Stripe
    ADD COLUMN billing_email TEXT,
    ADD COLUMN trial_ends_at DATETIME;  -- NULL quando trial acabar
```

---

## 5. Stripe Integration

### Configuração

```go
type Config struct {
    SecretKey      string // sk_live_... ou sk_test_...
    WebhookSecret  string // whsec_... do dashboard
    PriceLite     string // price_xxx
    PricePro      string
    PriceScale    string
    PriceEnterprise string
}
```

### Plano → Price mapping

```go
func PlanToPriceID(plan string) (string, error) {
    switch plan {
    case "lite":     return cfg.PriceLite, nil
    case "pro":      return cfg.PricePro, nil
    case "scale":    return cfg.PriceScale, nil
    case "enterprise": return cfg.PriceEnterprise, nil
    }
    return "", fmt.Errorf("plano desconhecido: %s", plan)
}
```

---

## 6. Critérios de aceitação

- [ ] Migration 015 adiciona colunas stripe na tabela ifs
- [ ] `billing.CreateCustomer(ctx, tenantID) → (customerID, error)`
- [ ] `billing.CreateSubscription(ctx, customerID, plan) → (subscriptionID, error)`
- [ ] `billing.GetPortalURL(ctx, customerID) → (url, error)`
- [ ] `billing.WebhookHandler` processa events e atualiza DB
- [ ] `billing.GetSubscriptionStatus(ctx, subscriptionID) → (status, error)`
- [ ] `go test ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] CHANGELOG entry v3.34.26

---

## 7. Segurança

- **Webhook signature**: validar `Stripe-Signature` header com `hmacSHA256`
- **Webhook secret** nunca em log
- **Admin only**: endpoints de billing protegidos por `X-Admin-Token`
- **Idempotency**: webhook handler idempotente (replay protection via event ID)
- **Dry-run mode**: se `STRIPE_SECRET_KEY` vazio, operações retornam erro jelas (não silenciam)

---

## 8. Riscos

| Risco | Mitigação |
|---|---|
| Stripe sandbox vs production keys | Config via env vars, fail-fast se mismatch |
| Webhook replay attack | Verificar event ID + timestamp |
| Plan downgrade não refletir imediatamente | Stripe webhooks são eventual-consistent; UI mostra status atual |
| Test mode vs production | Diferenciar por `STRIPE_TEST_MODE=true` |
