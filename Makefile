.PHONY: build test test-all test-integration clean install help lint coverage coverage-all

# Binary name
BINARY_NAME=go-acme-dns-manager
# Build directory
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOLINT=golangci-lint

# Default target executed when no arguments are provided
default: help

# Help target
help:
	@echo "Available targets:"
	@echo "  build            - Build the application binary"
	@echo "  test             - Run unit tests (excludes test utilities)"
	@echo "  test-all         - Run all tests including integration tests"
	@echo "  test-integration - Run integration tests with test utilities"
	@echo "  coverage         - Generate test coverage report (production code only)"
	@echo "  coverage-all     - Generate coverage including test utilities"
	@echo "  clean            - Remove build artifacts"
	@echo "  install          - Install the application to GOPATH/bin"
	@echo "  lint             - Run the linter"
	@echo "  all              - Run clean, lint, test, and build"
	@echo "  release          - Build for all supported platforms (requires goreleaser)"

# Build the binary
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(eval VERSION := local-version-$(shell date +%Y-%m-%d-%H:%M:%S))
	$(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME) (version: $(VERSION))"

# Run unit tests (excludes test utilities and mocks)
test:
	@echo "Running unit tests (production code only)..."
	$(GOTEST) -v ./...

# Run all tests including integration tests with test utilities
test-all:
	@echo "Running all tests including integration tests..."
	RUN_INTEGRATION_TESTS=1 $(GOTEST) -tags testutils -v ./...

# Run integration tests with test utilities
test-integration:
	@echo "Running integration tests with test utilities..."
	RUN_INTEGRATION_TESTS=1 $(GOTEST) -tags testutils -v ./pkg/manager/test_integration/...

# Generate test coverage report (production code only)
coverage:
	@echo "Generating test coverage report (production code only)..."
	$(GOTEST) -cover ./...

# Generate test coverage including test utilities
coverage-all:
	@echo "Generating test coverage including test utilities..."
	$(GOTEST) -tags testutils -cover ./...

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Install the application
install:
	@echo "Installing..."
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run all tasks
all: clean lint test build

# Build for all supported platforms (requires goreleaser)
release:
	@echo "Building release versions..."
	goreleaser build --snapshot --rm-dist
