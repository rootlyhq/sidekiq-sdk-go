package sidekiq

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RetrySet represents Sidekiq's retry set.
// Jobs here have failed and are waiting to be retried.
type RetrySet struct {
	*JobSet
}

// NewRetrySet creates a new RetrySet.
func NewRetrySet(client *Client) *RetrySet {
	return &RetrySet{
		JobSet: newJobSet(client, keyRetry),
	}
}

// RetryAll moves all jobs in the retry set back to their queues.
// Returns the number of jobs retried.
func (rs *RetrySet) RetryAll(ctx context.Context) (int64, error) {
	var count int64

	for {
		// Get a batch of entries
		entries, err := rs.client.rdb.ZRangeWithScores(ctx, rs.client.key(rs.name), 0, 99).Result()
		if err != nil {
			return count, wrapRedisErr("retry all", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(rs.JobSet, value, z.Score)
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

// KillAll moves all jobs in the retry set to the dead set.
// Returns the number of jobs killed.
func (rs *RetrySet) KillAll(ctx context.Context) (int64, error) {
	var count int64

	for {
		entries, err := rs.client.rdb.ZRangeWithScores(ctx, rs.client.key(rs.name), 0, 99).Result()
		if err != nil {
			return count, wrapRedisErr("kill all", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(rs.JobSet, value, z.Score)
			if err != nil {
				continue
			}

			if err := entry.Kill(ctx); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// DeleteOlderThan removes all entries older than the given time.
// Returns the number of entries deleted.
func (rs *RetrySet) DeleteOlderThan(ctx context.Context, t time.Time) (int64, error) {
	deleted, err := rs.client.rdb.ZRemRangeByScore(ctx, rs.client.key(rs.name),
		"-inf",
		formatScore(float64(t.Unix())),
	).Result()
	if err != nil {
		return 0, wrapRedisErr("delete older than", err)
	}
	return deleted, nil
}

// DeleteByClass removes all entries with the given worker class.
// Returns the number of entries deleted.
func (rs *RetrySet) DeleteByClass(ctx context.Context, class string) (int64, error) {
	var count int64

	err := rs.Each(ctx, func(entry *SortedEntry) bool {
		if entry.DisplayClass() == class {
			if err := entry.Delete(ctx); err == nil {
				count++
			}
		}
		return true
	})

	return count, err
}

// EnqueueJobs moves jobs with scores up to the given time to their queues.
// This is used by the scheduler to process jobs that are due.
// Returns the number of jobs enqueued.
func (rs *RetrySet) EnqueueJobs(ctx context.Context, now time.Time) (int64, error) {
	var count int64
	score := float64(now.Unix())

	for {
		entries, err := rs.client.rdb.ZRangeByScoreWithScores(ctx, rs.client.key(rs.name), &redis.ZRangeBy{
			Min:   "-inf",
			Max:   formatScore(score),
			Count: 100,
		}).Result()
		if err != nil {
			return count, wrapRedisErr("enqueue jobs", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(rs.JobSet, value, z.Score)
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
