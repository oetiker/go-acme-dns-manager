# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### New

### Changed

### Fixed

- Release only when tests have been successful!

## [0.4.1] - 2025-04-30

### New

### Changed
- Refactored command-line argument parsing for better code organization and testability
- Improved code structure by removing duplicate implementations in test files
- Enhanced test coverage to verify actual production code instead of test-only implementations

### Fixed
- Fixed inconsistent log formatting by standardizing on structured logger throughout the application
- Fixed user communication for DNS changes by separating log messages from user-facing output
- Fixed error handling in certificate argument parsing

## [0.4.0] - 2025-04-29

### Added
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

## [0.3.0] - 2025-04-28

### Changed
- Removed global `key_type` configuration option
- Added per-certificate `key_type` configuration in both CLI and config file
- Changed config naming to use consistent snake_case: `autoDomains` → `auto_domains`, `graceDays` → `grace_days`
- Improved certificate request parsing to support `/key_type=<type>` syntax
- Fixed account key storage path to follow Lego conventions with server-specific directories
- Fixed key type mapping to use proper Lego cryptographic constants
- Enhanced command-line interface to better handle certificate arguments with parameters

## [0.2.1] - 2025-04-26

### Added

- Comprehensive integration test framework
- Mock servers for ACME DNS and Let's Encrypt testing without external dependencies
- DNS resolver interface for better testability
- Certificate validation and generation utilities for testing
- Real-world certificate renewal testing infrastructure


## [0.2.0] - 2025-04-26


### Added
- Unit tests for critical components
- Added constants for file permissions and timeouts
- Comprehensive domain validation for certificate renewals
- Makefile for common development tasks
- GitHub Actions workflow for automated testing
### Changed
- Removed duplicated DNS provider setup code
- Improved code organization with better constants
- Enhanced error messages for certificate domain mismatches

## [0.1.0] - 2025-04-26

### Added
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
