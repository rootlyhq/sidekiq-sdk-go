package sidekiq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SortedEntry represents a job in a sorted set (schedule, retry, dead).
type SortedEntry struct {
	*JobRecord
	parent *JobSet
	score  float64
}

// newSortedEntry creates a new SortedEntry from raw data.
func newSortedEntry(parent *JobSet, value string, score float64) (*SortedEntry, error) {
	job, err := newJobRecord(value, "")
	if err != nil {
		return nil, err
	}
	return &SortedEntry{
		JobRecord: job,
		parent:    parent,
		score:     score,
	}, nil
}

// Score returns the entry's score (Unix timestamp).
func (e *SortedEntry) Score() float64 {
	return e.score
}

// At returns the scheduled time as a time.Time.
func (e *SortedEntry) At() time.Time {
	return time.Unix(int64(e.score), 0)
}

// Delete removes this entry from its parent set.
func (e *SortedEntry) Delete(ctx context.Context) error {
	removed, err := e.parent.DeleteByValue(ctx, e.Value())
	if err != nil {
		return err
	}
	if !removed {
		return ErrJobNotFound
	}
	return nil
}

// Reschedule moves this entry to a new time.
func (e *SortedEntry) Reschedule(ctx context.Context, at time.Time) error {
	// Remove from current position
	_, err := e.parent.client.rdb.ZRem(ctx, e.parent.client.key(e.parent.name), e.Value()).Result()
	if err != nil {
		return wrapRedisErr("reschedule", err)
	}

	// Add at new time
	err = e.parent.client.rdb.ZAdd(ctx, e.parent.client.key(e.parent.name), redis.Z{
		Score:  float64(at.Unix()),
		Member: e.Value(),
	}).Err()
	if err != nil {
		return wrapRedisErr("reschedule", err)
	}

	e.score = float64(at.Unix())
	return nil
}

// AddToQueue moves this entry to its target queue for immediate processing.
func (e *SortedEntry) AddToQueue(ctx context.Context) error {
	queueName := e.Queue()
	if queueName == "" {
		queueName = "default"
	}

	// Update the job data with current enqueued_at
	item := e.Item()
	item["enqueued_at"] = float64(time.Now().Unix())

	// Remove error fields if present
	delete(item, "error_message")
	delete(item, "error_class")
	delete(item, "error_backtrace")
	delete(item, "compressed_backtrace")
	delete(item, "failed_at")
	delete(item, "retry_count")
	delete(item, "retried_at")

	data, err := json.Marshal(item)
	if err != nil {
		return &JobParseError{Err: err}
	}

	pipe := e.parent.client.rdb.Pipeline()

	// Remove from sorted set
	pipe.ZRem(ctx, e.parent.client.key(e.parent.name), e.Value())

	// Add to queue
	pipe.SAdd(ctx, e.parent.client.key(keyQueues), queueName)
	pipe.LPush(ctx, e.parent.client.queueKey(queueName), string(data))

	_, err = pipe.Exec(ctx)
	return wrapRedisErr("add to queue", err)
}

// Retry moves this entry back to its queue immediately.
// This is an alias for AddToQueue.
func (e *SortedEntry) Retry(ctx context.Context) error {
	return e.AddToQueue(ctx)
}

// Kill moves this entry to the dead set.
func (e *SortedEntry) Kill(ctx context.Context) error {
	item := e.Item()

	// Update with current failed_at if not present
	if _, ok := item["failed_at"]; !ok {
		item["failed_at"] = float64(time.Now().Unix())
	}

	data, err := json.Marshal(item)
	if err != nil {
		return &JobParseError{Err: err}
	}

	pipe := e.parent.client.rdb.Pipeline()

	// Remove from current set
	pipe.ZRem(ctx, e.parent.client.key(e.parent.name), e.Value())

	// Add to dead set
	pipe.ZAdd(ctx, e.parent.client.key(keyDead), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(data),
	})

	_, err = pipe.Exec(ctx)
	return wrapRedisErr("kill job", err)
}

// HasError returns true if this entry has error information.
func (e *SortedEntry) HasError() bool {
	return e.ErrorMessage() != "" || e.ErrorClass() != ""
}

// formatScore formats a score for Redis range queries.
func init() {
	// Override the formatScore function with proper implementation
}

// formatScoreValue formats a float64 score for Redis commands.
func formatScoreValue(score float64) string {
	return fmt.Sprintf("%f", score)
}
