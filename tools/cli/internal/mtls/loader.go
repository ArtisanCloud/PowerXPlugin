package mtls

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ConfigFromPaths creates a Config using explicit certificate paths.
func ConfigFromPaths(certPath, keyPath, caPath string) (*Config, error) {
	certPath = expandPath(certPath)
	keyPath = expandPath(keyPath)
	caPath = expandPath(caPath)

	switch {
	case certPath == "":
		return nil, fmt.Errorf("mtls cert path is required")
	case keyPath == "":
		return nil, fmt.Errorf("mtls key path is required")
	case caPath == "":
		return nil, fmt.Errorf("mtls CA path is required")
	}

	cfg := DefaultConfig()
	cfg.CertPath = certPath
	cfg.KeyPath = keyPath
	cfg.CAPath = caPath
	return cfg, nil
}

// LoadConfigFromEnv loads an mTLS config from PX_MTLS_* environment variables.
// The second return value indicates whether env variables were present.
func LoadConfigFromEnv() (*Config, bool, error) {
	var (
		certPath = strings.TrimSpace(os.Getenv("PX_MTLS_CERT_PATH"))
		keyPath  = strings.TrimSpace(os.Getenv("PX_MTLS_KEY_PATH"))
		caPath   = strings.TrimSpace(os.Getenv("PX_MTLS_CA_PATH"))
		certDir  = strings.TrimSpace(os.Getenv("PX_MTLS_CERT_DIR"))
	)

	if certDir != "" {
		certDir = expandPath(certDir)
		if certPath == "" {
			certPath = filepath.Join(certDir, "client.crt")
		}
		if keyPath == "" {
			keyPath = filepath.Join(certDir, "client.key")
		}
		if caPath == "" {
			caPath = filepath.Join(certDir, "ca.crt")
		}
	}

	if certPath == "" && keyPath == "" && caPath == "" {
		return nil, false, nil
	}

	cfg, err := ConfigFromPaths(certPath, keyPath, caPath)
	if err != nil {
		return nil, true, err
	}

	if serverName := strings.TrimSpace(os.Getenv("PX_MTLS_SERVER_NAME")); serverName != "" {
		cfg.ServerName = serverName
	}

	if skip := strings.TrimSpace(os.Getenv("PX_MTLS_SKIP_VERIFY")); skip != "" {
		if val, err := strconv.ParseBool(skip); err == nil {
			cfg.InsecureSkipVerify = val
		}
	}

	if auto := strings.TrimSpace(os.Getenv("PX_MTLS_AUTO_ROTATE")); auto != "" {
		if val, err := strconv.ParseBool(auto); err == nil {
			cfg.AutoRotate = val
		}
	}

	if interval := strings.TrimSpace(os.Getenv("PX_MTLS_ROTATION_CHECK")); interval != "" {
		if dur, err := time.ParseDuration(interval); err == nil {
			cfg.RotationCheck = dur
		}
	}

	return cfg, true, nil
}

// TryLoadConfigFromDefaultDir attempts to load certs from ~/.px-plugin/certs.
func TryLoadConfigFromDefaultDir() (*Config, error) {
	dir := GetDefaultConfigPath()
	cfg, err := LoadConfigFromDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return os.ExpandEnv(path)
}
