# Testing the go-acme-dns-manager

This document describes the testing approach for the go-acme-dns-manager application.

## Test Structure

The test suite is organized into several tiers:

1. **Unit Tests**: Test individual functions and components in isolation
2. **Integration Tests**: Test the interaction between components with mocked external services
3. **End-to-End Tests**: Test the entire application flow with mocked external services

## Running Tests

### Basic Tests

To run basic unit tests:

```bash
make test
```

Or manually:

```bash
go test ./...
```

### Integration Tests

To run all tests including integration tests:

```bash
make test-all
```

Or manually:

```bash
RUN_INTEGRATION_TESTS=1 go test ./...
```

## Mock Servers

The test suite includes mock implementations of external services:

### Mock ACME DNS Server

Located in `internal/manager/testdata/mocks/acmedns_mock.go`, this mock server simulates the behavior of an ACME DNS server:

- `/register` endpoint for new account registration
- `/update` endpoint for DNS record updates
- Basic authentication validation
- TXT record storage and retrieval

### Mock ACME (Let's Encrypt) Server

Located in `internal/manager/testdata/mocks/acme_mock.go`, this mock server simulates the behavior of the Let's Encrypt ACME server:

- `/directory` endpoint for discovering API endpoints
- `/new-account` endpoint for account registration
- `/new-order` endpoint for certificate requests
- `/challenge` endpoint for challenge verification
- `/finalize-order` endpoint for certificate issuance
- `/certificate` endpoint for certificate delivery
- Self-signed certificate generation for testing

### Mock DNS Resolver

Located in `internal/manager/testdata/mocks/dns_mock.go`, this mock DNS resolver allows testing DNS verification without making actual DNS queries:

- Predefined CNAME responses
- Predefined error responses
- Implements the `DNSResolver` interface for easy substitution

## Test Helpers

Additional test helpers are available in `internal/manager/testdata/test_helpers`:

- `mock_lego.go`: Contains `MockLegoRun()` function that simulates the certificate generation process without making actual ACME calls

## Architecture for Testability

The code has been designed with testability in mind:

1. **Interface-Based Design**: Core components like the DNS resolver use interfaces to allow easy mocking
2. **Dependency Injection**: Functions accept dependencies that can be replaced with mocks during testing
3. **Exported Testing Helpers**: Functions like `VerifyWithResolver()` are exported to enable direct testing

## Adding New Tests

When adding new functionality, please follow these guidelines:

1. Add unit tests for all new functions
2. For code that interacts with external services, create appropriate mocks
3. Use the integration test framework to test the interaction between components
4. Update the mock servers if new endpoints or behaviors are needed
5. Consider using test-driven development (TDD) where applicable

## Continuous Integration

The project uses GitHub Actions to run tests automatically:

- Unit tests are run on every push
- Integration tests are run on every push
- Tests are also run before releases
