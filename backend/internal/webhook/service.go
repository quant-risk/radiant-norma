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
		deliveryID := newID()
		s.ds.Enqueue(deliveryID, id, event, payloadStr)

		// Fire-and-forget: worker processa a entrega.
		// Se secret existe, calcula HMAC-SHA256 do payload.
		_ = secret // used in deliverer
		_ = url    // used in deliverer
		_ = deliveryID
	}
}

// Deliver enqueues a delivery for processing by the background worker.
func (s *Service) Deliver(ctx context.Context, webhookID, event, payload string) string {
	id := newID()
	s.ds.Enqueue(id, webhookID, event, payload)
	return id
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
func (s *Service) ListDeliveries(ctx context.Context, ifID, webhookID string, limit int) ([]Delivery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT wd.id, wd.webhook_id, wd.event, wd.payload, wd.status,
		       COALESCE(wd.http_status, 0), wd.attempt, wd.created_at, COALESCE(wd.delivered_at, '1970-01-01')
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
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.Status,
			&d.HTTPStatus, &d.Attempt, &d.CreatedAt, &d.DeliveredAt); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ============================================================
// Event types
// ============================================================

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
