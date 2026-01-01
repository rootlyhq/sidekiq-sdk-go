# Sidekiq Go SDK - Implementation Plan

## Overview

This document outlines the plan to implement a Go SDK that provides the same API interface as Ruby's `sidekiq/api.rb`. This SDK will allow Go applications to interact with Sidekiq's Redis data structures for monitoring, administration, and job management.

**Goals:**
- Full API compatibility with Ruby's `Sidekiq::API`
- Idiomatic Go patterns (context, error handling, interfaces)
- Thread-safe and production-ready
- Support for Sidekiq 7.x+

---

## Repository Structure

```
github.com/rootlyhq/sidekiq-sdk-go-sdk/
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                      # CI pipeline (lint, test, build)
│   │   └── release.yml                 # Release automation with GoReleaser
│   └── dependabot.yml                  # Automated dependency updates
│
├── internal/
│   ├── json/
│   │   ├── json.go                     # JSON helpers, type coercion
│   │   └── json_test.go
│   └── testdata/
│       └── fixtures.go                 # Sidekiq Redis test fixtures
│
├── sidekiq.go                          # Client, configuration, public API
├── sidekiq_test.go
├── options.go                          # Functional options pattern
├── options_test.go
├── keys.go                             # Redis key constants and builders
├── keys_test.go
├── errors.go                           # Custom error types
│
├── stats.go                            # Stats and StatsHistory
├── stats_test.go
├── queue.go                            # Queue operations
├── queue_test.go
├── job.go                              # JobRecord (immutable job)
├── job_test.go
│
├── sorted_set.go                       # SortedSet and JobSet base types
├── sorted_set_test.go
├── sorted_entry.go                     # SortedEntry (job in sorted set)
├── sorted_entry_test.go
├── scheduled_set.go                    # ScheduledSet
├── scheduled_set_test.go
├── retry_set.go                        # RetrySet
├── retry_set_test.go
├── dead_set.go                         # DeadSet
├── dead_set_test.go
│
├── process.go                          # ProcessSet and Process
├── process_test.go
├── work.go                             # WorkSet and Work
├── work_test.go
│
├── enqueue.go                          # Job enqueuing
├── enqueue_test.go
├── scripts.go                          # Redis Lua scripts
├── scripts_test.go
│
├── doc.go                              # Package documentation
├── example_test.go                     # Testable examples for GoDoc
│
├── .editorconfig                       # Editor configuration
├── .gitignore                          # Git ignore rules
├── .golangci.yml                       # Linter configuration
├── .goreleaser.yaml                    # Release configuration
├── go.mod                              # Go module definition
├── go.sum                              # Dependency checksums
├── Makefile                            # Build automation
├── LICENSE                             # MIT License
├── README.md                           # User documentation
├── CLAUDE.md                           # Developer reference
└── CHANGELOG.md                        # Release history
```

---

## Configuration Files

### `.gitignore`

```gitignore
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, build with go test -c
*.test

# Output of the go coverage tool
*.out
coverage.html

# Go workspace
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Debug
debug.log

# Vendor (we use go modules)
vendor/
```

### `.editorconfig`

```editorconfig
root = true

[*]
charset = utf-8
end_of_line = lf
indent_style = tab
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab

[*.{yaml,yml}]
indent_style = space
indent_size = 2

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
```

### `.golangci.yml`

```yaml
run:
  timeout: 5m

output:
  formats:
    - format: colored-line-number

linters:
  enable:
    - bodyclose
    - gocritic
    - gocyclo
    - goimports
    - goprintffuncname
    - govet
    - ineffassign
    - misspell
    - nakedret
    - staticcheck
    - unconvert
    - unparam
    - unused
    - whitespace

linters-settings:
  gocyclo:
    min-complexity: 25
  goimports:
    local-prefixes: github.com/rootlyhq/sidekiq-sdk-go
  govet:
    enable-all: true
    disable:
      - fieldalignment

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gocyclo
        - unparam
    - path: internal/testdata/
      linters:
        - unused
```

### `.goreleaser.yaml`

```yaml
version: 2

project_name: sidekiq-go

builds:
  - skip: true  # Library only, no binary

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]

changelog:
  use: github
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
      - "^chore:"
  groups:
    - title: Features
      regexp: "^feat"
      order: 0
    - title: Bug Fixes
      regexp: "^fix"
      order: 1
    - title: Other
      order: 999

release:
  github:
    owner: rootlyhq
    name: sidekiq-go
  prerelease: auto
```

### `.github/dependabot.yml`

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
      day: monday
    open-pull-requests-limit: 5
    labels:
      - dependencies
      - go
    commit-message:
      prefix: "deps"

  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
      day: monday
    open-pull-requests-limit: 5
    labels:
      - dependencies
      - ci
    commit-message:
      prefix: "ci"
```

### `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    name: Test (${{ matrix.os }})
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-22.04, ubuntu-24.04, macos-latest, windows-latest]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Run tests
        run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
        if: matrix.os != 'windows-latest'

      - name: Run tests (Windows, no race)
        run: go test -coverprofile=coverage.out ./...
        if: matrix.os == 'windows-latest'

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        if: matrix.os == 'ubuntu-24.04'
        with:
          files: coverage.out
          token: ${{ secrets.CODECOV_TOKEN }}

  validate-tests-passed:
    name: All Tests Passed
    needs: [lint, test]
    runs-on: ubuntu-latest
    if: always()
    steps:
      - name: Check test results
        run: |
          if [[ "${{ needs.lint.result }}" != "success" || "${{ needs.test.result }}" != "success" ]]; then
            echo "Some tests failed"
            exit 1
          fi
          echo "All tests passed"
```

### `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    name: Release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### `Makefile`

```makefile
.PHONY: all build test lint fmt check clean coverage coverage-html version bump-patch bump-minor bump-major push-tag

# Get version from git tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

all: check

## Development

test:
	go test -count=1 -race ./...

coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

lint:
	golangci-lint run ./...

fmt:
	goimports -w -local github.com/rootlyhq/sidekiq-sdk-go .

check: fmt lint test

clean:
	rm -f coverage.out coverage.html

## Version management

version:
	@echo "Current version: $(VERSION)"
	@echo "Latest tag: $(shell git describe --tags --abbrev=0 2>/dev/null || echo 'none')"

bump-patch:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
	if [ -z "$$latest" ]; then latest="0.0.0"; fi; \
	new=$$(echo $$latest | awk -F. '{$$NF = $$NF + 1;} 1' OFS=.); \
	echo "Bumping version: v$$latest -> v$$new"; \
	git tag -a "v$$new" -m "Release v$$new"

bump-minor:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
	if [ -z "$$latest" ]; then latest="0.0.0"; fi; \
	new=$$(echo $$latest | awk -F. '{$$(NF-1) = $$(NF-1) + 1; $$NF = 0;} 1' OFS=.); \
	echo "Bumping version: v$$latest -> v$$new"; \
	git tag -a "v$$new" -m "Release v$$new"

bump-major:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
	if [ -z "$$latest" ]; then latest="0.0.0"; fi; \
	new=$$(echo $$latest | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' OFS=.); \
	echo "Bumping version: v$$latest -> v$$new"; \
	git tag -a "v$$new" -m "Release v$$new"

push-tag:
	git push origin --tags

release-patch: bump-patch push-tag
release-minor: bump-minor push-tag
release-major: bump-major push-tag
```

### `go.mod`

```go
module github.com/rootlyhq/sidekiq-sdk-go

go 1.24

require (
	github.com/alicebob/miniredis/v2 v2.33.0
	github.com/redis/go-redis/v9 v9.7.0
)
```

### `LICENSE`

```
MIT License

Copyright (c) 2025 Rootly Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## Testing Strategy

### Redis Mocking with miniredis

Use `github.com/alicebob/miniredis/v2` - a pure Go Redis server for unit tests. It implements Redis commands without needing a real Redis instance.

```go
// sidekiq_test.go
package sidekiq

import (
    "context"
    "testing"

    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

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
```

### Sidekiq Test Fixtures (`internal/testdata/fixtures.go`)

Create realistic Sidekiq data structures for testing:

```go
package testdata

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/alicebob/miniredis/v2"
)

// SidekiqFixtures provides helper methods to populate miniredis with Sidekiq data
type SidekiqFixtures struct {
    mr        *miniredis.Miniredis
    namespace string
}

func NewFixtures(mr *miniredis.Miniredis, namespace string) *SidekiqFixtures {
    return &SidekiqFixtures{mr: mr, namespace: namespace}
}

func (f *SidekiqFixtures) key(parts ...string) string {
    if f.namespace == "" {
        return strings.Join(parts, ":")
    }
    return f.namespace + ":" + strings.Join(parts, ":")
}

// Job payload structure matching Sidekiq format
type JobPayload struct {
    Class      string        `json:"class"`
    Args       []interface{} `json:"args"`
    JID        string        `json:"jid"`
    CreatedAt  float64       `json:"created_at"`
    EnqueuedAt float64       `json:"enqueued_at"`
    Queue      string        `json:"queue,omitempty"`
    Retry      interface{}   `json:"retry,omitempty"`
    Tags       []string      `json:"tags,omitempty"`
    // Error fields (for retry/dead)
    ErrorMessage string `json:"error_message,omitempty"`
    ErrorClass   string `json:"error_class,omitempty"`
    FailedAt     float64 `json:"failed_at,omitempty"`
    RetryCount   int     `json:"retry_count,omitempty"`
}

// ActiveJob wrapper payload
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

// Process info structure
type ProcessInfo struct {
    Hostname    string            `json:"hostname"`
    StartedAt   float64           `json:"started_at"`
    PID         int               `json:"pid"`
    Tag         string            `json:"tag"`
    Concurrency int               `json:"concurrency"`
    Queues      []string          `json:"queues"`
    Labels      []string          `json:"labels"`
    Identity    string            `json:"identity"`
    Version     string            `json:"version"`
    Busy        int               `json:"busy"`
}

// Work info structure
type WorkInfo struct {
    Queue   string                 `json:"queue"`
    RunAt   float64                `json:"run_at"`
    Payload map[string]interface{} `json:"payload"`
}

// SetupStats populates stat counters
func (f *SidekiqFixtures) SetupStats(processed, failed int64) {
    f.mr.Set(f.key("stat:processed"), fmt.Sprintf("%d", processed))
    f.mr.Set(f.key("stat:failed"), fmt.Sprintf("%d", failed))
}

// SetupDailyStats populates daily stat counters
func (f *SidekiqFixtures) SetupDailyStats(date time.Time, processed, failed int64) {
    dateStr := date.Format("2006-01-02")
    f.mr.Set(f.key("stat:processed:"+dateStr), fmt.Sprintf("%d", processed))
    f.mr.Set(f.key("stat:failed:"+dateStr), fmt.Sprintf("%d", failed))
}

// AddQueue creates a queue with jobs
func (f *SidekiqFixtures) AddQueue(name string, jobs ...JobPayload) {
    // Register queue
    f.mr.SAdd(f.key("queues"), name)

    // Add jobs to queue (LPUSH, so last job is first in queue)
    queueKey := f.key("queue:" + name)
    for i := len(jobs) - 1; i >= 0; i-- {
        job := jobs[i]
        if job.Queue == "" {
            job.Queue = name
        }
        data, _ := json.Marshal(job)
        f.mr.Lpush(queueKey, string(data))
    }
}

// AddScheduledJob adds a job to the schedule set
func (f *SidekiqFixtures) AddScheduledJob(at time.Time, job JobPayload) {
    data, _ := json.Marshal(job)
    f.mr.ZAdd(f.key("schedule"), float64(at.Unix()), string(data))
}

// AddRetryJob adds a job to the retry set
func (f *SidekiqFixtures) AddRetryJob(at time.Time, job JobPayload) {
    data, _ := json.Marshal(job)
    f.mr.ZAdd(f.key("retry"), float64(at.Unix()), string(data))
}

// AddDeadJob adds a job to the dead set
func (f *SidekiqFixtures) AddDeadJob(at time.Time, job JobPayload) {
    data, _ := json.Marshal(job)
    f.mr.ZAdd(f.key("dead"), float64(at.Unix()), string(data))
}

// AddProcess registers a Sidekiq process
func (f *SidekiqFixtures) AddProcess(info ProcessInfo) {
    f.mr.SAdd(f.key("processes"), info.Identity)
    data, _ := json.Marshal(info)
    f.mr.Set(f.key(info.Identity), string(data))
}

// AddWork registers work being done by a process
func (f *SidekiqFixtures) AddWork(identity, tid string, work WorkInfo) {
    data, _ := json.Marshal(work)
    f.mr.HSet(f.key(identity+":work"), tid, string(data))
}

// PauseQueue marks a queue as paused
func (f *SidekiqFixtures) PauseQueue(name string) {
    f.mr.Set(f.key("paused:"+name), "true")
}

// Sample data generators

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

func (f *SidekiqFixtures) SampleFailedJob(jid, class, errClass, errMsg string) JobPayload {
    job := f.SampleJob(jid, class)
    job.ErrorClass = errClass
    job.ErrorMessage = errMsg
    job.FailedAt = float64(time.Now().Unix())
    job.RetryCount = 1
    return job
}

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

func (f *SidekiqFixtures) SampleActiveJob(jid, wrappedClass string, args ...interface{}) ActiveJobPayload {
    now := float64(time.Now().Unix())
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
```

### Example Test Using Fixtures

```go
// stats_test.go
package sidekiq

import (
    "context"
    "testing"
    "time"

    "github.com/rootlyhq/sidekiq-sdk-go-sdk/internal/testdata"
)

func TestStats_Basic(t *testing.T) {
    client, mr := setupTestClient(t)
    fixtures := testdata.NewFixtures(mr, "")

    // Setup test data
    fixtures.SetupStats(1000, 50)
    fixtures.AddQueue("default",
        fixtures.SampleJob("job1", "MyWorker", "arg1"),
        fixtures.SampleJob("job2", "MyWorker", "arg2"),
    )
    fixtures.AddQueue("critical",
        fixtures.SampleJob("job3", "CriticalWorker"),
    )
    fixtures.AddScheduledJob(time.Now().Add(time.Hour),
        fixtures.SampleJob("scheduled1", "ScheduledWorker"),
    )
    fixtures.AddRetryJob(time.Now().Add(time.Minute),
        fixtures.SampleFailedJob("retry1", "FailingWorker", "StandardError", "oops"),
    )

    ctx := context.Background()
    stats, err := NewStats(ctx, client)
    if err != nil {
        t.Fatalf("NewStats() error = %v", err)
    }

    if got := stats.Processed(); got != 1000 {
        t.Errorf("Processed() = %d, want 1000", got)
    }
    if got := stats.Failed(); got != 50 {
        t.Errorf("Failed() = %d, want 50", got)
    }
    if got := stats.Enqueued(); got != 3 {
        t.Errorf("Enqueued() = %d, want 3", got)
    }
    if got := stats.ScheduledSize(); got != 1 {
        t.Errorf("ScheduledSize() = %d, want 1", got)
    }
    if got := stats.RetrySize(); got != 1 {
        t.Errorf("RetrySize() = %d, want 1", got)
    }

    queues := stats.Queues()
    if queues["default"] != 2 {
        t.Errorf("Queues()[default] = %d, want 2", queues["default"])
    }
    if queues["critical"] != 1 {
        t.Errorf("Queues()[critical] = %d, want 1", queues["critical"])
    }
}

func TestQueue_Each(t *testing.T) {
    client, mr := setupTestClient(t)
    fixtures := testdata.NewFixtures(mr, "")

    fixtures.AddQueue("default",
        fixtures.SampleJob("job1", "Worker1", "a"),
        fixtures.SampleJob("job2", "Worker2", "b"),
        fixtures.SampleJob("job3", "Worker3", "c"),
    )

    ctx := context.Background()
    queue := NewQueue(client, "default")

    var jobs []string
    err := queue.Each(ctx, func(job JobRecord) bool {
        jobs = append(jobs, job.JID())
        return true
    })
    if err != nil {
        t.Fatalf("Each() error = %v", err)
    }

    if len(jobs) != 3 {
        t.Errorf("got %d jobs, want 3", len(jobs))
    }
}

func TestQueue_Each_StopEarly(t *testing.T) {
    client, mr := setupTestClient(t)
    fixtures := testdata.NewFixtures(mr, "")

    fixtures.AddQueue("default",
        fixtures.SampleJob("job1", "Worker"),
        fixtures.SampleJob("job2", "Worker"),
        fixtures.SampleJob("job3", "Worker"),
    )

    ctx := context.Background()
    queue := NewQueue(client, "default")

    count := 0
    err := queue.Each(ctx, func(job JobRecord) bool {
        count++
        return count < 2 // stop after 2
    })
    if err != nil {
        t.Fatalf("Each() error = %v", err)
    }

    if count != 2 {
        t.Errorf("iterated %d times, want 2", count)
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

func TestNamespace(t *testing.T) {
    client, mr := setupTestClient(t)
    clientNS := NewClient(client.rdb, WithNamespace("myapp"))

    fixtures := testdata.NewFixtures(mr, "myapp")
    fixtures.SetupStats(500, 10)
    fixtures.AddQueue("default", fixtures.SampleJob("j1", "W"))

    ctx := context.Background()

    // Without namespace - should see nothing
    stats, _ := NewStats(ctx, client)
    if stats.Processed() != 0 {
        t.Errorf("non-namespaced client saw processed=%d, want 0", stats.Processed())
    }

    // With namespace - should see data
    statsNS, _ := NewStats(ctx, clientNS)
    if statsNS.Processed() != 500 {
        t.Errorf("namespaced client saw processed=%d, want 500", statsNS.Processed())
    }
}
```

### Integration Tests (Optional)

For testing against real Redis with actual Sidekiq data:

```go
//go:build integration

package sidekiq_test

import (
    "context"
    "os"
    "testing"

    "github.com/redis/go-redis/v9"
    sidekiq "github.com/rootlyhq/sidekiq-sdk-go"
)

func TestIntegration_RealRedis(t *testing.T) {
    redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        redisURL = "redis://localhost:6379"
    }

    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        t.Fatalf("invalid REDIS_URL: %v", err)
    }

    rdb := redis.NewClient(opt)
    defer rdb.Close()

    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    client := sidekiq.NewClient(rdb)

    // Test against real Sidekiq data
    stats, err := sidekiq.NewStats(ctx, client)
    if err != nil {
        t.Fatalf("NewStats() error = %v", err)
    }

    t.Logf("Processed: %d", stats.Processed())
    t.Logf("Failed: %d", stats.Failed())
    t.Logf("Enqueued: %d", stats.Enqueued())
}
```

Run integration tests:
```bash
REDIS_URL=redis://localhost:6379 go test -tags=integration ./...
```

---

## Documentation Templates

### `README.md`

```markdown
# sidekiq-go

[![Go Reference](https://pkg.go.dev/badge/github.com/rootlyhq/sidekiq-sdk-go.svg)](https://pkg.go.dev/github.com/rootlyhq/sidekiq-sdk-go)
[![CI](https://github.com/rootlyhq/sidekiq-sdk-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/rootlyhq/sidekiq-sdk-go-sdk/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rootlyhq/sidekiq-sdk-go)](https://goreportcard.com/report/github.com/rootlyhq/sidekiq-sdk-go)
[![codecov](https://codecov.io/gh/rootlyhq/sidekiq-go/branch/main/graph/badge.svg)](https://codecov.io/gh/rootlyhq/sidekiq-go)

Go SDK for interacting with [Sidekiq](https://sidekiq.org)'s Redis data structures. Monitor queues, manage jobs, and integrate Sidekiq metrics into your Go applications.

## Features

- **Full API compatibility** with Ruby's `Sidekiq::API`
- **Stats**: Processed/failed counts, queue sizes, latency metrics
- **Queues**: List, iterate, find jobs, clear queues
- **Sorted Sets**: Scheduled, Retry, Dead job management
- **Processes**: Monitor active Sidekiq workers
- **Job Enqueuing**: Push jobs to Sidekiq from Go
- **Namespace support**: Works with sidekiq-namespace gem
- **Thread-safe**: Safe for concurrent use

## Installation

```bash
go get github.com/rootlyhq/sidekiq-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/redis/go-redis/v9"
    sidekiq "github.com/rootlyhq/sidekiq-sdk-go"
)

func main() {
    ctx := context.Background()

    // Connect to Redis
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer rdb.Close()

    // Create Sidekiq client
    client := sidekiq.NewClient(rdb)

    // Get stats
    stats, err := sidekiq.NewStats(ctx, client)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Processed: %d\n", stats.Processed())
    fmt.Printf("Failed: %d\n", stats.Failed())
    fmt.Printf("Enqueued: %d\n", stats.Enqueued())
    fmt.Printf("Retry size: %d\n", stats.RetrySize())
}
```

## Usage

### Queue Operations

```go
// List all queues
queues, _ := sidekiq.AllQueues(ctx, client)
for _, q := range queues {
    size, _ := q.Size(ctx)
    latency, _ := q.Latency(ctx)
    fmt.Printf("%s: %d jobs, %.2fs latency\n", q.Name(), size, latency)
}

// Iterate jobs in a queue
queue := sidekiq.NewQueue(client, "default")
queue.Each(ctx, func(job sidekiq.JobRecord) bool {
    fmt.Printf("Job %s: %s\n", job.JID(), job.Class())
    return true // continue iteration
})

// Find specific job
job, _ := queue.FindJob(ctx, "abc123")

// Clear queue
deleted, _ := queue.Clear(ctx)
```

### Retry Management

```go
retrySet := sidekiq.NewRetrySet(client)

// Iterate retries
retrySet.Each(ctx, func(entry *sidekiq.SortedEntry) bool {
    fmt.Printf("%s failed: %s\n", entry.JID(), entry.ErrorMessage())
    return true
})

// Retry a job immediately
entry, _ := retrySet.FindJob(ctx, "xyz789")
entry.AddToQueue(ctx)

// Move to dead set
entry.Kill(ctx)

// Retry all
retrySet.RetryAll(ctx)
```

### Process Monitoring

```go
processSet := sidekiq.NewProcessSet(client)

processSet.Each(ctx, func(p *sidekiq.Process) bool {
    fmt.Printf("%s: %d/%d busy\n", p.Hostname(), p.Busy(), p.Concurrency())
    return true
})

// Signal processes to stop
processSet.Each(ctx, func(p *sidekiq.Process) bool {
    p.Quiet(ctx)
    return true
})
```

### Enqueue Jobs

```go
// Immediate execution
jid, _ := client.Enqueue(ctx, "MyWorker", []interface{}{"arg1", 42}, nil)

// Scheduled execution
jid, _ := client.EnqueueAt(ctx, time.Now().Add(time.Hour), "MyWorker",
    []interface{}{"arg1"},
    &sidekiq.EnqueueOptions{
        Queue: "critical",
        Retry: 5,
    })
```

### With Namespace

```go
client := sidekiq.NewClient(rdb, sidekiq.WithNamespace("myapp"))
```

## Compatibility

- Go 1.24+
- Sidekiq 7.x+
- Redis 6.x+

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Run all checks
make check

# Generate coverage report
make coverage-html
```

## License

MIT License - see [LICENSE](LICENSE)
```

### `CHANGELOG.md`

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - YYYY-MM-DD

### Added
- Initial release
- `Client` with Redis connection and namespace support
- `Stats` for cluster-wide metrics
- `Queue` for queue operations (size, latency, iterate, find, clear)
- `JobRecord` for accessing job data
- `ScheduledSet`, `RetrySet`, `DeadSet` for sorted set operations
- `SortedEntry` with reschedule, retry, kill operations
- `ProcessSet` and `Process` for worker monitoring
- `WorkSet` and `Work` for active job tracking
- `Enqueue`, `EnqueueAt`, `EnqueueIn` for job submission
- Namespace support via `WithNamespace` option
- Full test suite with miniredis
```

### `CLAUDE.md`

```markdown
# Sidekiq Go SDK - Developer Reference

## Overview

Go SDK for interacting with Sidekiq's Redis data structures. Provides monitoring, administration, and job management capabilities compatible with Ruby's `Sidekiq::API`.

## Tech Stack

- **Language**: Go 1.24+
- **Redis Client**: github.com/redis/go-redis/v9
- **Testing**: github.com/alicebob/miniredis/v2

## Project Structure

```
├── sidekiq.go          # Client, NewClient()
├── options.go          # WithNamespace(), WithLogger()
├── keys.go             # Redis key builders
├── errors.go           # ErrJobNotFound, etc.
├── stats.go            # Stats, GetHistory()
├── queue.go            # Queue, AllQueues()
├── job.go              # JobRecord
├── sorted_set.go       # SortedSet, JobSet
├── sorted_entry.go     # SortedEntry
├── scheduled_set.go    # ScheduledSet
├── retry_set.go        # RetrySet
├── dead_set.go         # DeadSet
├── process.go          # ProcessSet, Process
├── work.go             # WorkSet, Work
├── enqueue.go          # Enqueue(), EnqueueAt()
├── scripts.go          # Lua scripts
└── internal/
    ├── json/           # JSON utilities
    └── testdata/       # Test fixtures
```

## Build Commands

```bash
make test           # Run tests
make lint           # Run linter
make check          # fmt + lint + test
make coverage       # Generate coverage report
make coverage-html  # HTML coverage report
```

## Architecture

### Client

Central struct holding Redis connection and configuration.

```go
client := sidekiq.NewClient(rdb,
    sidekiq.WithNamespace("myapp"),
    sidekiq.WithLogger(logger),
)
```

### Redis Keys

All keys are built through the client to support namespacing:
- `stat:processed`, `stat:failed` - counters
- `queues` - SET of queue names
- `queue:{name}` - LIST of jobs
- `schedule`, `retry`, `dead` - ZSETs with timestamp scores
- `processes` - SET of process identities
- `{identity}:work` - HASH of active jobs

### Iteration Pattern

Callback-based iteration with early exit support:

```go
queue.Each(ctx, func(job JobRecord) bool {
    // return false to stop
    return true
})
```

## Testing

Uses miniredis for unit tests. Test fixtures in `internal/testdata/fixtures.go` provide helpers to populate Redis with realistic Sidekiq data.

```go
fixtures := testdata.NewFixtures(mr, "")
fixtures.SetupStats(1000, 50)
fixtures.AddQueue("default", fixtures.SampleJob("j1", "Worker"))
```

## Key Bindings (N/A - Library)

This is a library, not a CLI application.

## Debug

Enable Redis command logging:

```go
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
redis.SetLogger(log.New(os.Stderr, "redis: ", log.LstdFlags))
```
```

---

## Core Design Patterns

### 1. Client Configuration (Functional Options)

```go
type Client struct {
    rdb       redis.UniversalClient
    namespace string
    logger    Logger
}

// Functional options pattern
func NewClient(rdb redis.UniversalClient, opts ...Option) *Client

type Option func(*Client)

func WithNamespace(ns string) Option
func WithLogger(l Logger) Option
```

### 2. Interface-Based Design (for testing)

```go
// Logger interface for pluggable logging
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}
```

### 3. Error Handling

```go
// Custom error types for specific conditions
var (
    ErrJobNotFound     = errors.New("sidekiq: job not found")
    ErrQueueNotFound   = errors.New("sidekiq: queue not found")
    ErrProcessNotFound = errors.New("sidekiq: process not found")
    ErrInvalidJob      = errors.New("sidekiq: invalid job data")
)

// Wrap Redis errors with context
type RedisError struct {
    Op  string // Operation that failed
    Err error  // Underlying error
}

func (e *RedisError) Error() string {
    return fmt.Sprintf("sidekiq: %s: %v", e.Op, e.Err)
}

func (e *RedisError) Unwrap() error {
    return e.Err
}
```

### 4. Iterator Pattern

```go
// Callback-based iteration
func (q *Queue) Each(ctx context.Context, fn func(JobRecord) bool) error
```

**Decision:** Use callback-based for initial implementation (simple, idiomatic Go).

---

## Implementation Tasks

### Phase 1: Core Infrastructure

#### 1.1 Client & Configuration (`sidekiq.go`, `options.go`)
- [ ] `Client` struct with Redis connection and namespace
- [ ] `NewClient(rdb, ...Option)` constructor
- [ ] `WithNamespace(string)` option
- [ ] `WithLogger(Logger)` option
- [ ] `Close()` method (if client owns connection)
- [ ] Ping/health check method

#### 1.2 Redis Key Management (`keys.go`)
- [ ] Key constants with proper prefixing
- [ ] `(c *Client) key(parts ...string) string` - builds namespaced key
- [ ] `(c *Client) queueKey(name string) string` - builds `queue:{name}`
- [ ] `(c *Client) processKey(identity string) string`
- [ ] `(c *Client) workKey(identity string) string`

#### 1.3 Error Types (`errors.go`)
- [ ] Define sentinel errors
- [ ] `RedisError` wrapper type
- [ ] `JobParseError` for JSON parsing failures

#### 1.4 Internal Utilities (`internal/json/json.go`)
- [ ] `parseJob(data string) (map[string]interface{}, error)`
- [ ] `getString(m map[string]interface{}, key string) string`
- [ ] `getInt64(m map[string]interface{}, key string) int64`
- [ ] `getFloat64(m map[string]interface{}, key string) float64`
- [ ] `getTime(m map[string]interface{}, key string) time.Time`
- [ ] `getStringSlice(m map[string]interface{}, key string) []string`
- [ ] `decompressBacktrace(data string) ([]string, error)`

---

### Phase 2: Stats

#### 2.1 Stats Struct (`stats.go`)

```go
type Stats struct {
    processed           int64
    failed              int64
    scheduledSize       int64
    retrySize           int64
    deadSize            int64
    enqueued            int64
    processesSize       int64
    workersSize         int64
    defaultQueueLatency float64
    queues              map[string]int64
}
```

**Methods:**
- [ ] `NewStats(ctx, client) (*Stats, error)` - fetches all stats atomically
- [ ] `Processed() int64`
- [ ] `Failed() int64`
- [ ] `ScheduledSize() int64`
- [ ] `RetrySize() int64`
- [ ] `DeadSize() int64`
- [ ] `Enqueued() int64`
- [ ] `ProcessesSize() int64`
- [ ] `WorkersSize() int64`
- [ ] `DefaultQueueLatency() float64`
- [ ] `Queues() map[string]int64`
- [ ] `Reset(ctx, client, stats ...string) error`

#### 2.2 StatsHistory (`stats.go`)

```go
type DayStats struct {
    Date      time.Time
    Processed int64
    Failed    int64
}

func GetHistory(ctx context.Context, client *Client, days int, startDate time.Time) ([]DayStats, error)
```

---

### Phase 3: Queue

#### 3.1 Queue Struct (`queue.go`)

```go
type Queue struct {
    client *Client
    name   string
}
```

**Constructors:**
- [ ] `NewQueue(client, name) *Queue`
- [ ] `AllQueues(ctx, client) ([]*Queue, error)`

**Methods:**
- [ ] `Name() string`
- [ ] `Size(ctx) (int64, error)`
- [ ] `Paused(ctx) (bool, error)`
- [ ] `Pause(ctx) error`
- [ ] `Unpause(ctx) error`
- [ ] `Latency(ctx) (float64, error)`
- [ ] `Each(ctx, fn func(JobRecord) bool) error`
- [ ] `FindJob(ctx, jid string) (*JobRecord, error)`
- [ ] `Clear(ctx) (int64, error)`
- [ ] `Delete(ctx, job *JobRecord) error`

---

### Phase 4: JobRecord

#### 4.1 JobRecord Struct (`job.go`)

```go
type JobRecord struct {
    item  map[string]interface{}
    value string
    queue string
}
```

**Accessors:**
- [ ] `JID() string`
- [ ] `Queue() string`
- [ ] `Class() string`
- [ ] `DisplayClass() string` - unwraps ActiveJob
- [ ] `Args() []interface{}`
- [ ] `DisplayArgs() []interface{}` - unwraps ActiveJob
- [ ] `CreatedAt() time.Time`
- [ ] `EnqueuedAt() time.Time`
- [ ] `FailedAt() time.Time`
- [ ] `RetriedAt() time.Time`
- [ ] `RetryCount() int`
- [ ] `Tags() []string`
- [ ] `Bid() string`
- [ ] `ErrorMessage() string`
- [ ] `ErrorClass() string`
- [ ] `ErrorBacktrace() ([]string, error)`
- [ ] `Latency() float64`
- [ ] `Get(key string) interface{}`
- [ ] `Item() map[string]interface{}`
- [ ] `Value() string`

---

### Phase 5: Sorted Sets

#### 5.1 SortedSet Base (`sorted_set.go`)

```go
type SortedSet struct {
    client *Client
    name   string
}
```

- [ ] `Name() string`
- [ ] `Size(ctx) (int64, error)`
- [ ] `Clear(ctx) (int64, error)`

#### 5.2 JobSet (`sorted_set.go`)

```go
type JobSet struct {
    SortedSet
}
```

- [ ] `Each(ctx, fn func(*SortedEntry) bool) error`
- [ ] `EachByScore(ctx, min, max float64, fn func(*SortedEntry) bool) error`
- [ ] `Scan(ctx, match string, count int, fn func(*SortedEntry) bool) error`
- [ ] `FindJob(ctx, jid string) (*SortedEntry, error)`
- [ ] `Fetch(ctx, score float64, jid string) (*SortedEntry, error)`
- [ ] `Schedule(ctx, at time.Time, job map[string]interface{}) error`
- [ ] `DeleteByValue(ctx, value string) (bool, error)`
- [ ] `DeleteByScore(ctx, score float64) (int64, error)`

#### 5.3 SortedEntry (`sorted_entry.go`)

```go
type SortedEntry struct {
    JobRecord
    parent *JobSet
    score  float64
}
```

- [ ] `Score() float64`
- [ ] `At() time.Time`
- [ ] `Delete(ctx) error`
- [ ] `Reschedule(ctx, at time.Time) error`
- [ ] `AddToQueue(ctx) error`
- [ ] `Retry(ctx) error`
- [ ] `Kill(ctx) error`
- [ ] `HasError() bool`

---

### Phase 6: Specialized Sets

#### 6.1 ScheduledSet (`scheduled_set.go`)
- [ ] `NewScheduledSet(client) *ScheduledSet`

#### 6.2 RetrySet (`retry_set.go`)
- [ ] `NewRetrySet(client) *RetrySet`
- [ ] `RetryAll(ctx) (int64, error)`
- [ ] `KillAll(ctx) (int64, error)`

#### 6.3 DeadSet (`dead_set.go`)
- [ ] `NewDeadSet(client) *DeadSet`
- [ ] `Kill(ctx, job map[string]interface{}, opts *KillOptions) error`
- [ ] `Trim(ctx, maxJobs, maxAge int64) (int64, error)`
- [ ] `RetryAll(ctx) (int64, error)`

---

### Phase 7: Process Management

#### 7.1 ProcessSet (`process.go`)
- [ ] `NewProcessSet(client) *ProcessSet`
- [ ] `Size(ctx) (int64, error)`
- [ ] `Each(ctx, fn func(*Process) bool) error`
- [ ] `Get(ctx, identity string) (*Process, error)`
- [ ] `Cleanup(ctx) (int64, error)`
- [ ] `TotalConcurrency(ctx) (int64, error)`
- [ ] `TotalRSSKB(ctx) (int64, error)`
- [ ] `Leader(ctx) (string, error)`

#### 7.2 Process (`process.go`)
- [ ] `Identity() string`
- [ ] `Tag() string`
- [ ] `Labels() []string`
- [ ] `Hostname() string`
- [ ] `PID() int`
- [ ] `StartedAt() time.Time`
- [ ] `Queues() []string`
- [ ] `Capsules() map[string]CapsuleConfig`
- [ ] `Concurrency() int`
- [ ] `Version() string`
- [ ] `RSS() int64`
- [ ] `Busy() int`
- [ ] `Stopping() bool`
- [ ] `Quiet(ctx) error`
- [ ] `Stop(ctx) error`
- [ ] `DumpThreads(ctx) error`

---

### Phase 8: Work Tracking

#### 8.1 WorkSet (`work.go`)
- [ ] `NewWorkSet(client) *WorkSet`
- [ ] `Size(ctx) (int64, error)`
- [ ] `Each(ctx, fn func(*Work) bool) error`
- [ ] `FindWork(ctx, jid string) (*Work, error)`

#### 8.2 Work (`work.go`)
- [ ] `Queue() string`
- [ ] `RunAt() time.Time`
- [ ] `JID() string`
- [ ] `Job() *JobRecord`
- [ ] `Payload() map[string]interface{}`

---

### Phase 9: Job Enqueuing

#### 9.1 Enqueue Functions (`enqueue.go`)

```go
type EnqueueOptions struct {
    Queue     string
    At        time.Time
    In        time.Duration
    Retry     interface{}
    Backtrace int
    Tags      []string
}

func (c *Client) Enqueue(ctx context.Context, class string, args []interface{}, opts *EnqueueOptions) (string, error)
func (c *Client) EnqueueAt(ctx context.Context, at time.Time, class string, args []interface{}, opts *EnqueueOptions) (string, error)
func (c *Client) EnqueueIn(ctx context.Context, in time.Duration, class string, args []interface{}, opts *EnqueueOptions) (string, error)
```

---

### Phase 10: Lua Scripts (`scripts.go`)

- [ ] `requeue` - atomic move from sorted set to queue
- [ ] `kill` - atomic move to dead set
- [ ] `unpause_queue` - atomic unpause

---

### Phase 11: Repository Setup

- [ ] Initialize git repository
- [ ] Create all config files (.gitignore, .editorconfig, etc.)
- [ ] Setup GitHub workflows
- [ ] Configure Dependabot
- [ ] Setup GoReleaser
- [ ] Configure golangci-lint
- [ ] Create Makefile
- [ ] Write README.md
- [ ] Write CLAUDE.md
- [ ] Initialize CHANGELOG.md
- [ ] Add MIT LICENSE

---

## API Comparison Reference

| Ruby API | Go API | Notes |
|----------|--------|-------|
| `Sidekiq::Stats.new` | `NewStats(ctx, client)` | Returns struct |
| `Sidekiq::Stats::History.new(days, start)` | `GetHistory(ctx, client, days, start)` | Function |
| `Sidekiq::Queue.all` | `AllQueues(ctx, client)` | Returns slice |
| `Sidekiq::Queue.new(name)` | `NewQueue(client, name)` | Requires client |
| `queue.each { \|job\| }` | `queue.Each(ctx, func(j) bool)` | Callback |
| `Sidekiq::ScheduledSet.new` | `NewScheduledSet(client)` | |
| `Sidekiq::RetrySet.new` | `NewRetrySet(client)` | |
| `Sidekiq::DeadSet.new` | `NewDeadSet(client)` | |
| `Sidekiq::ProcessSet.new` | `NewProcessSet(client)` | |
| `Sidekiq::WorkSet.new` | `NewWorkSet(client)` | |

---

## Compatibility

**Minimum Requirements:**
- Go 1.24+
- Sidekiq 7.x+
- Redis 6.x+

| Feature | Sidekiq 7.x |
|---------|-------------|
| Basic queues | Yes |
| Sorted sets | Yes |
| Stats | Yes |
| Process info | Yes |
| Capsules | Yes |

---

## Implementation Priority

### P0 - MVP (Core Read Operations)
1. Repository setup (all config files)
2. Client & configuration
3. Keys & error handling
4. Stats
5. Queue (read + iterate)
6. JobRecord

### P1 - Job Management
7. SortedSet & JobSet
8. SortedEntry
9. ScheduledSet, RetrySet, DeadSet
10. Queue mutations (clear, delete)
11. Sorted set mutations (reschedule, retry, kill)

### P2 - Process Monitoring
12. ProcessSet
13. Process
14. WorkSet
15. Work

### P3 - Advanced Features
16. Job enqueuing
17. Lua scripts for atomicity
18. Stats history

### P4 - Polish
19. Comprehensive tests
20. Documentation
21. Examples
22. First release (v0.1.0)

---

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module path | `github.com/rootlyhq/sidekiq-sdk-go` | Clear naming convention |
| Go version | 1.24+ | Latest stable, improved stdlib |
| Sidekiq version | 7.x+ | Focus on modern API, Capsules support |
| Redis client | `redis.UniversalClient` | Supports standalone, sentinel, and cluster |
| Namespace handling | Explicit via `WithNamespace()` | Clear, no magic auto-detection |
| Iterator pattern | Callback-based | Wide compatibility, simple |
| Batch support | Out of scope | Sidekiq Pro/Enterprise feature |

---

## Notes

- This SDK is designed for monitoring and administration, not high-throughput job processing
- For Go-native job processing, consider Asynq or Machinery
- Full interoperability with Ruby Sidekiq 7.x+
- Test with both Sidekiq OSS and Enterprise data structures
- Consider adding OpenTelemetry tracing for observability in future versions
