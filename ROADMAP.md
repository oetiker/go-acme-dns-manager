# Go ACME DNS Manager Developer Roadmap

This document serves as a comprehensive guide for developers working on the go-acme-dns-manager project. Whether you're a new contributor or returning to the codebase, this roadmap provides all the essential information to get started quickly.

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

The project follows a modern, modular Go architecture with clean separation of concerns:

```
go-acme-dns-manager/
├── cmd/go-acme-dns-manager/  # Main application entry point
│   ├── main.go               # Clean 80-line main function (was 537 lines)
│   └── main_test.go          # Tests for command-line handling
├── pkg/                      # Modular packages with single responsibilities
│   ├── common/               # Shared interfaces, types, errors, context utilities
│   │   ├── interfaces.go     # Interface abstractions for all external dependencies
│   │   ├── errors.go         # Structured error handling with context and suggestions
│   │   ├── context.go        # Context utilities for timeouts and request tracing
│   │   └── types.go          # Common data types
│   ├── app/                  # Application lifecycle and configuration management
│   │   ├── application.go    # Main application struct with dependency injection
│   │   └── *_test.go         # Comprehensive tests with context support
│   └── manager/              # Core business logic (refactored from legacy)
│       ├── acmedns.go        # ACME DNS client implementation
│       ├── acme_accounts.go  # ACME user account management
│       ├── cert_storage.go   # Certificate file operations
│       ├── config.go         # Configuration handling
│       ├── dnsverify.go      # DNS verification logic
│       ├── legowrapper.go    # Core ACME operations
│       ├── logger.go         # Logger implementation
│       ├── schema.go         # JSON Schema for config validation
│       └── test_*/           # Comprehensive test suites
├── .github/workflows/        # CI/CD pipeline configurations
├── CHANGES.md                # Changelog for the project
├── ROADMAP.md               # Development roadmap and architecture guide
├── Makefile                  # Build and test commands
└── README.md                 # Project overview and user documentation
```

## Development Workflow

### 1. Understanding the Code

Start by understanding the application's core components in the new modular architecture:

- **Application Entry Point**: `cmd/go-acme-dns-manager/main.go` (clean 80-line main)
- **Application Lifecycle**: `pkg/app/application.go` (dependency injection, graceful shutdown)
- **Shared Interfaces**: `pkg/common/interfaces.go` (abstractions for all external dependencies)
- **Error Handling**: `pkg/common/errors.go` (structured errors with context and suggestions)
- **Context Management**: `pkg/common/context.go` (timeouts, cancellation, request tracing)
- **Configuration Management**: `pkg/manager/config.go` and `pkg/manager/schema.go`
- **ACME Operations**: `pkg/manager/legowrapper.go` and `pkg/manager/acme_accounts.go`
- **DNS Verification**: `pkg/manager/dnsverify.go`
- **Certificate Storage**: `pkg/manager/cert_storage.go`
- **Logging System**: `pkg/manager/logger.go` (consolidated from multiple files)

#### Configuration Schema Validation

The application uses JSON Schema validation to ensure proper configuration:

1. **Schema Definition**: Located in `pkg/manager/schema.go`
2. **Validation Process**:
   - When the configuration is loaded, it's validated against the JSON schema
   - Validation catches misspelled keys, unsupported structures, and invalid data types
   - Detailed error messages help users identify and fix configuration issues
3. **Benefits**:
   - Prevents silent failures from misspelled configuration keys
   - Enforces proper data types and value ranges
   - Gives users immediate feedback about configuration issues

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

#### Key Areas Covered by Tests

The project has test coverage for the following essential features:

1. Certificate renewal based on expiry date
2. Certificate renewal based on domain list differences (ensuring the certificate contains all requested domains)
3. Wildcard domain handling and domain name validation
4. DNS CNAME record verification
5. Configuration loading and validation

#### Mock Components

The test suite includes mock implementations for external dependencies:

- **Mock ACME DNS Server**: Simulates acme-dns server responses
- **Mock ACME Server**: Simulates Let's Encrypt API
- **Mock DNS Resolver**: Simulates DNS lookups without actual network requests

#### Test Coverage and Build Tags

The project has excellent test coverage with a clean separation between production code and test utilities:

**Current Test Coverage:**
- **pkg/common**: 91.5% coverage (shared interfaces and utilities)
- **pkg/app**: 71.2% coverage (application lifecycle management)
- **cmd/go-acme-dns-manager**: 64.3% coverage (main application entry)
- **pkg/manager**: 63.1% coverage (core business logic)

**Build Tags for Test Organization:**
Test utilities and mocks are excluded from coverage using build tags:
- Files in `pkg/manager/test_helpers/` and `pkg/manager/test_mocks/` use `//go:build testutils`
- Integration tests that depend on these utilities also use the `testutils` tag
- This ensures coverage reports only include production code

#### Running Tests

```bash
# Run unit tests (production code only, excludes test utilities)
make test

# Run all tests including integration tests with test utilities
make test-all

# Run only integration tests with test utilities
make test-integration

# Generate coverage report (production code only)
make coverage

# Generate coverage including test utilities
make coverage-all

# Manual test commands
go test ./...                           # Unit tests only
go test -tags testutils ./...           # All tests including utilities
RUN_INTEGRATION_TESTS=1 go test -tags testutils ./...  # Integration tests
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

1. Always work with the base domain first for account lookup and CNAME verification
   - Wildcard domains (`*.example.com`) and their base domains (`example.com`) always share the same ACME DNS account
   - The CNAME record is always set on the base domain (`_acme-challenge.example.com`)
   - Never try to look up wildcard domain accounts separately - they're the same as their base domain accounts
2. Use proper CNAME record formatting with BIND compatibility
3. Handle DNS timeouts and errors gracefully

### Certificate Management

For certificate operations:

1. Maintain backward compatibility with existing certificate storage
2. Consider certificate renewal thresholds and grace periods
3. Handle wildcard domains properly in domain validation:
   - Always normalize to the base domain for validation operations
   - Remember that a wildcard domain and its base domain require the same DNS validation

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

## Future Architectural Improvements

The following architectural improvements have been identified for future development to further enhance the codebase. Note that many foundational improvements (JSON Schema validation, graceful shutdown, request tracing, structured errors, and context support) have already been implemented in the recent refactoring.

### Current Priority

1. **Testing Architecture**
   - **Why**: While test coverage is good (63-91%), the test organization could be cleaner
   - **What**: Consolidate similar test patterns, improve mock reusability across packages
   - **Impact**: Easier maintenance and faster test development for new features

2. **Concurrency and Performance**
   - **Why**: Currently processes certificates sequentially, which is inefficient for large domain lists
   - **What**: Implement concurrent certificate processing with proper rate limiting to respect ACME server limits
   - **Impact**: Significantly faster execution for users managing many certificates

3. **Enhanced Configuration Flexibility**
   - **Why**: Current config mixes user preferences with implementation details
   - **What**: Separate user-facing configuration from internal settings, add profile-based configs
   - **Impact**: Simpler configuration for users, easier maintenance for developers

### Future Considerations

4. **Alternative Storage Backends** (if there's user demand)
   - **Why**: Some users might need certificates stored in specific locations (e.g., directly in application directories)
   - **What**: Simple pluggable storage interface - keep it minimal
   - **Impact**: Allows integration with existing deployment workflows
   - **Note**: Only implement if users actually request it

### Explicitly NOT Planned

The following features are **intentionally excluded** to keep the tool focused:

- **Web interfaces** - Use existing tools like Traefik, Caddy, or nginx for this
- **REST APIs** - CLI tools should stay CLI tools
- **Daemon mode** - Use cron/systemd timers instead
- **Enterprise features** - cert-manager, HashiCorp Vault, and commercial tools already handle enterprise needs
- **Notification systems** - Monitoring tools like Prometheus + Alertmanager do this better
- **Certificate templates/policies** - This is what proper certificate management platforms are for

**Philosophy**: This tool should remain a focused, reliable CLI utility that does one thing well - managing ACME DNS certificates with minimal configuration. Users who need enterprise features should use enterprise tools.

### Implementation Notes

- These improvements should be implemented incrementally to maintain stability
- Each change should include comprehensive tests to prevent regressions
- Maintain backward compatibility for configuration and storage formats
- Consider user feedback and real-world usage patterns when prioritizing features

## Getting Help

If you need assistance with the codebase:

1. Check the existing documentation in README.md
2. Review the code comments for specific implementation details
3. Examine the test files for usage examples of different components
4. Look at the CHANGES.md file to understand recent developments
