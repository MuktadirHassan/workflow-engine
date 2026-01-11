package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MuktadirHassan/workflow-engine/internal/db"
	"github.com/MuktadirHassan/workflow-engine/internal/worker"
)

// Responsibilities:
// Poll for jobs
// Acquire lease
// Execute exactly one job
// Update state
// Repeat

// It does NOT:
// Create jobs
// Decide what’s next
// Know about workflows

func main() {
	slog.Info("[worker] initializing worker")
	slog.Info("[worker] starting db migration")
	dbPath := os.Getenv("DB_PATH")
	workerID := os.Getenv("WORKER_ID")
	leaseSeconds := os.Getenv("LEASE_SECONDS")

	conn := db.MustOpen(dbPath)
	err := db.Migrate(conn)
	if err != nil {
		slog.Error("[worker] failed to migrate database", "error", err)
		return
	}
	slog.Info("[worker] database migration completed")

	w := worker.New(worker.Config{
		DB:           conn,
		WorkerID:     workerID,
		LeaseSeconds: leaseSeconds,
	})

	w.Run(context.Background())

}
