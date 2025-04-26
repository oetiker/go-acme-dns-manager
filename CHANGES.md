# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2025-04-26


### Added
- Unit tests for critical components
- Added constants for file permissions and timeouts
- Comprehensive domain validation for certificate renewals
- Makefile for common development tasks
- GitHub Actions workflow for automated testing
- Integration test framework

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
