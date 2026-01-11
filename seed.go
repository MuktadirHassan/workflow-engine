package main

import (
	// Seed a job for testing
	"database/sql"
	"log/slog"
	"os"

	"github.com/MuktadirHassan/video-processor/internal/db"
	"github.com/google/uuid"
)

func main() {
	conn := db.MustOpen(os.Getenv("DB_PATH"))
	err := db.Migrate(conn)
	if err != nil {
		panic(err)
	}

	seedJob(conn)
}

func seedJob(conn *sql.DB) {
	jobId := uuid.New().String()
	videoId := uuid.New().String()
	_, err := conn.Exec(`
		INSERT INTO jobs (
			id, video_id, job_type, state,
			attempt, max_attempts,
			input_path,
			created_at, updated_at
		) VALUES (
			'` + jobId + `', '` + videoId + `', 'fake',
			'pending',
			0, 3,
			'/tmp/input.mp4',
			DATETIME('now'), DATETIME('now')
		);
	`)
	if err != nil {
		slog.Error("failed to seed job", "error", err)
		return
	}

	slog.Info("seeded job into database")
}
