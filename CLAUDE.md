# Sidekiq Go SDK - Developer Reference

## Overview

Go SDK for interacting with Sidekiq's Redis data structures. Provides monitoring, administration, and job management capabilities compatible with Ruby's `Sidekiq::API`.

## Tech Stack

- **Language**: Go 1.24+
- **Redis Client**: github.com/redis/go-redis/v9
- **Testing**: github.com/alicebob/miniredis/v2

## Project Structure

```
├── sidekiq.go          # Client, NewClient()
├── options.go          # WithNamespace(), WithLogger()
├── keys.go             # Redis key builders
├── errors.go           # ErrJobNotFound, etc.
├── stats.go            # Stats, GetHistory()
├── queue.go            # Queue, AllQueues()
├── job.go              # JobRecord
├── sorted_set.go       # SortedSet, JobSet
├── sorted_entry.go     # SortedEntry
├── scheduled_set.go    # ScheduledSet
├── retry_set.go        # RetrySet
├── dead_set.go         # DeadSet
├── process.go          # ProcessSet, Process
├── work.go             # WorkSet, Work
├── enqueue.go          # Enqueue(), EnqueueAt()
├── scripts.go          # Lua scripts
└── internal/
    ├── json/           # JSON utilities
    └── testdata/       # Test fixtures
```

## Build Commands

```bash
make test           # Run tests
make lint           # Run linter
make check          # fmt + lint + test
make coverage       # Generate coverage report
make coverage-html  # HTML coverage report
```

## Architecture

### Client

Central struct holding Redis connection and configuration.

```go
client := sidekiq.NewClient(rdb,
    sidekiq.WithNamespace("myapp"),
    sidekiq.WithLogger(logger),
)
```

### Redis Keys

All keys are built through the client to support namespacing:
- `stat:processed`, `stat:failed` - counters
- `queues` - SET of queue names
- `queue:{name}` - LIST of jobs
- `schedule`, `retry`, `dead` - ZSETs with timestamp scores
- `processes` - SET of process identities
- `{identity}:work` - HASH of active jobs

### Iteration Pattern

Callback-based iteration with early exit support:

```go
queue.Each(ctx, func(job JobRecord) bool {
    // return false to stop
    return true
})
```

## Testing

Uses miniredis for unit tests. Test fixtures in `internal/testdata/fixtures.go` provide helpers to populate Redis with realistic Sidekiq data.

```go
fixtures := testdata.NewFixtures(mr, "")
fixtures.SetupStats(1000, 50)
fixtures.AddQueue("default", fixtures.SampleJob("j1", "Worker"))
```

## Key Types

### Stats
Snapshot of Sidekiq statistics. Create with `NewStats(ctx, client)`.

### Queue
Represents a Sidekiq queue. Operations: Size, Latency, Each, FindJob, Clear, Pause/Unpause.

### JobRecord
Immutable job data. Provides accessors for all standard Sidekiq job fields including ActiveJob unwrapping via DisplayClass/DisplayArgs.

### SortedEntry
Job in a sorted set (schedule, retry, dead). Can be rescheduled, moved to queue, or killed.

### Process
Active Sidekiq worker process. Provides info like hostname, concurrency, queues, and can send signals (Quiet, Stop).

### Work
Currently executing job. Tracks which process/thread is running it.

## Error Handling

```go
var (
    ErrJobNotFound     = errors.New("sidekiq: job not found")
    ErrQueueNotFound   = errors.New("sidekiq: queue not found")
    ErrProcessNotFound = errors.New("sidekiq: process not found")
    ErrInvalidJob      = errors.New("sidekiq: invalid job data")
)
```

Redis errors are wrapped in `RedisError` with operation context.

## Debug

Enable Redis command logging:

```go
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
redis.SetLogger(log.New(os.Stderr, "redis: ", log.LstdFlags))
```
