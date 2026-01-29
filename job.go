package sidekiq

import (
	"time"

	sjson "github.com/rootlyhq/sidekiq-sdk-go/internal/json"
)

// ActiveJob wrapper class name
const activeJobWrapperClass = "Sidekiq::ActiveJob::Wrapper"

// JobRecord represents an immutable Sidekiq job.
// It provides accessors for all standard job fields.
type JobRecord struct {
	item  map[string]interface{}
	value string
	queue string
}

// newJobRecord creates a new JobRecord from raw JSON data.
func newJobRecord(value string, queue string) (*JobRecord, error) {
	item, err := sjson.Parse(value)
	if err != nil {
		return nil, &JobParseError{Err: err}
	}
	return &JobRecord{
		item:  item,
		value: value,
		queue: queue,
	}, nil
}

// JID returns the unique job identifier.
func (j *JobRecord) JID() string {
	return sjson.GetString(j.item, "jid")
}

// Queue returns the queue name this job belongs to.
func (j *JobRecord) Queue() string {
	if j.queue != "" {
		return j.queue
	}
	return sjson.GetString(j.item, "queue")
}

// Class returns the worker class name.
func (j *JobRecord) Class() string {
	return sjson.GetString(j.item, "class")
}

// DisplayClass returns the worker class name, unwrapping ActiveJob if present.
func (j *JobRecord) DisplayClass() string {
	if j.Class() == activeJobWrapperClass {
		if wrapped := sjson.GetString(j.item, "wrapped"); wrapped != "" {
			return wrapped
		}
	}
	return j.Class()
}

// Args returns the job arguments.
func (j *JobRecord) Args() []interface{} {
	return sjson.GetSlice(j.item, "args")
}

// DisplayArgs returns the job arguments, unwrapping ActiveJob if present.
func (j *JobRecord) DisplayArgs() []interface{} {
	if j.Class() == activeJobWrapperClass {
		args := j.Args()
		if len(args) > 0 {
			if argMap, ok := args[0].(map[string]interface{}); ok {
				if arguments := sjson.GetSlice(argMap, "arguments"); arguments != nil {
					return arguments
				}
			}
		}
	}
	return j.Args()
}

// CreatedAt returns when the job was created.
func (j *JobRecord) CreatedAt() time.Time {
	return sjson.GetTime(j.item, "created_at")
}

// EnqueuedAt returns when the job was enqueued.
func (j *JobRecord) EnqueuedAt() time.Time {
	return sjson.GetTime(j.item, "enqueued_at")
}

// FailedAt returns when the job last failed, or zero time if never failed.
func (j *JobRecord) FailedAt() time.Time {
	return sjson.GetTime(j.item, "failed_at")
}

// RetriedAt returns when the job was last retried, or zero time if never retried.
func (j *JobRecord) RetriedAt() time.Time {
	return sjson.GetTime(j.item, "retried_at")
}

// RetryCount returns the number of times this job has been retried.
func (j *JobRecord) RetryCount() int {
	return int(sjson.GetInt64(j.item, "retry_count"))
}

// Tags returns the job's tags.
func (j *JobRecord) Tags() []string {
	return sjson.GetStringSlice(j.item, "tags")
}

// Bid returns the batch ID if this job is part of a batch.
func (j *JobRecord) Bid() string {
	return sjson.GetString(j.item, "bid")
}

// ErrorMessage returns the last error message, or empty string if no error.
func (j *JobRecord) ErrorMessage() string {
	return sjson.GetString(j.item, "error_message")
}

// ErrorClass returns the last error class name, or empty string if no error.
func (j *JobRecord) ErrorClass() string {
	return sjson.GetString(j.item, "error_class")
}

// ErrorBacktrace returns the decompressed backtrace if available.
func (j *JobRecord) ErrorBacktrace() ([]string, error) {
	compressed := sjson.GetString(j.item, "compressed_backtrace")
	if compressed != "" {
		return sjson.DecompressBacktrace(compressed)
	}

	// Try uncompressed backtrace
	return sjson.GetStringSlice(j.item, "backtrace"), nil
}

// Latency returns the time in seconds between when the job was enqueued and now.
func (j *JobRecord) Latency() float64 {
	enqueuedAt := j.EnqueuedAt()
	if enqueuedAt.IsZero() {
		return 0
	}
	return time.Since(enqueuedAt).Seconds()
}

// Get returns a raw value from the job's data by key.
func (j *JobRecord) Get(key string) interface{} {
	return j.item[key]
}

// Item returns the raw job data map.
func (j *JobRecord) Item() map[string]interface{} {
	return j.item
}

// Value returns the raw JSON string of the job.
func (j *JobRecord) Value() string {
	return j.value
}
