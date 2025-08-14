# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## CRITICAL: Read These First!

### Architecture Decision Records (ADRs)
**MUST READ**: The `docs/adr/` directory contains critical Architecture Decision Records that document core functionality that must NEVER be broken.

**Most Important**: [docs/adr/ADR-000-core-user-workflow.md](docs/adr/ADR-000-core-user-workflow.md)
- Documents the fundamental user workflows
- Breaking these workflows makes the program UNUSABLE
- Any changes MUST preserve these workflows

### Before Making ANY Changes:
1. Read ADR-000 to understand what users need the program to do
2. Run `make test-all` to ensure current functionality works
3. Make your changes
4. Run `make test-all` again to ensure nothing broke
5. If tests fail, you've broken core functionality - fix it!

### Core Purpose of This Program
This program helps users obtain and renew Let's Encrypt certificates using ACME-DNS for DNS-01 challenges. It has two modes:
- **Command-line mode**: Specify certificates directly as arguments
- **Auto mode**: Read certificates from config file (cron-friendly)

Both modes MUST support both initial certificate creation AND renewal.

@ROADMAP.md
@docs/adr/README.md
