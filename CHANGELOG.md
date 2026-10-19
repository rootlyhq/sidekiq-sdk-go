# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-10-20

### Changed
- **Minimum Go version is now 1.26** (was 1.24)
- Bump `github.com/redis/go-redis/v9` to v9.23.0
- Bump `github.com/alicebob/miniredis/v2` (test-only) to v2.39.0
- Use `any` instead of `interface{}` throughout (type alias, no API break)
- Apply Go modernizers (`range` over int, `maps.Copy`)
- Compare sentinel errors with `errors.Is` instead of `==`
- Enable `modernize`, `errorlint`, `intrange`, `copyloopvar`, `usestdlibvars` linters
- Docs now state Go 1.26+ (matches `go.mod`)

## [0.1.0] - 2025-01-01

### Added
- Initial release
- `Client` with Redis connection and namespace support
- `Stats` for cluster-wide metrics (processed, failed, enqueued, etc.)
- `Queue` for queue operations (size, latency, iterate, find, clear, pause)
- `JobRecord` for accessing job data with ActiveJob unwrapping
- `ScheduledSet`, `RetrySet`, `DeadSet` for sorted set operations
- `SortedEntry` with reschedule, retry, kill operations
- `ProcessSet` and `Process` for worker monitoring
- `WorkSet` and `Work` for active job tracking
- `Enqueue`, `EnqueueAt`, `EnqueueIn`, `EnqueueBulk` for job submission
- Namespace support via `WithNamespace` option
- Full test suite with miniredis
- CI/CD with GitHub Actions
- GoReleaser configuration for releases
