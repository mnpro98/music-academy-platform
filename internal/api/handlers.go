package api

import (
	"encoding/json"
	"music-academy-platform/internal/models"
	"net/http"
	"time"
)

type RegisterStudentRequest struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Gender      string  `json:"gender"`
	DateOfBirth string  `json:"date_of_birth"`
	Instrument  string  `json:"instrument"`
	Program     string  `json:"program"`
	AmountOwed  float64 `json:"amount_owed"`
}

func (s *Server) HandleRegisterStudent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterStudentRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
			return
		}

		if req.FirstName == "" || req.LastName == "" || req.Instrument == "" {
			http.Error(w, `{"error":"Missing mandatory registration fields"}`, http.StatusBadRequest)
			return
		}

		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err != nil {
			http.Error(w, `{"error":"Invalid date_of_birth format. Use YYYY-MM-DD"}`, http.StatusBadRequest)
			return
		}

		student := models.Student{
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			Gender:          req.Gender,
			DateOfBirth:     dob,
			InscriptionDate: time.Now(),
			Instrument:      req.Instrument,
			Program:         req.Program,
			AmountOwed:      req.AmountOwed,
			CreatedAt:       time.Now(),
		}

		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"Could not start transaction"}`, http.StatusInternalServerError)
			return
		}

		defer tx.Rollback()

		studentInsertQuery := `
			INSERT INTO students (first_name, last_name, gender, date_of_birth, inscription_date, instrument, program, amount_owed, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id;
		`
		err = tx.QueryRowContext(r.Context(), studentInsertQuery,
			student.FirstName, student.LastName, student.Gender, student.DateOfBirth,
			student.InscriptionDate, student.Instrument, student.Program, student.AmountOwed, student.CreatedAt,
		).Scan(&student.ID)

		if err != nil {
			http.Error(w, `{"error":"Failed to save student ledger record"}`, http.StatusInternalServerError)
			return
		}

		studentPayloadBytes, err := json.Marshal(student)
		if err != nil {
			http.Error(w, `{"error":"Failed to serialize event message"}`, http.StatusInternalServerError)
			return
		}

		outboxInsertQuery := `
			INSERT INTO outbox (event_type, payload, processed)
			VALUES ($1, $2, false);
		`
		_, err = tx.ExecContext(r.Context(), outboxInsertQuery, "StudentRegistered", studentPayloadBytes)
		if err != nil {
			http.Error(w, `{"error":"Failed to append transactional outbox log"}`, http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"Failed to commit transaction"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(student)
	}
}
