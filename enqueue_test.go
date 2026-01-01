package sidekiq

import (
	"context"
	"testing"
	"time"
)

func TestEnqueue_Immediate(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	jid, err := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1", 42}, nil)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if jid == "" {
		t.Error("Enqueue() returned empty JID")
	}
	if len(jid) != 24 {
		t.Errorf("JID length = %d, want 24", len(jid))
	}

	// Verify job is in queue
	queue := NewQueue(client, "default")
	size, _ := queue.Size(ctx)
	if size != 1 {
		t.Errorf("queue size = %d, want 1", size)
	}

	job, _ := queue.FindJob(ctx, jid)
	if job == nil {
		t.Fatal("job not found in queue")
	}
	if job.Class() != "MyWorker" {
		t.Errorf("Class() = %q, want %q", job.Class(), "MyWorker")
	}
}

func TestEnqueue_WithOptions(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	jid, err := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1"},
		&EnqueueOptions{
			Queue: "critical",
			Retry: 5,
			Tags:  []string{"important"},
		})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	queue := NewQueue(client, "critical")
	job, _ := queue.FindJob(ctx, jid)
	if job == nil {
		t.Fatal("job not found in critical queue")
	}

	tags := job.Tags()
	if len(tags) != 1 || tags[0] != "important" {
		t.Errorf("Tags() = %v, want [important]", tags)
	}
}

func TestEnqueueAt_Future(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	future := time.Now().Add(time.Hour)
	jid, err := client.EnqueueAt(ctx, future, "ScheduledWorker", []interface{}{"arg"}, nil)
	if err != nil {
		t.Fatalf("EnqueueAt() error = %v", err)
	}
	if jid == "" {
		t.Error("EnqueueAt() returned empty JID")
	}

	// Job should be in scheduled set, not queue
	queue := NewQueue(client, "default")
	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("queue size = %d, want 0 (job should be scheduled)", size)
	}

	schedSet := NewScheduledSet(client)
	schedSize, _ := schedSet.Size(ctx)
	if schedSize != 1 {
		t.Errorf("scheduled set size = %d, want 1", schedSize)
	}
}

func TestEnqueueAt_Past(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	jid, err := client.EnqueueAt(ctx, past, "Worker", []interface{}{}, nil)
	if err != nil {
		t.Fatalf("EnqueueAt() error = %v", err)
	}

	// Job should be in queue immediately since time is in past
	queue := NewQueue(client, "default")
	job, _ := queue.FindJob(ctx, jid)
	if job == nil {
		t.Error("job should be in queue for past time")
	}
}

func TestEnqueueIn(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	jid, err := client.EnqueueIn(ctx, time.Hour, "DelayedWorker", []interface{}{"arg"}, nil)
	if err != nil {
		t.Fatalf("EnqueueIn() error = %v", err)
	}
	if jid == "" {
		t.Error("EnqueueIn() returned empty JID")
	}

	schedSet := NewScheduledSet(client)
	entry, _ := schedSet.FindJob(ctx, jid)
	if entry == nil {
		t.Fatal("job not found in scheduled set")
	}
}

func TestEnqueue_NilArgs(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	jid, err := client.Enqueue(ctx, "Worker", nil, nil)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	queue := NewQueue(client, "default")
	job, _ := queue.FindJob(ctx, jid)
	if job == nil {
		t.Fatal("job not found")
	}

	args := job.Args()
	if args == nil {
		t.Error("Args() is nil, want empty slice")
	}
	if len(args) != 0 {
		t.Errorf("Args() length = %d, want 0", len(args))
	}
}

func TestEnqueueBulk(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	argsList := [][]interface{}{
		{"arg1"},
		{"arg2"},
		{"arg3"},
	}

	count, err := client.EnqueueBulk(ctx, "BulkWorker", argsList, nil)
	if err != nil {
		t.Fatalf("EnqueueBulk() error = %v", err)
	}
	if count != 3 {
		t.Errorf("EnqueueBulk() = %d, want 3", count)
	}

	queue := NewQueue(client, "default")
	size, _ := queue.Size(ctx)
	if size != 3 {
		t.Errorf("queue size = %d, want 3", size)
	}
}

func TestEnqueueBulk_Empty(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	count, err := client.EnqueueBulk(ctx, "Worker", nil, nil)
	if err != nil {
		t.Fatalf("EnqueueBulk() error = %v", err)
	}
	if count != 0 {
		t.Errorf("EnqueueBulk() = %d, want 0", count)
	}
}
