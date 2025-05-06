# Go ACME DNS Manager Developer Roadmap

This document serves as the primary guide for developers working on the go-acme-dns-manager project. Whether you're a new contributor or returning to the codebase, this roadmap provides all the essential information to get started quickly.

## Quick Start

1. **Clone the repository**:
   ```bash
   git clone https://github.com/oetiker/go-acme-dns-manager.git
   cd go-acme-dns-manager
   ```

2. **Build the application**:
   ```bash
   make build
   ```

3. **Run tests**:
   ```bash
   make test       # Basic unit tests
   make test-all   # All tests including integration tests
   ```

## Project Structure

The project follows a standard Go project layout:

```
go-acme-dns-manager/
├── cmd/go-acme-dns-manager/  # Main application entry point
│   ├── main.go               # Application entry point
│   └── main_test.go          # Tests for command-line handling
├── pkg/manager/              # Core business logic modules
│   ├── acmedns.go            # ACME DNS client implementation
│   ├── colorful_logger.go    # Colorful logging implementation
│   ├── config.go             # Configuration handling (including DNS resolver settings)
│   ├── constants.go          # Shared constants
│   ├── dnsverify.go          # DNS verification logic
│   ├── legowrapper.go        # Interface to Lego ACME client library
│   ├── logger.go             # Logger interface and implementation
│   └── ...
├── pkg/manager/test_mocks/   # Mock implementations for testing
│   ├── acmedns_mock.go       # Mock ACME DNS server
│   ├── acme_mock.go          # Mock Let's Encrypt server
│   └── dns_mock.go           # Mock DNS resolver
├── .github/workflows/        # CI/CD pipeline configurations
├── CHANGES.md                # Changelog for the project
├── Makefile                  # Build and test commands
└── README.md                 # Project overview and user documentation
```

## Development Workflow

### 1. Understanding the Code

Start by understanding the application's core components:

- **Configuration Management**: `pkg/manager/config.go`
- **Command-Line Interface**: `cmd/go-acme-dns-manager/main.go`
- **ACME DNS Integration**: `pkg/manager/acmedns.go`
- **Lego ACME Client**: `pkg/manager/legowrapper.go`
- **DNS Verification**: `pkg/manager/dnsverify.go`
- **Logging System**: `pkg/manager/logger.go` and `pkg/manager/colorful_logger.go`

#### Custom DNS Resolver Configuration

The application supports configuring a custom DNS resolver through:
1. **Config File**: Set `dns_resolver` in config.yaml
2. **Implementation**:
   - `dnsverify.go`: Uses the custom resolver for CNAME verification
   - `legowrapper.go`: Passes the resolver configuration to Lego via environment variables

### 2. Making Changes

When implementing a new feature or fixing a bug:

1. **Create a branch** for your changes
2. **Write tests first** whenever possible
3. **Implement your changes**
4. **Run tests** to verify functionality
5. **Update documentation** in relevant files

### 3. Documentation Requirements

All changes must be documented in the following places:

- **CHANGES.md**: Document what was changed, added, or fixed
  - Use categories: `New`, `Changed`, `Fixed`
  - Follow the established format of the file
- **README.md**: Update if the change affects user-facing features or configuration
- **Code comments**: Document complex logic or non-obvious behavior

### 4. Testing Requirements

The project has a robust testing framework:

#### Test Levels

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test component interactions with mocked external services
3. **End-to-End Tests**: Test full application flows

#### Architecture for Testability

The code has been specifically designed with testability in mind:

1. **Interface-Based Design**: Core components like the DNS resolver use interfaces to allow easy mocking
2. **Dependency Injection**: Functions accept dependencies that can be replaced with mocks during testing
3. **Exported Testing Helpers**: Functions like `VerifyWithResolver()` are exported to enable direct testing

#### Adding New Tests

When adding new functionality, please follow these guidelines:

1. Add unit tests for all new functions
2. For code that interacts with external services, create appropriate mocks
3. Use the integration test framework to test the interaction between components
4. Update the mock servers if new endpoints or behaviors are needed
5. Consider using test-driven development (TDD) where applicable

#### Mock Components

The test suite includes mock implementations for external dependencies:

- **Mock ACME DNS Server**: Simulates acme-dns server responses
- **Mock ACME Server**: Simulates Let's Encrypt API
- **Mock DNS Resolver**: Simulates DNS lookups without actual network requests

#### Running Tests

```bash
# Run unit tests only
make test

# Run all tests including integration tests
make test-all

# Run tests manually with Go test command
go test ./...

# Run integration tests manually
RUN_INTEGRATION_TESTS=1 go test ./...
```

### 5. Code Quality

Maintain high code quality by following these standards:

- Use **Go best practices** for code structure and error handling
- Store common values as **constants** in `constants.go`
- Use **dependency injection** for testable components
- Follow the **interface-based design** for components that interact with external services
- Run the linter before committing: `make lint`

The project uses several code quality tools:

```bash
# Run linting to check code style and find potential issues
make lint

# Build the application
make build

# Run all quality checks in sequence
make all
```

#### Constants and Configurations

Common values are stored as constants in `pkg/manager/constants.go` rather than being hard-coded throughout the application. These include:

- File permissions (e.g., `0600` for sensitive files)
- Default timeout values (e.g., DNS timeout, HTTP request timeout)
- Grace periods for certificate renewal

## Feature Implementation Guidelines

### Logging System

The application uses a structured logging system with multiple formats:

- **Go format**: Standard Go log format with timestamps
- **Emoji**: Colorful output with emoji indicators (terminal-friendly)
- **Color**: Colored text without emoji
- **ASCII**: Plain text without colors or emoji

When adding new log statements:

1. Use the appropriate log level: `Debug`, `Info`, `Warn`, `Error`
2. Use structured logging principles
3. Consider both terminal and non-terminal output formats

### DNS Challenge Handling

When working with DNS challenge logic:

1. Remember that wildcard domains share ACME DNS accounts with base domains
2. Use proper CNAME record formatting with BIND compatibility
3. Handle DNS timeouts and errors gracefully

### Certificate Management

For certificate operations:

1. Maintain backward compatibility with existing certificate storage
2. Consider certificate renewal thresholds and grace periods
3. Handle wildcard domains properly in domain validation

## Continuous Integration

The project uses GitHub Actions for CI/CD:

- **Automated tests** run on every push and pull request
  - Unit tests run on every push
  - Integration tests run on every push
- **Linting** ensures code quality
- **Build verification** checks for compilation errors
- Tests are also run as part of the release process

## Release Process

When preparing a release:

1. Ensure all changes are documented in CHANGES.md
2. Make sure the main branch is up to date.
3. Run the build and release workflow and pick the release level.

The release workflow will update the version and tag the code and create a release on github.

## Getting Help

If you need assistance with the codebase:

1. Check the existing documentation in README.md
2. Review the code comments for specific implementation details
3. Examine the test files for usage examples of different components
4. Look at the CHANGES.md file to understand recent developments
