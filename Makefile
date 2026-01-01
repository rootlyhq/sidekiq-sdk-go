.PHONY: all deps test lint fmt check clean coverage coverage-html version bump-patch bump-minor bump-major push-tag release-patch release-minor release-major

# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Default target
all: check

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v -count=1 -race ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "Total coverage: $$(go tool cover -func=coverage.out | grep total | awk '{print $$3}')"

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || ~/go/bin/golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

# Format code
fmt:
	go fmt ./...
	@which goimports > /dev/null 2>&1 && goimports -w -local github.com/rootlyhq/sidekiq-sdk-go . || echo "goimports not installed, skipping"

# Check for issues
check: fmt lint test

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean

# Show current version
version:
	@echo "Current version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Recent tags:"
	@git tag -l 'v*' --sort=-v:refname | head -5

# Get latest tag for version bumping
LATEST_TAG := $(shell git tag -l 'v*' --sort=-v:refname | head -1)
LATEST_TAG_OR_DEFAULT := $(if $(LATEST_TAG),$(LATEST_TAG),v0.0.0)

# Bump patch version (v1.0.0 -> v1.0.1)
bump-patch:
	@if [ -z "$(LATEST_TAG)" ]; then \
		NEW_TAG="v0.0.1"; \
	else \
		NEW_TAG=$$(echo $(LATEST_TAG) | awk -F. '{print $$1"."$$2"."$$3+1}'); \
	fi; \
	echo "Bumping $(LATEST_TAG_OR_DEFAULT) -> $$NEW_TAG"; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	echo "Created tag $$NEW_TAG (use 'make push-tag' to push)"

# Bump minor version (v1.0.0 -> v1.1.0)
bump-minor:
	@if [ -z "$(LATEST_TAG)" ]; then \
		NEW_TAG="v0.1.0"; \
	else \
		NEW_TAG=$$(echo $(LATEST_TAG) | awk -F. '{print $$1"."$$2+1".0"}'); \
	fi; \
	echo "Bumping $(LATEST_TAG_OR_DEFAULT) -> $$NEW_TAG"; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	echo "Created tag $$NEW_TAG (use 'make push-tag' to push)"

# Bump major version (v1.0.0 -> v2.0.0)
bump-major:
	@if [ -z "$(LATEST_TAG)" ]; then \
		NEW_TAG="v1.0.0"; \
	else \
		NEW_TAG=$$(echo $(LATEST_TAG) | sed 's/v//' | awk -F. '{print "v"$$1+1".0.0"}'); \
	fi; \
	echo "Bumping $(LATEST_TAG_OR_DEFAULT) -> $$NEW_TAG"; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	echo "Created tag $$NEW_TAG (use 'make push-tag' to push)"

# Push the latest tag to remote
push-tag:
	@TAG=$$(git tag -l 'v*' --sort=-v:refname | head -1); \
	if [ -z "$$TAG" ]; then \
		echo "No tags found"; \
		exit 1; \
	fi; \
	echo "Pushing $$TAG to origin..."; \
	git push origin $$TAG

# Create and push tag in one step
release-patch: bump-patch push-tag
release-minor: bump-minor push-tag
release-major: bump-major push-tag

# Help
help:
	@echo "Available targets:"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  test          - Run tests with race detection"
	@echo "  coverage      - Run tests with coverage"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format code with gofmt and goimports"
	@echo "  check         - Run fmt, lint, and test"
	@echo "  clean         - Remove build artifacts"
	@echo "  version       - Show version information"
	@echo "  bump-patch    - Bump patch version (v1.0.0 -> v1.0.1)"
	@echo "  bump-minor    - Bump minor version (v1.0.0 -> v1.1.0)"
	@echo "  bump-major    - Bump major version (v1.0.0 -> v2.0.0)"
	@echo "  push-tag      - Push latest tag to origin"
	@echo "  release-*     - Bump version and push tag"
