package sidekiq

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupTestClient creates a test client with miniredis.
func setupTestClient(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		rdb.Close()
	})

	client := NewClient(rdb)
	return client, mr
}

// setupTestClientWithNamespace creates a test client with a namespace.
func setupTestClientWithNamespace(t *testing.T, namespace string) (*Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		rdb.Close()
	})

	client := NewClient(rdb, WithNamespace(namespace))
	return client, mr
}

func TestNewClient(t *testing.T) {
	client, _ := setupTestClient(t)

	if client == nil {
		t.Fatal("expected client to not be nil")
	}
	if client.Redis() == nil {
		t.Fatal("expected Redis client to not be nil")
	}
}

func TestClient_WithNamespace(t *testing.T) {
	client, _ := setupTestClientWithNamespace(t, "myapp")

	if client.Namespace() != "myapp" {
		t.Errorf("Namespace() = %q, want %q", client.Namespace(), "myapp")
	}
}

func TestClient_Ping(t *testing.T) {
	client, _ := setupTestClient(t)
	ctx := context.Background()

	err := client.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestClient_Key(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		parts     []string
		want      string
	}{
		{
			name:      "no namespace",
			namespace: "",
			parts:     []string{"queue", "default"},
			want:      "queue:default",
		},
		{
			name:      "with namespace",
			namespace: "myapp",
			parts:     []string{"queue", "default"},
			want:      "myapp:queue:default",
		},
		{
			name:      "single part",
			namespace: "",
			parts:     []string{"queues"},
			want:      "queues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			defer rdb.Close()

			var client *Client
			if tt.namespace != "" {
				client = NewClient(rdb, WithNamespace(tt.namespace))
			} else {
				client = NewClient(rdb)
			}

			got := client.key(tt.parts...)
			if got != tt.want {
				t.Errorf("key(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}
