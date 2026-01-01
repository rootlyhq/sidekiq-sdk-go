package sidekiq

import (
	"context"
	"testing"

	"github.com/rootlyhq/sidekiq-sdk-go/internal/testdata"
)

func TestProcessSet_Size(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddProcess(fixtures.SampleProcess("worker1:1234:abc"))
	fixtures.AddProcess(fixtures.SampleProcess("worker2:5678:def"))

	ctx := context.Background()
	processSet := NewProcessSet(client)

	size, err := processSet.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 2 {
		t.Errorf("Size() = %d, want 2", size)
	}
}

func TestProcessSet_Each(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddProcess(fixtures.SampleProcess("worker1:1234:abc"))
	fixtures.AddProcess(fixtures.SampleProcess("worker2:5678:def"))

	ctx := context.Background()
	processSet := NewProcessSet(client)

	var identities []string
	err := processSet.Each(ctx, func(p *Process) bool {
		identities = append(identities, p.Identity())
		return true
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}

	if len(identities) != 2 {
		t.Errorf("Each() iterated %d processes, want 2", len(identities))
	}
}

func TestProcessSet_Get(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddProcess(fixtures.SampleProcess("worker1:1234:abc"))

	ctx := context.Background()
	processSet := NewProcessSet(client)

	proc, err := processSet.Get(ctx, "worker1:1234:abc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if proc.Identity() != "worker1:1234:abc" {
		t.Errorf("Identity() = %q, want %q", proc.Identity(), "worker1:1234:abc")
	}
	if proc.Hostname() != "worker-1" {
		t.Errorf("Hostname() = %q, want %q", proc.Hostname(), "worker-1")
	}
	if proc.PID() != 12345 {
		t.Errorf("PID() = %d, want 12345", proc.PID())
	}
	if proc.Concurrency() != 10 {
		t.Errorf("Concurrency() = %d, want 10", proc.Concurrency())
	}
	if proc.Version() != "7.0.0" {
		t.Errorf("Version() = %q, want %q", proc.Version(), "7.0.0")
	}
	if proc.Busy() != 3 {
		t.Errorf("Busy() = %d, want 3", proc.Busy())
	}
}

func TestProcessSet_Get_NotFound(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()
	processSet := NewProcessSet(client)

	_, err := processSet.Get(ctx, "nonexistent")
	if err != ErrProcessNotFound {
		t.Errorf("Get() error = %v, want ErrProcessNotFound", err)
	}
}

func TestProcessSet_TotalConcurrency(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	proc1 := fixtures.SampleProcess("w1:1:a")
	proc1.Concurrency = 10
	fixtures.AddProcess(proc1)

	proc2 := fixtures.SampleProcess("w2:2:b")
	proc2.Concurrency = 25
	fixtures.AddProcess(proc2)

	ctx := context.Background()
	processSet := NewProcessSet(client)

	total, err := processSet.TotalConcurrency(ctx)
	if err != nil {
		t.Fatalf("TotalConcurrency() error = %v", err)
	}
	if total != 35 {
		t.Errorf("TotalConcurrency() = %d, want 35", total)
	}
}

func TestProcess_Queues(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddProcess(fixtures.SampleProcess("worker1:1234:abc"))

	ctx := context.Background()
	processSet := NewProcessSet(client)

	proc, _ := processSet.Get(ctx, "worker1:1234:abc")
	queues := proc.Queues()

	if len(queues) != 2 {
		t.Errorf("Queues() length = %d, want 2", len(queues))
	}

	expected := map[string]bool{"default": true, "critical": true}
	for _, q := range queues {
		if !expected[q] {
			t.Errorf("unexpected queue %q", q)
		}
	}
}

func TestProcess_Labels(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddProcess(fixtures.SampleProcess("worker1:1234:abc"))

	ctx := context.Background()
	processSet := NewProcessSet(client)

	proc, _ := processSet.Get(ctx, "worker1:1234:abc")
	labels := proc.Labels()

	if len(labels) != 1 {
		t.Errorf("Labels() length = %d, want 1", len(labels))
	}
	if labels[0] != "production" {
		t.Errorf("Labels()[0] = %q, want %q", labels[0], "production")
	}
}
