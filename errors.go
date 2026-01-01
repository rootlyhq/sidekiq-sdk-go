package sidekiq

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions.
var (
	// ErrJobNotFound is returned when a job with the specified JID cannot be found.
	ErrJobNotFound = errors.New("sidekiq: job not found")

	// ErrQueueNotFound is returned when a queue with the specified name cannot be found.
	ErrQueueNotFound = errors.New("sidekiq: queue not found")

	// ErrProcessNotFound is returned when a process with the specified identity cannot be found.
	ErrProcessNotFound = errors.New("sidekiq: process not found")

	// ErrInvalidJob is returned when job data is malformed or missing required fields.
	ErrInvalidJob = errors.New("sidekiq: invalid job data")
)

// RedisError wraps Redis errors with operation context.
type RedisError struct {
	Op  string // Operation that failed (e.g., "get stats", "fetch queue")
	Err error  // Underlying error
}

func (e *RedisError) Error() string {
	return fmt.Sprintf("sidekiq: %s: %v", e.Op, e.Err)
}

func (e *RedisError) Unwrap() error {
	return e.Err
}

// JobParseError is returned when job JSON data cannot be parsed.
type JobParseError struct {
	JID string // Job ID if available
	Err error  // Underlying parse error
}

func (e *JobParseError) Error() string {
	if e.JID != "" {
		return fmt.Sprintf("sidekiq: failed to parse job %s: %v", e.JID, e.Err)
	}
	return fmt.Sprintf("sidekiq: failed to parse job: %v", e.Err)
}

func (e *JobParseError) Unwrap() error {
	return e.Err
}

// wrapRedisErr wraps a Redis error with operation context if it's not nil.
func wrapRedisErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return &RedisError{Op: op, Err: err}
}
