package coordinator

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/MuktadirHassan/video-processor/internal/jobs"
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
		slog.Info("[coordinator] checking for expired leases to reclaim")
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.repo.ReclaimExpiredLeases()
		if err != nil {
			slog.Error("[coordinator] failed to reclaim expired leases", "error", err)
		}

		slog.Info("[coordinator] expanding succeeded unexpanded jobs")
		c.expandSucceededJobs()
		slog.Info("[coordinator] finished expanding succeeded unexpanded jobs")

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
			c.repo.InsertJobIfNotExists(nextJob)
		}

		slog.Info("[coordinator] marking job as expanded", "job_id", job.ID)
		c.repo.MarkJobAsExpanded(job.ID)
	}
}

func (c *coordinator) nextJobsFor(job jobs.Job) []jobs.Job {
	switch job.JobType {
	case "validate":
		return []jobs.Job{
			jobs.New("metadata", job.OutputPath, job.ID),
		}
	default:
		return nil
	}
}
