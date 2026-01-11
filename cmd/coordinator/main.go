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
	conn := db.MustOpen(os.Getenv("DB_PATH"))
	if err := db.Migrate(conn); err != nil {
		panic(err)
	}

	c := coordinator.New(coordinator.Config{
		DB: conn,
	})

	c.Run(context.Background())
}
