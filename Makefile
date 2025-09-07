.PHONY: build clean version-info

# Version information
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION ?= $(shell go version | awk '{print $$3}')

# Build flags for version injection
LDFLAGS = -X 'main.Version=$(VERSION)' \
          -X 'main.CommitHash=$(COMMIT_HASH)' \
          -X 'main.GoVersion=$(GO_VERSION)'

# Build the cfgctl CLI tool
build:
	go build -ldflags "$(LDFLAGS)" -o cfgctl ./cmd/cfgctl

# Build without version injection (development)
build-dev:
	go build -o cfgctl ./cmd/cfgctl

# Build for release with optimizations
build-release:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -o cfgctl ./cmd/cfgctl

# Show version information that will be injected
version-info:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Go Version: $(GO_VERSION)"

# Clean build artifacts
clean:
	rm -f cfgctl
	go clean ./...