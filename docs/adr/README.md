# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the go-acme-dns-manager project.

## What are ADRs?

Architecture Decision Records capture important decisions about the architecture and behavior of the system. They document not just what was decided, but WHY it was decided, making them invaluable for understanding the system's design rationale.

## Critical ADRs

### [ADR-000: Core User Workflow](ADR-000-core-user-workflow.md)
**STATUS: Accepted**

Documents the fundamental user workflows that MUST be preserved. This is the most important ADR as it defines what the program actually does for users. Breaking these workflows makes the program unusable.

Key invariants:
- Automatic ACME-DNS account registration
- Clear DNS setup instructions
- Support for both command-line and auto modes
- Both initial creation and renewal workflows

## For AI Code Assistants

**IMPORTANT**: When making changes to this codebase, you MUST:

1. Read ADR-000 first to understand the core workflows
2. Ensure your changes don't break any documented workflows
3. Run tests that validate these workflows still work
4. If you need to change a workflow, create a new ADR explaining why

## Testing ADR Compliance

Run the workflow tests to ensure ADRs are still honored:

```bash
# Run all tests including workflow validation
make test-all

# Run specific workflow tests (when implemented)
go test ./... -run TestUserWorkflow
```

## Adding New ADRs

When adding a new ADR:
1. Use the next number in sequence (ADR-001, ADR-002, etc.)
2. Follow the format in ADR-000
3. Update this README with a summary
4. Link to the new ADR from relevant code with comments
