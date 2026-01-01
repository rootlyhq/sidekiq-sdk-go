package sidekiq

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Stats provides a snapshot of Sidekiq statistics.
// Create with NewStats which fetches all data atomically.
type Stats struct {
	client              *Client
	processed           int64
	failed              int64
	scheduledSize       int64
	retrySize           int64
	deadSize            int64
	enqueued            int64
	processesSize       int64
	workersSize         int64
	defaultQueueLatency float64
	queues              map[string]int64
}

// NewStats fetches current Sidekiq statistics.
// All stats are fetched in a single Redis pipeline for consistency.
func NewStats(ctx context.Context, client *Client) (*Stats, error) {
	pipe := client.rdb.Pipeline()

	processedCmd := pipe.Get(ctx, client.statKey("processed"))
	failedCmd := pipe.Get(ctx, client.statKey("failed"))
	scheduledCmd := pipe.ZCard(ctx, client.key(keySchedule))
	retryCmd := pipe.ZCard(ctx, client.key(keyRetry))
	deadCmd := pipe.ZCard(ctx, client.key(keyDead))
	queuesCmd := pipe.SMembers(ctx, client.key(keyQueues))
	processesCmd := pipe.SMembers(ctx, client.key(keyProcesses))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, wrapRedisErr("fetch stats", err)
	}

	s := &Stats{
		client:        client,
		scheduledSize: scheduledCmd.Val(),
		retrySize:     retryCmd.Val(),
		deadSize:      deadCmd.Val(),
		queues:        make(map[string]int64),
	}

	// Parse processed count
	if v, err := processedCmd.Result(); err == nil {
		s.processed, _ = strconv.ParseInt(v, 10, 64)
	}

	// Parse failed count
	if v, err := failedCmd.Result(); err == nil {
		s.failed, _ = strconv.ParseInt(v, 10, 64)
	}

	// Get queue sizes
	queueNames := queuesCmd.Val()
	if len(queueNames) > 0 {
		queuePipe := client.rdb.Pipeline()
		queueCmds := make([]*redis.IntCmd, len(queueNames))
		for i, name := range queueNames {
			queueCmds[i] = queuePipe.LLen(ctx, client.queueKey(name))
		}
		_, _ = queuePipe.Exec(ctx)

		for i, name := range queueNames {
			size := queueCmds[i].Val()
			s.queues[name] = size
			s.enqueued += size
		}
	}

	// Calculate default queue latency
	s.defaultQueueLatency = s.calculateLatency(ctx, "default")

	// Get processes count
	s.processesSize = int64(len(processesCmd.Val()))

	// Get workers count (sum of busy workers across all processes)
	processes := processesCmd.Val()
	if len(processes) > 0 {
		workPipe := client.rdb.Pipeline()
		workCmds := make([]*redis.IntCmd, len(processes))
		for i, identity := range processes {
			workCmds[i] = workPipe.HLen(ctx, client.workKey(identity))
		}
		_, _ = workPipe.Exec(ctx)

		for _, cmd := range workCmds {
			s.workersSize += cmd.Val()
		}
	}

	return s, nil
}

// calculateLatency calculates the latency for a queue by checking the oldest job.
func (s *Stats) calculateLatency(ctx context.Context, queueName string) float64 {
	result, err := s.client.rdb.LIndex(ctx, s.client.queueKey(queueName), -1).Result()
	if err != nil {
		return 0
	}

	job, err := newJobRecord(result, queueName)
	if err != nil {
		return 0
	}

	return job.Latency()
}

// Processed returns the total number of processed jobs.
func (s *Stats) Processed() int64 {
	return s.processed
}

// Failed returns the total number of failed jobs.
func (s *Stats) Failed() int64 {
	return s.failed
}

// ScheduledSize returns the number of jobs in the scheduled set.
func (s *Stats) ScheduledSize() int64 {
	return s.scheduledSize
}

// RetrySize returns the number of jobs in the retry set.
func (s *Stats) RetrySize() int64 {
	return s.retrySize
}

// DeadSize returns the number of jobs in the dead set.
func (s *Stats) DeadSize() int64 {
	return s.deadSize
}

// Enqueued returns the total number of jobs across all queues.
func (s *Stats) Enqueued() int64 {
	return s.enqueued
}

// ProcessesSize returns the number of active Sidekiq processes.
func (s *Stats) ProcessesSize() int64 {
	return s.processesSize
}

// WorkersSize returns the total number of busy workers across all processes.
func (s *Stats) WorkersSize() int64 {
	return s.workersSize
}

// DefaultQueueLatency returns the latency in seconds for the default queue.
func (s *Stats) DefaultQueueLatency() float64 {
	return s.defaultQueueLatency
}

// Queues returns a map of queue names to their sizes.
func (s *Stats) Queues() map[string]int64 {
	result := make(map[string]int64, len(s.queues))
	for k, v := range s.queues {
		result[k] = v
	}
	return result
}

// Reset clears the specified stat counters.
// Valid stat names are "processed" and "failed".
func Reset(ctx context.Context, client *Client, stats ...string) error {
	if len(stats) == 0 {
		stats = []string{"processed", "failed"}
	}

	pipe := client.rdb.Pipeline()
	for _, stat := range stats {
		pipe.Del(ctx, client.statKey(stat))
	}
	_, err := pipe.Exec(ctx)
	return wrapRedisErr("reset stats", err)
}

// DayStats holds statistics for a single day.
type DayStats struct {
	Date      time.Time
	Processed int64
	Failed    int64
}

// GetHistory fetches historical statistics for the specified number of days.
// The startDate parameter specifies the most recent day to include.
func GetHistory(ctx context.Context, client *Client, days int, startDate time.Time) ([]DayStats, error) {
	if days <= 0 {
		return nil, nil
	}

	pipe := client.rdb.Pipeline()
	processedCmds := make([]*redis.StringCmd, days)
	failedCmds := make([]*redis.StringCmd, days)
	dates := make([]time.Time, days)

	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, -i)
		dates[i] = date
		dateStr := date.Format("2006-01-02")
		processedCmds[i] = pipe.Get(ctx, client.statKey("processed:"+dateStr))
		failedCmds[i] = pipe.Get(ctx, client.statKey("failed:"+dateStr))
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, wrapRedisErr("fetch history", err)
	}

	result := make([]DayStats, days)
	for i := 0; i < days; i++ {
		result[i].Date = dates[i]
		if v, err := processedCmds[i].Result(); err == nil {
			result[i].Processed, _ = strconv.ParseInt(v, 10, 64)
		}
		if v, err := failedCmds[i].Result(); err == nil {
			result[i].Failed, _ = strconv.ParseInt(v, 10, 64)
		}
	}

	return result, nil
}
