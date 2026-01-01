// Package sidekiq provides a Go SDK for interacting with Sidekiq's Redis data structures.
//
// This SDK allows Go applications to monitor, administer, and manage Sidekiq jobs
// by directly interacting with the underlying Redis data structures. It provides
// full API compatibility with Ruby's Sidekiq::API module.
//
// # Quick Start
//
//	import (
//	    "context"
//	    "github.com/redis/go-redis/v9"
//	    sidekiq "github.com/rootlyhq/sidekiq-sdk-go"
//	)
//
//	func main() {
//	    ctx := context.Background()
//	    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	    client := sidekiq.NewClient(rdb)
//
//	    // Get stats
//	    stats, _ := sidekiq.NewStats(ctx, client)
//	    fmt.Printf("Processed: %d\n", stats.Processed())
//	}
//
// # Features
//
//   - Stats: Processed/failed counts, queue sizes, latency metrics
//   - Queues: List, iterate, find jobs, clear queues
//   - Sorted Sets: Scheduled, Retry, Dead job management
//   - Processes: Monitor active Sidekiq workers
//   - Job Enqueuing: Push jobs to Sidekiq from Go
//   - Namespace support: Works with sidekiq-namespace gem
//
// # Thread Safety
//
// All types in this package are safe for concurrent use by multiple goroutines.
// The underlying Redis client handles connection pooling.
//
// # Compatibility
//
//   - Go 1.24+
//   - Sidekiq 7.x+
//   - Redis 6.x+
package sidekiq
