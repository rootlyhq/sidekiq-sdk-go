package sidekiq

import (
	"context"
	"testing"
	"time"

	"github.com/rootlyhq/sidekiq-sdk-go/internal/testdata"
)

func TestScheduledSet_Basic(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	future := time.Now().Add(time.Hour)
	fixtures.AddScheduledJob(future, fixtures.SampleJob("sched1", "Worker1"))
	fixtures.AddScheduledJob(future.Add(time.Minute), fixtures.SampleJob("sched2", "Worker2"))

	ctx := context.Background()
	set := NewScheduledSet(client)

	size, err := set.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 2 {
		t.Errorf("Size() = %d, want 2", size)
	}
}

func TestRetrySet_FindJob(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddRetryJob(time.Now().Add(time.Minute),
		fixtures.SampleFailedJob("retry-abc", "FailWorker", "RuntimeError", "failed"),
	)
	fixtures.AddRetryJob(time.Now().Add(2*time.Minute),
		fixtures.SampleFailedJob("retry-xyz", "OtherWorker", "IOError", "timeout"),
	)

	ctx := context.Background()
	retrySet := NewRetrySet(client)

	entry, err := retrySet.FindJob(ctx, "retry-abc")
	if err != nil {
		t.Fatalf("FindJob() error = %v", err)
	}
	if entry == nil {
		t.Fatal("FindJob() returned nil")
	}
	if entry.JID() != "retry-abc" {
		t.Errorf("JID() = %q, want %q", entry.JID(), "retry-abc")
	}
	if entry.ErrorClass() != "RuntimeError" {
		t.Errorf("ErrorClass() = %q, want %q", entry.ErrorClass(), "RuntimeError")
	}
}

func TestRetrySet_Each(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	now := time.Now()
	fixtures.AddRetryJob(now.Add(time.Minute),
		fixtures.SampleFailedJob("r1", "W", "E", "m1"),
	)
	fixtures.AddRetryJob(now.Add(2*time.Minute),
		fixtures.SampleFailedJob("r2", "W", "E", "m2"),
	)
	fixtures.AddRetryJob(now.Add(3*time.Minute),
		fixtures.SampleFailedJob("r3", "W", "E", "m3"),
	)

	ctx := context.Background()
	retrySet := NewRetrySet(client)

	var jids []string
	err := retrySet.Each(ctx, func(entry *SortedEntry) bool {
		jids = append(jids, entry.JID())
		return true
	})
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}

	if len(jids) != 3 {
		t.Errorf("Each() iterated %d entries, want 3", len(jids))
	}
}

func TestDeadSet_Basic(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	now := time.Now()
	fixtures.AddDeadJob(now,
		fixtures.SampleFailedJob("dead1", "Worker", "Error", "msg"),
	)

	ctx := context.Background()
	deadSet := NewDeadSet(client)

	size, err := deadSet.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 1 {
		t.Errorf("Size() = %d, want 1", size)
	}
}

func TestSortedSet_Clear(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	fixtures.AddRetryJob(time.Now().Add(time.Minute),
		fixtures.SampleFailedJob("r1", "W", "E", "m"),
	)
	fixtures.AddRetryJob(time.Now().Add(2*time.Minute),
		fixtures.SampleFailedJob("r2", "W", "E", "m"),
	)

	ctx := context.Background()
	retrySet := NewRetrySet(client)

	deleted, err := retrySet.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if deleted != 2 {
		t.Errorf("Clear() = %d, want 2", deleted)
	}

	size, _ := retrySet.Size(ctx)
	if size != 0 {
		t.Errorf("after Clear(), Size() = %d, want 0", size)
	}
}

func TestSortedEntry_At(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	scheduledTime := time.Now().Add(time.Hour).Truncate(time.Second)
	fixtures.AddScheduledJob(scheduledTime, fixtures.SampleJob("s1", "W"))

	ctx := context.Background()
	set := NewScheduledSet(client)

	var entry *SortedEntry
	_ = set.Each(ctx, func(e *SortedEntry) bool {
		entry = e
		return false
	})

	if entry == nil {
		t.Fatal("no entry found")
	}

	at := entry.At().Truncate(time.Second)
	if !at.Equal(scheduledTime) {
		t.Errorf("At() = %v, want %v", at, scheduledTime)
	}
}

func TestSortedEntry_HasError(t *testing.T) {
	client, mr := setupTestClient(t)
	fixtures := testdata.NewFixtures(mr, "")

	// Job without error
	fixtures.AddScheduledJob(time.Now().Add(time.Hour),
		fixtures.SampleJob("s1", "Worker"),
	)

	// Job with error
	fixtures.AddRetryJob(time.Now().Add(time.Minute),
		fixtures.SampleFailedJob("r1", "Worker", "Error", "msg"),
	)

	ctx := context.Background()

	// Check scheduled job (no error)
	schedSet := NewScheduledSet(client)
	var schedEntry *SortedEntry
	_ = schedSet.Each(ctx, func(e *SortedEntry) bool {
		schedEntry = e
		return false
	})
	if schedEntry.HasError() {
		t.Error("scheduled entry HasError() = true, want false")
	}

	// Check retry job (has error)
	retrySet := NewRetrySet(client)
	var retryEntry *SortedEntry
	_ = retrySet.Each(ctx, func(e *SortedEntry) bool {
		retryEntry = e
		return false
	})
	if !retryEntry.HasError() {
		t.Error("retry entry HasError() = false, want true")
	}
}
