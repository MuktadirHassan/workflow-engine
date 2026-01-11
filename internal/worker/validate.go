package worker

import (
	"fmt"
	"os"

	"github.com/MuktadirHassan/video-processor/internal/jobs"
)

// ValidateJob performs cheap validation checks before any expensive work
// Rule: Every validation that can fail cheaply must fail before ffmpeg spins up
// This is cost-control, not correctness
func ValidateJob(job *jobs.Job) error {
	// Validate input file exists (cheap syscall)
	if err := validateInputFile(job.InputPath); err != nil {
		return fmt.Errorf("input validation failed: %w", err)
	}

	// Validate job hasn't exceeded max attempts (free memory check)
	if err := validateAttempts(job); err != nil {
		return fmt.Errorf("attempt validation failed: %w", err)
	}

	// Validate output path constraints (placeholder for future logic)
	if err := validateOutputPath(job.OutputPath); err != nil {
		return fmt.Errorf("output validation failed: %w", err)
	}

	return nil
}

// validateInputFile checks if the input file exists
// Cost: ~1 syscall, negligible compared to ffmpeg
func validateInputFile(inputPath string) error {
	if inputPath == "" {
		return fmt.Errorf("input path is empty")
	}

	// os.Stat is cheap - just a filesystem metadata lookup
	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inputPath)
		}
		return fmt.Errorf("failed to stat input file %s: %w", inputPath, err)
	}

	// Ensure it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("input path is a directory, not a file: %s", inputPath)
	}

	return nil
}

// validateAttempts checks if job has exceeded max retry attempts
// Cost: free (just memory comparison)
func validateAttempts(job *jobs.Job) error {
	if job.Attempt >= job.MaxAttempts {
		return fmt.Errorf("job has exceeded max attempts (%d >= %d)", job.Attempt, job.MaxAttempts)
	}
	return nil
}

// validateOutputPath validates output path constraints
// Cost: free (currently a no-op, placeholder for future logic)
// Examples of future checks:
// - Output path not already finalized
// - Output directory is writable
// - Sufficient disk space available
func validateOutputPath(outputPath *string) error {
	// No validation yet - placeholder for future expansion
	// When you add output validation, it should be cheap (e.g., checking a DB flag)
	// Don't do expensive filesystem operations here
	return nil
}
