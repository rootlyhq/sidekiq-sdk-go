# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
