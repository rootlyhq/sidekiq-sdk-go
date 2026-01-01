package sidekiq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// EnqueueOptions configures job enqueueing.
type EnqueueOptions struct {
	// Queue specifies the queue name. Defaults to "default".
	Queue string
	// Retry specifies the retry behavior.
	// Can be true (default retries), false (no retries), or an int (max retries).
	Retry interface{}
	// Tags are optional tags for the job.
	Tags []string
	// Backtrace specifies how many backtrace lines to keep on failure.
	Backtrace int
}

// Enqueue adds a job to a queue for immediate processing.
// Returns the job ID.
func (c *Client) Enqueue(ctx context.Context, class string, args []interface{}, opts *EnqueueOptions) (string, error) {
	return c.enqueue(ctx, class, args, opts, time.Time{})
}

// EnqueueAt adds a job to be processed at a specific time.
// Returns the job ID.
func (c *Client) EnqueueAt(ctx context.Context, at time.Time, class string, args []interface{}, opts *EnqueueOptions) (string, error) {
	return c.enqueue(ctx, class, args, opts, at)
}

// EnqueueIn adds a job to be processed after a duration.
// Returns the job ID.
func (c *Client) EnqueueIn(ctx context.Context, in time.Duration, class string, args []interface{}, opts *EnqueueOptions) (string, error) {
	return c.enqueue(ctx, class, args, opts, time.Now().Add(in))
}

// enqueue is the internal implementation of job enqueueing.
func (c *Client) enqueue(ctx context.Context, class string, args []interface{}, opts *EnqueueOptions, at time.Time) (string, error) {
	if opts == nil {
		opts = &EnqueueOptions{}
	}

	queue := opts.Queue
	if queue == "" {
		queue = "default"
	}

	if args == nil {
		args = []interface{}{}
	}

	jid, err := generateJID()
	if err != nil {
		return "", err
	}

	now := float64(time.Now().Unix())
	job := map[string]interface{}{
		"class":       class,
		"args":        args,
		"jid":         jid,
		"created_at":  now,
		"enqueued_at": now,
		"queue":       queue,
	}

	// Set retry
	if opts.Retry != nil {
		job["retry"] = opts.Retry
	} else {
		job["retry"] = true
	}

	// Set optional fields
	if len(opts.Tags) > 0 {
		job["tags"] = opts.Tags
	}
	if opts.Backtrace > 0 {
		job["backtrace"] = opts.Backtrace
	}

	data, err := json.Marshal(job)
	if err != nil {
		return "", &JobParseError{Err: err}
	}

	// Scheduled job
	if !at.IsZero() && at.After(time.Now()) {
		err = c.rdb.ZAdd(ctx, c.key(keySchedule), redis.Z{
			Score:  float64(at.Unix()),
			Member: string(data),
		}).Err()
		if err != nil {
			return "", wrapRedisErr("enqueue scheduled job", err)
		}
		return jid, nil
	}

	// Immediate job
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, c.key(keyQueues), queue)
	pipe.LPush(ctx, c.queueKey(queue), string(data))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", wrapRedisErr("enqueue job", err)
	}

	return jid, nil
}

// generateJID generates a random 24-character hex job ID.
func generateJID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// EnqueueBulk adds multiple jobs to a queue efficiently.
// Returns the number of jobs enqueued.
func (c *Client) EnqueueBulk(ctx context.Context, class string, argsList [][]interface{}, opts *EnqueueOptions) (int64, error) {
	if len(argsList) == 0 {
		return 0, nil
	}

	if opts == nil {
		opts = &EnqueueOptions{}
	}

	queue := opts.Queue
	if queue == "" {
		queue = "default"
	}

	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, c.key(keyQueues), queue)

	now := float64(time.Now().Unix())
	for _, args := range argsList {
		if args == nil {
			args = []interface{}{}
		}

		jid, err := generateJID()
		if err != nil {
			continue
		}

		job := map[string]interface{}{
			"class":       class,
			"args":        args,
			"jid":         jid,
			"created_at":  now,
			"enqueued_at": now,
			"queue":       queue,
		}

		if opts.Retry != nil {
			job["retry"] = opts.Retry
		} else {
			job["retry"] = true
		}

		if len(opts.Tags) > 0 {
			job["tags"] = opts.Tags
		}
		if opts.Backtrace > 0 {
			job["backtrace"] = opts.Backtrace
		}

		data, err := json.Marshal(job)
		if err != nil {
			continue
		}

		pipe.LPush(ctx, c.queueKey(queue), string(data))
	}

	results, err := pipe.Exec(ctx)
	if err != nil {
		return 0, wrapRedisErr("enqueue bulk", err)
	}

	// Count successful pushes (skip the SADD result)
	var count int64
	for i := 1; i < len(results); i++ {
		if results[i].Err() == nil {
			count++
		}
	}

	return count, nil
}
