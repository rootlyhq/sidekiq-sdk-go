package sidekiq

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Client is the main entry point for interacting with Sidekiq's Redis data.
// It holds the Redis connection and configuration options.
//
// Create a new client with NewClient:
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	client := sidekiq.NewClient(rdb)
type Client struct {
	rdb       redis.UniversalClient
	namespace string
	logger    Logger
}

// NewClient creates a new Sidekiq client with the given Redis connection.
//
// Example:
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	client := sidekiq.NewClient(rdb)
//
// With namespace:
//
//	client := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))
func NewClient(rdb redis.UniversalClient, opts ...Option) *Client {
	c := &Client{
		rdb:    rdb,
		logger: noopLogger{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Redis returns the underlying Redis client.
// This can be used for advanced operations not covered by this SDK.
func (c *Client) Redis() redis.UniversalClient {
	return c.rdb
}

// Namespace returns the configured namespace prefix, or empty string if none.
func (c *Client) Namespace() string {
	return c.namespace
}

// Ping verifies the Redis connection is working.
func (c *Client) Ping(ctx context.Context) error {
	return wrapRedisErr("ping", c.rdb.Ping(ctx).Err())
}
