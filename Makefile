.PHONY: build build-all clean test lint help deps

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE) -s -w"

DIST_DIR := ./dist
BINARY_NAME := gw-agent

help:
	@echo "Gateway Agent Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build       - Build binary for current platform"
	@echo "  build-all   - Build binaries for all target platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linters (requires golangci-lint)"
	@echo "  deps        - Download dependencies"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION     - Version string (default: git describe or 'dev')"
	@echo "  COMMIT      - Git commit hash (default: git rev-parse)"
	@echo "  BUILD_DATE  - Build timestamp (default: current UTC time)"

deps:
	go mod download
	go mod tidy

build: deps
	@echo "Building for current platform..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/agent
	@echo "Built: $(DIST_DIR)/$(BINARY_NAME)"

build-all: deps
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)/linux_amd64
	@mkdir -p $(DIST_DIR)/linux_arm64
	@mkdir -p $(DIST_DIR)/windows_amd64

	@echo "Building linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/linux_amd64/$(BINARY_NAME) ./cmd/agent

	@echo "Building linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/linux_arm64/$(BINARY_NAME) ./cmd/agent

	@echo "Building windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/windows_amd64/$(BINARY_NAME).exe ./cmd/agent

	@echo "Build complete!"
	@echo "Binaries:"
	@ls -lh $(DIST_DIR)/*/$(BINARY_NAME)* 2>/dev/null || ls -lh $(DIST_DIR)/*/$(BINARY_NAME).exe

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(DIST_DIR)
	go clean
	@echo "Clean complete!"

test: deps
	@echo "Running tests..."
	go test -v -race -cover ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, running go vet..."; \
		go vet ./...; \
	fi
