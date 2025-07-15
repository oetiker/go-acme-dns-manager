package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// mockLogger implements LoggerInterface for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, args ...interface{})             { m.debugMessages = append(m.debugMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Info(msg string, args ...interface{})              { m.infoMessages = append(m.infoMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Warn(msg string, args ...interface{})              { m.warnMessages = append(m.warnMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Error(msg string, args ...interface{})             { m.errorMessages = append(m.errorMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Debugf(format string, args ...interface{})         { m.debugMessages = append(m.debugMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Infof(format string, args ...interface{})          { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Warnf(format string, args ...interface{})          { m.warnMessages = append(m.warnMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Errorf(format string, args ...interface{})         { m.errorMessages = append(m.errorMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Importantf(format string, args ...interface{})     { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }

// Helper function to create a test config
func createTestConfig(tmpDir string) *manager.Config {
	return &manager.Config{
		CertStoragePath: tmpDir,
		AutoDomains: &manager.AutoDomainsConfig{
			GraceDays: 30, // 30 days before renewal
			Certs: map[string]manager.CertConfig{
				"example-cert": {
					Domains: []string{"example.com", "www.example.com"},
					KeyType: "rsa2048",
				},
				"wildcard-cert": {
					Domains: []string{"*.test.com", "test.com"},
					KeyType: "ec256",
				},
			},
		},
	}
}

func TestNewCertificateManager_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	if cm == nil {
		t.Fatal("Certificate manager should not be nil")
	}

	if cm.config != config {
		t.Error("Config should be set correctly")
	}

	if cm.logger != logger {
		t.Error("Logger should be set correctly")
	}

	// Verify log messages
	foundLoadingMessage := false
	foundSuccessMessage := false

	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Loading ACME DNS accounts from") {
			foundLoadingMessage = true
		}
		if strings.Contains(msg, "ACME DNS accounts loaded successfully") {
			foundSuccessMessage = true
		}
	}

	if !foundLoadingMessage {
		t.Error("Expected loading message not found")
	}
	if !foundSuccessMessage {
		t.Error("Expected success message not found")
	}
}

func TestProcessManualMode_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	args := []string{
		"test-cert@example.com,www.example.com",
		"another-cert@api.example.com/key_type=rsa4096",
	}

	err = cm.ProcessManualMode(ctx, args)
	if err != nil {
		t.Fatalf("ProcessManualMode failed: %v", err)
	}

	// Verify debug message was logged
	foundDebugMessage := false
	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Mode: Manual Specification") {
			foundDebugMessage = true
			break
		}
	}
	if !foundDebugMessage {
		t.Error("Expected debug message for manual mode not found")
	}
}

func TestProcessManualMode_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	// Invalid argument format
	args := []string{"invalid-format"}

	err = cm.ProcessManualMode(ctx, args)
	if err == nil {
		t.Fatal("Expected error for invalid argument format")
	}

	if !strings.Contains(err.Error(), "parsing argument") {
		t.Errorf("Expected parsing error, got: %s", err.Error())
	}
}

func TestProcessManualMode_DuplicateCertName(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	// Duplicate certificate names
	args := []string{
		"test-cert@example.com",
		"test-cert@api.example.com",
	}

	err = cm.ProcessManualMode(ctx, args)
	if err == nil {
		t.Fatal("Expected error for duplicate certificate name")
	}

	if !strings.Contains(err.Error(), "duplicate certificate name") {
		t.Errorf("Expected duplicate name error, got: %s", err.Error())
	}
}

func TestProcessAutoMode_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	err = cm.ProcessAutoMode(ctx)
	if err != nil {
		t.Fatalf("ProcessAutoMode failed: %v", err)
	}

	// Verify mode message
	foundModeMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Mode: Automatic") {
			foundModeMessage = true
			break
		}
	}
	if !foundModeMessage {
		t.Error("Expected automatic mode message not found")
	}
}

func TestProcessAutoMode_NoCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	config.AutoDomains = nil // No auto domains
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	err = cm.ProcessAutoMode(ctx)
	if err != nil {
		t.Fatalf("ProcessAutoMode failed: %v", err)
	}

	// Verify no certificates message
	foundNoopMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "No certificates defined") {
			foundNoopMessage = true
			break
		}
	}
	if !foundNoopMessage {
		t.Error("Expected 'no certificates' message not found")
	}
}

func TestProcessAutoMode_EmptyCerts(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	config.AutoDomains = &manager.AutoDomainsConfig{
		Certs: map[string]manager.CertConfig{}, // Empty certs map
	}
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	err = cm.ProcessAutoMode(ctx)
	if err != nil {
		t.Fatalf("ProcessAutoMode failed: %v", err)
	}

	// Verify no certificates message
	foundNoopMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "No certificates defined") {
			foundNoopMessage = true
			break
		}
	}
	if !foundNoopMessage {
		t.Error("Expected 'no certificates' message not found")
	}
}

func TestParseManualRequests_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	args := []string{
		"cert1@example.com,www.example.com",
		"cert2@api.example.com/key_type=rsa4096",
		"cert3@test.com",
	}

	requests, err := cm.parseManualRequests(args)
	if err != nil {
		t.Fatalf("parseManualRequests failed: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("Expected 3 requests, got %d", len(requests))
	}

	// Check first request
	if requests[0].Name != "cert1" {
		t.Errorf("Expected name 'cert1', got '%s'", requests[0].Name)
	}
	if len(requests[0].Domains) != 2 {
		t.Errorf("Expected 2 domains for cert1, got %d", len(requests[0].Domains))
	}
	if requests[0].KeyType != "" {
		t.Errorf("Expected empty key type for cert1, got '%s'", requests[0].KeyType)
	}

	// Check second request with key type
	if requests[1].Name != "cert2" {
		t.Errorf("Expected name 'cert2', got '%s'", requests[1].Name)
	}
	if requests[1].KeyType != "rsa4096" {
		t.Errorf("Expected key type 'rsa4096', got '%s'", requests[1].KeyType)
	}

	// Verify key type debug message
	foundKeyTypeMessage := false
	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Found key_type parameter: rsa4096") {
			foundKeyTypeMessage = true
			break
		}
	}
	if !foundKeyTypeMessage {
		t.Error("Expected key type debug message not found")
	}
}

func TestParseAutoRequests_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	requests := cm.parseAutoRequests()

	if len(requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(requests))
	}

	// Find example-cert request
	var exampleCert *CertRequest
	var wildcardCert *CertRequest

	for i := range requests {
		switch requests[i].Name {
		case "example-cert":
			exampleCert = &requests[i]
		case "wildcard-cert":
			wildcardCert = &requests[i]
		}
	}

	if exampleCert == nil {
		t.Fatal("example-cert request not found")
	}
	if wildcardCert == nil {
		t.Fatal("wildcard-cert request not found")
	}

	// Check example-cert
	if len(exampleCert.Domains) != 2 {
		t.Errorf("Expected 2 domains for example-cert, got %d", len(exampleCert.Domains))
	}
	if exampleCert.KeyType != "rsa2048" {
		t.Errorf("Expected key type 'rsa2048', got '%s'", exampleCert.KeyType)
	}

	// Check wildcard-cert
	if len(wildcardCert.Domains) != 2 {
		t.Errorf("Expected 2 domains for wildcard-cert, got %d", len(wildcardCert.Domains))
	}
	if wildcardCert.KeyType != "ec256" {
		t.Errorf("Expected key type 'ec256', got '%s'", wildcardCert.KeyType)
	}

	// Verify debug messages
	foundProcessingMessage := false
	foundKeyTypeMessage := false

	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Processing 2 certificate definition(s)") {
			foundProcessingMessage = true
		}
		if strings.Contains(msg, "will use key type:") {
			foundKeyTypeMessage = true
		}
	}

	if !foundProcessingMessage {
		t.Error("Expected processing message not found")
	}
	if !foundKeyTypeMessage {
		t.Error("Expected key type message not found")
	}
}

func TestProcessRequests_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	requests := []CertRequest{
		{Name: "test1", Domains: []string{"example.com"}, KeyType: "rsa2048"},
		{Name: "test2", Domains: []string{"api.example.com"}, KeyType: "ec256"},
	}

	err = cm.processRequests(ctx, requests)
	if err != nil {
		t.Fatalf("processRequests failed: %v", err)
	}

	// Verify pre-check message
	foundPrecheckMessage := false
	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Performing pre-checks for 2 requested certificates") {
			foundPrecheckMessage = true
			break
		}
	}
	if !foundPrecheckMessage {
		t.Error("Expected pre-check message not found")
	}
}

func TestProcessRequest_InitAction(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	err = cm.processRequest(ctx, req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("processRequest failed: %v", err)
	}

	// Verify processing and action messages
	foundProcessingMessage := false
	foundActionMessage := false
	foundInitMessage := false

	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Processing certificate: test-cert") {
			foundProcessingMessage = true
		}
	}

	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Certificate test-cert requires action: init") {
			foundActionMessage = true
		}
		if strings.Contains(msg, "Initializing certificate test-cert") {
			foundInitMessage = true
		}
	}

	if !foundProcessingMessage {
		t.Error("Expected processing message not found")
	}
	if !foundActionMessage {
		t.Error("Expected action message not found")
	}
	if !foundInitMessage {
		t.Error("Expected init message not found")
	}
}

func TestDetermineAction_NewCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "init" {
		t.Errorf("Expected action 'init' for new certificate, got '%s'", action)
	}

	// Verify debug message about missing metadata
	found := false
	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Certificate metadata not found") && strings.Contains(msg, "initializing new certificate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected debug message about missing metadata not found")
	}
}

func TestDetermineAction_MissingCertificateFile(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create only the metadata file without the actual certificate file
	certDir := filepath.Join(tmpDir, "certificates")
	err = os.MkdirAll(certDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	metadataPath := filepath.Join(certDir, "test-cert.json")
	err = os.WriteFile(metadataPath, []byte(`{"domain": "example.com"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "init" {
		t.Errorf("Expected action 'init' for missing certificate file, got '%s'", action)
	}

	// Verify debug message about missing certificate file
	found := false
	for _, msg := range logger.debugMessages {
		if strings.Contains(msg, "Certificate file not found") && strings.Contains(msg, "initializing new certificate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected debug message about missing certificate file not found")
	}
}

func TestDetermineAction_ValidCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create a valid certificate that doesn't need renewal
	err = createTestCertificateFiles(tmpDir, "test-cert", []string{"example.com"}, 90) // 90 days validity
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "skip" {
		t.Errorf("Expected action 'skip' for valid certificate, got '%s'", action)
	}

	// Verify info message about valid certificate
	found := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "is valid and doesn't need renewal") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected info message about valid certificate not found")
	}
}

func TestDetermineAction_ExpiredCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create an expired certificate
	err = createTestCertificateFiles(tmpDir, "test-cert", []string{"example.com"}, -1) // Expired 1 day ago
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "renew" {
		t.Errorf("Expected action 'renew' for expired certificate, got '%s'", action)
	}

	// Verify info message about renewal
	found := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "needs renewal") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected info message about renewal not found")
	}
}

func TestDetermineAction_CertificateNearExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create a certificate that expires within renewal threshold (30 days)
	err = createTestCertificateFiles(tmpDir, "test-cert", []string{"example.com"}, 15) // 15 days validity
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "renew" {
		t.Errorf("Expected action 'renew' for certificate near expiry, got '%s'", action)
	}

	// Verify info message about renewal
	found := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "needs renewal") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected info message about renewal not found")
	}
}

func TestDetermineAction_DomainChanges(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create a certificate with one domain
	err = createTestCertificateFiles(tmpDir, "test-cert", []string{"example.com"}, 90) // 90 days validity
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Request different domains
	req := CertRequest{Name: "test-cert", Domains: []string{"example.com", "www.example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	if action != "renew" {
		t.Errorf("Expected action 'renew' for domain changes, got '%s'", action)
	}

	// Verify info message about renewal
	found := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "needs renewal") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected info message about renewal not found")
	}
}

func TestDetermineAction_InvalidRenewalThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	// Pass invalid renewal threshold type
	_, err = cm.determineAction(req, "invalid-threshold")
	if err == nil {
		t.Error("Expected error for invalid renewal threshold type")
	}

	expectedErr := "invalid renewal threshold type"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestDetermineAction_CertificateCheckError(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create a malformed certificate file
	certDir := filepath.Join(tmpDir, "certificates")
	err = os.MkdirAll(certDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	certPath := filepath.Join(certDir, "test-cert.crt")
	err = os.WriteFile(certPath, []byte("invalid-certificate-content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write certificate file: %v", err)
	}

	// Create metadata file
	metadataPath := filepath.Join(certDir, "test-cert.json")
	err = os.WriteFile(metadataPath, []byte(`{"domain": "example.com"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}

	req := CertRequest{Name: "test-cert", Domains: []string{"example.com"}, KeyType: "rsa2048"}

	action, err := cm.determineAction(req, config.GetRenewalThreshold())
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	// Should return "renew" when certificate check fails
	if action != "renew" {
		t.Errorf("Expected action 'renew' for certificate check error, got '%s'", action)
	}

	// Verify warning message about certificate check error
	found := false
	for _, msg := range logger.warnMessages {
		if strings.Contains(msg, "Error checking certificate renewal status") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning message about certificate check error not found")
	}
}

// generateTestCertificate creates a self-signed certificate for testing
func generateTestCertificate(commonName string, dnsNames []string, notBefore, notAfter time.Time) ([]byte, []byte, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
	}

	// Add IP addresses if domains contain IP addresses
	for _, domain := range dnsNames {
		if ip := net.ParseIP(domain); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return certPEM, keyPEM, nil
}

// Helper function to create test certificate files
func createTestCertificateFiles(baseDir, certName string, domains []string, validityDays int) error {
	certDir := filepath.Join(baseDir, "certificates")
	err := os.MkdirAll(certDir, 0755)
	if err != nil {
		return err
	}

	// Generate certificate with specific validity period
	notBefore := time.Now().Add(-24 * time.Hour)
	notAfter := time.Now().Add(time.Duration(validityDays) * 24 * time.Hour)

	// Create a realistic test certificate using the existing manager function
	// This creates valid certificate files that work with CertificateNeedsRenewal
	certPEM, keyPEM, err := generateTestCertificate(domains[0], domains, notBefore, notAfter)
	if err != nil {
		return err
	}

	// Write certificate file
	certPath := filepath.Join(certDir, certName+".crt")
	err = os.WriteFile(certPath, certPEM, 0644)
	if err != nil {
		return err
	}

	// Write key file
	keyPath := filepath.Join(certDir, certName+".key")
	err = os.WriteFile(keyPath, keyPEM, 0600)
	if err != nil {
		return err
	}

	// Create metadata file
	metadata := fmt.Sprintf(`{
		"domain": "%s",
		"certUrl": "https://example.com/cert",
		"certStableUrl": "https://example.com/cert",
		"privateKey": "%s",
		"certificate": "%s"
	}`, domains[0],
		strings.ReplaceAll(string(keyPEM), "\n", "\\n"),
		strings.ReplaceAll(string(certPEM), "\n", "\\n"))

	metadataPath := filepath.Join(certDir, certName+".json")
	err = os.WriteFile(metadataPath, []byte(metadata), 0644)
	if err != nil {
		return err
	}

	return nil
}

func TestInitCertificate_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	req := CertRequest{Name: "test-cert", Domains: []string{"example.com", "www.example.com"}, KeyType: "rsa2048"}

	err = cm.initCertificate(ctx, req)
	if err != nil {
		t.Fatalf("initCertificate failed: %v", err)
	}

	// Verify init message
	foundInitMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Initializing certificate test-cert for domains [example.com www.example.com]") {
			foundInitMessage = true
			break
		}
	}
	if !foundInitMessage {
		t.Error("Expected init message not found")
	}
}

func TestRenewCertificate_Placeholder(t *testing.T) {
	tmpDir := t.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	ctx := context.Background()
	req := CertRequest{Name: "test-cert", Domains: []string{"example.com", "www.example.com"}, KeyType: "rsa2048"}

	err = cm.renewCertificate(ctx, req)
	if err != nil {
		t.Fatalf("renewCertificate failed: %v", err)
	}

	// Verify renew message
	foundRenewMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Renewing certificate test-cert for domains [example.com www.example.com]") {
			foundRenewMessage = true
			break
		}
	}
	if !foundRenewMessage {
		t.Error("Expected renew message not found")
	}
}

// Note: The current implementation of determineAction always returns "init"
// Tests for other actions (renew, skip) would require a more sophisticated implementation
// These tests focus on the current functionality

// Benchmark tests
func BenchmarkParseManualRequests(b *testing.B) {
	tmpDir := b.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		b.Fatalf("Failed to create certificate manager: %v", err)
	}

	args := []string{
		"cert1@example.com,www.example.com",
		"cert2@api.example.com/key_type=rsa4096",
		"cert3@test.com/key_type=ec256",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.parseManualRequests(args)
	}
}

func BenchmarkParseAutoRequests(b *testing.B) {
	tmpDir := b.TempDir()
	config := createTestConfig(tmpDir)
	logger := &mockLogger{}

	cm, err := NewCertificateManager(config, logger)
	if err != nil {
		b.Fatalf("Failed to create certificate manager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.parseAutoRequests()
	}
}
