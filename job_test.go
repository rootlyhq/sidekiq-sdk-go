package sidekiq

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJobRecord_Basic(t *testing.T) {
	now := time.Now()
	job := map[string]interface{}{
		"class":       "MyWorker",
		"args":        []interface{}{"arg1", 42},
		"jid":         "abc123",
		"queue":       "default",
		"created_at":  float64(now.Unix()),
		"enqueued_at": float64(now.Unix()),
		"retry":       true,
		"tags":        []interface{}{"important", "batch-1"},
	}
	data, _ := json.Marshal(job)

	record, err := newJobRecord(string(data), "default")
	if err != nil {
		t.Fatalf("newJobRecord() error = %v", err)
	}

	if record.JID() != "abc123" {
		t.Errorf("JID() = %q, want %q", record.JID(), "abc123")
	}
	if record.Class() != "MyWorker" {
		t.Errorf("Class() = %q, want %q", record.Class(), "MyWorker")
	}
	if record.Queue() != "default" {
		t.Errorf("Queue() = %q, want %q", record.Queue(), "default")
	}

	args := record.Args()
	if len(args) != 2 {
		t.Errorf("Args() length = %d, want 2", len(args))
	}

	tags := record.Tags()
	if len(tags) != 2 {
		t.Errorf("Tags() length = %d, want 2", len(tags))
	}
}

func TestJobRecord_ActiveJob(t *testing.T) {
	now := time.Now()
	job := map[string]interface{}{
		"class":   "Sidekiq::ActiveJob::Wrapper",
		"wrapped": "MyActiveJob",
		"args": []interface{}{
			map[string]interface{}{
				"job_class": "MyActiveJob",
				"arguments": []interface{}{"real_arg1", "real_arg2"},
			},
		},
		"jid":         "xyz789",
		"queue":       "default",
		"created_at":  float64(now.Unix()),
		"enqueued_at": float64(now.Unix()),
	}
	data, _ := json.Marshal(job)

	record, err := newJobRecord(string(data), "default")
	if err != nil {
		t.Fatalf("newJobRecord() error = %v", err)
	}

	// Raw class is the wrapper
	if record.Class() != "Sidekiq::ActiveJob::Wrapper" {
		t.Errorf("Class() = %q, want wrapper class", record.Class())
	}

	// DisplayClass unwraps
	if record.DisplayClass() != "MyActiveJob" {
		t.Errorf("DisplayClass() = %q, want %q", record.DisplayClass(), "MyActiveJob")
	}

	// DisplayArgs unwraps
	displayArgs := record.DisplayArgs()
	if len(displayArgs) != 2 {
		t.Errorf("DisplayArgs() length = %d, want 2", len(displayArgs))
	}
}

func TestJobRecord_FailedJob(t *testing.T) {
	now := time.Now()
	job := map[string]interface{}{
		"class":         "FailingWorker",
		"args":          []interface{}{},
		"jid":           "fail123",
		"queue":         "default",
		"created_at":    float64(now.Unix()),
		"enqueued_at":   float64(now.Unix()),
		"failed_at":     float64(now.Unix()),
		"error_class":   "RuntimeError",
		"error_message": "Something went wrong",
		"retry_count":   3,
	}
	data, _ := json.Marshal(job)

	record, err := newJobRecord(string(data), "default")
	if err != nil {
		t.Fatalf("newJobRecord() error = %v", err)
	}

	if record.ErrorClass() != "RuntimeError" {
		t.Errorf("ErrorClass() = %q, want %q", record.ErrorClass(), "RuntimeError")
	}
	if record.ErrorMessage() != "Something went wrong" {
		t.Errorf("ErrorMessage() = %q, want %q", record.ErrorMessage(), "Something went wrong")
	}
	if record.RetryCount() != 3 {
		t.Errorf("RetryCount() = %d, want 3", record.RetryCount())
	}
	if record.FailedAt().IsZero() {
		t.Error("FailedAt() is zero, want non-zero")
	}
}

func TestJobRecord_InvalidJSON(t *testing.T) {
	_, err := newJobRecord("invalid json", "default")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJobRecord_Get(t *testing.T) {
	job := map[string]interface{}{
		"class":         "Worker",
		"args":          []interface{}{},
		"jid":           "get123",
		"custom_field":  "custom_value",
		"custom_number": float64(42),
	}
	data, _ := json.Marshal(job)

	record, err := newJobRecord(string(data), "")
	if err != nil {
		t.Fatalf("newJobRecord() error = %v", err)
	}

	if v := record.Get("custom_field"); v != "custom_value" {
		t.Errorf("Get(custom_field) = %v, want %q", v, "custom_value")
	}
	if v := record.Get("custom_number"); v != float64(42) {
		t.Errorf("Get(custom_number) = %v, want 42", v)
	}
	if v := record.Get("nonexistent"); v != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", v)
	}
}
