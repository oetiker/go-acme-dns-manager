.PHONY: build test clean install help lint

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
	@echo "  build     - Build the application binary"
	@echo "  test      - Run all tests"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install the application to GOPATH/bin"
	@echo "  lint      - Run the linter"
	@echo "  all       - Run clean, lint, test, and build"
	@echo "  release   - Build for all supported platforms (requires goreleaser)"

# Build the binary
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(eval VERSION := local-version-$(shell date +%Y-%m-%d-%H:%M:%S))
	$(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME) (version: $(VERSION))"

# Run unit tests (excluding integration tests)
test:
	@echo "Running unit tests..."
	RUN_INTEGRATION_TESTS=1  $(GOTEST) -v ./...

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
