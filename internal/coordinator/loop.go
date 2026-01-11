package coordinator

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/MuktadirHassan/workflow-engine/internal/jobs"
)

type Config struct {
	DB *sql.DB
}

type coordinator struct {
	repo jobs.Repo
}

func New(cfg Config) *coordinator {
	return &coordinator{
		repo: jobs.Repo{Db: cfg.DB},
	}
}

// This is idempotency for orchestration.
// Succeeded + expanded = false
// → coordinator must act

// Succeeded + expanded = true
// → coordinator must ignore forever

func (c *coordinator) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// check invariants
		violations, err := c.repo.CheckInvariants()
		if err != nil {
			slog.Error("[coordinator] failed to check invariants", "error", err)
		}
		for _, v := range violations {
			slog.Error("[coordinator] INVARIANT VIOLATION", "job_id", v.JobID, "violation", v.Violation)
		}

		// observe stuck jobs
		stuckJobs, err := c.repo.ListStuckJobs(10 * time.Second)
		if err != nil {
			slog.Error("[coordinator] failed to list stuck jobs", "error", err)
			continue
		}

		slog.Info("[coordinator] found stuck jobs", "count", len(stuckJobs))

		// Release tokens for expired leases BEFORE reclaiming the leases
		// This prevents token leaks when workers crash
		err = c.repo.ReleaseTokensForExpiredLeases()
		if err != nil {
			slog.Error("[coordinator] failed to release tokens for expired leases", "error", err)
		}

		err = c.repo.ReclaimExpiredLeases()
		if err != nil {
			slog.Error("[coordinator] failed to reclaim expired leases", "error", err)
		}

		c.expandSucceededJobs()

		slog.Info("[coordinator] sleeping for 10 seconds before next lease check")
		time.Sleep(10 * time.Second)
	}
}

func (c *coordinator) expandSucceededJobs() {
	jobs, err := c.repo.FindSucceededUnexpandedJobs()
	if err != nil {
		slog.Error("[coordinator] failed to fetch succeeded unexpanded jobs", "error", err)
		return
	}

	for _, job := range jobs {
		nextJobs := c.nextJobsFor(job)

		for _, nextJob := range nextJobs {
			slog.Info("[coordinator] inserting next job", "parent_job_id", job.ID, "job_type", nextJob.JobType)
			err := c.repo.InsertJobIfNotExists(nextJob)
			if err != nil {
				slog.Error("[coordinator] failed to insert next job", "error", err)
				return
			}
		}
		// simulate killing the job after insert
		// panic("simulate crash before marking job as expanded")
		slog.Info("[coordinator] marking job as expanded", "job_id", job.ID)
		err := c.repo.MarkJobAsExpanded(job.ID)
		if err != nil {
			slog.Error("[coordinator] failed to mark job as expanded", "error", err)
		}
	}
}

func (c *coordinator) nextJobsFor(job jobs.Job) []jobs.Job {
	switch job.JobType {
	case "validate":
		outputPath := ""
		if job.OutputPath != nil {
			outputPath = *job.OutputPath
		}
		return []jobs.Job{
			jobs.NewDeterministicChildJob("metadata", outputPath, job.ID),
		}
	case "metadata":
		return []jobs.Job{
			jobs.NewDeterministicChildJob("thumbnail", job.InputPath, job.ID),
		}
	case "thumbnail":
		return []jobs.Job{
			jobs.NewDeterministicChildJob("encode", job.InputPath, job.ID),
		}
	default:
		return nil
	}
}
