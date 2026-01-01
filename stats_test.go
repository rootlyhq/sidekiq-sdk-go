package sidekiq

import (
	"context"
	"testing"
	"time"

	"github.com/rootlyhq/sidekiq-sdk-go/internal/testdata"
)

func TestStats_Basic(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	// Setup test data
	fixtures.SetupStats(1000, 50)
	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "MyWorker", "arg1"),
		fixtures.SampleJob("job2", "MyWorker", "arg2"),
	)
	fixtures.AddQueue("critical",
		fixtures.SampleJob("job3", "CriticalWorker"),
	)
	fixtures.AddScheduledJob(time.Now().Add(time.Hour),
		fixtures.SampleJob("scheduled1", "ScheduledWorker"),
	)
	fixtures.AddRetryJob(time.Now().Add(time.Minute),
		fixtures.SampleFailedJob("retry1", "FailingWorker", "StandardError", "oops"),
	)
	fixtures.AddDeadJob(time.Now(),
		fixtures.SampleFailedJob("dead1", "DeadWorker", "FatalError", "boom"),
	)

	ctx := context.Background()
	stats, err := NewStats(ctx, client)
	if err != nil {
		t.Fatalf("NewStats() error = %v", err)
	}

	if got := stats.Processed(); got != 1000 {
		t.Errorf("Processed() = %d, want 1000", got)
	}
	if got := stats.Failed(); got != 50 {
		t.Errorf("Failed() = %d, want 50", got)
	}
	if got := stats.Enqueued(); got != 3 {
		t.Errorf("Enqueued() = %d, want 3", got)
	}
	if got := stats.ScheduledSize(); got != 1 {
		t.Errorf("ScheduledSize() = %d, want 1", got)
	}
	if got := stats.RetrySize(); got != 1 {
		t.Errorf("RetrySize() = %d, want 1", got)
	}
	if got := stats.DeadSize(); got != 1 {
		t.Errorf("DeadSize() = %d, want 1", got)
	}

	queues := stats.Queues()
	if queues["default"] != 2 {
		t.Errorf("Queues()[default] = %d, want 2", queues["default"])
	}
	if queues["critical"] != 1 {
		t.Errorf("Queues()[critical] = %d, want 1", queues["critical"])
	}
}

func TestStats_Empty(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	stats, err := NewStats(ctx, client)
	if err != nil {
		t.Fatalf("NewStats() error = %v", err)
	}

	if got := stats.Processed(); got != 0 {
		t.Errorf("Processed() = %d, want 0", got)
	}
	if got := stats.Failed(); got != 0 {
		t.Errorf("Failed() = %d, want 0", got)
	}
	if got := stats.Enqueued(); got != 0 {
		t.Errorf("Enqueued() = %d, want 0", got)
	}
}

func TestStats_WithNamespace(t *testing.T) {
	client, mr := setupTestClient(t)
	clientNS := NewClient(client.Redis(), WithNamespace("myapp"))

	fixtures := testdata.NewFixtures(mr, "myapp")
	fixtures.SetupStats(500, 10)
	fixtures.AddQueue("default", fixtures.SampleJob("j1", "W"))

	ctx := context.Background()

	// Without namespace - should see nothing
	stats, _ := NewStats(ctx, client)
	if stats.Processed() != 0 {
		t.Errorf("non-namespaced client saw processed=%d, want 0", stats.Processed())
	}

	// With namespace - should see data
	statsNS, _ := NewStats(ctx, clientNS)
	if statsNS.Processed() != 500 {
		t.Errorf("namespaced client saw processed=%d, want 500", statsNS.Processed())
	}
}

func TestGetHistory(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	today := time.Now().Truncate(24 * time.Hour)
	fixtures.SetupDailyStats(today, 100, 5)
	fixtures.SetupDailyStats(today.AddDate(0, 0, -1), 90, 3)
	fixtures.SetupDailyStats(today.AddDate(0, 0, -2), 80, 2)

	ctx := context.Background()
	history, err := GetHistory(ctx, client, 3, today)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(history) != 3 {
		t.Fatalf("GetHistory() returned %d items, want 3", len(history))
	}

	if history[0].Processed != 100 {
		t.Errorf("history[0].Processed = %d, want 100", history[0].Processed)
	}
	if history[1].Processed != 90 {
		t.Errorf("history[1].Processed = %d, want 90", history[1].Processed)
	}
	if history[2].Processed != 80 {
		t.Errorf("history[2].Processed = %d, want 80", history[2].Processed)
	}
}

func TestReset(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")
	fixtures.SetupStats(1000, 50)

	ctx := context.Background()

	// Verify stats exist
	stats, _ := NewStats(ctx, client)
	if stats.Processed() != 1000 {
		t.Errorf("before reset: Processed() = %d, want 1000", stats.Processed())
	}

	// Reset
	if err := Reset(ctx, client); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	// Verify stats are cleared
	stats, _ = NewStats(ctx, client)
	if stats.Processed() != 0 {
		t.Errorf("after reset: Processed() = %d, want 0", stats.Processed())
	}
	if stats.Failed() != 0 {
		t.Errorf("after reset: Failed() = %d, want 0", stats.Failed())
	}
}
