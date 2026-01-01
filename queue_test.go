package sidekiq

import (
	"context"
	"testing"

	"github.com/rootlyhq/sidekiq-sdk-go/internal/testdata"
)

func TestQueue_Basic(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker1", "a"),
		fixtures.SampleJob("job2", "Worker2", "b"),
		fixtures.SampleJob("job3", "Worker3", "c"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	if queue.Name() != "default" {
		t.Errorf("Name() = %q, want %q", queue.Name(), "default")
	}

	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 3 {
		t.Errorf("Size() = %d, want 3", size)
	}
}

func TestQueue_Each(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker1", "a"),
		fixtures.SampleJob("job2", "Worker2", "b"),
		fixtures.SampleJob("job3", "Worker3", "c"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	var jobs []string
	err := queue.Each(ctx, func(job JobRecord) bool {
		jobs = append(jobs, job.JID())
		return true
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("got %d jobs, want 3", len(jobs))
	}
}

func TestQueue_Each_StopEarly(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker"),
		fixtures.SampleJob("job2", "Worker"),
		fixtures.SampleJob("job3", "Worker"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	count := 0
	err := queue.Each(ctx, func(job JobRecord) bool {
		count++
		return count < 2 // stop after 2
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}

	if count != 2 {
		t.Errorf("iterated %d times, want 2", count)
	}
}

func TestQueue_FindJob(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker1"),
		fixtures.SampleJob("job2", "Worker2"),
		fixtures.SampleJob("job3", "Worker3"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	job, err := queue.FindJob(ctx, "job2")
	if err != nil {
		t.Fatalf("FindJob() error = %v", err)
	}
	if job == nil {
		t.Fatal("FindJob() returned nil")
	}
	if job.JID() != "job2" {
		t.Errorf("JID() = %q, want %q", job.JID(), "job2")
	}
	if job.Class() != "Worker2" {
		t.Errorf("Class() = %q, want %q", job.Class(), "Worker2")
	}
}

func TestQueue_FindJob_NotFound(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker1"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	job, err := queue.FindJob(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindJob() error = %v", err)
	}
	if job != nil {
		t.Errorf("FindJob() = %v, want nil", job)
	}
}

func TestQueue_Clear(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default",
		fixtures.SampleJob("job1", "Worker"),
		fixtures.SampleJob("job2", "Worker"),
	)

	ctx := context.Background()
	queue := NewQueue(client, "default")

	deleted, err := queue.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if deleted != 2 {
		t.Errorf("Clear() = %d, want 2", deleted)
	}

	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("after Clear(), Size() = %d, want 0", size)
	}
}

func TestQueue_Pause(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")
	fixtures.AddQueue("default")

	ctx := context.Background()
	queue := NewQueue(client, "default")

	// Initially not paused
	paused, err := queue.Paused(ctx)
	if err != nil {
		t.Fatalf("Paused() error = %v", err)
	}
	if paused {
		t.Error("Paused() = true, want false")
	}

	// Pause
	if err := queue.Pause(ctx); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	paused, _ = queue.Paused(ctx)
	if !paused {
		t.Error("after Pause(), Paused() = false, want true")
	}

	// Unpause
	if err := queue.Unpause(ctx); err != nil {
		t.Fatalf("Unpause() error = %v", err)
	}

	paused, _ = queue.Paused(ctx)
	if paused {
		t.Error("after Unpause(), Paused() = true, want false")
	}
}

func TestAllQueues(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddQueue("default", fixtures.SampleJob("j1", "W"))
	fixtures.AddQueue("critical", fixtures.SampleJob("j2", "W"))
	fixtures.AddQueue("low", fixtures.SampleJob("j3", "W"))

	ctx := context.Background()
	queues, err := AllQueues(ctx, client)
	if err != nil {
		t.Fatalf("AllQueues() error = %v", err)
	}

	if len(queues) != 3 {
		t.Errorf("AllQueues() returned %d queues, want 3", len(queues))
	}

	names := make(map[string]bool)
	for _, q := range queues {
		names[q.Name()] = true
	}

	for _, name := range []string{"default", "critical", "low"} {
		if !names[name] {
			t.Errorf("AllQueues() missing queue %q", name)
		}
	}
}
