package jobs

import (
	"database/sql"
	"log/slog"
	"time"
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
		SELECT id, video_id, job_type, state, attempt, max_attempts, lease_owner, lease_expires_at, 
		       input_path, output_path, error, created_at, updated_at, expanded, parent_job_id
		FROM jobs
		WHERE lease_owner = ? AND state = 'running' AND id = ?
		ORDER BY updated_at DESC
		LIMIT 1;
	`
	row := r.Db.QueryRow(query, workerID, jobID)
	var job Job
	err = row.Scan(&job.ID, &job.VideoID, &job.JobType, &job.State, &job.Attempt, &job.MaxAttempts,
		&job.LeaseOwner, &job.LeaseExpiresAt, &job.InputPath, &job.OutputPath, &job.Error,
		&job.CreatedAt, &job.UpdatedAt, &job.Expanded, &job.ParentJobID)

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
	rows, err := r.Db.Exec(query)
	if err != nil {
		return err
	}

	rowsAffected, _ := rows.RowsAffected()
	if rowsAffected > 0 {
		slog.Info("[coordinator] reclaimed expired leases", "count", rowsAffected)
	}
	return nil
}

func (r *Repo) FindSucceededUnexpandedJobs() ([]Job, error) {
	query := `
	SELECT id, video_id, job_type, state, attempt, max_attempts, lease_owner, lease_expires_at, 
	       input_path, output_path, error, created_at, updated_at, expanded, parent_job_id
	FROM jobs
	WHERE state = 'succeeded' AND expanded = FALSE
	`
	rows, err := r.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var job []Job
	for rows.Next() {
		var j Job
		err := rows.Scan(&j.ID, &j.VideoID, &j.JobType, &j.State, &j.Attempt, &j.MaxAttempts,
			&j.LeaseOwner, &j.LeaseExpiresAt, &j.InputPath, &j.OutputPath, &j.Error,
			&j.CreatedAt, &j.UpdatedAt, &j.Expanded, &j.ParentJobID)
		if err != nil {
			return nil, err
		}
		job = append(job, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return job, nil
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
	INSERT INTO jobs (id, video_id, job_type, state, attempt, max_attempts, input_path, output_path, 
	                  expanded, parent_job_id, created_at, updated_at)
	SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, DATETIME('now'), DATETIME('now')
	WHERE NOT EXISTS (SELECT 1 FROM jobs WHERE id = ?);
	`
	res, err := r.Db.Exec(query, job.ID, job.VideoID, job.JobType, job.State, job.Attempt, job.MaxAttempts,
		job.InputPath, job.OutputPath, job.Expanded, job.ParentJobID, job.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		slog.Info("[coordinator] job already exists, skipping insert", "job_id", job.ID)
	}

	return err
}

func (r *Repo) ListStuckJobs(gracePeriod time.Duration) ([]Job, error) {
	// stuck = running AND lease_expires_at < now - grace_period
	query := `
	SELECT id, video_id, job_type, state, attempt, max_attempts, lease_owner, lease_expires_at, 
	       input_path, output_path, error, created_at, updated_at, expanded, parent_job_id
	FROM jobs
	WHERE state = 'running' AND lease_expires_at < DATETIME('now', '-' || ? || ' seconds');
	`

	rows, err := r.Db.Query(query, gracePeriod.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var job []Job
	for rows.Next() {
		var j Job
		err := rows.Scan(&j.ID, &j.VideoID, &j.JobType, &j.State, &j.Attempt, &j.MaxAttempts,
			&j.LeaseOwner, &j.LeaseExpiresAt, &j.InputPath, &j.OutputPath, &j.Error,
			&j.CreatedAt, &j.UpdatedAt, &j.Expanded, &j.ParentJobID)
		if err != nil {
			return nil, err
		}
		job = append(job, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return job, nil
}

type InvariantViolation struct {
	JobID     string
	Violation string
}

func (r *Repo) CheckInvariants() ([]InvariantViolation, error) {
	query := `SELECT id, violation FROM job_invariants_violations`
	rows, err := r.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []InvariantViolation
	for rows.Next() {
		var v InvariantViolation
		if err := rows.Scan(&v.JobID, &v.Violation); err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}
	return violations, rows.Err()
}

// AcquireResourceToken attempts to acquire a token for the given resource.
// Returns true if token was acquired, false if no capacity available.
// This is atomic - the increment and capacity check happen in a single UPDATE.
func (r *Repo) AcquireResourceToken(resourceName string) (bool, error) {
	query := `
	UPDATE resource_limits
	SET current_inflight = current_inflight + 1,
	    updated_at = DATETIME('now')
	WHERE resource_name = ?
	  AND current_inflight < max_concurrency;
	`
	res, err := r.Db.Exec(query, resourceName)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// If rows affected = 1, we successfully acquired the token
	// If rows affected = 0, capacity was full
	acquired := rows == 1

	if acquired {
		slog.Info("[repo] acquired resource token", "resource", resourceName)
	} else {
		slog.Warn("[repo] no capacity for resource token", "resource", resourceName)
	}

	return acquired, nil
}

// ReleaseResourceToken releases a token for the given resource.
// This is idempotent - safe to call multiple times.
// Uses MAX to prevent negative counts from bugs (SQLite compatible).
func (r *Repo) ReleaseResourceToken(resourceName string) error {
	query := `
	UPDATE resource_limits
	SET current_inflight = MAX(current_inflight - 1, 0),
	    updated_at = DATETIME('now')
	WHERE resource_name = ?;
	`
	_, err := r.Db.Exec(query, resourceName)
	if err != nil {
		return err
	}

	slog.Info("[repo] released resource token", "resource", resourceName)
	return nil
}

// ReleaseTokensForExpiredLeases releases tokens for jobs with expired leases.
// Called by coordinator during lease reclamation to prevent token leaks.
func (r *Repo) ReleaseTokensForExpiredLeases() error {
	// First, get the count of expired leases per job type
	query := `
	UPDATE resource_limits
	SET current_inflight = MAX(
	    current_inflight - (
	        SELECT COUNT(*)
	        FROM jobs
	        WHERE state = 'running'
	          AND lease_expires_at < CURRENT_TIMESTAMP
	          AND job_type = resource_limits.resource_name
	    ),
	    0
	),
	updated_at = DATETIME('now');
	`
	res, err := r.Db.Exec(query)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows > 0 {
		slog.Info("[repo] released tokens for expired leases", "resources_updated", rows)
	}

	return nil
}
