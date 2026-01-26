.PHONY: build install clean snapshot help test test-integration test-integration-docker test-integration-docker-build test-integration-docker-shell test-integration-local-dangerous test-integration-verbose test-integration-run

# Build variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION := $(shell go version | awk '{print $$3}')

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.Date=$(DATE) \
           -X main.GoVersion=$(GOVERSION)

help:
	@echo "Available targets:"
	@echo "  build                         - Build rocha binary with version information"
	@echo "  install                       - Build and install to ~/.local/bin"
	@echo "  snapshot                      - Test GoReleaser locally (no publish)"
	@echo "  clean                         - Remove built binaries and dist/"
	@echo "  test                          - Run all tests (defaults to Docker)"
	@echo "  test-integration              - Run integration tests in Docker (safe, default)"
	@echo "  test-integration-docker-build - Build the Docker test image"
	@echo "  test-integration-docker-shell - Open interactive shell in test container"
	@echo "  test-integration-verbose      - Run integration tests with no cache"
	@echo "  test-integration-run          - Run specific test: make test-integration-run TEST=TestName"
	@echo "  test-integration-local-dangerous - Run tests on host (WARNING: modifies system files)"

build:
	@echo "Building rocha..."
	go build -ldflags "$(LDFLAGS)" -o rocha ./cmd
	@echo "Build complete: ./rocha"

install: build
	@echo "Installing to ~/.local/bin/rocha..."
	@mkdir -p ~/.local/bin
	@cp rocha ~/.local/bin/rocha
	@echo "Installation complete"

snapshot:
	@echo "Running GoReleaser in snapshot mode..."
	goreleaser release --snapshot --clean
	@echo "Snapshot build complete in dist/"

clean:
	@echo "Cleaning..."
	@rm -f rocha
	@rm -rf dist/
	@echo "Clean complete"

# Docker test configuration
DOCKER_IMAGE := rocha-integration-tests

# Test targets
# Default: run integration tests in Docker (safe)
test: test-integration

test-integration: test-integration-docker

test-integration-docker-build:
	@echo "Building integration test Docker image..."
	docker build -t $(DOCKER_IMAGE) -f test/integration/Dockerfile .

test-integration-docker: test-integration-docker-build
	@echo "Running integration tests in Docker..."
	docker run --rm $(DOCKER_IMAGE) go test -v ./test/integration/...

test-integration-docker-shell: test-integration-docker-build
	@echo "Starting interactive shell in test container..."
	docker run --rm -it $(DOCKER_IMAGE) /bin/bash

# WARNING: Runs tests directly on host - can modify system files!
# DO NOT use in CI or automated agents
test-integration-local-dangerous:
	@echo "WARNING: Running integration tests locally (not in container)"
	@echo "WARNING: This may modify shell config, create files, etc."
	@sleep 2
	cd test/integration && go test -v ./...

test-integration-verbose:
	@echo "Running integration tests (no cache)..."
	docker run --rm $(DOCKER_IMAGE) go test -v -count=1 ./test/integration/...

test-integration-run:
	@echo "Running: $(TEST)"
	docker run --rm $(DOCKER_IMAGE) go test -v -run $(TEST) ./test/integration/...
