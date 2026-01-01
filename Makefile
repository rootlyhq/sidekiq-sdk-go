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
