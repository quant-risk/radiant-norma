package webhook

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"
)

// Dispatcher enqueues webhook deliveries and processes them asynchronously.
type Dispatcher struct {
	db    *sql.DB
	queue chan deliveryJob
	wg    sync.WaitGroup
}

// deliveryJob represents a pending delivery.
type deliveryJob struct {
	ID        string
	WebhookID string
	Event     string
	Payload   string
}

// NewDispatcher creates a dispatcher with a background worker pool.
func NewDispatcher(db *sql.DB) *Dispatcher {
	d := &Dispatcher{
		db:    db,
		queue: make(chan deliveryJob, 1000),
	}
	// 4 workers process deliveries.
	for i := 0; i < 4; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}
	return d
}

// Enqueue adds a delivery job to the queue.
func (d *Dispatcher) Enqueue(id, webhookID, event, payload string) {
	select {
	case d.queue <- deliveryJob{ID: id, WebhookID: webhookID, Event: event, Payload: payload}:
	default:
		// queue full — drop (should not happen in practice)
	}
}

// worker processes delivery jobs.
func (d *Dispatcher) worker(idx int) {
	defer d.wg.Done()
	for job := range d.queue {
		d.processJob(job)
	}
}

func (d *Dispatcher) processJob(job deliveryJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Insert delivery record first.
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempt)
		 VALUES (?, ?, ?, ?, 'pending', 0)`,
		job.ID, job.WebhookID, job.Event, job.Payload)
	if err != nil {
		slog.Error("webhook deliver: failed to insert delivery record", "id", job.ID, "err", err)
		return
	}

	// Fetch webhook URL + secret.
	var url, secret string
	err = d.db.QueryRowContext(ctx,
		"SELECT url, secret FROM webhooks WHERE id = ? AND active = 1", job.WebhookID,
	).Scan(&url, &secret)

	if err != nil {
		if err == sql.ErrNoRows {
			d.markDone(job.ID, "failed", 0, "webhook not found")
		} else {
			// Real DB error — mark failed so delivery isn't stuck in pending.
			d.markDone(job.ID, "failed", 0, err.Error())
		}
		return
	}

	// Attempt delivery.
	statusCode, respBody, delErr := deliver(ctx, url, job.Event, job.Payload, secret)

	if delErr != nil {
		if isRetryable(delErr) {
			d.scheduleRetry(job, statusCode, respBody)
			return
		}
		d.markDone(job.ID, "failed", statusCode, respBody)
		return
	}

	d.markDone(job.ID, "success", statusCode, respBody)
}

func (d *Dispatcher) markDone(id, status string, httpStatus int, respBody string) {
	if len(respBody) > 2048 {
		respBody = respBody[:2048]
	}
	_, _ = d.db.ExecContext(context.Background(),
		`UPDATE webhook_deliveries SET status=?, http_status=?, response_body=?,
		 delivered_at=CURRENT_TIMESTAMP WHERE id=?`,
		status, httpStatus, respBody, id)
}

func (d *Dispatcher) scheduleRetry(job deliveryJob, httpStatus int, respBody string) {
	if len(respBody) > 2048 {
		respBody = respBody[:2048]
	}
	attempt := 1
	_ = d.db.QueryRowContext(context.Background(),
		"SELECT attempt FROM webhook_deliveries WHERE id=?", job.ID,
	).Scan(&attempt)

	if attempt >= 5 {
		d.markDone(job.ID, "failed", httpStatus, respBody)
		return
	}

	backoffs := []time.Duration{1, 5, 15, 30, 60}
	delay := time.Minute
	if attempt < len(backoffs) {
		delay = backoffs[attempt]
	}
	nextRetry := time.Now().Add(delay)

	_, _ = d.db.ExecContext(context.Background(),
		`UPDATE webhook_deliveries SET status='retrying', attempt=?, http_status=?,
		 response_body=?, next_retry_at=? WHERE id=?`,
		attempt+1, httpStatus, respBody, nextRetry, job.ID)
}

// Close gracefully shuts down the dispatcher.
func (d *Dispatcher) Close() {
	close(d.queue)
	d.wg.Wait()
}
