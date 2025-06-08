package common

import (
	"context"
	"net/http"
)

// LoggerInterface defines the logging interface used throughout the application
// This allows for dependency injection and better testability
type LoggerInterface interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Importantf(format string, args ...interface{})
}

// HTTPClientInterface defines the interface for HTTP client operations
// This allows for mocking HTTP requests in tests and supports context cancellation
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileSystemInterface defines the interface for file system operations
// This allows for mocking file operations in tests
type FileSystemInterface interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm FileMode) error
	MkdirAll(path string, perm FileMode) error
	Stat(name string) (FileInfo, error)
	Remove(name string) error
}

// FileMode represents file permission mode (os.FileMode)
type FileMode uint32

// FileInfo represents file information (os.FileInfo)
type FileInfo interface {
	Name() string
	Size() int64
	Mode() FileMode
	IsDir() bool
}

// ACMEClientInterface defines the interface for ACME client operations
// This abstracts the go-acme/lego library for better testability
type ACMEClientInterface interface {
	Obtain(ctx context.Context, request CertificateRequest) (*CertificateResource, error)
	Renew(ctx context.Context, cert CertificateResource) (*CertificateResource, error)
}

// CertificateRequest represents a certificate request (abstracts lego types)
type CertificateRequest struct {
	Domains []string
	Bundle  bool
}

// CertificateResource represents a certificate resource (abstracts lego types)
type CertificateResource struct {
	Domain            string
	CertURL           string
	CertStableURL     string
	PrivateKey        []byte
	Certificate       []byte
	IssuerCertificate []byte
	CSR               []byte
}

// StorageInterface defines the interface for certificate and account storage
// This allows for different storage backends (file, database, etc.)
type StorageInterface interface {
	SaveCertificate(ctx context.Context, certName string, cert *CertificateResource) error
	LoadCertificate(ctx context.Context, certName string) (*CertificateResource, error)
	CertificateExists(ctx context.Context, certName string) bool
	SaveAccount(ctx context.Context, domain string, account *AcmeDnsAccount) error
	LoadAccount(ctx context.Context, domain string) (*AcmeDnsAccount, error)
	AccountExists(ctx context.Context, domain string) bool
}

// DNSResolver defines the interface for DNS resolution
type DNSResolver interface {
	LookupCNAME(ctx context.Context, host string) (string, error)
}

// ContextKey represents context keys used in the application
type ContextKey string

const (
	// ContextKeyTimeout is used for operation timeouts
	ContextKeyTimeout ContextKey = "timeout"
	// ContextKeyRequestID is used for request tracing
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyDomain is used for domain-specific operations
	ContextKeyDomain ContextKey = "domain"
	// ContextKeyOperation is used to track the current operation
	ContextKeyOperation ContextKey = "operation"
)

// Verify that our concrete types implement the interfaces
var _ HTTPClientInterface = (*http.Client)(nil)
