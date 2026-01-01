package sidekiq

import (
	"context"
	"testing"
	"time"

	"github.com/rootlyhq/sidekiq-sdk-go/internal/testdata"
)

func TestWorkSet_Size(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	// Add a process with work
	proc := fixtures.SampleProcess("worker-1:1:abc")
	fixtures.AddProcess(proc)
	fixtures.AddWork(proc.Identity, "tid1", fixtures.SampleWork("work1", "Worker1", "default"))
	fixtures.AddWork(proc.Identity, "tid2", fixtures.SampleWork("work2", "Worker2", "critical"))

	ctx := context.Background()
	workSet := NewWorkSet(client)

	size, err := workSet.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 2 {
		t.Errorf("Size() = %d, want 2", size)
	}
}

func TestWorkSet_Each(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	proc := fixtures.SampleProcess("worker-1:1:xyz")
	fixtures.AddProcess(proc)
	fixtures.AddWork(proc.Identity, "tid1", fixtures.SampleWork("j1", "W1", "default"))
	fixtures.AddWork(proc.Identity, "tid2", fixtures.SampleWork("j2", "W2", "critical"))

	ctx := context.Background()
	workSet := NewWorkSet(client)

	var jids []string
	err := workSet.Each(ctx, func(w *Work) bool {
		jids = append(jids, w.JID())
		return true
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}

	if len(jids) != 2 {
		t.Errorf("Each() iterated %d items, want 2", len(jids))
	}
}

func TestWork_Fields(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	proc := fixtures.SampleProcess("worker-1:1:test")
	fixtures.AddProcess(proc)
	fixtures.AddWork(proc.Identity, "tid1", fixtures.SampleWork("job123", "MyWorker", "high"))

	ctx := context.Background()
	workSet := NewWorkSet(client)

	var work *Work
	_ = workSet.Each(ctx, func(w *Work) bool {
		work = w
		return false // stop after first
	})

	if work == nil {
		t.Fatal("no work found")
	}

	if work.JID() != "job123" {
		t.Errorf("JID() = %q, want %q", work.JID(), "job123")
	}
	if work.Queue() != "high" {
		t.Errorf("Queue() = %q, want %q", work.Queue(), "high")
	}
	if work.RunAt().IsZero() {
		t.Error("RunAt() should not be zero")
	}
}

func TestWorkSet_Empty(t *testing.T) {
	client, _ := setupTestClient(t)

	ctx := context.Background()
	workSet := NewWorkSet(client)

	size, err := workSet.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 0 {
		t.Errorf("Size() = %d, want 0", size)
	}

	count := 0
	err = workSet.Each(ctx, func(w *Work) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Each() iterated %d items, want 0", count)
	}
}

func TestWorkSet_MultipleProcesses(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	// Add two processes with work
	proc1 := fixtures.SampleProcess("worker-1:1:aaa")
	proc2 := fixtures.SampleProcess("worker-2:2:bbb")
	proc2.Hostname = "worker-2"

	fixtures.AddProcess(proc1)
	fixtures.AddProcess(proc2)

	fixtures.AddWork(proc1.Identity, "t1", fixtures.SampleWork("j1", "W", "default"))
	fixtures.AddWork(proc2.Identity, "t2", fixtures.SampleWork("j2", "W", "default"))
	fixtures.AddWork(proc2.Identity, "t3", fixtures.SampleWork("j3", "W", "critical"))

	ctx := context.Background()
	workSet := NewWorkSet(client)

	size, err := workSet.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 3 {
		t.Errorf("Size() = %d, want 3", size)
	}
}

func TestWork_Job(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	proc := fixtures.SampleProcess("worker-1:1:job")
	fixtures.AddProcess(proc)
	fixtures.AddWork(proc.Identity, "tid1", testdata.WorkInfo{
		Queue: "default",
		RunAt: float64(time.Now().Unix()),
		Payload: map[string]interface{}{
			"jid":         "work-job-123",
			"class":       "TestWorker",
			"args":        []interface{}{"arg1", 42},
			"queue":       "default",
			"created_at":  float64(time.Now().Unix()),
			"enqueued_at": float64(time.Now().Unix()),
		},
	})

	ctx := context.Background()
	workSet := NewWorkSet(client)

	var work *Work
	_ = workSet.Each(ctx, func(w *Work) bool {
		work = w
		return false
	})

	if work == nil {
		t.Fatal("no work found")
	}

	job := work.Job()
	if job == nil {
		t.Fatal("Job() returned nil")
	}

	if job.JID() != "work-job-123" {
		t.Errorf("Job().JID() = %q, want %q", job.JID(), "work-job-123")
	}
	if job.Class() != "TestWorker" {
		t.Errorf("Job().Class() = %q, want %q", job.Class(), "TestWorker")
	}
}
