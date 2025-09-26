.PHONY: build test test-all test-integration clean install help lint coverage coverage-all build-mock test-mock-e2e

# Binary names
BINARY_NAME=go-acme-dns-manager
MOCK_BINARY_NAME=go-acme-dns-manager-mock
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
	@echo "  build-mock       - Build the mock version for testing"
	@echo "  test             - Run unit tests (excludes test utilities)"
	@echo "  test-all         - Run all tests including integration tests"
	@echo "  test-integration - Run integration tests with test utilities"
	@echo "  test-mock-e2e    - Run end-to-end tests with mock binary"
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

# Build the mock version for testing
build-mock:
	@echo "Building mock version for testing..."
	@mkdir -p $(BUILD_DIR)
	$(eval MOCK_VERSION := mock-$(shell date +%Y%m%d-%H%M%S))
	$(GOBUILD) -tags testutils -ldflags "-X main.version=$(MOCK_VERSION)" -o $(BUILD_DIR)/$(MOCK_BINARY_NAME) ./cmd/$(MOCK_BINARY_NAME)
	@echo "Mock binary built: $(BUILD_DIR)/$(MOCK_BINARY_NAME) (version: $(MOCK_VERSION))"
	@echo "🧪 This binary always runs with mock servers - no real ACME calls!"

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

# Run end-to-end tests with mock binary
test-mock-e2e: build-mock
	@echo "Running E2E tests with mock binary..."
	RUN_INTEGRATION_TESTS=1 $(GOTEST) -tags testutils -v ./pkg/manager/test_integration/ -run TestMockBinary

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
