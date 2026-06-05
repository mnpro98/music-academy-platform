package worker

import (
	"context"
	"log"
	"time"

	"music-academy-platform/internal/db"
	"music-academy-platform/internal/models"

	"github.com/nats-io/nats.go"
)

// OutboxPublisher manages the background polling cycle and streaming operations.
type OutboxPublisher struct {
	db        *db.DB
	natsConn  *nats.Conn
	jsContext nats.JetStreamContext
	interval  time.Duration
}

// NewOutboxPublisher initializes the publisher with database and NATS connections.
func NewOutboxPublisher(database *db.DB, natsURL string, pollInterval time.Duration) (*OutboxPublisher, error) {
	// Connect to the central NATS broker
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(5),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// Initialize the JetStream management context for persistent streaming capabilities
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	// Ensure the target NATS Stream exists before starting the worker
	// This ensures that any messages published to 'academy.student.*' are stored securely.
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "ACADEMY_EVENTS",
		Subjects: []string{"academy.student.*"},
		Storage:  nats.FileStorage, // Retains messages safely on disk inside the broker
	})
	if err != nil {
		log.Printf("Note: Stream initialization check returned: %v (It may already exist)\n", err)
	}

	return &OutboxPublisher{
		db:        database,
		natsConn:  nc,
		jsContext: js,
		interval:  pollInterval,
	}, nil
}

// Close gracefully terminates network connections when the worker shuts down.
func (p *OutboxPublisher) Close() {
	if p.natsConn != nil {
		log.Println("Disconnecting cleanly from NATS broker...")
		p.natsConn.Close()
	}
}

// Start boots up the ticking operational loop. It blocks until the context is canceled.
func (p *OutboxPublisher) Start(ctx context.Context) {
	log.Printf("Outbox background worker started. Scanning every %v for unprocessed logs...\n", p.interval)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox publisher shutting down gracefully...")
			return
		case <-ticker.C:
			if err := p.ProcessOutbox(ctx); err != nil {
				log.Printf("Error processing outbox: %v\n", err)
			}
		}
	}
}

// ProcessOutbox scans the database for unprocessed events and streams them downstream.
func (p *OutboxPublisher) ProcessOutbox(ctx context.Context) error {
	// 1. Open an isolation transaction block for safe multi-row updates
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 2. Query unprocessed events.
	// HIGH CONCURRENCY TWEAK: 'FOR UPDATE SKIP LOCKED' prevents multiple instances
	// of this worker from picking up or locking the exact same events simultaneously.
	query := `
		SELECT id, event_type, payload
		FROM outbox
		WHERE processed = false
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED;
	`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []models.OutboxEvent
	for rows.Next() {
		var event models.OutboxEvent
		if err := rows.Scan(&event.ID, &event.EventType, &event.Payload); err != nil {
			return err
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return tx.Commit()
	} // No data to process during this cycle

	log.Printf("Detected %d unprocessed registration events. Streaming to NATS JetStream...\n", len(events))

	// 3. Iterate over the events and publish them to the message broker
	updateQuery := `UPDATE outbox SET processed = true, processed_at = $1 WHERE id = $2;`

	for _, event := range events {
		subject := "academy.student." + event.EventType

		// Publish the raw JSON payload to NATS JetStream and await an explicit Acknowledgement (ACK)
		_, err = p.jsContext.Publish(subject, event.Payload)
		if err != nil {
			// If NATS is temporarily unavailable, skip this cycle.
			// Thanks to the outbox pattern, events remain completely safe in PostgreSQL.
			log.Printf("Streaming failed for event UUID %s: %v. Retrying in next cycle.\n", event.ID, err)
			return err
		}

		// 4. Update the outbox row to mark it as successfully processed
		_, err = tx.ExecContext(ctx, updateQuery, time.Now(), event.ID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Successfully streamed and acknowledged %d events.\n", len(events))
	return nil
}
