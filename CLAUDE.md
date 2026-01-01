# Sidekiq Go SDK - Developer Reference

## Overview

Go SDK for interacting with Sidekiq's Redis data structures. Provides monitoring, administration, and job management capabilities compatible with Ruby's `Sidekiq::API`.

## Tech Stack

- **Language**: Go 1.24+
- **Redis Client**: github.com/redis/go-redis/v9
- **Testing**: github.com/alicebob/miniredis/v2
- **Linting**: golangci-lint

## Project Structure

```
├── sidekiq.go            # Client, NewClient()
├── sidekiq_test.go
├── options.go            # WithNamespace(), WithLogger()
├── keys.go               # Redis key builders
├── errors.go             # ErrJobNotFound, RedisError, etc.
├── doc.go                # Package documentation
│
├── stats.go              # Stats, GetHistory(), Reset()
├── stats_test.go
├── queue.go              # Queue, AllQueues()
├── queue_test.go
├── job.go                # JobRecord (immutable job data)
├── job_test.go
│
├── sorted_set.go         # SortedSet, JobSet base types
├── sorted_entry.go       # SortedEntry (job in sorted set)
├── sorted_set_test.go
├── scheduled_set.go      # ScheduledSet
├── retry_set.go          # RetrySet with RetryAll, KillAll
├── dead_set.go           # DeadSet with RetryAll
│
├── process.go            # ProcessSet, Process
├── process_test.go
├── work.go               # WorkSet, Work
├── work_test.go
│
├── enqueue.go            # Enqueue, EnqueueAt, EnqueueIn, EnqueueBulk
├── enqueue_test.go
├── scripts.go            # Lua scripts for atomic operations
│
├── example_test.go       # GoDoc examples
│
└── internal/
    ├── json/
    │   ├── json.go       # JSON parsing utilities
    │   └── json_test.go
    └── testdata/
        └── fixtures.go   # Test fixtures (SidekiqFixtures)
```

## Build Commands

```bash
make deps           # Download dependencies
make test           # Run tests with race detection
make lint           # Run golangci-lint
make check          # fmt + lint + test
make coverage       # Generate coverage report
make coverage-html  # HTML coverage report
make version        # Show version info
make bump-patch     # Create patch release tag
make bump-minor     # Create minor release tag
```

## Architecture

### Client

Central struct holding Redis connection and configuration. Created via functional options pattern.

```go
client := sidekiq.NewClient(rdb,
    sidekiq.WithNamespace("myapp"),
    sidekiq.WithLogger(logger),
)
```

### Redis Keys

All keys are built through the client to support namespacing:

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `stat:processed` | STRING | Total processed count |
| `stat:failed` | STRING | Total failed count |
| `stat:processed:YYYY-MM-DD` | STRING | Daily processed count |
| `stat:failed:YYYY-MM-DD` | STRING | Daily failed count |
| `queues` | SET | Queue names |
| `queue:{name}` | LIST | Jobs in queue (FIFO) |
| `schedule` | ZSET | Scheduled jobs (score = timestamp) |
| `retry` | ZSET | Failed jobs awaiting retry |
| `dead` | ZSET | Dead jobs (morgue) |
| `processes` | SET | Active process identities |
| `{identity}` | STRING | Process info JSON |
| `{identity}:work` | HASH | Active jobs per thread |
| `paused:{queue}` | STRING | Queue paused flag |

### Iteration Pattern

All collections use callback-based iteration with early exit support:

```go
queue.Each(ctx, func(job JobRecord) bool {
    // Process job
    // Return false to stop iteration
    return true
})
```

### Error Handling

```go
// Sentinel errors
var (
    ErrJobNotFound     = errors.New("sidekiq: job not found")
    ErrQueueNotFound   = errors.New("sidekiq: queue not found")
    ErrProcessNotFound = errors.New("sidekiq: process not found")
    ErrInvalidJob      = errors.New("sidekiq: invalid job data")
)

// Redis errors wrapped with context
type RedisError struct {
    Op  string
    Err error
}

// Job parsing errors
type JobParseError struct {
    Err error
}
```

## Key Types

### Stats
Snapshot of Sidekiq cluster statistics. Immutable after creation.

Methods: `Processed()`, `Failed()`, `Enqueued()`, `ScheduledSize()`, `RetrySize()`, `DeadSize()`, `ProcessesSize()`, `WorkersSize()`, `DefaultQueueLatency()`, `Queues()`

### Queue
Represents a Sidekiq queue backed by a Redis LIST.

Methods: `Name()`, `Size()`, `Latency()`, `Paused()`, `Pause()`, `Unpause()`, `Each()`, `FindJob()`, `Clear()`, `Delete()`

### JobRecord
Immutable job data. Provides accessors for all standard Sidekiq job fields.

Key methods:
- `JID()`, `Queue()`, `Class()`, `Args()` - Basic fields
- `DisplayClass()`, `DisplayArgs()` - Unwraps ActiveJob wrappers
- `CreatedAt()`, `EnqueuedAt()` - Timestamps
- `ErrorClass()`, `ErrorMessage()`, `ErrorBacktrace()` - Error info
- `RetryCount()`, `FailedAt()`, `RetriedAt()` - Retry info
- `Tags()`, `Bid()` - Metadata
- `Item()` - Raw job data map
- `Value()` - Original JSON string

### SortedEntry
Job in a sorted set (schedule, retry, dead). Embeds JobRecord and adds:

Methods: `Score()`, `At()`, `Delete()`, `Reschedule()`, `AddToQueue()`, `Retry()`, `Kill()`, `HasError()`

### Process
Active Sidekiq worker process.

Methods: `Identity()`, `Hostname()`, `PID()`, `Tag()`, `StartedAt()`, `Queues()`, `Labels()`, `Concurrency()`, `Busy()`, `Version()`, `Stopping()`, `Quiet()`, `Stop()`, `DumpThreads()`

### Work
Currently executing job.

Methods: `Queue()`, `RunAt()`, `JID()`, `Job()`, `Payload()`

## Testing

Uses miniredis for unit tests - no real Redis required. Test fixtures in `internal/testdata/fixtures.go`:

```go
client, mr := setupTestClient(t)
fixtures := testdata.NewFixtures(mr, "")

// Setup test data
fixtures.SetupStats(1000, 50)
fixtures.AddQueue("default", fixtures.SampleJob("j1", "Worker"))
fixtures.AddRetryJob(time.Now(), fixtures.SampleFailedJob("r1", "W", "Error", "msg"))
fixtures.AddProcess(fixtures.SampleProcess("worker-1:1:abc"))
fixtures.AddWork("worker-1:1:abc", "tid1", fixtures.SampleWork("w1", "Worker", "default"))
```

## Lua Scripts

Located in `scripts.go`. Used for atomic operations:

- `requeueScript` - Move job from sorted set to queue atomically
- `killScript` - Move job to dead set with trimming

## Adding New Features

1. Add implementation in appropriate file
2. Add tests using miniredis fixtures
3. Update example_test.go if user-facing
4. Run `make check` to verify
5. Update CHANGELOG.md

## Debug

Enable Redis command logging:

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Log all Redis commands
rdb.AddHook(&redis.LoggingHook{})
```

Or use the SDK's logger option:

```go
client := sidekiq.NewClient(rdb, sidekiq.WithLogger(myLogger))
```
