package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/MuktadirHassan/video-processor/internal/jobs"
)

// One architectural smell to address soon (not now)
// Your worker currently:
// Knows about SQL
// Knows about jobs.Job
// Eventually, you'll want:
// Worker logic to be testable without a DB
// Job handlers to be pure functions
// That comes later.
// For now, this is acceptable for a PoC

type Config struct {
	DB           *sql.DB
	WorkerID     string
	LeaseSeconds string
}

type worker struct {
	repo         jobs.Repo
	workerID     string
	leaseSeconds string
}

func New(cfg Config) *worker {
	return &worker{
		repo:         jobs.Repo{Db: cfg.DB},
		workerID:     cfg.WorkerID,
		leaseSeconds: cfg.LeaseSeconds,
	}
}

func (w *worker) Run(ctx context.Context) {
	// ask db, is there a job i can lease?
	// if no, sleep
	// if yes, execute exactly one job
	// mark success or failure
	// loop forever

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.repo.AcquireLease(w.workerID, w.leaseSeconds)
		if err != nil {
			slog.Error("[worker] failed to acquire lease", "error", err)
			continue
		}

		if job == nil {
			slog.Info("[worker] no pending jobs found, sleeping...")
			time.Sleep(2 * time.Second)
			continue
		}

		slog.Info("[worker] acquired lease for job", "job_id", job.ID)

		// Execute job with token management
		// Extracted to separate function to properly scope defer
		w.executeJobWithToken(ctx, job)
	}
}

// executeJobWithToken handles token acquisition, job execution, and token release
// The defer is scoped to this function, not the loop, preventing accumulation
func (w *worker) executeJobWithToken(ctx context.Context, job *jobs.Job) {
	// Try to acquire resource token for this job type
	// This provides backpressure - if no capacity, we fail fast
	acquired, err := w.repo.AcquireResourceToken(string(job.JobType))
	if err != nil {
		slog.Error("[worker] failed to acquire resource token", "job_id", job.ID, "error", err)
		// Release the lease since we can't execute
		w.repo.FailJob(job.ID, w.workerID, err)
		return
	}

	if !acquired {
		// No capacity available - this is intentional backpressure
		slog.Warn("[worker] no resource capacity, releasing lease", "job_id", job.ID, "job_type", job.JobType)
		// Put job back to pending
		w.repo.FailJob(job.ID, w.workerID, nil)
		time.Sleep(2 * time.Second)
		return
	}

	// CRITICAL: Use defer to guarantee token release even on panic
	// This prevents token leaks
	// Now properly scoped to this function, not the loop
	defer func() {
		releaseErr := w.repo.ReleaseResourceToken(string(job.JobType))
		if releaseErr != nil {
			slog.Error("[worker] failed to release resource token", "job_id", job.ID, "error", releaseErr)
		}
	}()

	err = w.handleJob(ctx, job)

	if err != nil {
		slog.Error("[worker] job failed", "job_id", job.ID, "error", err)
		w.repo.FailJob(job.ID, w.workerID, err)
	} else {
		slog.Info("[worker] job succeeded", "job_id", job.ID)
		w.repo.SucceedJob(job.ID, w.workerID)
	}
}
