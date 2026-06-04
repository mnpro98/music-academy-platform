package models

import (
	"encoding/json"
	"time"
)

type Student struct {
	ID              int       `json:"id" db:"id"`
	FirstName       string    `json:"first_name" db:"first_name"`
	LastName        string    `json:"last_name" db:"last_name"`
	Gender          string    `json:"gender" db:"gender"`
	DateOfBirth     time.Time `json:"date_of_birth" db:"date_of_birth"`
	InscriptionDate time.Time `json:"inscription_date" db:"inscription_date"`
	Instrument      string    `json:"instrument" db:"instrument"`
	Program         string    `json:"program" db:"program"`
	AmountOwed      float64   `json:"amount_owed" db:"amount_owed"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type OutboxEvent struct {
	ID          string          `json:"id" db:"id"`
	EventType   string          `json:"event_type" db:"event_type"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	Processed   bool            `json:"processed" db:"processed"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
}
