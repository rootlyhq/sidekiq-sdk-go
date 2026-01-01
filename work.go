package sidekiq

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	sjson "github.com/rootlyhq/sidekiq-sdk-go/internal/json"
)

// WorkSet represents all currently executing jobs across all processes.
type WorkSet struct {
	client *Client
}

// NewWorkSet creates a new WorkSet.
func NewWorkSet(client *Client) *WorkSet {
	return &WorkSet{client: client}
}

// Size returns the total number of executing jobs.
func (ws *WorkSet) Size(ctx context.Context) (int64, error) {
	identities, err := ws.client.rdb.SMembers(ctx, ws.client.key(keyProcesses)).Result()
	if err != nil {
		return 0, wrapRedisErr("get work count", err)
	}

	if len(identities) == 0 {
		return 0, nil
	}

	pipe := ws.client.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(identities))
	for i, identity := range identities {
		cmds[i] = pipe.HLen(ctx, ws.client.workKey(identity))
	}
	_, _ = pipe.Exec(ctx)

	var total int64
	for _, cmd := range cmds {
		total += cmd.Val()
	}
	return total, nil
}

// Each iterates over all executing jobs.
func (ws *WorkSet) Each(ctx context.Context, fn func(*Work) bool) error {
	identities, err := ws.client.rdb.SMembers(ctx, ws.client.key(keyProcesses)).Result()
	if err != nil {
		return wrapRedisErr("list work", err)
	}

	for _, identity := range identities {
		work, err := ws.client.rdb.HGetAll(ctx, ws.client.workKey(identity)).Result()
		if err != nil {
			continue
		}

		for tid, data := range work {
			w, err := newWork(ws.client, identity, tid, data)
			if err != nil {
				continue
			}
			if !fn(w) {
				return nil
			}
		}
	}

	return nil
}

// FindWork finds a job by JID across all processes.
func (ws *WorkSet) FindWork(ctx context.Context, jid string) (*Work, error) {
	var found *Work
	err := ws.Each(ctx, func(w *Work) bool {
		if w.JID() == jid {
			found = w
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// Work represents a currently executing job.
type Work struct {
	client   *Client
	process  string
	tid      string
	info     map[string]interface{}
	payload  map[string]interface{}
	rawValue string
}

// newWork creates a new Work from raw data.
func newWork(client *Client, process, tid, data string) (*Work, error) {
	info, err := sjson.Parse(data)
	if err != nil {
		return nil, &JobParseError{Err: err}
	}

	var payload map[string]interface{}
	if p := sjson.GetMap(info, "payload"); p != nil {
		payload = p
	}

	return &Work{
		client:   client,
		process:  process,
		tid:      tid,
		info:     info,
		payload:  payload,
		rawValue: data,
	}, nil
}

// Process returns the process identity executing this job.
func (w *Work) Process() string {
	return w.process
}

// TID returns the thread ID executing this job.
func (w *Work) TID() string {
	return w.tid
}

// Queue returns the queue this job came from.
func (w *Work) Queue() string {
	return sjson.GetString(w.info, "queue")
}

// RunAt returns when this job started executing.
func (w *Work) RunAt() time.Time {
	return sjson.GetTime(w.info, "run_at")
}

// JID returns the job ID.
func (w *Work) JID() string {
	if w.payload != nil {
		return sjson.GetString(w.payload, "jid")
	}
	return ""
}

// Job returns the JobRecord for this work.
func (w *Work) Job() *JobRecord {
	if w.payload == nil {
		return nil
	}
	data, err := sjson.Marshal(w.payload)
	if err != nil {
		return nil
	}
	job, err := newJobRecord(data, w.Queue())
	if err != nil {
		return nil
	}
	return job
}

// Payload returns the raw job payload.
func (w *Work) Payload() map[string]interface{} {
	return w.payload
}

// Info returns the raw work info.
func (w *Work) Info() map[string]interface{} {
	return w.info
}
