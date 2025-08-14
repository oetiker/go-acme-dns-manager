# ADR-000: Core User Workflow - Certificate Lifecycle Management

## Status
Accepted

## Context
Users need to obtain and renew Let's Encrypt certificates using ACME-DNS for DNS-01 challenges.
The program provides TWO distinct operational modes, each supporting both initial certificate
creation and renewal.

## Decision
The program implements these invariant workflows:

### Mode 1: Command-Line Mode (Manual Specification)
User specifies certificates directly on the command line.

#### Initial Certificate Creation
1. User runs: `go-acme-dns-manager cert1@example.com,www.example.com`
2. Program automatically:
   - Detects certificate doesn't exist
   - Creates ACME-DNS account(s) if missing (automatic registration!)
   - Shows DNS CNAME instructions for all domains needing setup
   - Exits cleanly for user to configure DNS
3. User configures DNS records
4. User runs same command again
5. Program:
   - Verifies DNS is correctly configured
   - Obtains certificate from Let's Encrypt
   - Saves certificate files

#### Certificate Renewal (Command-Line Mode)
1. User runs: `go-acme-dns-manager cert1@example.com,www.example.com`
2. Program automatically:
   - Detects existing certificate
   - Checks expiry and domain coverage
   - If renewal needed:
     - Uses existing ACME-DNS accounts
     - Renews certificate with Let's Encrypt
   - If no renewal needed:
     - Reports certificate is up-to-date
     - Exits cleanly

### Mode 2: Automatic Mode (Config-Driven)
User runs with `-auto` flag, certificates defined in config file.

#### Initial Certificate Creation (Auto Mode)
1. User configures `auto_domains` section in config.yaml
2. User runs: `go-acme-dns-manager -auto`
3. For each certificate in config that doesn't exist:
   - Creates ACME-DNS account(s) if missing
   - Collects all domains needing DNS setup
4. If any DNS setup needed:
   - Shows consolidated DNS instructions for ALL certificates
   - Exits cleanly for user to configure DNS
5. User configures DNS records
6. User runs same command again
7. Program obtains all certificates

#### Certificate Renewal (Auto Mode)
1. User runs: `go-acme-dns-manager -auto` (typically via cron)
2. Program automatically:
   - Checks all certificates defined in config
   - For each certificate:
     - Checks expiry (default: renew if <30 days)
     - Checks domain coverage matches config
   - Renews those needing renewal
   - Skips those up-to-date
   - Reports summary of actions taken

### Common Workflow Invariants (Both Modes)
1. **ACME-DNS accounts are ALWAYS auto-created** - user never manually registers
2. **DNS instructions are ALWAYS shown when needed** - clear, actionable CNAME records
3. **Program ALWAYS exits cleanly after showing DNS instructions** - no hanging
4. **Wildcard and base domains ALWAYS share ACME-DNS accounts** - `*.example.com` and `example.com` use same account
5. **Quiet mode ALWAYS shows critical warnings** - DNS instructions must appear even with `-quiet`

## Consequences
- ANY change that breaks these workflows makes the program unusable for its core purpose
- Both modes MUST support both initial creation and renewal
- The program MUST be cron-friendly (auto mode with -quiet)
- Tests MUST verify complete user journeys for both modes
- Refactoring MUST preserve these workflows intact

## Test Coverage Requirements

The following tests validate these workflows (located in `pkg/manager/test_integration/workflow_basic_test.go`):

- `TestUserWorkflow_BasicCommandLine` - Validates command-line mode initial certificate workflow
- `TestUserWorkflow_BasicAutoMode` - Validates auto mode initial certificate workflow
- `TestUserWorkflow_Renewal_CommandLine` - Validates renewal detection in command-line mode
- `TestUserWorkflow_Renewal_AutoMode` - Validates renewal detection in auto mode (including selective renewal)

These tests use mocked ACME operations to verify the complete user workflows work as documented. They ensure both modes support both initial creation and renewal as required.

## Implementation References
- Command-line parsing: `cmd/go-acme-dns-manager/main.go`
- Mode detection: `pkg/app/application.go` - `ValidateMode()`
- Certificate processing: `pkg/app/cert_manager.go` - `ProcessManualMode()` and `ProcessAutoMode()`
- ACME-DNS registration: `pkg/manager/legowrapper.go` - `PreCheckAcmeDNS()`
- DNS instructions display: `pkg/manager/legowrapper.go` - lines 133-143
- Quiet mode handling: `pkg/manager/logger.go` - `LogLevelQuiet` mapping

## Version History
- 2025-08-13: Initial documentation of core workflows
