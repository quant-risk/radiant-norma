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

// EnqueueAndInsert creates the delivery record and enqueues it.
// Phase 5: fixes the missing INSERT that caused processJob to never find records.
func (d *Dispatcher) EnqueueAndInsert(webhookID, event, payload string) string {
	id := newID()
	_, err := d.db.ExecContext(context.Background(),
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempt, created_at)
		 VALUES (?, ?, ?, ?, 'pending', 0, CURRENT_TIMESTAMP)`,
		id, webhookID, event, payload)
	if err != nil {
		slog.Error("webhook enqueue: failed to insert delivery record",
			"webhook_id", webhookID, "event", event, "err", err)
		return id
	}
	d.Enqueue(id, webhookID, event, payload)
	return id
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

	// Ensure delivery record exists with status=pending. Uses UPDATE so the
	// row (created by EnqueueAndInsert) is preserved — avoiding the
	// DELETE+INSERT cycle of INSERT OR REPLACE which races with concurrent
	// goroutines on the same SQLite connection in the test environment.
	res, err := d.db.ExecContext(ctx,
		`UPDATE webhook_deliveries SET status='pending' WHERE id=? AND webhook_id=?`,
		job.ID, job.WebhookID)
	if err != nil {
		slog.Error("webhook deliver: failed to update delivery record", "id", job.ID, "err", err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		slog.Warn("webhook deliver: delivery record not found, skipping", "id", job.ID, "webhook_id", job.WebhookID)
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
		// Phase 5: retryable decision based on error + status code.
		if isRetryable(delErr, statusCode) {
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
