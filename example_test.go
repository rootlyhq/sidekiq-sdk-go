package sidekiq_test

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	sidekiq "github.com/rootlyhq/sidekiq-sdk-go"
)

func ExampleNewClient() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Create a basic client
	client := sidekiq.NewClient(rdb)
	_ = client

	// Create a client with namespace
	clientNS := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))
	_ = clientNS
}

func ExampleNewStats() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	stats, err := sidekiq.NewStats(ctx, client)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Processed: %d\n", stats.Processed())
	fmt.Printf("Failed: %d\n", stats.Failed())
	fmt.Printf("Enqueued: %d\n", stats.Enqueued())
	fmt.Printf("Retry Size: %d\n", stats.RetrySize())
	fmt.Printf("Dead Size: %d\n", stats.DeadSize())
}

func ExampleNewQueue() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// Get a specific queue
	queue := sidekiq.NewQueue(client, "default")

	// Get queue size
	size, _ := queue.Size(ctx)
	fmt.Printf("Queue size: %d\n", size)

	// Get queue latency
	latency, _ := queue.Latency(ctx)
	fmt.Printf("Latency: %.2f seconds\n", latency)

	// Iterate over jobs in the queue
	_ = queue.Each(ctx, func(job sidekiq.JobRecord) bool {
		fmt.Printf("Job %s: %s\n", job.JID(), job.DisplayClass())
		return true // continue iteration
	})
}

func ExampleAllQueues() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// List all queues
	queues, err := sidekiq.AllQueues(ctx, client)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, q := range queues {
		size, _ := q.Size(ctx)
		fmt.Printf("%s: %d jobs\n", q.Name(), size)
	}
}

func ExampleNewRetrySet() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	retrySet := sidekiq.NewRetrySet(client)

	// Get size
	size, _ := retrySet.Size(ctx)
	fmt.Printf("Retry set size: %d\n", size)

	// Iterate over retries
	_ = retrySet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
		fmt.Printf("Job %s failed: %s\n", entry.JID(), entry.ErrorMessage())
		fmt.Printf("Scheduled retry at: %s\n", entry.At().Format(time.RFC3339))
		return true
	})

	// Find specific job
	entry, err := retrySet.FindJob(ctx, "some-jid")
	if err == nil && entry != nil {
		// Retry immediately
		_ = entry.AddToQueue(ctx)
	}
}

func ExampleNewDeadSet() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	deadSet := sidekiq.NewDeadSet(client)

	// Get size
	size, _ := deadSet.Size(ctx)
	fmt.Printf("Dead set size: %d\n", size)

	// Iterate over dead jobs
	_ = deadSet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
		fmt.Printf("Dead job %s: %s - %s\n",
			entry.JID(),
			entry.ErrorClass(),
			entry.ErrorMessage())
		return true
	})
}

func ExampleNewProcessSet() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	processSet := sidekiq.NewProcessSet(client)

	// Get number of processes
	size, _ := processSet.Size(ctx)
	fmt.Printf("Active processes: %d\n", size)

	// Iterate over processes
	_ = processSet.Each(ctx, func(p *sidekiq.Process) bool {
		fmt.Printf("%s (%s): %d/%d busy\n",
			p.Identity(),
			p.Hostname(),
			p.Busy(),
			p.Concurrency())
		return true
	})
}

func ExampleClient_Enqueue() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// Enqueue a job for immediate processing
	jid, err := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1", 42}, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Enqueued job: %s\n", jid)

	// Enqueue with options
	jid, err = client.Enqueue(ctx, "CriticalWorker", []interface{}{"data"},
		&sidekiq.EnqueueOptions{
			Queue: "critical",
			Retry: 5,
			Tags:  []string{"important"},
		})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Enqueued critical job: %s\n", jid)
}

func ExampleClient_EnqueueAt() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// Schedule a job for 1 hour from now
	scheduleTime := time.Now().Add(time.Hour)
	jid, err := client.EnqueueAt(ctx, scheduleTime, "ScheduledWorker",
		[]interface{}{"scheduled_data"}, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Scheduled job %s for %s\n", jid, scheduleTime.Format(time.RFC3339))
}

func ExampleClient_EnqueueIn() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// Schedule a job to run in 30 minutes
	jid, err := client.EnqueueIn(ctx, 30*time.Minute, "DelayedWorker",
		[]interface{}{"delayed_data"}, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Job %s will run in 30 minutes\n", jid)
}

func ExampleGetHistory() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	client := sidekiq.NewClient(rdb)
	ctx := context.Background()

	// Get stats for the last 7 days
	history, err := sidekiq.GetHistory(ctx, client, 7, time.Now())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, day := range history {
		fmt.Printf("%s: processed=%d, failed=%d\n",
			day.Date.Format("2006-01-02"),
			day.Processed,
			day.Failed)
	}
}
