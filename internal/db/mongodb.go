package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureMongoIndexes temporarily dials the MongoDB instance to assert required pipeline indices.
func EnsureMongoIndexes(ctx context.Context, mongoURI string) error {
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// Create an explicit allocation ceiling timeout for setup execution
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to establish migration connection to mongodb: %w", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Warning: Non-critical failure during mongo driver disconnection: %v\n", err)
		}
	}()

	collection := client.Database("academy_analytics").Collection("student_ledgers")

	// Construct the unique search pointer index configuration
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "student_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	log.Println("Asserting runtime search index: academy_analytics.student_ledgers -> { student_id: 1 }...")
	_, err = collection.Indexes().CreateOne(connectCtx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create programmatic unique index target: %w", err)
	}

	log.Println("MongoDB system collection validation completely synchronized.")
	return nil
}
