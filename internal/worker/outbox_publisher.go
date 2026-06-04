package worker

import (
	"context"
	"log"
	"time"

	"music-academy-platform/internal/db"
	"music-academy-platform/internal/models"

	"github.com/nats-io/nats.go"
)

type OutboxPublisher struct {
	db        *db.DB
	natsConn  *nats.Conn
	jsContext nats.JetStreamContext
	interval  time.Duration
}

func NewOutboxPublisher(database *db.DB, natsURL string, pollInterval time.Duration) (*OutboxPublisher, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(5),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "ACADEMY_EVENTS",
		Subjects: []string{"academy.student.*"},
		Storage:  nats.FileStorage,
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

func (p *OutboxPublisher) Close() {
	if p.natsConn != nil {
		log.Println("Disconnecting cleanly from NATS broker...")
		p.natsConn.Close()
	}
}

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

func (p *OutboxPublisher) ProcessOutbox(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	}

	log.Printf("Detected %d unprocessed registration events. Streaming to NATS JetStream...\n", len(events))

	updateQuery := `UPDATE outbox SET processed = true, processed_at = $1 WHERE id = $2;`

	for _, event := range events {
		subject := "academy.student." + event.EventType

		_, err = p.jsContext.Publish(subject, event.Payload)
		if err != nil {
			log.Printf("Streaming failed for event UUID %s: %v. Retrying in next cycle.\n", event.ID, err)
			return err
		}

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
