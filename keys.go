package sidekiq

import "strings"

// Redis key names used by Sidekiq.
const (
	keyQueues    = "queues"
	keyProcesses = "processes"
	keySchedule  = "schedule"
	keyRetry     = "retry"
	keyDead      = "dead"
)

// key builds a namespaced Redis key from the given parts.
func (c *Client) key(parts ...string) string {
	key := strings.Join(parts, ":")
	if c.namespace == "" {
		return key
	}
	return c.namespace + ":" + key
}

// queueKey returns the Redis key for a queue's job list.
func (c *Client) queueKey(name string) string {
	return c.key("queue", name)
}

// processKey returns the Redis key for a process's info hash.
func (c *Client) processKey(identity string) string {
	return c.key(identity)
}

// workKey returns the Redis key for a process's work hash.
func (c *Client) workKey(identity string) string {
	return c.key(identity, "work")
}

// statKey returns the Redis key for a stat counter.
func (c *Client) statKey(name string) string {
	return c.key("stat", name)
}

// pausedKey returns the Redis key for a queue's paused flag.
func (c *Client) pausedKey(name string) string {
	return c.key("paused", name)
}
