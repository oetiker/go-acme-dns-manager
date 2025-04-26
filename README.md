# Go ACME DNS Manager

This tool automates the process of obtaining and renewing Let's Encrypt certificates using the DNS-01 challenge with an `acme-dns` server. It is a Go application and directly integrates the `go-acme/lego` library.

## Features

*   Registers new domains with your `acme-dns` server automatically.
*   Stores `acme-dns` credentials securely in a separate JSON file (`<lego_storage_path>/acme-dns-accounts.json`).
*   Verifies required `_acme-challenge` CNAME records using Go's native DNS resolver.
*   Obtains new certificates (`init` action).
*   Renews existing certificates (`renew` action).
*   Uses a YAML configuration file (`config.yaml`) for main settings.
*   Generates a default `config.yaml` on first run if it doesn't exist.
*   Supports manual certificate requests via command-line arguments (`cert-name@domain,...`).
*   Supports automated renewals via config file (`autoDomains` section) and `-auto-renew` flag.
*   Automatically determines `init` or `renew` action based on certificate existence.
*   Self-contained binary with minimal external dependencies.

## Installation / Building

1.  **Prerequisites:** Ensure you have Go installed (version 1.18 or later recommended).
2.  **Clone Repository:** Clone the parent repository containing this tool.
3.  **Build:** Navigate to this directory (`lego/go-acme-dns-manager`) and run:
    ```bash
    go build .
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

# List of domains to include in the certificate (REMOVED - Use command-line args or autoDomains section)

# Storage for acme-dns account credentials is now in a separate JSON file:
# See '<cert_storage_path>/acme-dns-accounts.json'

# Path where Let's Encrypt certificates, account info, and acme-dns credentials will be stored.
# Relative paths are relative to the directory containing this config file.
# Default is '.lego' inside the config file directory.
cert_storage_path: ".lego" # Renamed from lego_storage_path

# Optional section for configuring automatic mode via the -auto flag.
#autoDomains:
#  graceDays: 30 # Renew certs expiring within this many days (default: 30)
#  certs:
#    # The key here (e.g., 'my-main-site') is the name used for certificate files
#    # stored in '<cert_storage_path>/certificates/my-main-site.crt' etc.
#    my-main-site:
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
*   `autoDomains`: (Optional) Section for configuring automated mode (`-auto` flag).
    *   `graceDays`: Number of days before expiry to trigger a renewal attempt (default: 30).
    *   `certs`: A map where keys are the desired certificate names (used for filenames) and values contain a list of domains for that certificate. Wildcard domains (e.g., `*.example.com`) are supported.

## Usage

The tool operates in two main modes:

**1. Manual Mode:** Specify certificate requests directly on the command line.

```bash
# Request a new certificate (or renew if exists) named 'cert1' for example.com and www.example.com
./go-acme-dns-manager -config my.yaml cert1@example.com,www.example.com

# Request/Renew multiple certificates in one command
./go-acme-dns-manager -config my.yaml cert1@a.com,b.com cert2@c.com cert3@d.com,e.com,f.com

# Use a specific config file location
./go-acme-dns-manager -config /etc/go-acme-dns-manager/config.yaml cert1@example.com
```

*   The format is `cert-name@domain1,domain2,...`. Wildcard domains (e.g., `*.example.com`) are supported in the domain list.
*   **Shorthand:** For single-domain certificates, you can omit the `cert-name@` prefix and just provide the domain name (e.g., `example.com`). The tool will use the domain name as the certificate name in this case (e.g., saving files as `example.com.crt`, `example.com.key`).
*   The `cert-name` (explicit or implied) is used for storing certificate files.
*   The tool automatically determines if it needs to perform an initial request (`init`) or a renewal (`renew`) based on whether certificate files for that `cert-name` already exist in the `cert_storage_path`.
*   **Important:** If requesting a certificate name that already exists, the tool checks if the primary domain matches the existing certificate. It currently *does not* verify if the full list of Subject Alternative Names (SANs) matches the existing certificate due to limitations in reading SANs from the stored metadata file. Ensure your request matches the intended certificate.

**2. Automatic Mode:** Use the `-auto` flag to process certificates defined in the `autoDomains` section of the config file.

```bash
# Check all certificates defined in config.yaml and init/renew if necessary
./go-acme-dns-manager -config my.yaml -auto
```

*   This mode requires the `autoDomains` section to be configured in `config.yaml`.
*   No certificate arguments should be provided on the command line.
*   The tool iterates through each certificate defined under `autoDomains.certs`.
*   For each certificate, it checks if the `.crt` file exists and if its expiry date is within the configured `graceDays`.
*   If the certificate doesn't exist or is nearing expiry, it performs an `init` or `renew` action. Otherwise, it skips the certificate.

**General Workflow (applies to both modes for each certificate processed):**

1.  **ACME DNS Check/Registration:**
    *   The tool checks `<cert_storage_path>/acme-dns-accounts.json` for credentials for each domain in the current certificate request.
    *   If credentials are missing for a domain, it registers with the `acme-dns` server.
    *   The required `_acme-challenge.yourdomain.com CNAME ...` record is printed.
    *   The tool saves the new credentials to `<cert_storage_path>/acme-dns-accounts.json` and **exits**.
    *   **You must manually create the CNAME record(s) in your DNS zone and run the command again.**
2.  **CNAME Verification:**
    *   Once credentials exist, the tool verifies the CNAME records point correctly. If any are incorrect, it prints the required record and exits.
    *   **You must manually correct the CNAME record(s) and run the command again.**
3.  **Certificate Action (`init` or `renew`):**
    *   If all CNAMEs are valid, it contacts Let's Encrypt via the `lego` library to obtain/renew the certificate using the `acme-dns` provider.
    *   The action (`init` or `renew`) is determined automatically based on file existence.
    *   Certificates and the Let's Encrypt account key are saved in the `cert_storage_path`.

## Development and Testing

For information on developing, testing, and contributing to this project, please see [DEVELOPMENT.md](DEVELOPMENT.md).

## Cron Job Example (Automatic Mode)

To automate initial creation and renewal using the configuration file:

```cron
# Run daily at 3:30 AM, check configured certs and init/renew if needed
30 3 * * * /path/to/go-acme-dns-manager -auto -config /path/to/config.yaml >> /var/log/go-acme-dns-manager.log 2>&1
```

(Adjust paths and logging as needed).
