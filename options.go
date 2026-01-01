package sidekiq

// Option configures a Client.
type Option func(*Client)

// WithNamespace sets the Redis key namespace prefix.
// This is compatible with the sidekiq-namespace Ruby gem.
//
// Example:
//
//	client := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))
//	// Keys will be prefixed: "myapp:queue:default", "myapp:stat:processed", etc.
func WithNamespace(ns string) Option {
	return func(c *Client) {
		c.namespace = ns
	}
}

// WithLogger sets a custom logger for the client.
// If not set, no logging is performed.
func WithLogger(l Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// Logger is the interface for pluggable logging.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}
