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


        CREATE TABLE IF NOT EXISTS resource_limits (
            resource_name TEXT PRIMARY KEY,  -- e.g., 'encode', 'thumbnail', 'metadata'
            max_concurrency INTEGER NOT NULL,
            current_inflight INTEGER NOT NULL DEFAULT 0,
            updated_at DATETIME NOT NULL
        );

        -- Seed initial resource limits
        INSERT OR IGNORE INTO resource_limits (resource_name, max_concurrency, updated_at)
        VALUES 
            ('validate', 5, DATETIME('now')),
            ('metadata', 10, DATETIME('now')),
            ('thumbnail', 3, DATETIME('now')),
            ('encode', 2, DATETIME('now'));

        DROP VIEW IF EXISTS job_invariants_violations;
        CREATE VIEW job_invariants_violations AS
        -- Violation 1: Job running with expired lease
        SELECT id, 'running_expired_lease' as violation
        FROM jobs
        WHERE state = 'running' AND lease_expires_at < datetime('now')

        UNION ALL

        -- Violation 2: Job succeeded but not expanded for > 30s
        SELECT id, 'succeeded_not_expanded' as violation
        FROM jobs
        WHERE state = 'succeeded' 
          AND expanded = FALSE 
          AND updated_at < datetime('now', '-30 seconds')

        UNION ALL

        -- Violation 3: Job exhausted attempts but not dead or succeeded
        SELECT id, 'max_attempts_exceeded' as violation
        FROM jobs
        WHERE attempt >= max_attempts 
          AND state NOT IN ('dead', 'succeeded');
        `)
	// CREATE UNIQUE INDEX uniq_parent_job_type
	// ON jobs(parent_job_id, job_type);
	return err
}
