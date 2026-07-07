// Package billing implementa integração com Stripe para gestão de planos e subscriptions.
package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SubscriptionService gerencia subscriptions no contexto do tenant.
type SubscriptionService struct {
	db     *sql.DB
	client *Client
}

// NewSubscriptionService cria um novo SubscriptionService.
func NewSubscriptionService(db *sql.DB, client *Client) *SubscriptionService {
	return &SubscriptionService{db: db, client: client}
}

// TenantBillingInfo representa o estado de billing de um tenant.
type TenantBillingInfo struct {
	TenantID             string
	CNPJ                 string
	Plano                Plan
	StripeCustomerID     string
	StripeSubscriptionID string
	StripePlanID         string
	BillingEmail         string
	TrialEndsAt          *time.Time
	SubscriptionStatus   string // active, trialing, past_due, canceled
}

// GetTenantBillingInfo retorna informações de billing do tenant.
func (s *SubscriptionService) GetTenantBillingInfo(ctx context.Context, tenantID string) (*TenantBillingInfo, error) {
	var info TenantBillingInfo
	var trialEndsAt sql.NullTime
	var stripeCustomerID, stripeSubscriptionID, stripePlanID, billingEmail sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT
			id, cnpj, plano,
			COALESCE(stripe_customer_id, ''),
			COALESCE(stripe_subscription_id, ''),
			COALESCE(stripe_plan_id, ''),
			COALESCE(billing_email, ''),
			trial_ends_at
		FROM ifs
		WHERE id = ? AND deleted_at IS NULL
	`, tenantID).Scan(
		&info.TenantID,
		&info.CNPJ,
		&info.Plano,
		&stripeCustomerID,
		&stripeSubscriptionID,
		&stripePlanID,
		&billingEmail,
		&trialEndsAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant %s não encontrado", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	info.StripeCustomerID = stripeCustomerID.String
	info.StripeSubscriptionID = stripeSubscriptionID.String
	info.StripePlanID = stripePlanID.String
	info.BillingEmail = billingEmail.String
	if trialEndsAt.Valid {
		info.TrialEndsAt = &trialEndsAt.Time
	}

	// Se tem subscription, busca status atual do Stripe.
	if info.StripeSubscriptionID != "" && s.client != nil {
		sub, err := s.client.GetSubscription(ctx, info.StripeSubscriptionID)
		if err == nil {
			info.SubscriptionStatus = sub.Status
		}
	}

	return &info, nil
}

// CreateCustomerAndSubscription cria Customer e Subscription no Stripe
// e persiste os IDs no banco.
func (s *SubscriptionService) CreateCustomerAndSubscription(ctx context.Context, tenantID string, email, name string, plan Plan) (*TenantBillingInfo, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id é obrigatório")
	}
	if email == "" {
		return nil, errors.New("email é obrigatório")
	}
	if !IsValidPlan(plan) {
		return nil, fmt.Errorf("plano inválido: %s", plan)
	}

	// 1. Cria Customer no Stripe.
	customer, err := s.client.CreateCustomer(ctx, CreateCustomerInput{
		Email:    email,
		Name:     name,
		TenantID: tenantID,
		Metadata: map[string]string{"tenant_id": tenantID},
	})
	if err != nil {
		return nil, fmt.Errorf("criar customer: %w", err)
	}

	// 2. Cria Subscription no Stripe.
	subscription, err := s.client.CreateSubscription(ctx, CreateSubscriptionInput{
		CustomerID: customer.ID,
		Plan:       plan,
		Metadata:   map[string]string{"tenant_id": tenantID},
	})
	if err != nil {
		return nil, fmt.Errorf("criar subscription: %w", err)
	}

	// 3. Persiste no banco.
	_, err = s.db.ExecContext(ctx, `
		UPDATE ifs SET
			stripe_customer_id = ?,
			stripe_subscription_id = ?,
			stripe_plan_id = ?,
			billing_email = ?,
			trial_ends_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, customer.ID, subscription.ID, subscription.PriceID, email, subscription.TrialEndsAt, tenantID)
	if err != nil {
		return nil, fmt.Errorf("persistir billing info: %w", err)
	}

	// Atualiza plano no schema.
	if err := s.updateTenantPlan(ctx, tenantID, plan); err != nil {
		return nil, fmt.Errorf("update plano: %w", err)
	}

	return &TenantBillingInfo{
		TenantID:             tenantID,
		Plano:                plan,
		StripeCustomerID:     customer.ID,
		StripeSubscriptionID: subscription.ID,
		StripePlanID:         subscription.PriceID,
		BillingEmail:         email,
		TrialEndsAt:          subscription.TrialEndsAt,
		SubscriptionStatus:   subscription.Status,
	}, nil
}

// GetPortalURL gera URL do Stripe Customer Portal.
func (s *SubscriptionService) GetPortalURL(ctx context.Context, tenantID, returnURL string) (string, error) {
	info, err := s.GetTenantBillingInfo(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if info.StripeCustomerID == "" {
		return "", errors.New("cliente Stripe não configurado para este tenant")
	}

	return s.client.GetPortalURL(ctx, info.StripeCustomerID, returnURL)
}

// CancelSubscription cancela a subscription do tenant.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, tenantID string) error {
	info, err := s.GetTenantBillingInfo(ctx, tenantID)
	if err != nil {
		return err
	}
	if info.StripeSubscriptionID == "" {
		return errors.New("nenhuma subscription ativa")
	}

	if err := s.client.CancelSubscription(ctx, info.StripeSubscriptionID); err != nil {
		return fmt.Errorf("cancelar subscription: %w", err)
	}

	// Atualiza DB — plano volta para lite.
	_, err = s.db.ExecContext(ctx, `
		UPDATE ifs SET
			stripe_subscription_id = NULL,
			stripe_plan_id = NULL,
			trial_ends_at = NULL,
			plano = 'lite',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, tenantID)
	if err != nil {
		return fmt.Errorf("update db: %w", err)
	}

	return nil
}

// updateTenantPlan atualiza o plano no schema da tabela ifs.
func (s *SubscriptionService) updateTenantPlan(ctx context.Context, tenantID string, plan Plan) error {
	planStr := string(plan)
	_, err := s.db.ExecContext(ctx, `
		UPDATE ifs SET plano = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, planStr, tenantID)
	return err
}
