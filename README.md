# Go ACME DNS Manager

A focused, reliable CLI tool for obtaining and renewing Let's Encrypt certificates using the DNS-01 challenge with an `acme-dns` server.

## Philosophy

This tool follows the Unix philosophy of "do one thing and do it well." It is designed to be:

- **Simple**: Minimal configuration, clear command-line interface
- **Reliable**: Robust error handling, comprehensive testing, graceful failure modes
- **Focused**: Certificate management only - no web interfaces, APIs, or enterprise features

**What this tool IS:**
- A CLI utility for ACME DNS certificate management with excellent wildcard support
- Perfect for centralized certificate generation with secure distribution workflows
- Ideal for organizations that want to generate certificates in one secure location and distribute them safely
- Designed for users who want something more robust than basic scripts but simpler than enterprise solutions

**What this tool is NOT:**
- An enterprise certificate management platform (use cert-manager, HashiCorp Vault, or commercial tools)
- A web service or daemon (use cron/systemd timers for automation)
- A monitoring or notification system (integrate with existing monitoring tools)

If you need enterprise features, web interfaces, or complex workflows, this tool is probably not for you.

## Use Case: Secure Wildcard Certificate Management

This tool was specifically designed to solve the wildcard certificate security problem:

**The Challenge**: Wildcard certificates (`*.example.com`) are powerful but dangerous if the private keys are distributed widely. Traditional approaches either:
- Give every service the ability to generate certificates (security risk)
- Use complex enterprise certificate management (overkill for smaller setups)

**Our Solution**: Centralized generation with secure distribution:
1. **Generate once**: Use this tool in a secure, central location to generate wildcard certificates
2. **Distribute safely**: Push certificates from the central location to services that need them
3. **Minimize exposure**: Only the central certificate generation system needs ACME credentials
4. **Double protection**: The acme-dns server itself is protected by both credentials and IP access restrictions
5. **Automate renewals**: Services receive updated certificates without needing ACME access

This approach gives you the convenience of wildcard certificates without distributing certificate generation capabilities across your infrastructure.

## Use Case: Internal Services with Valid Certificates

Another key use case is obtaining valid, browser-trusted certificates for internal services:

**The Challenge**: Internal services (databases, monitoring tools, APIs) running on private networks need valid TLS certificates for:
- Browser trust (no certificate warnings)
- API client validation
- Security best practices
- Compliance requirements

**Traditional Solutions Fall Short**:
- **Self-signed certificates**: Require manual trust store management, browser warnings
- **Internal CAs**: Complex PKI setup, client configuration, certificate distribution
- **HTTP-01 challenges**: Require internet-reachable services (won't work for internal hosts)

**Our Solution**: DNS-01 challenges work perfectly for internal services:
1. **Internet connectivity not required**: DNS-01 validation doesn't need the service to be publicly accessible
2. **Real certificates**: Get proper Let's Encrypt certificates that browsers and tools trust automatically
3. **Centralized management**: Generate certificates for `internal.example.com`, `monitoring.example.com`, etc. from one secure location
4. **Domain flexibility**: Use real domains or subdomains that resolve internally but aren't publicly accessible
5. **Secure access control**: Only the central certificate generation system can access acme-dns (protected by credentials + IP restrictions)

Example: Your internal monitoring dashboard at `https://grafana.internal.example.com` gets a real, trusted certificate even though the server is only accessible on your private network.

## Features

*   Registers new domains with your `acme-dns` server automatically.
*   Stores `acme-dns` credentials securely in a separate JSON file (`<lego_storage_path>/acme-dns-accounts.json`).
*   Verifies required `_acme-challenge` CNAME records using Go's native DNS resolver.
*   Obtains new certificates (`init` action).
*   Renews existing certificates (`renew` action).
*   Uses a YAML configuration file (`config.yaml`) for main settings.
*   Generates a default `config.yaml` on first run if it doesn't exist.
*   Supports manual certificate requests via command-line arguments (`cert-name@domain,...`).
*   Supports automated renewals via config file (`auto_domains` section) and `-auto` flag.
*   Automatically determines `init` or `renew` action based on certificate existence.
*   Self-contained binary with minimal external dependencies.
*   Configurable logging with support for different formats (Go, Emoji, Color, ASCII) and levels.
*   Smart terminal detection to provide user-friendly output when attached to a TTY.
*   Proper wildcard domain handling that shares ACME DNS accounts between wildcard and base domains.
*   BIND-style formatted DNS CNAME records for easy copying into zone files.

## Installation

### Option 1: Download Pre-compiled Binary (Recommended)

Download the latest release for your platform from:
https://github.com/oetiker/go-acme-dns-manager/releases

Available for Linux, macOS, Windows, and Illumos in the appropriate archive format.

### Option 2: Build from Source

1.  **Prerequisites:** Ensure you have Go installed (version 1.18 or later recommended).
2.  **Clone Repository:**
    ```bash
    git clone https://github.com/oetiker/go-acme-dns-manager.git
    cd go-acme-dns-manager
    ```
3.  **Build:**
    ```bash
    make build
    ```
    This will create the `go-acme-dns-manager` executable in the current directory.

## Configuration (`config.yaml`)

The application uses a `config.yaml` file for all its settings. By default, it looks for this file in the same directory as the executable. You can specify a different path using the `-config` flag.

If the specified configuration file is not found, the tool will exit with an error. You can print a default configuration template to standard output using the `-print-config-template` flag:

```bash
./go-acme-dns-manager -print-config-template > config.yaml
```

**You must edit the generated file** with your specific details before running the tool.

```yaml
# Configuration for go-acme-dns-manager

# Email address for Let's Encrypt registration and notifications
email: "your-email@example.com" # <-- EDIT THIS

# Let's Encrypt ACME server URL
# Production: https://acme-v02.api.letsencrypt.org/directory
# Staging: https://acme-staging-v02.api.letsencrypt.org/directory
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory" # <-- Use production URL when ready (Renamed from lego_server)

# Key type for the certificate (e.g., rsa2048, rsa4096, ec256, ec384)
key_type: "ec256"

# URL of your acme-dns server (e.g., https://acme-dns.example.com)
acme_dns_server: "https://acme-dns.oetiker.ch" # <-- EDIT THIS if different

# DNS resolver to use for CNAME verification checks (optional, uses system default if empty)
# Example: "1.1.1.1:53" or "8.8.8.8"
dns_resolver: ""

# List of domains to include in the certificate (REMOVED - Use command-line args or auto_domains section)

# Storage for acme-dns account credentials is now in a separate JSON file:
# See '<cert_storage_path>/acme-dns-accounts.json'

# Path where Let's Encrypt certificates, account info, and acme-dns credentials will be stored.
# Relative paths are relative to the directory containing this config file.
# Default is '.lego' inside the config file directory.
cert_storage_path: ".lego" # Renamed from lego_storage_path

# Optional section for configuring automatic mode via the -auto flag.
#auto_domains:
#  graceDays: 30 # Renew certs expiring within this many days (default: 30)
#  certs:
#    # The key here (e.g., 'my-main-site') is the name used for certificate files
#    # stored in '<cert_storage_path>/certificates/my-main-site.crt' etc.
#    my-main-site:
#      key_type: "ec256"       # Optional: Override global key_type for this cert
#      domains:
#        - example.com         # First domain is the Common Name (CN)
#        - www.example.com
#    another-service:
#      domains:
#        - service.example.com
```

**Key Configuration Fields:**

*   `email`: Your email address for Let's Encrypt.
*   `acme_server`: The ACME server URL. Use the staging URL for testing. (Renamed from `lego_server`)
*   `key_type`: The type of private key to generate for your Let's Encrypt account and certificates.
*   `acme_dns_server`: The base URL of your running `acme-dns` instance.
*   `dns_resolver`: (Optional) Specify a DNS server for CNAME checks. If empty, the system's default resolver is used.
*   `cert_storage_path`: Directory where the Let's Encrypt account key (`account.key`), registration info (`account.json`), obtained certificates (within a `certificates` subdirectory named after the certificate name), and the `acme-dns` account credentials (`acme-dns-accounts.json`) will be stored. Relative paths are based on the `config.yaml` location. (Renamed from `lego_storage_path`)
*   `challenge_timeout`: (Optional) Timeout duration for ACME challenges (e.g., DNS propagation checks). Uses Go duration format (e.g., "10m", "5m30s"). Defaults to "10m".
*   `http_timeout`: (Optional) Timeout duration for HTTP requests made to the ACME server. Uses Go duration format (e.g., "30s", "1m"). Defaults to "30s".
*   `auto_domains`: (Optional) Section for configuring automatic renewals.
    *   `grace_days`: Number of days before expiry to trigger renewal (default: 30).
    *   `certs`: A map where keys are certificate names (used for filenames) and values define the domains and optional `key_type` for each certificate.
        *   `domains`: A list of domain names to include in the certificate. The first domain is the Common Name (CN).
        *   `key_type`: (Optional) Override the default key_type of rsa4096 for this specific certificate.

## Usage

The tool operates in two main modes:

**1. Manual Mode:** Specify certificate requests directly on the command line.

```bash
# Request a new certificate (or renew if exists) named 'cert1' for example.com and www.example.com
./go-acme-dns-manager -config my.yaml cert1@example.com,www.example.com

# Request/Renew multiple certificates in one command seting a special key_type for the last one
./go-acme-dns-manager -config my.yaml cert1@a.com,b.com cert2@c.com cert3@d.com,e.com,f.com/key_type=ec256

# Use a specific config file location
./go-acme-dns-manager -config /etc/go-acme-dns-manager/config.yaml cert1@example.com
```

*   The format is `cert-name@domain1,domain2,...`. Wildcard domains (e.g., `*.example.com`) are supported in the domain list.
*   **Shorthand:** For single-domain certificates, you can omit the `cert-name@` prefix and just provide the domain name (e.g., `example.com`). The tool will use the domain name as the certificate name in this case (e.g., saving files as `example.com.crt`, `example.com.key`).
*   The `cert-name` (explicit or implied) is used for storing certificate files.
*   **Wildcard Domains:** For wildcard certificates (e.g., `*.example.com`), the tool:
    *   Creates appropriate CNAME records pointing to `_acme-challenge.example.com` (base domain, no wildcard)
    *   Shares ACME DNS accounts between wildcard and base domains to simplify management
    *   Properly handles challenge verification for both wildcard and base domains
*   The tool automatically determines if it needs to perform an initial request (`init`) or a renewal (`renew`) based on whether certificate files for that `cert-name` already exist in the `cert_storage_path`.
*   **Important:** If requesting a certificate name that already exists, the tool checks if the primary domain matches the existing certificate. It currently *does not* verify if the full list of Subject Alternative Names (SANs) matches the existing certificate due to limitations in reading SANs from the stored metadata file. Ensure your request matches the intended certificate.

**2. Automatic Mode:** Use the `-auto` flag to process certificates defined in the `auto_domains` section of the config file.

```bash
# Check all certificates defined in config.yaml and init/renew if necessary
./go-acme-dns-manager -config my.yaml -auto

# Use specific logging options
./go-acme-dns-manager -config my.yaml -log-level=debug -log-format=color cert1@example.com
```

*   This mode requires the `auto_domains` section to be configured in `config.yaml`.
*   No certificate arguments should be provided on the command line.
*   The tool iterates through each certificate defined under `auto_domains.certs`.
*   For each certificate, it checks if the `.crt` file exists and if its expiry date is within the configured `grace_days`.

**3. Logging Options:** Control the verbosity and output format of logging.

```bash
# Use debug level logging with colorful output
./go-acme-dns-manager -config my.yaml -log-level=debug -log-format=color cert1@example.com

# Use machine-readable logs (good for automation/cron jobs)
./go-acme-dns-manager -config my.yaml -log-format=go -auto
```

*   `-log-level`: Set the minimum log level to display (default: "info")
    *   `debug`: Show all detailed debugging information
    *   `info`: Show standard operational information (default)
    *   `warn`: Show warnings and errors only
    *   `error`: Show only errors
*   `-log-format`: Set the output format for logs
    *   `go`: Standard Go log format with timestamps (machine-readable)
    *   `emoji`: Colorful output with emoji indicators (default when connected to a terminal)
    *   `color`: Colored text output without emoji
    *   `ascii`: Plain text output without colors or emoji
*   `-debug`: Enable debug-level logging (shorthand for `-log-level=debug`)
*   `-quiet`: Reduce output in auto mode (useful for cron jobs, shows only errors and important messages)
*   The tool automatically detects if it's connected to a terminal and selects an appropriate format (emoji when connected to a TTY, go format otherwise) unless explicitly overridden by the `-log-format` flag.
*   If the certificate doesn't exist or is nearing expiry, it performs an `init` or `renew` action. Otherwise, it skips the certificate.

**General Workflow (applies to both modes for each certificate processed):**

1.  **ACME DNS Check/Registration:**
    *   The tool checks `<cert_storage_path>/acme-dns-accounts.json` for credentials for each domain in the current certificate request.
    *   If credentials are missing for a domain, it registers with the `acme-dns` server.
    *   The required `_acme-challenge.yourdomain.com CNAME ...` record is printed in BIND-compatible format.
    *   For wildcard domains (`*.example.com`), it correctly uses the base domain (`example.com`) for the challenge record.
    *   Wildcard and base domains share the same ACME DNS account, simplifying certificate management.
    *   The tool saves the new credentials to `<cert_storage_path>/acme-dns-accounts.json` and **exits**.
    *   **You must manually create the CNAME record(s) in your DNS zone and run the command again.**
2.  **CNAME Verification:**
    *   Once credentials exist, the tool verifies the CNAME records point correctly. If any are incorrect, it prints the required record and exits.
    *   **You must manually correct the CNAME record(s) and run the command again.**
3.  **Certificate Action (`init` or `renew`):**
    *   If all CNAMEs are valid, it contacts Let's Encrypt via the `lego` library to obtain/renew the certificate using the `acme-dns` provider.
    *   The action (`init` or `renew`) is determined automatically based on file existence.
    *   Certificates and the Let's Encrypt account key are saved in the `cert_storage_path`.

**DNS Output Format:**

When DNS records need to be created or modified, the tool provides BIND-compatible output:

```
===== REQUIRED DNS CHANGES =====
Add the following CNAME records to your DNS:

; example.com, www.example.com
_acme-challenge.example.com. IN CNAME 1234abcd-wxyz-9876-asdf.acme-dns.yourdomain.com.

; *.example.org (wildcard)
_acme-challenge.example.org. IN CNAME 5678efgh-ijkl-5432-mnop.acme-dns.yourdomain.com.
```

*   Records are grouped by target to minimize duplicate entries
*   Comments indicate which domain(s) each record serves
*   Wildcard domains are clearly marked
*   Full domain names include trailing dots for BIND compatibility
*   Records are ready to copy directly into zone files

## Development and Testing

This project includes a comprehensive testing framework that allows testing both individual components and the entire certificate lifecycle with mock servers. This approach enables testing of ACME DNS and Let's Encrypt interactions without needing actual external services.

For detailed information on development and testing, see [ROADMAP.md](ROADMAP.md). This document serves as the comprehensive guide for contributors, covering:

- Project structure and organization
- Development workflow
- Testing approach and requirements
- Code quality standards
- Feature implementation guidelines
- Continuous integration
- Release process

### Running Tests

```bash
# Run unit tests
make test

# Run all tests including integration tests
make test-all
```

## Cron Job Example (Automatic Mode)

To automate initial creation and renewal using the configuration file:

```cron
# Run daily at 3:30 AM, check configured certs and init/renew if needed
30 3 * * * /path/to/go-acme-dns-manager -auto -config /path/to/config.yaml >> /var/log/go-acme-dns-manager.log 2>&1
```

(Adjust paths and logging as needed).
