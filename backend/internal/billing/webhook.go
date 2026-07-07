// Package billing implementa integração com Stripe para gestão de planos e subscriptions.
package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// WebhookHandler processa eventos Stripe e sincroniza com o DB.
//
// Endpoints:
//   - POST /api/v1/webhooks/stripe
//
// Eventos processados:
//   - customer.subscription.created
//   - customer.subscription.updated
//   - customer.subscription.deleted
//   - invoice.payment_succeeded
//   - invoice.payment_failed
type WebhookHandler struct {
	db     *sql.DB
	client *Client
	logger *slog.Logger
}

// NewWebhookHandler cria um novo WebhookHandler.
func NewWebhookHandler(db *sql.DB, client *Client) *WebhookHandler {
	return &WebhookHandler{
		db:     db,
		client: client,
		logger: slog.Default(),
	}
}

// SetLogger injeta logger customizado.
func (h *WebhookHandler) SetLogger(l *slog.Logger) {
	h.logger = l
}

// Handle processa um webhook Stripe.
func (h *WebhookHandler) Handle(ctx context.Context, r *http.Request) error {
	// Lê body raw para verificação de assinatura.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Verifica assinatura do webhook.
	sigHeader := r.Header.Get("Stripe-Signature")
	if err := h.client.VerifyWebhookSignature(body, sigHeader); err != nil {
		h.logger.Warn("webhook signature invalid", "err", err)
		return fmt.Errorf("assinatura inválida: %w", err)
	}

	// Parse evento Stripe.
	var event StripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("parse event: %w", err)
	}

	// Ignora eventos já processados (idempotência).
	processed, err := h.isEventProcessed(ctx, event.ID)
	if err != nil {
		h.logger.Warn("check event processed failed", "event_id", event.ID, "err", err)
	}
	if processed {
		h.logger.Debug("event already processed", "event_id", event.ID)
		return nil
	}

	// Processa o evento.
	if err := h.processEvent(ctx, &event, body); err != nil {
		h.logger.Error("process event failed",
			"event_id", event.ID,
			"event_type", event.Type,
			"err", err)
		return err
	}

	// Marca como processado.
	if err := h.markEventProcessed(ctx, &event, body); err != nil {
		h.logger.Error("mark event processed failed", "event_id", event.ID, "err", err)
		// Não retorna erro — evento foi processado com sucesso
	}

	h.logger.Info("webhook processed",
		"event_id", event.ID,
		"event_type", event.Type)

	return nil
}

// StripeEvent representa um evento genérico do Stripe.
type StripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    json.RawMessage `json:"data"`
}

// processEvent delega para o handler específico do evento.
func (h *WebhookHandler) processEvent(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	switch event.Type {
	case "customer.subscription.created":
		return h.handleSubscriptionCreated(ctx, event, rawBody)
	case "customer.subscription.updated":
		return h.handleSubscriptionUpdated(ctx, event, rawBody)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(ctx, event, rawBody)
	case "invoice.payment_succeeded":
		return h.handleInvoicePaymentSucceeded(ctx, event, rawBody)
	case "invoice.payment_failed":
		return h.handleInvoicePaymentFailed(ctx, event, rawBody)
	default:
		h.logger.Debug("unhandled event type", "type", event.Type)
		return nil
	}
}

// handleSubscriptionCreated processa criação de subscription.
func (h *WebhookHandler) handleSubscriptionCreated(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	var data struct {
		Object struct {
			ID               string `json:"id"`
			Customer         string `json:"customer"`
			Status           string `json:"status"`
			TrialEnd         int64  `json:"trial_end"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
			Items            struct {
				Data []struct {
					Price struct {
						ID string `json:"id"`
					} `json:"price"`
				} `json:"data"`
			} `json:"items"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	plan, _ := h.client.PriceIDToPlan(data.Object.Items.Data[0].Price.ID)

	var trialEndsAt interface{}
	if data.Object.TrialEnd > 0 {
		trialEndsAt = time.Unix(data.Object.TrialEnd, 0)
	}

	// Atualiza IFS com subscription info.
	// Encontramos o tenant pelo stripe_customer_id.
	_, err := h.db.ExecContext(ctx, `
		UPDATE ifs SET
			stripe_subscription_id = ?,
			stripe_plan_id = ?,
			trial_ends_at = ?
		WHERE stripe_customer_id = ?
	`, data.Object.ID, data.Object.Items.Data[0].Price.ID, trialEndsAt, data.Object.Customer)

	if err != nil {
		return fmt.Errorf("update ifs: %w", err)
	}

	h.logger.Info("subscription created",
		"subscription_id", data.Object.ID,
		"customer_id", data.Object.Customer,
		"plan", plan)

	return nil
}

// handleSubscriptionUpdated processa atualização de subscription.
func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	var data struct {
		Object struct {
			ID               string `json:"id"`
			Customer         string `json:"customer"`
			Status           string `json:"status"`
			TrialEnd         int64  `json:"trial_end"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
			Items            struct {
				Data []struct {
					Price struct {
						ID string `json:"id"`
					} `json:"price"`
				} `json:"data"`
			} `json:"items"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	plan, _ := h.client.PriceIDToPlan(data.Object.Items.Data[0].Price.ID)

	var trialEndsAt interface{}
	if data.Object.TrialEnd > 0 {
		trialEndsAt = time.Unix(data.Object.TrialEnd, 0)
	}

	_, err := h.db.ExecContext(ctx, `
		UPDATE ifs SET
			stripe_subscription_id = ?,
			stripe_plan_id = ?,
			trial_ends_at = ?
		WHERE stripe_customer_id = ?
	`, data.Object.ID, data.Object.Items.Data[0].Price.ID, trialEndsAt, data.Object.Customer)

	if err != nil {
		return fmt.Errorf("update ifs: %w", err)
	}

	h.logger.Info("subscription updated",
		"subscription_id", data.Object.ID,
		"status", data.Object.Status,
		"plan", plan)

	return nil
}

// handleSubscriptionDeleted processa cancelamento de subscription.
func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	var data struct {
		Object struct {
			ID       string `json:"id"`
			Customer string `json:"customer"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Não deletamos dados — apenas limpamos subscription info.
	// O plano fica como "lite" por default.
	_, err := h.db.ExecContext(ctx, `
		UPDATE ifs SET
			stripe_subscription_id = NULL,
			stripe_plan_id = NULL,
			trial_ends_at = NULL,
			plano = 'lite'
		WHERE stripe_customer_id = ?
	`, data.Object.Customer)

	if err != nil {
		return fmt.Errorf("update ifs: %w", err)
	}

	h.logger.Info("subscription deleted",
		"subscription_id", data.Object.ID,
		"customer_id", data.Object.Customer)

	return nil
}

// handleInvoicePaymentSucceeded processa pagamento bem-sucedido.
func (h *WebhookHandler) handleInvoicePaymentSucceeded(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	var data struct {
		Object struct {
			ID           string `json:"id"`
			Customer     string `json:"customer"`
			Subscription string `json:"subscription"`
			AmountPaid   int64  `json:"amount_paid"`
			Currency     string `json:"currency"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	h.logger.Info("invoice payment succeeded",
		"invoice_id", data.Object.ID,
		"customer_id", data.Object.Customer,
		"amount", data.Object.AmountPaid,
		"currency", data.Object.Currency)

	// Invoice payment succeeded é informativo — subscription já está ativa.
	// Não precisa atualizar nada no schema.
	return nil
}

// handleInvoicePaymentFailed processa pagamento falho.
func (h *WebhookHandler) handleInvoicePaymentFailed(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	var data struct {
		Object struct {
			ID              string `json:"id"`
			Customer        string `json:"customer"`
			Subscription    string `json:"subscription"`
			AmountDue       int64  `json:"amount_due"`
			Currency        string `json:"currency"`
			NextPaymentTime int64  `json:"next_payment_attempt"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	h.logger.Warn("invoice payment failed",
		"invoice_id", data.Object.ID,
		"customer_id", data.Object.Customer,
		"amount_due", data.Object.AmountDue,
		"currency", data.Object.Currency,
		"next_payment_attempt", data.Object.NextPaymentTime)

	// Payment failed não bloqueia — Stripe tentará novamente.
	// O status da subscription só muda se payment falhar após múltiplas tentativas.
	return nil
}

// isEventProcessed verifica se o evento já foi processado.
func (h *WebhookHandler) isEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var count int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM billing_events
		WHERE stripe_event_id = ? AND event_type LIKE 'stripe%'
	`, eventID).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return count > 0, nil
}

// markEventProcessed marca o evento como processado no billing_events.
func (h *WebhookHandler) markEventProcessed(ctx context.Context, event *StripeEvent, rawBody []byte) error {
	// Extrai customer_id do payload do evento para encontrar tenant real.
	var customerID string
	var data struct {
		Object struct {
			Customer string `json:"customer"`
		} `json:"object"`
	}
	if json.Unmarshal(event.Data, &data) == nil {
		customerID = data.Object.Customer
	}

	// Busca tenant_id real pelo stripe_customer_id.
	var tenantID string
	if customerID != "" {
		_ = h.db.QueryRowContext(ctx, `
			SELECT id FROM ifs WHERE stripe_customer_id = ?
		`, customerID).Scan(&tenantID)
	}

	_, err := h.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO billing_events (tenant_id, event_type, stripe_event_id, payload, processed_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, tenantID, event.Type, event.ID, string(rawBody))
	return err
}
