package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewClient tests creating a new mTLS client
func TestNewClient(t *testing.T) {
	// Create temporary directory for test certs
	tmpDir, err := os.MkdirTemp("", "mtls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create config
	config := &Config{
		CertPath:   filepath.Join(tmpDir, "client.crt"),
		KeyPath:    filepath.Join(tmpDir, "client.key"),
		CAPath:     filepath.Join(tmpDir, "ca.crt"),
		ServerName: "test.example.com",
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Verify client is created
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	// Verify TLS config
	tlsConfig := client.GetTLSConfig()
	if tlsConfig == nil {
		t.Fatal("GetTLSConfig returned nil")
	}

	// Verify certificates are loaded
	if len(tlsConfig.Certificates) == 0 {
		t.Fatal("No certificates loaded")
	}
}

// TestLoadConfigFromDir tests loading config from directory
func TestLoadConfigFromDir(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Load config
	config, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir failed: %v", err)
	}

	// Verify config
	if config == nil {
		t.Fatal("LoadConfigFromDir returned nil")
	}

	// Verify paths
	if config.CertPath == "" {
		t.Error("CertPath is empty")
	}
	if config.KeyPath == "" {
		t.Error("KeyPath is empty")
	}
	if config.CAPath == "" {
		t.Error("CAPath is empty")
	}

	// Verify file existence
	if _, err := os.Stat(config.CertPath); err != nil {
		t.Errorf("Client cert file does not exist: %v", err)
	}
	if _, err := os.Stat(config.KeyPath); err != nil {
		t.Errorf("Client key file does not exist: %v", err)
	}
	if _, err := os.Stat(config.CAPath); err != nil {
		t.Errorf("CA cert file does not exist: %v", err)
	}
}

// TestLoadConfigFromDir_NonExistent tests loading config from non-existent directory
func TestLoadConfigFromDir_NonExistent(t *testing.T) {
	_, err := LoadConfigFromDir("/non/existent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

// TestGetCertificateInfo tests getting certificate information
func TestGetCertificateInfo(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-info-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create config
	config := &Config{
		CertPath: filepath.Join(tmpDir, "client.crt"),
		KeyPath:  filepath.Join(tmpDir, "client.key"),
		CAPath:   filepath.Join(tmpDir, "ca.crt"),
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Get certificate info
	info, err := client.GetCertificateInfo()
	if err != nil {
		t.Fatalf("GetCertificateInfo failed: %v", err)
	}

	// Verify info
	if info == nil {
		t.Fatal("GetCertificateInfo returned nil")
	}
	if info.Subject == "" {
		t.Error("Subject is empty")
	}
	if info.Issuer == "" {
		t.Error("Issuer is empty")
	}
	if info.SerialNumber == "" {
		t.Error("SerialNumber is empty")
	}
	if info.Hash == "" {
		t.Error("Hash is empty")
	}
}

// TestCheckValidity tests certificate validity checking
func TestCheckValidity(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-validity-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create config
	config := &Config{
		CertPath: filepath.Join(tmpDir, "client.crt"),
		KeyPath:  filepath.Join(tmpDir, "client.key"),
		CAPath:   filepath.Join(tmpDir, "ca.crt"),
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Check validity
	err = client.CheckValidity()
	if err != nil {
		t.Errorf("CheckValidity failed: %v", err)
	}
}

// TestGetDaysUntilExpiry tests getting days until expiry
func TestGetDaysUntilExpiry(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-expiry-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create config
	config := &Config{
		CertPath: filepath.Join(tmpDir, "client.crt"),
		KeyPath:  filepath.Join(tmpDir, "client.key"),
		CAPath:   filepath.Join(tmpDir, "ca.crt"),
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Get days until expiry
	days, err := client.GetDaysUntilExpiry()
	if err != nil {
		t.Fatalf("GetDaysUntilExpiry failed: %v", err)
	}

	// Verify days
	if days < 0 {
		t.Errorf("Days until expiry is negative: %d", days)
	}
	if days > 365 {
		t.Errorf("Days until expiry is unexpectedly large: %d", days)
	}

	t.Logf("Certificate expires in %d days", days)
}

// TestReloadCertificates tests reloading certificates
func TestReloadCertificates(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-reload-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create config
	config := &Config{
		CertPath: filepath.Join(tmpDir, "client.crt"),
		KeyPath:  filepath.Join(tmpDir, "client.key"),
		CAPath:   filepath.Join(tmpDir, "ca.crt"),
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Get initial hash
	initialHash := client.CertificateHash()
	if initialHash == "" {
		t.Error("Initial hash is empty")
	}

	// Reload certificates
	err = client.ReloadCertificates()
	if err != nil {
		t.Errorf("ReloadCertificates failed: %v", err)
	}

	// Get new hash
	newHash := client.CertificateHash()
	if newHash == "" {
		t.Error("New hash is empty")
	}
}

// TestEnsureConfigDir tests ensuring config directory exists
func TestEnsureConfigDir(t *testing.T) {
	// Save current home directory
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	// Create temporary home
	tmpHome, err := os.MkdirTemp("", "mtls-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Set temporary home
	os.Setenv("HOME", tmpHome)

	// Ensure config dir
	dir, err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Verify directory exists
	if info, err := os.Stat(dir); err != nil {
		t.Errorf("Config directory does not exist: %v", err)
	} else if !info.IsDir() {
		t.Error("Config path is not a directory")
	}

	// Verify default path
	expectedDir := filepath.Join(tmpHome, ".px-plugin", "certs")
	if dir != expectedDir {
		t.Errorf("Expected dir %s, got %s", expectedDir, dir)
	}
}

// TestDefaultConfig tests creating default config
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Verify defaults
	if config.MinVersion == 0 {
		t.Error("MinVersion is zero")
	}
	if config.MaxVersion == 0 {
		t.Error("MaxVersion is zero")
	}
	if config.RotationCheck == 0 {
		t.Error("RotationCheck is zero")
	}
	if !config.AutoRotate {
		t.Error("AutoRotate is false")
	}
}

// generateTestCerts generates test certificates for testing
func generateTestCerts(dir string) error {
	// Generate CA
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2023),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Generate CA private key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create CA certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// Save CA certificate
	caCertOut, err := os.Create(filepath.Join(dir, "ca.crt"))
	if err != nil {
		return err
	}
	defer caCertOut.Close()
	pem.Encode(caCertOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})

	// Generate client certificate
	client := &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization:  []string{"Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Generate client private key
	clientPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create client certificate
	clientBytes, err := x509.CreateCertificate(rand.Reader, client, ca, &clientPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// Save client certificate
	clientCertOut, err := os.Create(filepath.Join(dir, "client.crt"))
	if err != nil {
		return err
	}
	defer clientCertOut.Close()
	pem.Encode(clientCertOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientBytes})

	// Save client private key
	clientKeyOut, err := os.Create(filepath.Join(dir, "client.key"))
	if err != nil {
		return err
	}
	defer clientKeyOut.Close()
	clientKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientPrivKey)}
	pem.Encode(clientKeyOut, clientKeyPEM)

	return nil
}

// TestLoadConfigFromDir_MissingFiles tests error handling for missing files
func TestLoadConfigFromDir_MissingFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mtls-missing-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to load config without certificates
	_, err = LoadConfigFromDir(tmpDir)
	if err == nil {
		t.Error("Expected error for missing certificates, got nil")
	}
}

// TestNewClient_InvalidConfig tests error handling for invalid config
func TestNewClient_InvalidConfig(t *testing.T) {
	// Test nil config
	_, err := NewClient(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}

	// Test non-existent cert path
	config := &Config{
		CertPath: "/non/existent/cert.crt",
		KeyPath:  "/non/existent/key.key",
		CAPath:   "/non/existent/ca.crt",
	}
	_, err = NewClient(config)
	if err == nil {
		t.Error("Expected error for non-existent certs, got nil")
	}
}
