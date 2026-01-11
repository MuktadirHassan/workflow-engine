package jobs

import (
	"time"
)

type JobState string

const (
	StatePending   JobState = "pending"
	StateRunning   JobState = "running"
	StateSucceeded JobState = "succeeded"
	StateDead      JobState = "dead"
)

type JobType string

const (
	JobTypeValidate JobType = "validate"
	JobTypeMetadata JobType = "metadata"
)

type Job struct {
	ID             string
	VideoID        string
	JobType        JobType
	State          JobState
	Attempt        int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	InputPath      string
	OutputPath     string
	Error          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func New(jobType JobType, outputPath string, jobID string) Job {
	return Job{
		ID:          jobID,
		JobType:     jobType,
		State:       StatePending,
		Attempt:     0,
		MaxAttempts: 3,
		InputPath:   outputPath,
	}
}
