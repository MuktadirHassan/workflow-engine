package main

import (
	"context"
	"os"

	"github.com/MuktadirHassan/video-processor/internal/coordinator"
	"github.com/MuktadirHassan/video-processor/internal/db"
)

// Responsibilities:
// Scan for completed jobs
// Decide next job(s)
// Create new jobs
// Reap expired leases

// It does NOT:
// Touch video files
// Execute work

func main() {
	db := db.MustOpen(os.Getenv("DB_PATH"))

	c := coordinator.New(coordinator.Config{
		DB: db,
	})

	c.Run(context.Background())
}
