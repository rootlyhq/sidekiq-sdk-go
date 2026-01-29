# sidekiq-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/rootlyhq/sidekiq-sdk-go.svg)](https://pkg.go.dev/github.com/rootlyhq/sidekiq-sdk-go)
[![CI](https://github.com/rootlyhq/sidekiq-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/rootlyhq/sidekiq-sdk-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rootlyhq/sidekiq-sdk-go)](https://goreportcard.com/report/github.com/rootlyhq/sidekiq-sdk-go)

Go SDK for interacting with [Sidekiq](https://sidekiq.org)'s Redis data structures. Monitor queues, manage jobs, and integrate Sidekiq metrics into your Go applications.

## Features

- **Full API compatibility** with Ruby's `Sidekiq::API`
- **Stats**: Processed/failed counts, queue sizes, latency metrics, historical data
- **Queues**: List, iterate, find jobs, clear, pause/unpause
- **Sorted Sets**: Scheduled, Retry, Dead job management with reschedule/retry/kill
- **Processes**: Monitor active Sidekiq workers, send quiet/stop signals
- **Work Tracking**: See currently executing jobs across all workers
- **Job Enqueuing**: Push jobs to Sidekiq from Go (immediate, scheduled, bulk)
- **Namespace support**: Works with sidekiq-namespace gem
- **Thread-safe**: Safe for concurrent use

## Installation

```bash
go get github.com/rootlyhq/sidekiq-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/redis/go-redis/v9"
    sidekiq "github.com/rootlyhq/sidekiq-sdk-go"
)

func main() {
    ctx := context.Background()

    // Connect to Redis
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer rdb.Close()

    // Create Sidekiq client
    client := sidekiq.NewClient(rdb)

    // Get stats
    stats, err := sidekiq.NewStats(ctx, client)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Processed: %d\n", stats.Processed())
    fmt.Printf("Failed: %d\n", stats.Failed())
    fmt.Printf("Enqueued: %d\n", stats.Enqueued())
    fmt.Printf("Scheduled: %d\n", stats.ScheduledSize())
    fmt.Printf("Retries: %d\n", stats.RetrySize())
    fmt.Printf("Dead: %d\n", stats.DeadSize())
}
```

## Usage

### Queue Operations

```go
// List all queues with sizes
queues, _ := sidekiq.AllQueues(ctx, client)
for _, q := range queues {
    size, _ := q.Size(ctx)
    latency, _ := q.Latency(ctx)
    paused, _ := q.Paused(ctx)
    fmt.Printf("%s: %d jobs, %.2fs latency, paused=%v\n", q.Name(), size, latency, paused)
}

// Work with a specific queue
queue := sidekiq.NewQueue(client, "default")

// Iterate jobs in a queue
queue.Each(ctx, func(job sidekiq.JobRecord) bool {
    fmt.Printf("Job %s: %s %v\n", job.JID(), job.DisplayClass(), job.DisplayArgs())
    return true // continue iteration
})

// Find specific job by JID
job, err := queue.FindJob(ctx, "abc123")
if err == nil {
    fmt.Printf("Found: %s\n", job.Class())
}

// Clear all jobs from queue
deleted, _ := queue.Clear(ctx)
fmt.Printf("Deleted %d jobs\n", deleted)

// Pause/unpause queue processing
queue.Pause(ctx)
queue.Unpause(ctx)
```

### Scheduled Jobs

```go
scheduledSet := sidekiq.NewScheduledSet(client)

// Get count of scheduled jobs
size, _ := scheduledSet.Size(ctx)
fmt.Printf("Scheduled jobs: %d\n", size)

// Iterate scheduled jobs
scheduledSet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
    fmt.Printf("Job %s scheduled for %s\n", entry.JID(), entry.At().Format(time.RFC3339))
    return true
})

// Find and reschedule a job
entry, _ := scheduledSet.FindJob(ctx, "job-id")
if entry != nil {
    // Move to 1 hour from now
    entry.Reschedule(ctx, time.Now().Add(time.Hour))

    // Or execute immediately
    entry.AddToQueue(ctx)

    // Or delete it
    entry.Delete(ctx)
}
```

### Retry Management

```go
retrySet := sidekiq.NewRetrySet(client)

// Iterate failed jobs awaiting retry
retrySet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
    fmt.Printf("Job %s failed: [%s] %s\n",
        entry.JID(),
        entry.ErrorClass(),
        entry.ErrorMessage())
    fmt.Printf("  Retry count: %d, Next retry: %s\n",
        entry.RetryCount(),
        entry.At().Format(time.RFC3339))
    return true
})

// Find and handle a specific failed job
entry, _ := retrySet.FindJob(ctx, "xyz789")
if entry != nil {
    // Retry immediately
    entry.AddToQueue(ctx)

    // Or move to dead set (give up)
    entry.Kill(ctx)

    // Or reschedule retry
    entry.Reschedule(ctx, time.Now().Add(time.Hour))
}

// Bulk operations
retrySet.RetryAll(ctx) // Retry all failed jobs immediately
retrySet.KillAll(ctx)  // Move all to dead set
retrySet.Clear(ctx)    // Delete all retry jobs
```

### Dead Set (Morgue)

```go
deadSet := sidekiq.NewDeadSet(client)

// Iterate dead jobs
deadSet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
    fmt.Printf("Dead job %s: %s - %s\n",
        entry.JID(),
        entry.ErrorClass(),
        entry.ErrorMessage())
    return true
})

// Resurrect a dead job
entry, _ := deadSet.FindJob(ctx, "dead-job-id")
if entry != nil {
    entry.AddToQueue(ctx) // Send back to its original queue
}

// Bulk operations
deadSet.RetryAll(ctx) // Resurrect all dead jobs
deadSet.Clear(ctx)    // Permanently delete all dead jobs
```

### Process Monitoring

```go
processSet := sidekiq.NewProcessSet(client)

// Get total concurrency across all workers
total, _ := processSet.TotalConcurrency(ctx)
fmt.Printf("Total worker threads: %d\n", total)

// Iterate active processes
processSet.Each(ctx, func(p *sidekiq.Process) bool {
    fmt.Printf("Process %s on %s\n", p.Identity(), p.Hostname())
    fmt.Printf("  PID: %d, Started: %s\n", p.PID(), p.StartedAt().Format(time.RFC3339))
    fmt.Printf("  Concurrency: %d, Busy: %d\n", p.Concurrency(), p.Busy())
    fmt.Printf("  Queues: %v\n", p.Queues())
    fmt.Printf("  Version: %s\n", p.Version())
    return true
})

// Send signals to processes
processSet.Each(ctx, func(p *sidekiq.Process) bool {
    p.Quiet(ctx) // Stop fetching new jobs
    // p.Stop(ctx) // Shutdown gracefully
    return true
})
```

### Work Tracking (Active Jobs)

```go
workSet := sidekiq.NewWorkSet(client)

// Get count of currently executing jobs
size, _ := workSet.Size(ctx)
fmt.Printf("Jobs currently executing: %d\n", size)

// Iterate active work
workSet.Each(ctx, func(w *sidekiq.Work) bool {
    fmt.Printf("Job %s running on queue %s since %s\n",
        w.JID(),
        w.Queue(),
        w.RunAt().Format(time.RFC3339))

    // Get full job details
    job := w.Job()
    if job != nil {
        fmt.Printf("  Class: %s, Args: %v\n", job.DisplayClass(), job.DisplayArgs())
    }
    return true
})
```

### Enqueue Jobs

```go
// Immediate execution
jid, err := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1", 42}, nil)
fmt.Printf("Enqueued job: %s\n", jid)

// With options
jid, _ = client.Enqueue(ctx, "MyWorker", []interface{}{"data"}, &sidekiq.EnqueueOptions{
    Queue: "critical",
    Retry: 5,
    Tags:  []string{"important", "user-123"},
})

// Scheduled execution (absolute time)
jid, _ = client.EnqueueAt(ctx, time.Now().Add(time.Hour), "MyWorker",
    []interface{}{"scheduled"}, nil)

// Delayed execution (relative duration)
jid, _ = client.EnqueueIn(ctx, 30*time.Minute, "MyWorker",
    []interface{}{"delayed"}, nil)

// Bulk enqueue (efficient for many jobs)
jobs := [][]interface{}{
    {"user-1", "action-a"},
    {"user-2", "action-b"},
    {"user-3", "action-c"},
}
count, _ := client.EnqueueBulk(ctx, "BulkWorker", jobs, &sidekiq.EnqueueOptions{
    Queue: "bulk",
})
fmt.Printf("Enqueued %d jobs\n", count)
```

### Stats History

```go
// Get last 30 days of statistics
history, _ := sidekiq.GetHistory(ctx, client, 30, time.Now())
for _, day := range history {
    fmt.Printf("%s: %d processed, %d failed (%.2f%% failure rate)\n",
        day.Date.Format("2006-01-02"),
        day.Processed,
        day.Failed,
        float64(day.Failed)/float64(day.Processed+1)*100)
}

// Reset statistics
sidekiq.Reset(ctx, client, "processed", "failed")
```

### With Namespace

```go
// For apps using sidekiq-namespace gem
client := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))

// All operations will use "myapp:" prefix for Redis keys
stats, _ := sidekiq.NewStats(ctx, client)
```

## Compatibility

- Go 1.24+
- Sidekiq 7.x+
- Redis 6.x+

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Run all checks (fmt + lint + test)
make check

# Generate coverage report
make coverage-html

# Show current version
make version
```

## License

MIT License © 2026 Rootly Inc. - see [LICENCE.txt](LICENCE.txt)
