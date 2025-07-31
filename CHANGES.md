# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### New

### Changed

### Fixed

## 0.7.3 - 2025-07-31
### New
- **DNS setup instructions**: Restored helpful CNAME record setup instructions that were lost during previous refactoring
  - Shows BIND-compatible DNS instructions when new ACME DNS accounts are registered
  - Shows BIND-compatible DNS instructions when existing CNAME records are missing or invalid
  - Supports both regular domains (`example.com`) and wildcard domains (`*.example.com`) with appropriate labels
  - Uses the format "===== REQUIRED DNS CHANGES =====" matching the README documentation
  - Provides actionable DNS setup guidance in all scenarios where CNAME configuration is needed

## 0.7.2 - 2025-07-15
### Fixed
- **Test suite failures in CI**: Fixed app package tests that were making real ACME server calls instead of using mocks
  - Updated `CertificateManager` to use dependency injection for the ACME client function
  - Added `SetLegoRunner()` method to allow tests to inject a mock ACME client
  - All app package tests now use `mockLegoRunner` instead of calling Let's Encrypt staging server
  - Prevents test failures in CI environments where external network calls are problematic
  - Maintains separation between unit tests and integration tests
- **Wildcard certificate renewal failure**: Fixed critical issue where wildcard certificates could not be renewed
  - The renewal logic was looking for certificate files using the literal wildcard domain name `*.example.com`
  - Certificate files are actually stored using underscore naming convention `_.example.com`
  - Fixed `legowrapper.go` to use `certName` instead of `primaryDomain` when looking for certificate files
  - Wildcard certificates now renew correctly without "certificate file not found" errors
  - Maintains consistency between certificate storage and lookup naming conventions
- **Application hanging on exit**: Fixed critical issue where the application would hang indefinitely after completing its work
  - The `Run()` method now properly calls `Shutdown()` after completing certificate processing
  - Fixed early exit paths (version flag, config template flag) to call `Shutdown()` before returning
  - Made `Shutdown()` method thread-safe using `sync.Once` to prevent panic from multiple calls
  - Application now exits cleanly in all scenarios instead of hanging on `WaitForShutdown()`
  - Improved user experience by eliminating the need to manually terminate the application
- **Test suite improvements**: Enhanced test coverage to catch application lifecycle issues
  - Added `TestApplication_FullLifecycle` test that verifies complete `Run()` + `WaitForShutdown()` flow
  - Fixed integration test that was working around the hanging bug instead of exposing it
  - Re-enabled `TestApplication_WaitForShutdown_Extended` test with proper implementation
  - Test suite now properly validates application shutdown behavior and would catch similar issues

## 0.7.1 - 2025-07-15
### Fixed
- **Auto-detection of new certificates**: Fixed issue where the program would fail when trying to renew a certificate that doesn't exist yet
  - The `determineAction` function now properly checks for certificate file existence before deciding between "init" and "renew" actions
  - New certificates are automatically detected and initialized using "init" action instead of failing with "renew"
  - Prevents errors like "certificate file not found for primary domain" when working with new certificates
  - Improves user experience by eliminating the need to manually run 'init' first for new certificates

## 0.7.0 - 2025-06-08
### New
- **Comprehensive Test Coverage**: Added extensive test suites achieving excellent coverage across all modules:
  - **pkg/common**: 91.5% coverage with shared interfaces and utility testing
  - **pkg/app**: 71.2% coverage with application lifecycle management testing
  - **cmd/go-acme-dns-manager**: 64.3% coverage with main application entry testing
  - **pkg/manager**: 63.5% coverage with core business logic testing
  - Added comprehensive test cases covering edge cases, error conditions, and integration scenarios
- **Test Utilities with Build Tags**: Implemented clean separation of test code using Go build tags:
  - Test utilities and mocks in `pkg/manager/test_helpers/` and `pkg/manager/test_mocks/` use `//go:build testutils`
  - Production code coverage reports exclude test utilities for accurate metrics
  - Integration tests can be run with `-tags testutils` flag when needed
- **Enhanced Makefile**: Added comprehensive test targets:
  - `make test` - Run unit tests (production code only)
  - `make test-all` - Run all tests including integration tests
  - `make test-integration` - Run integration tests with test utilities
  - `make coverage` - Generate coverage report (production code only)
  - `make coverage-all` - Generate coverage including test utilities
- **Major Dead Code Cleanup**: Removed ~1,000+ lines of unused code created during refactoring:
  - **Removed duplicate files**: `pkg/manager/testdata/` directory, `pkg/manager/cnameutil.go`, `pkg/manager/colorful_logger.go`
  - **Cleaned up test organization**: Moved test utilities to proper locations with build tags
  - **Consolidated logging**: Merged `colorful_logger.go` into single `logger.go` file
  - **Removed unused test file**: `pkg/manager/test_mocks/simple.go`

### Changed
- **Major architectural modernization** - Transformed 537-line monolithic main function into clean 80-line main with testable architecture:
  - **Dependency Injection**: Eliminated global state and implemented proper dependency injection throughout
  - **Modular Packages**: Created new focused packages (`pkg/common/`, `pkg/app/`) for shared interfaces and application lifecycle
  - **Interface Abstractions**: Created comprehensive interfaces for all external dependencies enabling better testability and mocking
  - **Context Support**: Implemented full `context.Context` support for cancellation, timeouts, and request tracing
  - **Graceful Shutdown**: Added proper signal handling (SIGINT/SIGTERM) and graceful shutdown capabilities
  - **Structured Error Handling**: Replaced basic errors with detailed ApplicationError types including context, suggestions, and debugging information
  - **Request Tracing**: Added unique request IDs for debugging and monitoring
  - **Enhanced User Experience**: Improved error messages with actionable suggestions and helpful guidance
- Refactored codebase for better maintainability by consolidating and creating new files:
  - Consolidated `logger.go` and `colorful_logger.go` into single `logger.go` file
  - Created new focused modules for better organization:
    - `pkg/app/application.go` - Application lifecycle and dependency injection
    - `pkg/app/cert_manager.go` - Certificate management operations
    - `pkg/common/` - Shared interfaces, types, errors, and context utilities
    - `pkg/manager/acme_accounts.go` - ACME user account management
    - `pkg/manager/cert_storage.go` - Certificate file operations
- Improved code organization following Go best practices for file structure and single responsibility principle
- **Testing Excellence**: Added 42 new test cases for architectural improvements with 100% backward compatibility

### Fixed
- **Critical Logger Bug**: Fixed serious logging level bug in `pkg/logger/logger.go`:
  - `SetLevel()` method incorrectly mapped `LogLevelInfo` to `slog.LevelWarn` instead of `slog.LevelInfo`
  - Fixed `LogLevelQuiet` error logging logic to properly allow error messages while suppressing other levels
  - Enhanced `SimpleHandler` to properly respect log level filtering before outputting messages
  - Added comprehensive tests that caught these issues and prevent regressions

## 0.6.2 - 2025-05-12
### Changed
- Improved domain handling by always prioritizing base domains in account lookup and CNAME verification
- Simplified account lookup logic to eliminate redundant wildcard/base domain checks
- Updated ROADMAP.md to better explain the relationship between wildcard domains and base domains

### Fixed
- Fixed issue where the system would fail when wildcard domains were listed before their base domains in auto_domains configuration
- Simplified wildcard domain handling in CNAME verification logic
- Fixed issue where valid CNAME records were incorrectly marked as requiring manual updates when registering new accounts
- Eliminated unnecessary backward compatibility code in account lookup

## 0.6.1 - 2025-05-07
### Changed
- Refactored codebase to extract business logic from main.go into manager package
- Moved certificate checking, parsing, and CNAME record management into dedicated modules
- Improved code organization and separation of concerns
- Refactored wildcard domain handling to improve CNAME verification logic
- Enhanced test coverage for wildcard domain handling and extracted modules

### Fixed
- Fixed issue where certificate renewal wasn't actually performed when domains don't match, despite the program claiming it would do so. The expiry check was overriding the domain mismatch renewal decision.
- Fixed redundant CNAME verification for wildcard domains when the base domain has already been verified

## 0.6.0 - 2025-05-07

### New
- Added integration test to verify certificate renewal logic when required domains don't match actual certificate domains
- Added JSON Schema validation for configuration files to detect misspelled keys and invalid structures
- Added detailed error messages for configuration validation issues

### Changed
- Improved error reporting for configuration issues with specific validation messages
- Simplified configuration validation by removing redundant manual checks in favor of JSON Schema validation
- Reduced redundant information in configuration error messages

### Fixed
- Cleaned up codebase by removing unused validation functions and fixing linting issues
- Fix tests failing because invalid config was used

## 0.5.3 - 2025-05-05

### Fixed

- Fixed custom DNS resolver configuration not being passed to Lego client
- Fixed nil pointer dereference when verifying CNAME records in domain sharing scenarios
- Restricted Lego to DNS-01 challenge type only (disabled HTTP-01 and TLS-ALPN-01)

## 0.5.2 - 2025-05-05

### New
- Added version information display when running the binary
- Added `-version` flag to explicitly show version information
- Added timestamped version for local builds (`local-version-YYYY-MM-DD-HH:MM:SS`)
- Added automated test for handling of wildcard domains.

### Changed
- Modified Makefile to inject version information during builds
- Updated release workflow to inject actual version number in release builds

### Fixed
- Fixed handling of wildcard domains to properly accept that * and non star
  domains use a single dns entry

## 0.5.1 - 2025-05-05

### New
- Added proper domain name validation according to RFC standards

### Changed
- Domain name validation is now stricter and rejects malformed domains

### Fixed
- Fixed issue where invalid domain names would be accepted
- Domain names are now verified to follow DNS RFC 1035 standards
- Wildcard domains are now properly validated (only allowed in format `*.domain.tld`)


## 0.5.0 - 2025-05-05

### New
- Added `-log-level` flag to control verbosity (debug|info|warn|error)
- Added `-log-format` flag to control output format (go|emoji|color|ascii)
- Added automatic terminal detection to use emoji format when connected to a TTY
- Added colorful logger with emoji support for human-friendly output

### Changed
- Improved DNS CNAME output format to be more copy-paste friendly
- Modified wildcard domain handling to share ACME DNS accounts between wildcard and base domains

### Fixed

- Fixed wildcard domain handling for DNS challenges to use the base domain correctly
- Fixed various log messages to use the proper logging system instead of direct prints
- Fixed multiple linting errors, including unchecked error return values in test files and file operations
- Improved code quality by using tagged switch statements where appropriate
- Corrected ineffectual variable assignments in the main application

## 0.4.3 - 2025-04-30

### Fixed

- Release only when tests have been successful!

## 0.4.1 - 2025-04-30

### Changed
- Refactored command-line argument parsing for better code organization and testability
- Improved code structure by removing duplicate implementations in test files
- Enhanced test coverage to verify actual production code instead of test-only implementations

### Fixed
- Fixed inconsistent log formatting by standardizing on structured logger throughout the application
- Fixed user communication for DNS changes by separating log messages from user-facing output
- Fixed error handling in certificate argument parsing

## 0.4.0 - 2025-04-29

### New
- Configurable timeouts for ACME challenges (`challenge_timeout`) and ACME server HTTP requests (`http_timeout`) in `config.yaml`.
- Added `-quiet` mode flag to reduce output in auto mode (useful for cron jobs)
- Added a summary report of required CNAME changes at the end of execution
- Added proper handling of wildcard domains (`*.domain.com`) for CNAME records
- Added structured logging framework using Go's standard library `slog`
- Added `-debug` flag to enable detailed debug logging

### Changed
- Fixed wildcard domain handling to use the correct base domain for CNAME records
- Improved domain verification to force renewal when certificate doesn't contain all requested domains
- Enhanced usability by consolidating CNAME setup instructions at the end of execution
- Refactored logging system to use a consistent interface with support for different log levels

## 0.3.0 - 2025-04-28

### Changed
- Removed global `key_type` configuration option
- Added per-certificate `key_type` configuration in both CLI and config file
- Changed config naming to use consistent snake_case: `autoDomains` → `auto_domains`, `graceDays` → `grace_days`
- Improved certificate request parsing to support `/key_type=<type>` syntax
- Fixed account key storage path to follow Lego conventions with server-specific directories
- Fixed key type mapping to use proper Lego cryptographic constants
- Enhanced command-line interface to better handle certificate arguments with parameters

## 0.2.1 - 2025-04-26

### New

- Comprehensive integration test framework
- Mock servers for ACME DNS and Let's Encrypt testing without external dependencies
- DNS resolver interface for better testability
- Certificate validation and generation utilities for testing
- Real-world certificate renewal testing infrastructure


## 0.2.0 - 2025-04-26


### New
- Unit tests for critical components
- Added constants for file permissions and timeouts
- Comprehensive domain validation for certificate renewals
- Makefile for common development tasks
- GitHub Actions workflow for automated testing
### Changed
- Removed duplicated DNS provider setup code
- Improved code organization with better constants
- Enhanced error messages for certificate domain mismatches

## 0.1.0 - 2025-04-26

### New
- Initial project structure based on refactoring from Python script.
- Core functionality for ACME DNS registration and certificate management using Lego library.
- Configuration via `config.yaml`.
- Separate storage for ACME DNS accounts in JSON (`<cert_storage_path>/acme-dns-accounts.json`).
- Command-line interface supporting manual mode (`cert-name@domain,...` and shorthand `domain`) and automatic mode (`-auto`).
- Automatic determination of `init`/`renew` actions.
- Conflict detection for certificate names and primary domains.
- `-print-config-template` flag to output default configuration.

### Changed
- Renamed config keys: `lego_server` -> `acme_server`, `lego_storage_path` -> `cert_storage_path`.
- Renamed flag: `-auto-renew` -> `-auto`.
- Tool now errors if config file is missing instead of auto-generating.

### Removed
- Dependency on external `dig` and `podman` tools.
- Old `-action` and `-d` command-line flags.
- Top-level `domains` list from `config.yaml`.
