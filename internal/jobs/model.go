package jobs

import (
	"time"

	"github.com/google/uuid"
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
	LeaseOwner     *string
	LeaseExpiresAt *time.Time
	InputPath      string
	OutputPath     *string
	Error          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Expanded       bool
	ParentJobID    *string
}

func New(jobType string, inputPath string, parentID string) Job {
	return Job{
		ID:          uuid.NewString(), // ← critical
		JobType:     JobType(jobType),
		State:       StatePending,
		Attempt:     0,
		MaxAttempts: 3,
		InputPath:   inputPath,
		Expanded:    false,
		ParentJobID: &parentID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}
