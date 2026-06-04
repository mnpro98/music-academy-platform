package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type SimulatedStudent struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Gender      string  `json:"gender"`
	DateOfBirth string  `json:"date_of_birth"`
	Instrument  string  `json:"instrument"`
	Program     string  `json:"program"`
	AmountOwed  float64 `json:"amount_owed"`
}

var firstNames = []string{"Mauricio", "Valeria", "Ernesto", "Sofia", "Alejandro", "Camila", "Diego", "Isabella"}
var lastNames = []string{"Nanez", "Gonzalez", "Mejilla", "Pro", "Rodriguez", "Martinez", "Castillo", "Sanchez"}
var instruments = []string{"vocals", "guitar", "piano", "drums", "bass"}
var programs = []string{"basic", "full"}
var genders = []string{"male", "female"}

func main() {
	log.Println("Starting Automated Music Academy Registration Generator...")

	apiHost := getEnv("INGESTION_API_HOST", "http://localhost:8080")
	targetURL := fmt.Sprintf("%s/api/v1/students", apiHost)

	// Read the generation interval configuration or use 250ms as a fallback
	intervalStr := getEnv("GENERATION_INTERVAL", "250ms")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Printf("Warning: Invalid GENERATION_INTERVAL '%s' (%v). Falling back to 250ms.\n", intervalStr, err)
		interval = 250 * time.Millisecond
	}

	// Initialize the ticker using the dynamically calculated duration
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Printf("Target API endpoint verified: %s. Initiating generation loop...\n", targetURL)

	for {
		select {
		case <-stopSignal:
			log.Println("Safely stopping mock generation service.")
			return
		case <-ticker.C:
			student := generateMockStudent(rng)

			payloadBytes, err := json.Marshal(student)
			if err != nil {
				log.Printf("Serialization error: %v\n", err)
				continue
			}

			resp, err := httpClient.Post(targetURL, "application/json", bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Network delivery failure (Is Ingestion API alive?): %v\n", err)
				continue
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusCreated {
				log.Printf("[✓] Ingested: %s %s | Owed: $%.2f | Instrument: %s\n",
					student.FirstName, student.LastName, student.AmountOwed, student.Instrument)
			} else {
				log.Printf("[X] API rejected registration payload with Status Code: %d\n", resp.StatusCode)
			}
		}
	}
}

func generateMockStudent(r *rand.Rand) SimulatedStudent {
	birthYear := r.Intn(38) + 1980
	birthMonth := r.Intn(12) + 1

	var balance float64
	if r.Float32() > 0.30 {
		balance = float64((r.Intn(6) + 1) * 1000) // Generates $1000, $2000... up to $6000
	} else {
		balance = 0.0 // Active verification record for the pipeline filter bypass rules
	}

	return SimulatedStudent{
		FirstName:   firstNames[r.Intn(len(firstNames))],
		LastName:    lastNames[r.Intn(len(lastNames))],
		Gender:      genders[r.Intn(len(genders))],
		DateOfBirth: fmt.Sprintf("%d-%02d-01", birthYear, birthMonth),
		Instrument:  instruments[r.Intn(len(instruments))],
		Program:     programs[r.Intn(len(programs))],
		AmountOwed:  balance,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
