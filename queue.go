package sidekiq

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Queue represents a Sidekiq queue.
type Queue struct {
	client *Client
	name   string
}

// NewQueue creates a new Queue instance.
func NewQueue(client *Client, name string) *Queue {
	return &Queue{
		client: client,
		name:   name,
	}
}

// AllQueues returns all known queues.
func AllQueues(ctx context.Context, client *Client) ([]*Queue, error) {
	names, err := client.rdb.SMembers(ctx, client.key(keyQueues)).Result()
	if err != nil {
		return nil, wrapRedisErr("list queues", err)
	}

	queues := make([]*Queue, len(names))
	for i, name := range names {
		queues[i] = NewQueue(client, name)
	}
	return queues, nil
}

// Name returns the queue name.
func (q *Queue) Name() string {
	return q.name
}

// Size returns the number of jobs in the queue.
func (q *Queue) Size(ctx context.Context) (int64, error) {
	size, err := q.client.rdb.LLen(ctx, q.client.queueKey(q.name)).Result()
	if err != nil {
		return 0, wrapRedisErr("get queue size", err)
	}
	return size, nil
}

// Paused returns whether the queue is paused.
func (q *Queue) Paused(ctx context.Context) (bool, error) {
	exists, err := q.client.rdb.Exists(ctx, q.client.pausedKey(q.name)).Result()
	if err != nil {
		return false, wrapRedisErr("check paused", err)
	}
	return exists > 0, nil
}

// Pause pauses the queue.
func (q *Queue) Pause(ctx context.Context) error {
	err := q.client.rdb.Set(ctx, q.client.pausedKey(q.name), "true", 0).Err()
	return wrapRedisErr("pause queue", err)
}

// Unpause unpauses the queue.
func (q *Queue) Unpause(ctx context.Context) error {
	err := q.client.rdb.Del(ctx, q.client.pausedKey(q.name)).Err()
	return wrapRedisErr("unpause queue", err)
}

// Latency returns the latency in seconds of the oldest job in the queue.
// Returns 0 if the queue is empty.
func (q *Queue) Latency(ctx context.Context) (float64, error) {
	result, err := q.client.rdb.LIndex(ctx, q.client.queueKey(q.name), -1).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, wrapRedisErr("get latency", err)
	}

	job, err := newJobRecord(result, q.name)
	if err != nil {
		return 0, err
	}
	return job.Latency(), nil
}

// Each iterates over all jobs in the queue.
// The callback function receives each job and should return true to continue
// or false to stop iteration.
func (q *Queue) Each(ctx context.Context, fn func(JobRecord) bool) error {
	const pageSize = 100
	offset := int64(0)

	for {
		jobs, err := q.client.rdb.LRange(ctx, q.client.queueKey(q.name), offset, offset+pageSize-1).Result()
		if err != nil {
			return wrapRedisErr("iterate queue", err)
		}

		if len(jobs) == 0 {
			break
		}

		for _, jobData := range jobs {
			job, err := newJobRecord(jobData, q.name)
			if err != nil {
				continue // Skip malformed jobs
			}
			if !fn(*job) {
				return nil
			}
		}

		if int64(len(jobs)) < pageSize {
			break
		}
		offset += pageSize
	}

	return nil
}

// FindJob finds a job by JID in the queue.
// Returns nil if not found.
func (q *Queue) FindJob(ctx context.Context, jid string) (*JobRecord, error) {
	var found *JobRecord
	err := q.Each(ctx, func(job JobRecord) bool {
		if job.JID() == jid {
			found = &job
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// Clear removes all jobs from the queue.
// Returns the number of jobs deleted.
func (q *Queue) Clear(ctx context.Context) (int64, error) {
	key := q.client.queueKey(q.name)

	// Get the current size before clearing
	size, err := q.client.rdb.LLen(ctx, key).Result()
	if err != nil {
		return 0, wrapRedisErr("clear queue", err)
	}

	// Delete the queue
	err = q.client.rdb.Del(ctx, key).Err()
	if err != nil {
		return 0, wrapRedisErr("clear queue", err)
	}

	return size, nil
}

// Delete removes a specific job from the queue.
// Returns an error if the job is not found.
func (q *Queue) Delete(ctx context.Context, job *JobRecord) error {
	removed, err := q.client.rdb.LRem(ctx, q.client.queueKey(q.name), 1, job.Value()).Result()
	if err != nil {
		return wrapRedisErr("delete job", err)
	}
	if removed == 0 {
		return ErrJobNotFound
	}
	return nil
}

// DeleteJob removes a job by JID from the queue.
// Returns an error if the job is not found.
func (q *Queue) DeleteJob(ctx context.Context, jid string) error {
	job, err := q.FindJob(ctx, jid)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFound
	}
	return q.Delete(ctx, job)
}
