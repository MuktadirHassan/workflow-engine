package jobs

import (
	"database/sql"
	"log/slog"
)

type Repo struct {
	Db *sql.DB
}

//  CREATE TABLE IF NOT EXISTS jobs (
//             id TEXT PRIMARY KEY,
//             video_id TEXT NOT NULL,
//             job_type TEXT NOT NULL,
//             state TEXT NOT NULL,
//             attempt INTEGER NOT NULL,
//             max_attempts INTEGER NOT NULL,
//             lease_owner TEXT,
//             lease_expires_at DATETIME,
//             input_path TEXT NOT NULL,
//             output_path TEXT,
//             error TEXT,
//             created_at DATETIME NOT NULL,
//             updated_at DATETIME NOT NULL
//         );

func (r *Repo) AcquireLease(workerID string, leaseSeconds string) (*Job, error) {
	selectPendingJobsQuery := `
		SELECT id FROM jobs
		WHERE state = 'pending'
		ORDER BY created_at
		LIMIT 1;
	`
	var jobID string
	j := r.Db.QueryRow(selectPendingJobsQuery)
	err := j.Scan(&jobID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE jobs
		SET
			state = 'running',
			lease_owner = ?,
			lease_expires_at = DATETIME('now', '+' || ? || ' seconds'),
			updated_at = DATETIME('now')
		WHERE id = ?
		AND state = 'pending';
	`
	res, err := r.Db.Exec(query, workerID, leaseSeconds, jobID)
	rows, _ := res.RowsAffected()
	if rows == 0 {
		slog.Info("[worker] failed to acquire lease, another worker might have taken it")
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// Fetch the leased job details
	query = `
		SELECT id, state, lease_owner, lease_expires_at, created_at, updated_at
		FROM jobs
		WHERE lease_owner = ? AND state = 'running' AND id = ?
		ORDER BY updated_at DESC
		LIMIT 1;
	`
	row := r.Db.QueryRow(query, workerID, jobID)
	var job Job
	err = row.Scan(&job.ID, &job.State, &job.LeaseOwner, &job.LeaseExpiresAt, &job.CreatedAt, &job.UpdatedAt)

	// sql: no rows in result set
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &job, err
}

func (r *Repo) FailJob(jobID string, workerID string, jobErr error) error {
	query := `
		UPDATE jobs
		SET
			attempt = attempt + 1,
			state = CASE
				WHEN attempt + 1 >= max_attempts THEN 'dead'
				ELSE 'pending'
			END,
			lease_owner = NULL,
			lease_expires_at = NULL,
			error = ?,
			updated_at = DATETIME('now')
		WHERE id = ?
		AND lease_owner = ?;
	`
	res, err := r.Db.Exec(query, jobErr.Error(), jobID, workerID)
	rows, _ := res.RowsAffected()
	if rows == 0 {
		slog.Warn("[worker] lost lease while completing job...")
	}
	return err
}

func (r *Repo) SucceedJob(jobID string, workerID string) error {
	query := `
	UPDATE jobs
	SET
		state = 'succeeded',
		lease_owner = NULL,
		lease_expires_at = NULL,
		updated_at = DATETIME('now')
	WHERE id = ?
	AND lease_owner = ?;`
	res, err := r.Db.Exec(query, jobID, workerID)
	rows, _ := res.RowsAffected()
	if rows == 0 {
		slog.Warn("[worker] lost lease while completing job...")
	}
	return err
}

func (r *Repo) ReclaimExpiredLeases() error {
	query := `
	UPDATE jobs 
	SET state = 'pending', 
		lease_owner = NULL, 
		lease_expires_at = NULL,
		attempt = attempt + 1
	WHERE state = 'running' 
	AND lease_expires_at < CURRENT_TIMESTAMP
	AND attempt < max_attempts;
	`
	_, err := r.Db.Exec(query)
	return err
}

func (r *Repo) FindSucceededUnexpandedJobs() ([]Job, error) {
	query := `
	SELECT id, state, lease_owner, lease_expires_at, created_at, updated_at
	FROM jobs
	WHERE state = 'succeeded' AND expanded = FALSE
	`
	rows, err := r.Db.Query(query)
	var job []Job
	for rows.Next() {
		var j Job
		err := rows.Scan(&j.ID, &j.State, &j.LeaseOwner, &j.LeaseExpiresAt, &j.CreatedAt, &j.UpdatedAt)
		if err != nil {
			return nil, err
		}
		job = append(job, j)
	}

	// sql: no rows in result set
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return job, err
}

func (r *Repo) MarkJobAsExpanded(jobID string) error {
	query := `
	UPDATE jobs
	SET expanded = TRUE,
		updated_at = DATETIME('now')
	WHERE id = ?;
	`
	_, err := r.Db.Exec(query, jobID)
	return err
}

func (r *Repo) InsertJobIfNotExists(job Job) error {
	query := `
	INSERT INTO jobs (id, video_id, job_type, state, attempt, max_attempts, input_path, output_path, created_at, updated_at)
	SELECT ?, ?, ?, ?, ?, ?, ?, ?, DATETIME('now'), DATETIME('now')
	WHERE NOT EXISTS (SELECT 1 FROM jobs WHERE id = ?);
	`
	_, err := r.Db.Exec(query, job.ID, job.VideoID, job.JobType, job.State, job.Attempt, job.MaxAttempts, job.InputPath, job.OutputPath, job.ID)
	return err
}
