package mtls

import (
	"fmt"
	"os"
	"path/filepath"
)

// Setup creates the mTLS configuration directory and example files
func Setup() error {
	// Ensure config directory exists
	dir, err := EnsureConfigDir()
	if err != nil {
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}

	// Check if certificates already exist
	certPath := filepath.Join(dir, "client.crt")
	if _, err := os.Stat(certPath); err == nil {
		// Certificates already exist
		return nil
	}

	fmt.Println("Setting up mTLS configuration...")
	fmt.Printf("mTLS config directory: %s\n", dir)
	fmt.Println()
	fmt.Println("To set up mTLS, you need to:")
	fmt.Println("1. Obtain a client certificate and key from your CA")
	fmt.Println("2. Obtain the CA certificate")
	fmt.Println("3. Place the files in the config directory:")
	fmt.Printf("   - Client certificate: %s\n", filepath.Join(dir, "client.crt"))
	fmt.Printf("   - Client key:         %s\n", filepath.Join(dir, "client.key"))
	fmt.Printf("   - CA certificate:     %s\n", filepath.Join(dir, "ca.crt"))
	fmt.Println()
	fmt.Println("For testing, you can generate self-signed certificates:")
	fmt.Println("  openssl req -x509 -newkey rsa:2048 -keyout client.key -out client.crt -days 365 -nodes")
	fmt.Println("  openssl req -x509 -newkey rsa:2048 -keyout ca.key -out ca.crt -days 365 -nodes -subj \"/CN=Test CA\"")

	return nil
}

// VerifySetup verifies that the mTLS configuration is properly set up
func VerifySetup() error {
	// Try to load config
	config, err := LoadConfigFromDir(GetDefaultConfigPath())
	if err != nil {
		return fmt.Errorf("mTLS not configured: %w", err)
	}

	// Create a temporary client to verify certificates
	_, err = NewClient(config)
	if err != nil {
		return fmt.Errorf("mTLS configuration is invalid: %w", err)
	}

	fmt.Println("mTLS configuration is valid!")
	return nil
}

// PrintInfo prints mTLS configuration information
func PrintInfo() error {
	dir := GetDefaultConfigPath()
	if dir == "" {
		return fmt.Errorf("cannot determine mTLS config directory")
	}

	fmt.Printf("mTLS Configuration Directory: %s\n", dir)
	fmt.Println()

	// List files
	files := []string{"client.crt", "client.key", "ca.crt"}
	for _, file := range files {
		path := filepath.Join(dir, file)
		if info, err := os.Stat(path); err != nil {
			fmt.Printf("  %s: NOT FOUND\n", file)
		} else {
			fmt.Printf("  %s: EXISTS (%d bytes)\n", file, info.Size())
		}
	}

	return nil
}
