.PHONY: build install clean snapshot help test test-integration test-integration-verbose test-integration-run

# Build variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION := $(shell go version | awk '{print $$3}')

LDFLAGS := -X rocha/version.Version=$(VERSION) \
           -X rocha/version.Commit=$(COMMIT) \
           -X rocha/version.Date=$(DATE) \
           -X rocha/version.GoVersion=$(GOVERSION)

help:
	@echo "Available targets:"
	@echo "  build                    - Build rocha binary with version information"
	@echo "  install                  - Build and install to ~/.local/bin"
	@echo "  snapshot                 - Test GoReleaser locally (no publish)"
	@echo "  clean                    - Remove built binaries and dist/"
	@echo "  test                     - Run all tests (alias for test-integration)"
	@echo "  test-integration         - Run integration tests"
	@echo "  test-integration-verbose - Run integration tests with no cache"
	@echo "  test-integration-run     - Run specific test: make test-integration-run TEST=TestName"

build:
	@echo "Building rocha..."
	go build -ldflags "$(LDFLAGS)" -o rocha .
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

# Test targets
test: test-integration

test-integration:
	@echo "Running integration tests..."
	@cd test/integration && go test -v ./...
	@echo "Integration tests complete"

test-integration-verbose:
	@echo "Running integration tests (no cache)..."
	@cd test/integration && go test -v -count=1 ./...
	@echo "Integration tests complete"

test-integration-run:
	@echo "Running: $(TEST)"
	@cd test/integration && go test -v -run $(TEST) ./...
