package sidekiq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// DeadSet represents Sidekiq's dead job set (morgue).
// Jobs here have exhausted their retries and will not be automatically retried.
type DeadSet struct {
	*JobSet
}

// NewDeadSet creates a new DeadSet.
func NewDeadSet(client *Client) *DeadSet {
	return &DeadSet{
		JobSet: newJobSet(client, keyDead),
	}
}

// KillOptions configures how a job is killed.
type KillOptions struct {
	// NotifyFailure sends a death notification (Sidekiq Pro feature).
	NotifyFailure bool
	// ErrorMessage overrides the error message.
	ErrorMessage string
	// ErrorClass overrides the error class.
	ErrorClass string
}

// Kill adds a job to the dead set.
func (ds *DeadSet) Kill(ctx context.Context, job map[string]interface{}, opts *KillOptions) error {
	now := time.Now()

	// Set error info
	if opts != nil {
		if opts.ErrorMessage != "" {
			job["error_message"] = opts.ErrorMessage
		}
		if opts.ErrorClass != "" {
			job["error_class"] = opts.ErrorClass
		}
	}

	// Set failed_at if not present
	if _, ok := job["failed_at"]; !ok {
		job["failed_at"] = float64(now.Unix())
	}

	data, err := json.Marshal(job)
	if err != nil {
		return &JobParseError{Err: err}
	}

	err = ds.client.rdb.ZAdd(ctx, ds.client.key(ds.name), redis.Z{
		Score:  float64(now.Unix()),
		Member: string(data),
	}).Err()
	if err != nil {
		return wrapRedisErr("kill job", err)
	}

	// Trim the dead set if it's getting too large
	return ds.Trim(ctx, DefaultDeadSetMaxJobs, DefaultDeadSetMaxAge)
}

// Default limits for the dead set.
const (
	// DefaultDeadSetMaxJobs is the default maximum number of jobs in the dead set.
	DefaultDeadSetMaxJobs = 10000
	// DefaultDeadSetMaxAge is the default maximum age of jobs in the dead set (6 months).
	DefaultDeadSetMaxAge = 180 * 24 * time.Hour
)

// Trim removes old entries from the dead set.
// It removes entries older than maxAge and keeps only maxJobs entries.
func (ds *DeadSet) Trim(ctx context.Context, maxJobs int64, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge).Unix()

	pipe := ds.client.rdb.Pipeline()

	// Remove entries older than maxAge
	pipe.ZRemRangeByScore(ctx, ds.client.key(ds.name), "-inf", formatScore(float64(cutoff)))

	// Keep only maxJobs entries (remove oldest if over limit)
	pipe.ZRemRangeByRank(ctx, ds.client.key(ds.name), 0, -maxJobs-1)

	_, err := pipe.Exec(ctx)
	return wrapRedisErr("trim dead set", err)
}

// RetryAll moves all jobs in the dead set back to their queues.
// Returns the number of jobs retried.
func (ds *DeadSet) RetryAll(ctx context.Context) (int64, error) {
	var count int64

	for {
		entries, err := ds.client.rdb.ZRangeWithScores(ctx, ds.client.key(ds.name), 0, 99).Result()
		if err != nil {
			return count, wrapRedisErr("retry all", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(ds.JobSet, value, z.Score)
			if err != nil {
				continue
			}

			if err := entry.AddToQueue(ctx); err == nil {
				count++
			}
		}
	}

	return count, nil
}
