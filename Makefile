# CloudView Makefile

# Variables
BINARY_NAME=cloudview
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS=-ldflags "-X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.version=${VERSION} \
				  -X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.buildTime=${BUILD_TIME} \
				  -X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.gitCommit=${GIT_COMMIT}"

# Directories
BUILD_DIR=build
DIST_DIR=dist

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

.PHONY: all build clean test deps fmt vet lint install uninstall help

# Default target
all: clean deps fmt vet test build

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/main.go
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/main.go
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmark tests
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install binary to GOPATH/bin
install: build
	@echo "Installing ${BINARY_NAME}..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Uninstall binary from GOPATH/bin
uninstall:
	@echo "Uninstalling ${BINARY_NAME}..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Run the application
run: build
	@echo "Running ${BINARY_NAME}..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run the application in development mode
dev:
	@echo "Running in development mode..."
	$(GOCMD) run ./cmd/main.go

# Generate mocks (requires mockgen)
mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=pkg/providers/interface.go -destination=test/mocks/provider_mock.go; \
	else \
		echo "mockgen not installed. Install it with: go install github.com/golang/mock/mockgen@latest"; \
	fi

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Check for outdated dependencies
deps-check:
	@echo "Checking for outdated dependencies..."
	$(GOCMD) list -u -m all

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Initialize new module (for first time setup)
init:
	@echo "Initializing Go module..."
	$(GOMOD) init github.com/Tsahi-Elkayam/cloudview

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t cloudview:$(VERSION) .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it cloudview:$(VERSION)

# Show help
help:
	@echo "CloudView Makefile Commands:"
	@echo ""
	@echo "Build commands:"
	@echo "  build        Build the binary"
	@echo "  build-all    Build for multiple platforms"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  uninstall    Remove binary from GOPATH/bin"
	@echo ""
	@echo "Development commands:"
	@echo "  run          Build and run the application"
	@echo "  dev          Run in development mode"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  lint         Lint code (requires golangci-lint)"
	@echo ""
	@echo "Testing commands:"
	@echo "  test         Run tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  benchmark    Run benchmark tests"
	@echo "  mocks        Generate mocks (requires mockgen)"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps         Download dependencies"
	@echo "  deps-check   Check for outdated dependencies"
	@echo "  deps-update  Update dependencies"
	@echo ""
	@echo "Security:"
	@echo "  security     Run security scan (requires gosec)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean        Clean build artifacts"
	@echo "  help         Show this help message"