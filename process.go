package sidekiq

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	sjson "github.com/rootlyhq/sidekiq-sdk-go/internal/json"
)

// ProcessSet represents the set of active Sidekiq processes.
type ProcessSet struct {
	client *Client
}

// NewProcessSet creates a new ProcessSet.
func NewProcessSet(client *Client) *ProcessSet {
	return &ProcessSet{client: client}
}

// Size returns the number of active processes.
func (ps *ProcessSet) Size(ctx context.Context) (int64, error) {
	size, err := ps.client.rdb.SCard(ctx, ps.client.key(keyProcesses)).Result()
	if err != nil {
		return 0, wrapRedisErr("get process count", err)
	}
	return size, nil
}

// Each iterates over all active processes.
func (ps *ProcessSet) Each(ctx context.Context, fn func(*Process) bool) error {
	identities, err := ps.client.rdb.SMembers(ctx, ps.client.key(keyProcesses)).Result()
	if err != nil {
		return wrapRedisErr("list processes", err)
	}

	if len(identities) == 0 {
		return nil
	}

	// Fetch all process info in parallel
	pipe := ps.client.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(identities))
	for i, identity := range identities {
		cmds[i] = pipe.Get(ctx, ps.client.processKey(identity))
	}
	_, _ = pipe.Exec(ctx)

	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue // Process info expired
		}
		if err != nil {
			continue
		}

		proc, err := newProcess(ps.client, identities[i], data)
		if err != nil {
			continue
		}

		if !fn(proc) {
			return nil
		}
	}

	return nil
}

// Get retrieves a specific process by identity.
func (ps *ProcessSet) Get(ctx context.Context, identity string) (*Process, error) {
	data, err := ps.client.rdb.Get(ctx, ps.client.processKey(identity)).Result()
	if err == redis.Nil {
		return nil, ErrProcessNotFound
	}
	if err != nil {
		return nil, wrapRedisErr("get process", err)
	}

	return newProcess(ps.client, identity, data)
}

// Cleanup removes stale process entries.
// Returns the number of processes removed.
func (ps *ProcessSet) Cleanup(ctx context.Context) (int64, error) {
	identities, err := ps.client.rdb.SMembers(ctx, ps.client.key(keyProcesses)).Result()
	if err != nil {
		return 0, wrapRedisErr("cleanup processes", err)
	}

	var removed int64
	for _, identity := range identities {
		exists, err := ps.client.rdb.Exists(ctx, ps.client.processKey(identity)).Result()
		if err != nil {
			continue
		}
		if exists == 0 {
			ps.client.rdb.SRem(ctx, ps.client.key(keyProcesses), identity)
			removed++
		}
	}

	return removed, nil
}

// TotalConcurrency returns the sum of concurrency across all processes.
func (ps *ProcessSet) TotalConcurrency(ctx context.Context) (int64, error) {
	var total int64
	err := ps.Each(ctx, func(p *Process) bool {
		total += int64(p.Concurrency())
		return true
	})
	return total, err
}

// TotalRSSKB returns the sum of RSS memory across all processes.
func (ps *ProcessSet) TotalRSSKB(ctx context.Context) (int64, error) {
	var total int64
	err := ps.Each(ctx, func(p *Process) bool {
		total += p.RSS()
		return true
	})
	return total, err
}

// Leader returns the identity of the current leader process.
func (ps *ProcessSet) Leader(ctx context.Context) (string, error) {
	leader, err := ps.client.rdb.Get(ctx, ps.client.key("leader")).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", wrapRedisErr("get leader", err)
	}
	return leader, nil
}

// Process represents an active Sidekiq process.
type Process struct {
	client   *Client
	identity string
	info     map[string]interface{}
}

// newProcess creates a new Process from raw data.
func newProcess(client *Client, identity, data string) (*Process, error) {
	info, err := sjson.Parse(data)
	if err != nil {
		return nil, &JobParseError{Err: err}
	}
	return &Process{
		client:   client,
		identity: identity,
		info:     info,
	}, nil
}

// Identity returns the unique process identifier.
func (p *Process) Identity() string {
	return p.identity
}

// Tag returns the process tag.
func (p *Process) Tag() string {
	return sjson.GetString(p.info, "tag")
}

// Labels returns the process labels.
func (p *Process) Labels() []string {
	return sjson.GetStringSlice(p.info, "labels")
}

// Hostname returns the hostname where the process is running.
func (p *Process) Hostname() string {
	return sjson.GetString(p.info, "hostname")
}

// PID returns the process ID.
func (p *Process) PID() int {
	return int(sjson.GetInt64(p.info, "pid"))
}

// StartedAt returns when the process started.
func (p *Process) StartedAt() time.Time {
	return sjson.GetTime(p.info, "started_at")
}

// Queues returns the list of queues this process is working on.
func (p *Process) Queues() []string {
	return sjson.GetStringSlice(p.info, "queues")
}

// CapsuleConfig represents a Sidekiq 7 capsule configuration.
type CapsuleConfig struct {
	Concurrency int
	Queues      []string
}

// Capsules returns the capsule configurations (Sidekiq 7+).
func (p *Process) Capsules() map[string]CapsuleConfig {
	capsules := sjson.GetMap(p.info, "config")
	if capsules == nil {
		return nil
	}

	result := make(map[string]CapsuleConfig)
	for name, v := range capsules {
		if cfg, ok := v.(map[string]interface{}); ok {
			result[name] = CapsuleConfig{
				Concurrency: int(sjson.GetInt64(cfg, "concurrency")),
				Queues:      sjson.GetStringSlice(cfg, "queues"),
			}
		}
	}
	return result
}

// Concurrency returns the total concurrency of the process.
func (p *Process) Concurrency() int {
	return int(sjson.GetInt64(p.info, "concurrency"))
}

// Version returns the Sidekiq version.
func (p *Process) Version() string {
	return sjson.GetString(p.info, "version")
}

// RSS returns the resident set size in KB.
func (p *Process) RSS() int64 {
	return sjson.GetInt64(p.info, "rss")
}

// Busy returns the number of busy threads.
func (p *Process) Busy() int {
	return int(sjson.GetInt64(p.info, "busy"))
}

// Stopping returns true if the process is shutting down.
func (p *Process) Stopping() bool {
	return sjson.GetBool(p.info, "quiet")
}

// Quiet signals the process to stop fetching new work.
func (p *Process) Quiet(ctx context.Context) error {
	return p.signal(ctx, "TSTP")
}

// Stop signals the process to terminate.
func (p *Process) Stop(ctx context.Context) error {
	return p.signal(ctx, "TERM")
}

// DumpThreads signals the process to dump thread backtraces.
func (p *Process) DumpThreads(ctx context.Context) error {
	return p.signal(ctx, "TTIN")
}

// signal sends a signal to the process via Redis pub/sub.
func (p *Process) signal(ctx context.Context, sig string) error {
	channel := p.client.key(p.identity, "-signals")
	err := p.client.rdb.Publish(ctx, channel, sig).Err()
	return wrapRedisErr("signal process", err)
}

// Info returns the raw process info map.
func (p *Process) Info() map[string]interface{} {
	return p.info
}
