// Package testdata provides test fixtures for Sidekiq data structures.
package testdata

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// SidekiqFixtures provides helper methods to populate miniredis with Sidekiq data.
type SidekiqFixtures struct {
	mr        *miniredis.Miniredis
	namespace string
}

// NewFixtures creates a new SidekiqFixtures instance.
func NewFixtures(mr *miniredis.Miniredis, namespace string) *SidekiqFixtures {
	return &SidekiqFixtures{mr: mr, namespace: namespace}
}

func (f *SidekiqFixtures) key(parts ...string) string {
	key := strings.Join(parts, ":")
	if f.namespace == "" {
		return key
	}
	return f.namespace + ":" + key
}

// JobPayload represents a Sidekiq job payload.
type JobPayload struct {
	Class        string        `json:"class"`
	Args         []interface{} `json:"args"`
	JID          string        `json:"jid"`
	CreatedAt    float64       `json:"created_at"`
	EnqueuedAt   float64       `json:"enqueued_at"`
	Queue        string        `json:"queue,omitempty"`
	Retry        interface{}   `json:"retry,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	ErrorClass   string        `json:"error_class,omitempty"`
	FailedAt     float64       `json:"failed_at,omitempty"`
	RetryCount   int           `json:"retry_count,omitempty"`
}

// ActiveJobPayload represents an ActiveJob wrapper payload.
type ActiveJobPayload struct {
	Class   string `json:"class"`
	Wrapped string `json:"wrapped"`
	Queue   string `json:"queue"`
	Args    []struct {
		JobClass  string        `json:"job_class"`
		Arguments []interface{} `json:"arguments"`
	} `json:"args"`
	JID        string  `json:"jid"`
	CreatedAt  float64 `json:"created_at"`
	EnqueuedAt float64 `json:"enqueued_at"`
}

// ProcessInfo represents a Sidekiq process.
type ProcessInfo struct {
	Hostname    string   `json:"hostname"`
	StartedAt   float64  `json:"started_at"`
	PID         int      `json:"pid"`
	Tag         string   `json:"tag"`
	Concurrency int      `json:"concurrency"`
	Queues      []string `json:"queues"`
	Labels      []string `json:"labels"`
	Identity    string   `json:"identity"`
	Version     string   `json:"version"`
	Busy        int      `json:"busy"`
}

// WorkInfo represents work being executed.
type WorkInfo struct {
	Queue   string                 `json:"queue"`
	RunAt   float64                `json:"run_at"`
	Payload map[string]interface{} `json:"payload"`
}

// SetupStats populates stat counters.
func (f *SidekiqFixtures) SetupStats(processed, failed int64) {
	f.mr.Set(f.key("stat", "processed"), fmt.Sprintf("%d", processed))
	f.mr.Set(f.key("stat", "failed"), fmt.Sprintf("%d", failed))
}

// SetupDailyStats populates daily stat counters.
func (f *SidekiqFixtures) SetupDailyStats(date time.Time, processed, failed int64) {
	dateStr := date.Format("2006-01-02")
	f.mr.Set(f.key("stat", "processed:"+dateStr), fmt.Sprintf("%d", processed))
	f.mr.Set(f.key("stat", "failed:"+dateStr), fmt.Sprintf("%d", failed))
}

// AddQueue creates a queue with jobs.
func (f *SidekiqFixtures) AddQueue(name string, jobs ...JobPayload) {
	f.mr.SAdd(f.key("queues"), name)

	queueKey := f.key("queue", name)
	for i := len(jobs) - 1; i >= 0; i-- {
		job := jobs[i]
		if job.Queue == "" {
			job.Queue = name
		}
		data, _ := json.Marshal(job)
		f.mr.Lpush(queueKey, string(data))
	}
}

// AddScheduledJob adds a job to the schedule set.
func (f *SidekiqFixtures) AddScheduledJob(at time.Time, job JobPayload) {
	data, _ := json.Marshal(job)
	f.mr.ZAdd(f.key("schedule"), float64(at.Unix()), string(data))
}

// AddRetryJob adds a job to the retry set.
func (f *SidekiqFixtures) AddRetryJob(at time.Time, job JobPayload) {
	data, _ := json.Marshal(job)
	f.mr.ZAdd(f.key("retry"), float64(at.Unix()), string(data))
}

// AddDeadJob adds a job to the dead set.
func (f *SidekiqFixtures) AddDeadJob(at time.Time, job JobPayload) {
	data, _ := json.Marshal(job)
	f.mr.ZAdd(f.key("dead"), float64(at.Unix()), string(data))
}

// AddProcess registers a Sidekiq process.
func (f *SidekiqFixtures) AddProcess(info ProcessInfo) {
	f.mr.SAdd(f.key("processes"), info.Identity)
	data, _ := json.Marshal(info)
	f.mr.Set(f.key(info.Identity), string(data))
}

// AddWork registers work being done by a process.
func (f *SidekiqFixtures) AddWork(identity, tid string, work WorkInfo) {
	data, _ := json.Marshal(work)
	f.mr.HSet(f.key(identity, "work"), tid, string(data))
}

// PauseQueue marks a queue as paused.
func (f *SidekiqFixtures) PauseQueue(name string) {
	f.mr.Set(f.key("paused", name), "true")
}

// SampleJob creates a sample job payload.
func (f *SidekiqFixtures) SampleJob(jid, class string, args ...interface{}) JobPayload {
	now := float64(time.Now().Unix())
	if args == nil {
		args = []interface{}{}
	}
	return JobPayload{
		Class:      class,
		Args:       args,
		JID:        jid,
		CreatedAt:  now,
		EnqueuedAt: now,
		Retry:      true,
	}
}

// SampleFailedJob creates a sample failed job payload.
func (f *SidekiqFixtures) SampleFailedJob(jid, class, errClass, errMsg string) JobPayload {
	job := f.SampleJob(jid, class)
	job.ErrorClass = errClass
	job.ErrorMessage = errMsg
	job.FailedAt = float64(time.Now().Unix())
	job.RetryCount = 1
	return job
}

// SampleProcess creates a sample process info.
func (f *SidekiqFixtures) SampleProcess(identity string) ProcessInfo {
	return ProcessInfo{
		Hostname:    "worker-1",
		StartedAt:   float64(time.Now().Add(-time.Hour).Unix()),
		PID:         12345,
		Tag:         "default",
		Concurrency: 10,
		Queues:      []string{"default", "critical"},
		Labels:      []string{"production"},
		Identity:    identity,
		Version:     "7.0.0",
		Busy:        3,
	}
}

// SampleActiveJob creates a sample ActiveJob wrapper payload.
func (f *SidekiqFixtures) SampleActiveJob(jid, wrappedClass string, args ...interface{}) ActiveJobPayload {
	now := float64(time.Now().Unix())
	if args == nil {
		args = []interface{}{}
	}
	return ActiveJobPayload{
		Class:   "ActiveJob::QueueAdapters::SidekiqAdapter::JobWrapper",
		Wrapped: wrappedClass,
		Queue:   "default",
		Args: []struct {
			JobClass  string        `json:"job_class"`
			Arguments []interface{} `json:"arguments"`
		}{
			{JobClass: wrappedClass, Arguments: args},
		},
		JID:        jid,
		CreatedAt:  now,
		EnqueuedAt: now,
	}
}
