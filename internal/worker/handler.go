package worker

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/MuktadirHassan/video-processor/internal/jobs"
)

func (w *worker) handleJob(ctx context.Context, job *jobs.Job) error {
	slog.Info("[worker] executing job", "job_id", job.ID)
	// sleep for 20-25 seconds
	n := rand.Int() % 6 // 0-5
	time.Sleep(time.Duration(5+n) * time.Second)

	slog.Info("[worker] completed job", "job_id", job.ID)

	return nil
}
