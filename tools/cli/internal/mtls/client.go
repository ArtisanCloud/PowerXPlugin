package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config holds mTLS configuration
type Config struct {
	// Certificate paths
	CertPath string
	KeyPath  string
	CAPath   string

	// TLS configuration
	ServerName         string
	InsecureSkipVerify bool
	MinVersion         uint16
	MaxVersion         uint16

	// Certificate rotation
	AutoRotate      bool
	RotationCheck   time.Duration
	LastLoadTime    time.Time
	CertificateHash string
}

// Client handles mTLS client configuration
type Client struct {
	config    *Config
	tlsConfig *tls.Config
	cert      tls.Certificate
	caPool    *x509.CertPool
	mu        sync.RWMutex
	rotator   *Rotator
}

// NewClient creates a new mTLS client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("mTLS config is required")
	}

	client := &Client{
		config: config,
	}
	client.rotator = &Rotator{
		client: client,
		stop:   make(chan struct{}),
	}

	// Load certificates
	if err := client.loadCertificates(); err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	// Create TLS config
	if err := client.buildTLSConfig(); err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Start certificate rotator if enabled
	if config.AutoRotate {
		go client.rotator.start()
	}

	return client, nil
}

// Close stops the certificate rotator
func (c *Client) Close() {
	if c.rotator != nil {
		close(c.rotator.stop)
		c.rotator = nil
	}
}

// GetTLSConfig returns the TLS configuration
func (c *Client) GetTLSConfig() *tls.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsConfig
}

// ReloadCertificates reloads certificates from disk
func (c *Client) ReloadCertificates() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Load certificates
	if err := c.loadCertificates(); err != nil {
		return fmt.Errorf("failed to reload certificates: %w", err)
	}

	// Rebuild TLS config
	if err := c.buildTLSConfig(); err != nil {
		return fmt.Errorf("failed to rebuild TLS config: %w", err)
	}

	return nil
}

// GetCertificateInfo returns certificate information
func (c *Client) GetCertificateInfo() (*CertInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cert.Leaf == nil {
		// Parse certificate if not already parsed
		leaf, err := x509.ParseCertificate(c.cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		c.cert.Leaf = leaf
	}

	return &CertInfo{
		Subject:      c.cert.Leaf.Subject.String(),
		Issuer:       c.cert.Leaf.Issuer.String(),
		NotBefore:    c.cert.Leaf.NotBefore,
		NotAfter:     c.cert.Leaf.NotAfter,
		DNSNames:     c.cert.Leaf.DNSNames,
		SerialNumber: c.cert.Leaf.SerialNumber.String(),
		Hash:         c.CertificateHash(),
	}, nil
}

// CertificateHash returns the hash of the current certificate
func (c *Client) CertificateHash() string {
	return c.config.CertificateHash
}

// loadCertificates loads client certificate, key, and CA
func (c *Client) loadCertificates() error {
	var err error

	// Load client certificate and key
	if c.config.CertPath != "" && c.config.KeyPath != "" {
		c.cert, err = tls.LoadX509KeyPair(c.config.CertPath, c.config.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load client cert/key: %w", err)
		}
	}

	// Load CA certificate
	if c.config.CAPath != "" {
		c.caPool = x509.NewCertPool()
		caCert, err := os.ReadFile(c.config.CAPath)
		if err != nil {
			return fmt.Errorf("failed to read CA cert: %w", err)
		}

		if !c.caPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA cert")
		}
	}

	// Calculate certificate hash
	if len(c.cert.Certificate) > 0 {
		c.config.CertificateHash = hashCertificate(c.cert.Certificate[0])
		c.config.LastLoadTime = time.Now()
	}

	return nil
}

// buildTLSConfig builds the TLS configuration
func (c *Client) buildTLSConfig() error {
	config := &tls.Config{
		Certificates: []tls.Certificate{c.cert},
		MinVersion:   c.config.MinVersion,
		MaxVersion:   c.config.MaxVersion,
		ServerName:   c.config.ServerName,
	}

	// Set CA pool if loaded
	if c.caPool != nil {
		config.RootCAs = c.caPool
	}

	// Handle insecure skip verify
	if c.config.InsecureSkipVerify {
		config.InsecureSkipVerify = true
		config.VerifyConnection = nil
	} else {
		// Custom verification
		config.VerifyConnection = c.verifyConnection
	}

	c.tlsConfig = config
	return nil
}

// verifyConnection verifies the TLS connection
func (c *Client) verifyConnection(state tls.ConnectionState) error {
	// Verify peer certificate
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates presented")
	}

	// Verify certificate chain
	opts := x509.VerifyOptions{
		DNSName: c.config.ServerName,
		Roots:   c.caPool,
	}

	if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	// Check expiration
	now := time.Now()
	for _, cert := range state.PeerCertificates {
		if now.Before(cert.NotBefore) {
			return fmt.Errorf("certificate is not yet valid")
		}
		if now.After(cert.NotAfter) {
			return fmt.Errorf("certificate has expired")
		}
	}

	return nil
}

// Rotator handles automatic certificate rotation
type Rotator struct {
	client *Client
	stop   chan struct{}
}

// start starts the certificate rotation monitor
func (r *Rotator) start() {
	ticker := time.NewTicker(r.client.config.RotationCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.checkAndRotate()
		case <-r.stop:
			return
		}
	}
}

// checkAndRotate checks if certificates have changed and rotates if necessary
func (r *Rotator) checkAndRotate() {
	// Get current certificate hash
	currentHash := r.client.CertificateHash()

	// Reload certificate to check for changes
	newCert, err := tls.LoadX509KeyPair(r.client.config.CertPath, r.client.config.KeyPath)
	if err != nil {
		fmt.Printf("Warning: failed to load certificate for rotation check: %v\n", err)
		return
	}

	// Calculate new hash
	newHash := hashCertificate(newCert.Certificate[0])

	// Check if certificate has changed
	if newHash != currentHash {
		fmt.Println("Certificate rotation detected, reloading...")

		if err := r.client.ReloadCertificates(); err != nil {
			fmt.Printf("Warning: failed to reload rotated certificate: %v\n", err)
		} else {
			fmt.Println("Certificate rotated successfully")
		}
	}
}

// CertInfo holds certificate information
type CertInfo struct {
	Subject      string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	DNSNames     []string
	SerialNumber string
	Hash         string
}

// DefaultConfig returns a default mTLS configuration
func DefaultConfig() *Config {
	return &Config{
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		RotationCheck:      5 * time.Minute,
		AutoRotate:         true,
		InsecureSkipVerify: false,
	}
}

// LoadConfigFromDir loads mTLS configuration from a directory
func LoadConfigFromDir(dir string) (*Config, error) {
	config := DefaultConfig()

	// Check if directory exists
	if info, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("mTLS config directory does not exist: %s", dir)
		}
		return nil, fmt.Errorf("failed to stat mTLS config directory: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("mTLS config path is not a directory: %s", dir)
	}

	// Load paths
	config.CertPath = filepath.Join(dir, "client.crt")
	config.KeyPath = filepath.Join(dir, "client.key")
	config.CAPath = filepath.Join(dir, "ca.crt")

	// Check if files exist
	if _, err := os.Stat(config.CertPath); err != nil {
		return nil, fmt.Errorf("client certificate not found: %w", err)
	}
	if _, err := os.Stat(config.KeyPath); err != nil {
		return nil, fmt.Errorf("client key not found: %w", err)
	}
	if _, err := os.Stat(config.CAPath); err != nil {
		return nil, fmt.Errorf("CA certificate not found: %w", err)
	}

	return config, nil
}

// hashCertificate calculates a hash of the certificate
func hashCertificate(certDER []byte) string {
	// Simple hash for demonstration
	// In production, use a proper cryptographic hash
	hash := 0
	for _, b := range certDER {
		hash = hash*31 + int(b)
	}
	return fmt.Sprintf("%d", hash)
}

// GetDefaultConfigPath returns the default mTLS config path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".px-plugin", "certs")
}

// EnsureConfigDir creates the mTLS config directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	dir := GetDefaultConfigPath()
	if dir == "" {
		return "", fmt.Errorf("failed to get home directory")
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create mTLS config directory: %w", err)
	}

	return dir, nil
}

// CheckValidity checks if the current certificate is valid
func (c *Client) CheckValidity() error {
	info, err := c.GetCertificateInfo()
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %w", err)
	}

	now := time.Now()
	if now.Before(info.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", info.NotBefore.Format(time.RFC3339))
	}
	if now.After(info.NotAfter) {
		return fmt.Errorf("certificate has expired (expired at %s)", info.NotAfter.Format(time.RFC3339))
	}

	// Check if certificate expires soon (within 7 days)
	expiryThreshold := now.Add(7 * 24 * time.Hour)
	if info.NotAfter.Before(expiryThreshold) {
		return fmt.Errorf("certificate expires soon (at %s)", info.NotAfter.Format(time.RFC3339))
	}

	return nil
}

// GetDaysUntilExpiry returns the number of days until the certificate expires
func (c *Client) GetDaysUntilExpiry() (int, error) {
	info, err := c.GetCertificateInfo()
	if err != nil {
		return 0, fmt.Errorf("failed to get certificate info: %w", err)
	}

	now := time.Now()
	expiry := info.NotAfter
	duration := expiry.Sub(now)
	days := int(duration.Hours() / 24)

	return days, nil
}
