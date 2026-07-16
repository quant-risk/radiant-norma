// Package webhook implementa webhooks outbound da Radiant Norma.
//
// Permite que tenants registrem callbacks HTTP que são chamados quando eventos
// ocorrem (ex: validação completada, schema mudou, radar detectou mudança).
//
// Eventos disponíveis:
//   - validation.completed  — documento validado (sucesso ou com erros)
//   - schema.changed        — novo schema versionado
//   - radar.change_detected — radar detectou mudança de layout
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Now returns current time — injectable for testing.
var Now = time.Now

// Service gerencia webhooks de um tenant.
type Service struct {
	db *sql.DB
	ds *Dispatcher
}

// NewService cria um webhook service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db, ds: NewDispatcher(db)}
}

// NewServiceWithoutDispatcher cria um webhook service sem dispatcher de fundo.
// Útil para testes onde não se deseja processamento assíncrono (evita race
// conditions com SQLite in-memory).
func NewServiceWithoutDispatcher(db *sql.DB) *Service {
	return &Service{db: db, ds: nil}
}

// Webhook representa um webhook registrado.
type Webhook struct {
	ID          string    `json:"id"`
	IFID        string    `json:"if_id"`
	URL         string    `json:"url"`
	Events      []string  `json:"events"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// List retorna todos os webhooks ativos de um tenant.
func (s *Service) List(ctx context.Context, ifID string) ([]Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, if_id, url, events, description, active, created_at, updated_at
		FROM webhooks
		WHERE if_id = ? AND active = 1
		ORDER BY created_at DESC
	`, ifID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var out []Webhook
	for rows.Next() {
		var w Webhook
		var eventsCSV, desc sql.NullString
		if err := rows.Scan(&w.ID, &w.IFID, &w.URL, &eventsCSV, &desc, &w.Active, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		w.Events = strings.Split(eventsCSV.String, ",")
		w.Description = desc.String
		out = append(out, w)
	}
	return out, rows.Err()
}

// Register cria um novo webhook.
func (s *Service) Register(ctx context.Context, ifID, url, events, description, secret string) (*Webhook, error) {
	if err := validateURL(url); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	events = strings.TrimSpace(strings.ToLower(events))
	if events == "" {
		events = "*"
	}

	id := newID()
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, if_id, url, events, description, secret, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, id, ifID, url, events, description, secret, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert webhook: %w", err)
	}

	return &Webhook{
		ID:          id,
		IFID:        ifID,
		URL:         url,
		Events:      strings.Split(events, ","),
		Description: description,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Delete desativa um webhook (soft delete).
func (s *Service) Delete(ctx context.Context, ifID, webhookID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE webhooks SET active = 0, updated_at = ? WHERE id = ? AND if_id = ? AND active = 1
	`, time.Now(), webhookID, ifID)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("webhook não encontrado")
	}
	return nil
}

// Dispatch dispara um evento para todos os webhooks registrados.
func (s *Service) Dispatch(ctx context.Context, ifID, event string, payload any) {
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := string(payloadBytes)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, secret FROM webhooks
		WHERE if_id = ? AND active = 1
		AND (events = '*' OR ',' || events || ',' LIKE '%,' || ? || ',%')
	`, ifID, event)
	if err != nil {
		slog.Error("webhook dispatch query failed", "if_id", ifID, "event", event, "err", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, url, secret string
		if err := rows.Scan(&id, &url, &secret); err != nil {
			continue
		}
		if s.ds != nil {
			// Phase 5: use EnqueueAndInsert to create delivery record before enqueuing.
			s.ds.EnqueueAndInsert(id, event, payloadStr)
		}

		// Fire-and-forget: worker processa a entrega.
		// Se secret existe, calcula HMAC-SHA256 do payload.
		_ = secret // used in deliverer
		_ = url    // used in deliverer
	}
}

// Deliver enqueues a delivery for processing by the background worker.
// Phase 5: now uses EnqueueAndInsert to create the delivery record before enqueuing.
func (s *Service) Deliver(webhookID, event, payload string) string {
	if s.ds == nil {
		return ""
	}
	return s.ds.EnqueueAndInsert(webhookID, event, payload)
}

// Delivery represents a webhook delivery attempt.
type Delivery struct {
	ID          string    `json:"id"`
	WebhookID   string    `json:"webhook_id"`
	Event       string    `json:"event"`
	Payload     string    `json:"payload"`
	Status      string    `json:"status"`
	HTTPStatus  int       `json:"http_status,omitempty"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"created_at"`
	DeliveredAt time.Time `json:"delivered_at,omitempty"`
}

// ListDeliveries returns recent deliveries for a webhook.
// Returns ErrWebhookNotFound if the webhook doesn't exist or belongs to another tenant.
func (s *Service) ListDeliveries(ctx context.Context, ifID, webhookID string, limit int) ([]Delivery, error) {
	// First verify the webhook belongs to this tenant.
	var exists int
	if err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM webhooks WHERE id = ? AND if_id = ? AND active = 1`,
		webhookID, ifID).Scan(&exists); err == sql.ErrNoRows {
		return nil, fmt.Errorf("webhook not found")
	} else if err != nil {
		return nil, fmt.Errorf("check webhook: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT wd.id, wd.webhook_id, wd.event, wd.payload, wd.status,
		       COALESCE(wd.http_status, 0), wd.attempt, wd.created_at, wd.delivered_at
		FROM webhook_deliveries wd
		JOIN webhooks w ON w.id = wd.webhook_id
		WHERE w.if_id = ? AND wd.webhook_id = ?
		ORDER BY wd.created_at DESC
		LIMIT ?
	`, ifID, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()

	var out []Delivery
	for rows.Next() {
		var d Delivery
		var deliveredAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.Status,
			&d.HTTPStatus, &d.Attempt, &d.CreatedAt, &deliveredAt); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		if deliveredAt.Valid {
			d.DeliveredAt = deliveredAt.Time
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDelivery returns a single delivery by ID (tenant-scoped).
func (s *Service) GetDelivery(ctx context.Context, ifID, webhookID, deliveryID string) (*Delivery, error) {
	var d Delivery
	var deliveredAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT wd.id, wd.webhook_id, wd.event, wd.payload, wd.status,
		       COALESCE(wd.http_status, 0), wd.attempt, wd.created_at, wd.delivered_at
		FROM webhook_deliveries wd
		JOIN webhooks w ON w.id = wd.webhook_id
		WHERE w.if_id = ? AND wd.webhook_id = ? AND wd.id = ?
	`, ifID, webhookID, deliveryID).Scan(
		&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.Status,
		&d.HTTPStatus, &d.Attempt, &d.CreatedAt, &deliveredAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delivery not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get delivery: %w", err)
	}
	if deliveredAt.Valid {
		d.DeliveredAt = deliveredAt.Time
	}
	return &d, nil
}

// RetryDelivery resets a failed delivery back to 'pending' so the dispatcher
// will re-attempt it. Only works on deliveries in 'failed' status.
// Tenant-scoped via IFID check on the webhook.
func (s *Service) RetryDelivery(ctx context.Context, ifID, webhookID, deliveryID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status='pending', attempt=0, http_status=0, response_body=NULL
		WHERE id = ?
		AND webhook_id = ?
		AND status = 'failed'
		AND EXISTS (SELECT 1 FROM webhooks WHERE id = ? AND if_id = ?)
	`, deliveryID, webhookID, webhookID, ifID)
	if err != nil {
		return fmt.Errorf("retry delivery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("delivery not found or not in failed status")
	}
	// Re-enqueue via dispatcher if available.
	if s.ds != nil {
		var payload, event string
		_ = s.db.QueryRowContext(ctx,
			"SELECT payload, event FROM webhook_deliveries WHERE id=?", deliveryID,
		).Scan(&payload, &event)
		if payload != "" {
			s.ds.Enqueue(deliveryID, webhookID, event, payload)
		}
	}
	return nil
}

// DispatchSchemaChanged fires a schema.changed webhook event.
func (s *Service) DispatchSchemaChanged(ctx context.Context, ifID, cadoc, effectiveFrom, changelog string) {
	if s == nil {
		return
	}
	evt := EventSchemaChanged{}
	evt.WebhookBase = WebhookBase{Event: "schema.changed", Timestamp: Now(), IFID: ifID}
	evt.Cadoc = cadoc
	evt.EffectiveFrom = effectiveFrom
	evt.Changelog = changelog
	s.Dispatch(ctx, ifID, "schema.changed", evt)
}

// DispatchRadarChangeDetected fires a radar.change_detected webhook event.
func (s *Service) DispatchRadarChangeDetected(ctx context.Context, ifID, cadoc, scanID string, changes []Change) {
	if s == nil {
		return
	}
	evt := EventRadarChangeDetected{}
	evt.WebhookBase = WebhookBase{Event: "radar.change_detected", Timestamp: Now(), IFID: ifID}
	evt.Cadoc = cadoc
	evt.ScanID = scanID
	evt.Changes = changes
	s.Dispatch(ctx, ifID, "radar.change_detected", evt)
}

// DispatchSubmissionAccepted fires a submission.accepted webhook event.
func (s *Service) DispatchSubmissionAccepted(ctx context.Context, ifID, cadoc, dataBase, protocolo, xmlHash string) {
	if s == nil {
		return
	}
	evt := EventSubmissionAccepted{}
	evt.WebhookBase = WebhookBase{Event: "submission.accepted", Timestamp: Now(), IFID: ifID}
	evt.Cadoc = cadoc
	evt.DataBase = dataBase
	evt.Protocolo = protocolo
	evt.XMLHash = xmlHash
	s.Dispatch(ctx, ifID, "submission.accepted", evt)
}

// DispatchSubmissionRejected fires a submission.rejected webhook event.
func (s *Service) DispatchSubmissionRejected(ctx context.Context, ifID, cadoc, dataBase, protocolo, reason string) {
	if s == nil {
		return
	}
	evt := EventSubmissionRejected{}
	evt.WebhookBase = WebhookBase{Event: "submission.rejected", Timestamp: Now(), IFID: ifID}
	evt.Cadoc = cadoc
	evt.DataBase = dataBase
	evt.Protocolo = protocolo
	evt.Reason = reason
	s.Dispatch(ctx, ifID, "submission.rejected", evt)
}

// ============================================================
// Event types
// ====================================

// EventValidationCompleted fired when a CADOC validation finishes.
type EventValidationCompleted struct {
	WebhookBase
	Cadoc      string `json:"cadoc"`
	DataBase   string `json:"data_base"`
	Valid      bool   `json:"valid"`
	ErrorCount int    `json:"error_count"`
	WarnCount  int    `json:"warning_count"`
	XMLHash    string `json:"xml_hash"`
}

// EventSchemaChanged fired when a new schema version is published.
type EventSchemaChanged struct {
	WebhookBase
	Cadoc         string `json:"cadoc"`
	EffectiveFrom string `json:"effective_from"`
	Changelog     string `json:"changelog,omitempty"`
}

// EventRadarChangeDetected fired when radar finds layout changes.
type EventRadarChangeDetected struct {
	WebhookBase
	Cadoc   string   `json:"cadoc"`
	ScanID  string   `json:"scan_id"`
	Changes []Change `json:"changes"`
}

// Change represents a detected layout change.
type Change struct {
	Tag      string `json:"tag"`
	Attr     string `json:"attr,omitempty"`
	Kind     string `json:"kind"` // added, removed, modified
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// EventSubmissionAccepted fired when a submission is accepted by BACEN STA.
type EventSubmissionAccepted struct {
	WebhookBase
	Cadoc     string `json:"cadoc"`
	DataBase  string `json:"data_base"`
	Protocolo string `json:"protocolo"`
	XMLHash   string `json:"xml_hash,omitempty"`
}

// EventSubmissionRejected fired when a submission is rejected by BACEN STA.
type EventSubmissionRejected struct {
	WebhookBase
	Cadoc     string `json:"cadoc"`
	DataBase  string `json:"data_base"`
	Protocolo string `json:"protocolo"`
	Reason    string `json:"reason,omitempty"`
}

// WebhookBase is embedded in every event.
type WebhookBase struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	IFID      string    `json:"if_id"`
}

// SignPayload returns HMAC-SHA256 signature of payload given a secret.
func SignPayload(payload, secret string) string {
	if secret == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// ============================================================
// Validation helpers
// ============================================================

func validateURL(rawURL string) error {
	if len(rawURL) < 10 || len(rawURL) > 2048 {
		return fmt.Errorf("url length must be 10-2048 chars")
	}
	if !(strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://")) {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}
