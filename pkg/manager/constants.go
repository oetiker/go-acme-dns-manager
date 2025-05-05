package manager

import "time"

// Constants for file permissions
const (
	// DirPermissions defines permissions for directories (0750)
	DirPermissions = 0750

	// PrivateKeyPermissions defines permissions for private key files (0600)
	PrivateKeyPermissions = 0600

	// CertificatePermissions defines permissions for certificate files (0644)
	CertificatePermissions = 0644

	// DefaultGraceDays defines the default renewal period in days
	DefaultGraceDays = 30

	// DefaultDNSTimeout defines the timeout for DNS operations in seconds
	DefaultDNSTimeout = 15

	// DefaultKeyType defines the default certificate key type
	DefaultKeyType = "rsa4096"

	// DefaultChallengeTimeout is the default timeout for ACME challenges
	DefaultChallengeTimeout = 10 * time.Minute
	// DefaultHTTPTimeout is the default timeout for HTTP requests to the ACME server
	DefaultHTTPTimeout = 30 * time.Second
)
