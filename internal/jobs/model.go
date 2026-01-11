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

// Namespace UUID for generating deterministic job IDs
var jobNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func NewDeterministicChildJob(jobType string, inputPath string, parentID string) Job {
	// Generate deterministic ID from parent + job type for idempotency
	// This ensures the same child job always gets the same ID,
	// preventing duplicates if coordinator crashes before marking parent as expanded
	deterministicID := uuid.NewSHA1(jobNamespace, []byte(parentID+jobType)).String()

	return Job{
		ID:          deterministicID,
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
