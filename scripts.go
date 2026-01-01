package sidekiq

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Lua scripts for atomic operations.
// These scripts ensure atomicity of multi-step operations.

// requeueScript atomically moves a job from a sorted set to a queue.
var requeueScript = redis.NewScript(`
local from_key = KEYS[1]
local to_key = KEYS[2]
local queues_key = KEYS[3]
local queue_name = ARGV[1]
local job_value = ARGV[2]

local removed = redis.call('ZREM', from_key, job_value)
if removed == 0 then
    return 0
end

redis.call('SADD', queues_key, queue_name)
redis.call('LPUSH', to_key, job_value)
return 1
`)

// killScript atomically moves a job from a sorted set to the dead set.
var killScript = redis.NewScript(`
local from_key = KEYS[1]
local dead_key = KEYS[2]
local job_value = ARGV[1]
local score = ARGV[2]
local max_jobs = tonumber(ARGV[3])
local max_age = tonumber(ARGV[4])
local now = tonumber(ARGV[5])

local removed = redis.call('ZREM', from_key, job_value)
if removed == 0 then
    return 0
end

redis.call('ZADD', dead_key, score, job_value)

-- Trim by age
local cutoff = now - max_age
redis.call('ZREMRANGEBYSCORE', dead_key, '-inf', cutoff)

-- Trim by count
local count = redis.call('ZCARD', dead_key)
if count > max_jobs then
    redis.call('ZREMRANGEBYRANK', dead_key, 0, count - max_jobs - 1)
end

return 1
`)

// enqueueJobsScript atomically moves due jobs from a sorted set to their queues.
var enqueueJobsScript = redis.NewScript(`
local sorted_set = KEYS[1]
local queues_key = KEYS[2]
local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])

local jobs = redis.call('ZRANGEBYSCORE', sorted_set, '-inf', now, 'LIMIT', 0, limit)
local count = 0

for _, job in ipairs(jobs) do
    local data = cjson.decode(job)
    local queue = data.queue or 'default'

    redis.call('ZREM', sorted_set, job)
    redis.call('SADD', queues_key, queue)
    redis.call('LPUSH', 'queue:' .. queue, job)
    count = count + 1
end

return count
`)

// RequeueJob atomically moves a job from a sorted set to its queue.
func (c *Client) RequeueJob(ctx context.Context, setName, jobValue string) error {
	job, err := newJobRecord(jobValue, "")
	if err != nil {
		return err
	}

	queue := job.Queue()
	if queue == "" {
		queue = "default"
	}

	keys := []string{
		c.key(setName),
		c.queueKey(queue),
		c.key(keyQueues),
	}
	args := []interface{}{queue, jobValue}

	result, err := requeueScript.Run(ctx, c.rdb, keys, args...).Int()
	if err != nil {
		return wrapRedisErr("requeue job", err)
	}
	if result == 0 {
		return ErrJobNotFound
	}
	return nil
}

// KillJob atomically moves a job to the dead set with trimming.
func (c *Client) KillJob(ctx context.Context, setName, jobValue string, score float64) error {
	keys := []string{
		c.key(setName),
		c.key(keyDead),
	}
	args := []interface{}{
		jobValue,
		score,
		DefaultDeadSetMaxJobs,
		int64(DefaultDeadSetMaxAge.Seconds()),
		ctx.Value("now"), // for testing, defaults to nil
	}

	result, err := killScript.Run(ctx, c.rdb, keys, args...).Int()
	if err != nil {
		return wrapRedisErr("kill job", err)
	}
	if result == 0 {
		return ErrJobNotFound
	}
	return nil
}
