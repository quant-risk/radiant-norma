// Package billing implementa integração com Stripe para gestão de planos e subscriptions.
//
// Suporta os planos Lite, Pro, Scale e Enterprise com trial de 14 dias.
// Webhooks sincronizam status de subscription com o banco de dados.
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Plan representa um plano disponível.
type Plan string

const (
	PlanLite       Plan = "lite"
	PlanPro        Plan = "pro"
	PlanScale      Plan = "scale"
	PlanEnterprise Plan = "enterprise"
)

// IsValidPlan retorna true se o plano é válido.
func IsValidPlan(p Plan) bool {
	switch p {
	case PlanLite, PlanPro, PlanScale, PlanEnterprise:
		return true
	}
	return false
}

// Config é a configuração do cliente Stripe.
type Config struct {
	SecretKey       string // sk_live_... ou sk_test_...
	WebhookSecret   string // whsec_... do dashboard
	PriceLite       string // price_xxx
	PricePro        string
	PriceScale      string
	PriceEnterprise string
	TrialDays       int // dias de trial (default 14)
	IsTestMode      bool
}

// Client é o cliente Stripe para operações de billing.
type Client struct {
	cfg  Config
	hc   *http.Client
	base string
}

// NewClient cria um novo cliente Stripe.
func NewClient(cfg Config) (*Client, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("Stripe secret key não configurada")
	}
	if cfg.TrialDays == 0 {
		cfg.TrialDays = 14
	}

	return &Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout: 30 * time.Second,
		},
		base: "https://api.stripe.com",
	}, nil
}

// Customer representa um Customer Stripe.
type Customer struct {
	ID        string
	Email     string
	Name      string
	TenantID  string // nosso ID interno
	CreatedAt time.Time
}

// CreateCustomerInput é o input para criar Customer.
type CreateCustomerInput struct {
	Email    string
	Name     string
	TenantID string
	Metadata map[string]string
}

// CreateCustomer cria um novo Customer no Stripe.
func (c *Client) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error) {
	if input.Email == "" {
		return nil, errors.New("email é obrigatório")
	}

	body := strings.Builder{}
	body.WriteString("email=" + input.Email)
	body.WriteString("&name=" + input.Name)
	if input.TenantID != "" {
		body.WriteString("&metadata[tenant_id]=" + input.TenantID)
	}
	for k, v := range input.Metadata {
		body.WriteString(fmt.Sprintf("&metadata[%s]=%s", k, v))
	}

	url := c.base + "/v1/customers"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body.String()))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Stripe CreateCustomer: status %d: %s", resp.StatusCode, string(respBody))
	}

	var stripeResp struct {
		ID       string            `json:"id"`
		Email    string            `json:"email"`
		Name     string            `json:"name"`
		Created  int64             `json:"created"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stripeResp); err != nil {
		return nil, err
	}

	// Extrai tenant_id dos metadados se presente.
	tenantID := ""
	if stripeResp.Metadata != nil {
		tenantID = stripeResp.Metadata["tenant_id"]
	}

	return &Customer{
		ID:        stripeResp.ID,
		Email:     stripeResp.Email,
		Name:      stripeResp.Name,
		TenantID:  tenantID,
		CreatedAt: time.Unix(stripeResp.Created, 0),
	}, nil
}

// Subscription representa uma Subscription Stripe.
type Subscription struct {
	ID               string
	CustomerID       string
	Status           string // active, trialing, past_due, canceled
	Plan             Plan
	PriceID          string
	TrialEndsAt      *time.Time
	CurrentPeriodEnd time.Time
	CreatedAt        time.Time
}

// CreateSubscriptionInput é o input para criar Subscription.
type CreateSubscriptionInput struct {
	CustomerID string
	Plan       Plan
	Metadata   map[string]string
}

// CreateSubscription cria uma nova Subscription no Stripe com trial.
func (c *Client) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*Subscription, error) {
	if input.CustomerID == "" {
		return nil, errors.New("customer_id é obrigatório")
	}
	if !IsValidPlan(input.Plan) {
		return nil, fmt.Errorf("plano inválido: %s", input.Plan)
	}

	priceID, err := c.PlanToPriceID(input.Plan)
	if err != nil {
		return nil, err
	}

	body := strings.Builder{}
	body.WriteString("customer=" + input.CustomerID)
	body.WriteString("&items[0][price]=" + priceID)
	body.WriteString(fmt.Sprintf("&trial_period_days=%d", c.cfg.TrialDays))
	for k, v := range input.Metadata {
		body.WriteString(fmt.Sprintf("&metadata[%s]=%s", k, v))
	}

	url := c.base + "/v1/subscriptions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body.String()))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Stripe CreateSubscription: status %d: %s", resp.StatusCode, string(respBody))
	}

	return parseSubscriptionResponse(resp, c)
}

// GetSubscription retorna uma Subscription pelo ID.
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	url := c.base + "/v1/subscriptions/" + subscriptionID
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Stripe GetSubscription: status %d: %s", resp.StatusCode, string(respBody))
	}

	return parseSubscriptionResponse(resp, c)
}

// GetPortalURL gera URL do Stripe Customer Portal para o Customer.
func (c *Client) GetPortalURL(ctx context.Context, customerID, returnURL string) (string, error) {
	if customerID == "" {
		return "", errors.New("customer_id é obrigatório")
	}
	if returnURL == "" {
		return "", errors.New("return_url é obrigatório")
	}

	body := strings.Builder{}
	body.WriteString("customer=" + customerID)
	body.WriteString("&return_url=" + returnURL)

	url := c.base + "/v1/billing_portal/sessions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body.String()))
	if err != nil {
		return "", err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Stripe GetPortalURL: status %d: %s", resp.StatusCode, string(respBody))
	}

	var portalResp struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&portalResp); err != nil {
		return "", err
	}

	return portalResp.URL, nil
}

// CancelSubscription cancela uma Subscription.
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string) error {
	url := c.base + "/v1/subscriptions/" + subscriptionID
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Stripe CancelSubscription: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// parseSubscriptionResponse parseia resposta de subscription do Stripe.
func parseSubscriptionResponse(resp *http.Response, c *Client) (*Subscription, error) {
	var subResp struct {
		ID               string `json:"id"`
		Customer         string `json:"customer"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
		Created          int64  `json:"created"`
		TrialEnd         int64  `json:"trial_end"`
		Items            struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&subResp); err != nil {
		return nil, err
	}

	plan, _ := c.PriceIDToPlan(subResp.Items.Data[0].Price.ID)

	var trialEnds *time.Time
	if subResp.TrialEnd > 0 {
		t := time.Unix(subResp.TrialEnd, 0)
		trialEnds = &t
	}

	return &Subscription{
		ID:               subResp.ID,
		CustomerID:       subResp.Customer,
		Status:           subResp.Status,
		Plan:             plan,
		PriceID:          subResp.Items.Data[0].Price.ID,
		TrialEndsAt:      trialEnds,
		CurrentPeriodEnd: time.Unix(subResp.CurrentPeriodEnd, 0),
		CreatedAt:        time.Unix(subResp.Created, 0),
	}, nil
}

// PlanToPriceID retorna o Price ID do Stripe para um plano.
func (c *Client) PlanToPriceID(plan Plan) (string, error) {
	switch plan {
	case PlanLite:
		return c.cfg.PriceLite, nil
	case PlanPro:
		return c.cfg.PricePro, nil
	case PlanScale:
		return c.cfg.PriceScale, nil
	case PlanEnterprise:
		return c.cfg.PriceEnterprise, nil
	}
	return "", fmt.Errorf("plano desconhecido: %s", plan)
}

// PriceIDToPlan retorna o plano para um Price ID do Stripe.
func (c *Client) PriceIDToPlan(priceID string) (Plan, error) {
	switch priceID {
	case c.cfg.PriceLite:
		return PlanLite, nil
	case c.cfg.PricePro:
		return PlanPro, nil
	case c.cfg.PriceScale:
		return PlanScale, nil
	case c.cfg.PriceEnterprise:
		return PlanEnterprise, nil
	}
	return "", fmt.Errorf("price_id desconhecido: %s", priceID)
}

func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.cfg.SecretKey)
	req.Header.Set("Stripe-Version", "2024-12-18.acacia")
}

// VerifyWebhookSignature verifica a assinatura de um webhook Stripe.
// Retorna nil se válida, erro caso contrário.
func (c *Client) VerifyWebhookSignature(payload []byte, sigHeader string) error {
	if c.cfg.WebhookSecret == "" {
		return errors.New("webhook secret não configurado")
	}

	// Parse: "t=timestamp,v1=signature"
	var timestamp, signature string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signature = kv[1]
		}
	}

	if timestamp == "" || signature == "" {
		return errors.New("assinatura malformada")
	}

	// Verifica timestamp (máximo 5 minutos).
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("timestamp inválido")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return errors.New("webhook timestamp muito antigo")
	}

	// Computa expected: HMAC-SHA256(timestamp + "." + payload, secret).
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(c.cfg.WebhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmacEqual([]byte(expected), []byte(signature)) {
		return errors.New("assinatura inválida")
	}

	return nil
}

// hmacEqual compara HMACs em tempo constante para evitar timing attacks.
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
