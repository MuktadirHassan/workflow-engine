package db

import "database/sql"

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS jobs (
            id TEXT PRIMARY KEY,
            video_id TEXT NOT NULL,
            job_type TEXT NOT NULL,
            state TEXT NOT NULL,
            attempt INTEGER NOT NULL,
            max_attempts INTEGER NOT NULL,
            lease_owner TEXT,
            lease_expires_at DATETIME,
            input_path TEXT NOT NULL,
            output_path TEXT,
            error TEXT,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            expanded BOOLEAN DEFAULT FALSE,
            parent_job_id TEXT
        );

        
        `)
	// CREATE UNIQUE INDEX uniq_parent_job_type
	// ON jobs(parent_job_id, job_type);
	return err
}
