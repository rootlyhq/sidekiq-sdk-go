# sidekiq-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/rootlyhq/sidekiq-sdk-go.svg)](https://pkg.go.dev/github.com/rootlyhq/sidekiq-sdk-go)
[![CI](https://github.com/rootlyhq/sidekiq-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/rootlyhq/sidekiq-sdk-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rootlyhq/sidekiq-sdk-go)](https://goreportcard.com/report/github.com/rootlyhq/sidekiq-sdk-go)

Go SDK for interacting with [Sidekiq](https://sidekiq.org)'s Redis data structures. Monitor queues, manage jobs, and integrate Sidekiq metrics into your Go applications.

## Features

- **Full API compatibility** with Ruby's `Sidekiq::API`
- **Stats**: Processed/failed counts, queue sizes, latency metrics
- **Queues**: List, iterate, find jobs, clear queues
- **Sorted Sets**: Scheduled, Retry, Dead job management
- **Processes**: Monitor active Sidekiq workers
- **Job Enqueuing**: Push jobs to Sidekiq from Go
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
    fmt.Printf("Retry size: %d\n", stats.RetrySize())
}
```

## Usage

### Queue Operations

```go
// List all queues
queues, _ := sidekiq.AllQueues(ctx, client)
for _, q := range queues {
    size, _ := q.Size(ctx)
    latency, _ := q.Latency(ctx)
    fmt.Printf("%s: %d jobs, %.2fs latency\n", q.Name(), size, latency)
}

// Iterate jobs in a queue
queue := sidekiq.NewQueue(client, "default")
queue.Each(ctx, func(job sidekiq.JobRecord) bool {
    fmt.Printf("Job %s: %s\n", job.JID(), job.Class())
    return true // continue iteration
})

// Find specific job
job, _ := queue.FindJob(ctx, "abc123")

// Clear queue
deleted, _ := queue.Clear(ctx)

// Pause/unpause queue
queue.Pause(ctx)
queue.Unpause(ctx)
```

### Retry Management

```go
retrySet := sidekiq.NewRetrySet(client)

// Iterate retries
retrySet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
    fmt.Printf("%s failed: %s\n", entry.JID(), entry.ErrorMessage())
    return true
})

// Retry a job immediately
entry, _ := retrySet.FindJob(ctx, "xyz789")
entry.AddToQueue(ctx)

// Move to dead set
entry.Kill(ctx)

// Retry all
retrySet.RetryAll(ctx)
```

### Process Monitoring

```go
processSet := sidekiq.NewProcessSet(client)

processSet.Each(ctx, func(p *sidekiq.Process) bool {
    fmt.Printf("%s: %d/%d busy\n", p.Hostname(), p.Busy(), p.Concurrency())
    return true
})

// Signal processes to stop
processSet.Each(ctx, func(p *sidekiq.Process) bool {
    p.Quiet(ctx)
    return true
})
```

### Enqueue Jobs

```go
// Immediate execution
jid, _ := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1", 42}, nil)

// Scheduled execution
jid, _ := client.EnqueueAt(ctx, time.Now().Add(time.Hour), "MyWorker",
    []interface{}{"arg1"},
    &sidekiq.EnqueueOptions{
        Queue: "critical",
        Retry: 5,
    })

// Delayed execution
jid, _ := client.EnqueueIn(ctx, 30*time.Minute, "MyWorker", []interface{}{}, nil)

// Bulk enqueue
count, _ := client.EnqueueBulk(ctx, "BulkWorker", [][]interface{}{
    {"arg1"},
    {"arg2"},
    {"arg3"},
}, nil)
```

### With Namespace

```go
client := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))
```

### Stats History

```go
// Get last 7 days of statistics
history, _ := sidekiq.GetHistory(ctx, client, 7, time.Now())
for _, day := range history {
    fmt.Printf("%s: %d processed, %d failed\n",
        day.Date.Format("2006-01-02"),
        day.Processed,
        day.Failed)
}
```

## Compatibility

- Go 1.24+
- Sidekiq 7.x+
- Redis 6.x+

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Run all checks
make check

# Generate coverage report
make coverage-html
```

## License

MIT License - see [LICENSE](LICENSE)
