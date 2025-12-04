package devapi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/powerx-plugin/cli/internal/mtls"
)

// TestDevClient_MTLS tests mTLS integration with DevClient
func TestDevClient_MTLS(t *testing.T) {
	// Create temporary directory for test certs
	tmpDir, err := os.MkdirTemp("", "devapi-mtls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create mTLS config
	mtlsConfig := &mtls.Config{
		CertPath:   filepath.Join(tmpDir, "client.crt"),
		KeyPath:    filepath.Join(tmpDir, "client.key"),
		CAPath:     filepath.Join(tmpDir, "ca.crt"),
		ServerName: "test.example.com",
	}

	// Create mTLS client
	mtlsClient, err := mtls.NewClient(mtlsConfig)
	if err != nil {
		t.Fatalf("Failed to create mTLS client: %v", err)
	}
	defer mtlsClient.Close()

	// Verify TLS config available
	if mtlsClient.GetTLSConfig() == nil {
		t.Fatal("expected TLS config to be initialized")
	}

	// Get certificate info
	info, err := mtlsClient.GetCertificateInfo()
	if err != nil {
		t.Fatalf("GetCertificateInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Certificate info is nil")
	}
	t.Logf("Certificate: %s", info.Subject)

	// Check certificate validity
	err = mtlsClient.CheckValidity()
	if err != nil {
		t.Errorf("Certificate validity check failed: %v", err)
	}
}

// TestDevClient_WithMTLS tests DevClient with mTLS
func TestDevClient_WithMTLS(t *testing.T) {
	// Create temporary directory for test certs
	tmpDir, err := os.MkdirTemp("", "devapi-mtls-client-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificates
	err = generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Create DevClient with mTLS
	opts := ClientOptions{
		BaseURL:        "https://api.example.com",
		MTLSCertPath:   filepath.Join(tmpDir, "client.crt"),
		MTLSKeyPath:    filepath.Join(tmpDir, "client.key"),
		MTLSCACertPath: filepath.Join(tmpDir, "ca.crt"),
		Timeout:        5 * time.Second,
	}

	client := NewClient(opts)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	defer client.Close()

	// Verify mTLS is enabled
	if !client.IsMTLSEnabled() {
		t.Error("Expected mTLS to be enabled")
	}

	// Get mTLS info
	info, err := client.GetMTLSInfo()
	if err != nil {
		t.Fatalf("GetMTLSInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("mTLS info is nil")
	}
	t.Logf("mTLS Certificate: %s", info.Subject)

	// Check certificate
	err = client.CheckMTLSCertificate()
	if err != nil {
		t.Errorf("mTLS certificate check failed: %v", err)
	}
}

// TestDevClient_WithoutMTLS tests DevClient without mTLS
func TestDevClient_WithoutMTLS(t *testing.T) {
	// Create DevClient without mTLS
	opts := ClientOptions{
		BaseURL: "https://api.example.com",
		Timeout: 5 * time.Second,
	}

	client := NewClient(opts)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	defer client.Close()

	// Verify mTLS is not enabled
	if client.IsMTLSEnabled() {
		t.Error("Expected mTLS to be disabled")
	}

	// Get mTLS info should fail
	_, err := client.GetMTLSInfo()
	if err == nil {
		t.Error("Expected error when mTLS is disabled, got nil")
	}
}

// TestMTLS_Setup tests the mTLS setup function
func TestMTLS_Setup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Test setup creates directory
	err := mtls.Setup()
	if err != nil {
		t.Errorf("Setup failed: %v", err)
	}

	// Test setup can be called multiple times
	err = mtls.Setup()
	if err != nil {
		t.Errorf("Setup failed on second call: %v", err)
	}
}

// TestMTLS_PrintInfo tests the mTLS info printing
func TestMTLS_PrintInfo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Setup first
	err := mtls.Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Print info should work even without certs
	err = mtls.PrintInfo()
	if err != nil {
		t.Errorf("PrintInfo failed: %v", err)
	}
}

// TestMTLS_VerifySetup tests the mTLS setup verification
func TestMTLS_VerifySetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Setup first
	err := mtls.Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Verify setup should fail without certs
	err = mtls.VerifySetup()
	if err == nil {
		t.Error("Expected verification to fail without certs, got nil")
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
