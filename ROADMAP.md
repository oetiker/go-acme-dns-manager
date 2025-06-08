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

### 5. Code Quality and Best Practices

Maintain high code quality by following these standards and best practices specific to this ACME DNS manager application:

#### Go Idioms for Perl/Python/JavaScript/TypeScript Developers

If you're coming from a Perl, Python, JavaScript, or TypeScript background, here are key Go patterns used in this project that may be unfamiliar:

1. **Explicit Error Handling** (vs. exceptions/try-catch/promises):
   ```go
   // Go: Explicit error checking (not try/catch)
   cert, err := loadCertificate(path)
   if err != nil {
       return fmt.Errorf("failed to load certificate: %w", err)
   }
   // Continue with cert...
   ```
   Unlike try/catch (JS/TS/Python), eval/die (Perl), or promise rejections (JS), Go returns errors as values that must be explicitly checked.

2. **Interfaces for Dependency Injection** (vs. duck typing):
   ```go
   // Go: Explicit interface definition (in pkg/common/interfaces.go)
   type DNSResolver interface {
       LookupCNAME(ctx context.Context, host string) (string, error)
   }

   // Function accepts interface, not concrete type
   func verifyDNS(resolver DNSResolver, domain string) error {
       cname, err := resolver.LookupCNAME(ctx, "_acme-challenge."+domain)
       if err != nil {
           return fmt.Errorf("DNS lookup failed: %w", err)
       }
       // Process cname...
   }

   // How it's used in practice - dependency injection at creation:
   func NewCertManager(config *Config, resolver DNSResolver) *CertManager {
       return &CertManager{
           config:   config,
           resolver: resolver, // Interface injected, not concrete type
       }
   }

   // Called like this in main():
   dnsResolver := &net.Resolver{} // Real implementation
   certManager := NewCertManager(config, dnsResolver)

   // Or in tests with a mock:
   mockResolver := &MockDNSResolver{} // Test implementation
   certManager := NewCertManager(config, mockResolver)
   ```
   Go uses explicit interfaces for dependency injection rather than duck typing (Python/JS), TypeScript interfaces (compile-time only), or Perl's implicit behavior. This allows easy testing with mocks.

3. **Struct Methods** (vs. object-oriented classes):
   ```go
   // Go: Struct definition (like a class, but simpler)
   type CertManager struct {
       config   *Config
       logger   Logger
       resolver DNSResolver
   }

   // Method with receiver - the (cm *CertManager) part is the "self"
   func (cm *CertManager) RenewCertificate(ctx context.Context, domain string) error {
       // cm.config, cm.logger, cm.resolver are all accessible
       cm.logger.Info("Starting certificate renewal for domain", domain)

       // Call other methods on the same struct
       if !cm.needsRenewal(domain) {
           return nil
       }

       return cm.obtainNewCertificate(ctx, domain)
   }

   // Private method (lowercase name) - only accessible within same package
   func (cm *CertManager) needsRenewal(domain string) bool {
       // Implementation...
   }

   // How it's called in practice:
   func main() {
       certManager := NewCertManager(config, logger, resolver)

       // Call method on the instance - looks familiar to Python/Perl users
       err := certManager.RenewCertificate(ctx, "example.com")
       if err != nil {
           log.Fatal(err)
       }
   }
   ```
   Go uses struct methods with explicit receivers rather than class methods (JS/TS/Python) or Perl objects. The receiver `(cm *CertManager)` is equivalent to `this` (JS/TS), `self` (Python), or Perl's object reference.

4. **Package-Level Organization** (vs. modules/namespaces):
   ```go
   // IMPORTANT: All files in same directory share the SAME namespace/package

   // File: pkg/manager/config.go
   package manager

   type Config struct {
       Domains []string
   }

   func LoadConfig(path string) (*Config, error) { } // Public (capital L)
   func validatePath(path string) bool { }           // Private (lowercase v)

   // File: pkg/manager/cert_storage.go
   package manager // Same package name!

   // Can directly use Config struct and validatePath() function from config.go
   // No imports needed - they're in the same package namespace
   func SaveCertificate(config *Config, cert []byte) error {
       if !validatePath(config.CertPath) { // Direct access to private function
           return errors.New("invalid path")
       }
       // Save certificate...
   }

   // File: pkg/manager/cert_manager.go
   package manager // Same package again!

   type CertManager struct {
       config *Config // Direct access to Config type
   }

   func NewCertManager() *CertManager {
       config, _ := LoadConfig("config.yaml") // Direct access to LoadConfig
       return &CertManager{config: config}
   }
   ```
   **Key insight**: Files in the same package work as if they were ONE BIG FILE. You can call private functions and access private types across files in the same package without imports. This is very different from ES6 modules (JS/TS), Python modules, or Perl packages where each file is separate.

5. **Defer for Cleanup** (vs. finally/destructors):
   ```go
   // Go: defer ensures cleanup happens
   func processFile(path string) error {
       file, err := os.Open(path)
       if err != nil {
           return err
       }
       defer file.Close() // Always runs, even on early return

       // Process file...
       return nil
   }
   ```
   Go's `defer` is more explicit than finally blocks (JS/TS/Python), context managers (Python), or END blocks (Perl).

6. **Context for Cancellation** (vs. global signals):
   ```go
   // Go: Context flows through the entire call chain explicitly

   // Top level - main() creates context with timeout
   func main() {
       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
       defer cancel() // Always call cancel to free resources

       app := NewApplication(config)
       err := app.RenewAllCertificates(ctx) // Pass context down
       if err != nil {
           log.Fatal(err)
       }
   }

   // Application level - passes context to manager
   func (app *Application) RenewAllCertificates(ctx context.Context) error {
       for _, domain := range app.config.Domains {
           // Check if we've been cancelled before each domain
           select {
           case <-ctx.Done():
               return fmt.Errorf("operation cancelled: %w", ctx.Err())
           default:
           }

           // Pass context to next level
           err := app.certManager.RenewCertificate(ctx, domain)
           if err != nil {
               return err
           }
       }
       return nil
   }

   // Manager level - passes context to ACME operations
   func (cm *CertManager) RenewCertificate(ctx context.Context, domain string) error {
       // Context flows down to all operations that might take time
       valid, err := cm.verifyDNS(ctx, domain)        // DNS lookup respects timeout
       if err != nil {
           return err
       }

       return cm.acmeClient.ObtainCertificate(ctx, domain) // ACME call respects timeout
   }

   // DNS verification also respects context
   func (cm *CertManager) verifyDNS(ctx context.Context, domain string) (bool, error) {
       // The context cancellation propagates all the way down to network calls
       cname, err := cm.resolver.LookupCNAME(ctx, "_acme-challenge."+domain)
       if err != nil {
           return false, err
       }
       // Process result...
   }
   ```
   **Key insight**: Context flows explicitly through every function call that might take time or do I/O. If main() times out or user hits Ctrl+C, ALL operations stop gracefully. This is very different from AbortController (JS/TS), signal handlers (Python/Perl), or threading events.

7. **Type Safety** (vs. dynamic typing):
   ```go
   // Go: Explicit type declarations
   type Config struct {
       ACMEServerURL string        `yaml:"acme_server_url"`
       Domains       []string      `yaml:"domains"`
       RenewalDays   int          `yaml:"renewal_days"`
   }

   // Compile-time type checking prevents many runtime errors
   ```
   Go catches type errors at compile time rather than runtime, similar to TypeScript but stricter, unlike JavaScript/Python/Perl dynamic typing.

8. **No Inheritance** (composition over inheritance):
   ```go
   // Go: Composition, not inheritance
   type Application struct {
       config      *Config
       certManager *CertManager  // Embedded, not inherited
       logger      Logger
   }

   // Use composition to build complex types
   ```
   Go favors composition and interfaces over class inheritance (JS/TS/Python) or Perl's object systems.

#### General Go Best Practices

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

#### Security Best Practices

Given the sensitive nature of certificate management, follow these security guidelines:

1. **Secrets Handling**:
   - Never log private keys, passwords, or API tokens
   - Use secure file permissions (`0600`) for certificate files and private keys
   - Clear sensitive data from memory when possible
   - Use `defer` statements to ensure cleanup of sensitive resources

2. **Input Validation**:
   - Always validate domain names using proper DNS validation
   - Sanitize file paths to prevent directory traversal attacks
   - Validate configuration values using JSON Schema (see `schema.go`)
   - Use structured error handling with context for debugging

3. **Network Security**:
   - Use HTTPS for all external API calls
   - Implement proper timeout handling for network operations
   - Validate TLS certificates when making outbound connections
   - Handle DNS resolution errors gracefully

#### Error Handling Patterns

Follow the established error handling patterns in this codebase:

1. **Structured Errors** (see `pkg/common/errors.go`):
   ```go
   // Use structured errors with context
   return fmt.Errorf("failed to save certificate for domain %s: %w", domain, err)

   // Include suggestions for user action
   return &ValidationError{
       Field:       "domains",
       Value:       domain,
       Message:     "invalid domain format",
       Suggestion:  "ensure domain follows RFC format (e.g., example.com)",
   }
   ```

2. **Context Propagation**:
   - Always pass and respect context for cancellation
   - Use context timeouts for network operations
   - Add request tracing information where helpful

3. **Graceful Degradation**:
   - Log errors but continue processing other domains when possible
   - Provide meaningful error messages to users
   - Include recovery suggestions in error messages

#### Testing Best Practices

1. **Test Organization**:
   - Unit tests in `*_test.go` files alongside production code
   - Integration tests in `test_integration/` directory
   - Test utilities in `test_helpers/` with `//go:build testutils` tag
   - Mock implementations in `test_mocks/` with appropriate build tags

2. **Test Data**:
   - Use clearly fake test data (see `.gitleaks.toml` configuration)
   - Generate certificates using test helpers rather than hardcoding
   - Use table-driven tests for multiple scenarios
   - Test both success and failure cases

3. **Mock Usage**:
   - Mock external dependencies (DNS, ACME servers, file system)
   - Use interfaces to enable easy mocking
   - Test error conditions by controlling mock behavior

#### Domain and Certificate Handling

1. **Domain Normalization**:
   - Always work with base domains for ACME DNS account operations
   - Wildcard domains (`*.example.com`) and base domains (`example.com`) share accounts
   - Use consistent domain validation across the codebase

2. **Certificate Lifecycle**:
   - Check certificate expiry before attempting renewal
   - Compare domain lists to determine if certificate needs updating
   - Handle certificate parsing errors gracefully
   - Use atomic file operations for certificate storage

#### Configuration and Schema

1. **Configuration Management**:
   - Use JSON Schema validation for all configuration (see `schema.go`)
   - Provide clear error messages for configuration issues
   - Support both file-based and programmatic configuration
   - Validate configuration early in application startup

2. **Schema Evolution**:
   - Maintain backward compatibility when updating configuration schema
   - Document configuration changes in CHANGES.md
   - Use optional fields with sensible defaults for new features

#### Logging Standards

1. **Log Levels**:
   - `Debug`: Detailed information for troubleshooting
   - `Info`: General application flow and important events
   - `Warn`: Recoverable issues that users should be aware of
   - `Error`: Serious problems that prevent normal operation

2. **Structured Logging**:
   - Include relevant context (domain, operation, timestamps)
   - Use consistent field names across log statements
   - Support multiple output formats (emoji, color, ASCII, Go format)
   - Never log sensitive information (private keys, passwords)

#### Performance Considerations

1. **Efficient Operations**:
   - Use connection pooling for HTTP clients
   - Implement appropriate timeouts for all network operations
   - Consider concurrent processing for multiple domains (future enhancement)
   - Cache DNS lookups when appropriate

2. **Resource Management**:
   - Close file handles and network connections properly
   - Use `defer` statements for cleanup
   - Avoid memory leaks in long-running operations
   - Consider memory usage when processing large certificate lists

#### Constants and Configurations

Common values are stored as constants in `pkg/manager/constants.go` rather than being hard-coded throughout the application. These include:

- File permissions (e.g., `0600` for sensitive files)
- Default timeout values (e.g., DNS timeout, HTTP request timeout)
- Grace periods for certificate renewal
- Default retry counts and backoff strategies

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
