package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type iamConfig struct {
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
		Schema string `yaml:"schema"`
	} `yaml:"database"`
	Context struct {
		ProviderMode string `yaml:"provider_mode"`
	} `yaml:"context"`
	baseDir string
}

var schemaNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func resolveEntryPath(entry string) (string, error) {
	if entry == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determine working directory: %w", err)
		}
		entry = cwd
	}

	absPath, err := filepath.Abs(entry)
	if err != nil {
		return "", fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("entry path invalid: %w", err)
	}
	return absPath, nil
}

func resolveIAMConfigPath(entry, override string) (string, error) {
	if override != "" {
		path := override
		if !filepath.IsAbs(path) {
			path = filepath.Join(entry, path)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("config path invalid: %w", err)
		}
		return abs, nil
	}

	candidates := []string{
		filepath.Join(entry, "backend", "etc", "config.yaml"),
		filepath.Join(entry, "backend", "etc", "config.example.yaml"),
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return cand, nil
		}
	}
	return "", errors.New("backend/etc/config.yaml not found (set --config to override)")
}

func loadIAMConfig(path string) (*iamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := &iamConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.baseDir = filepath.Dir(path)
	return cfg, nil
}

func connectIAMDatabase(cfg *iamConfig) (*gorm.DB, *sql.DB, string, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	dsn := strings.TrimSpace(cfg.Database.DSN)

	var dialector gorm.Dialector
	var normalizedDriver string
	var err error

	switch driver {
	case "", "postgres", "postgresql", "pgx":
		if dsn == "" {
			return nil, nil, "", errors.New("database.dsn is required for postgres driver")
		}
		dialector = postgres.New(postgres.Config{DSN: dsn})
		normalizedDriver = "postgres"
	case "sqlite", "sqlite3":
		if dsn == "" {
			dsn = "file:powerx-plugin.db?cache=shared&_fk=1"
		}
		dsn, err = normalizeSQLiteDSN(dsn, cfg.baseDir)
		if err != nil {
			return nil, nil, "", err
		}
		dialector = sqlite.Open(dsn)
		normalizedDriver = "sqlite"
	case "memory":
		if dsn == "" {
			dsn = "file::memory:?cache=shared"
		}
		dialector = sqlite.Open(dsn)
		normalizedDriver = "sqlite"
	default:
		return nil, nil, "", fmt.Errorf("unsupported database driver: %s", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		return nil, nil, "", fmt.Errorf("connect database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, "", fmt.Errorf("wrap database: %w", err)
	}
	return db, sqlDB, normalizedDriver, nil
}

func normalizeSQLiteDSN(dsn, baseDir string) (string, error) {
	if strings.HasPrefix(dsn, "file:") {
		rest := strings.TrimPrefix(dsn, "file:")
		if strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "/") || strings.HasPrefix(rest, "\\") {
			return dsn, nil
		}
		parts := strings.SplitN(rest, "?", 2)
		pathPart := parts[0]
		abs := filepath.Clean(filepath.Join(baseDir, pathPart))
		if len(parts) == 2 {
			return fmt.Sprintf("file:%s?%s", abs, parts[1]), nil
		}
		return "file:" + abs, nil
	}

	if filepath.IsAbs(dsn) {
		return dsn, nil
	}
	abs := filepath.Clean(filepath.Join(baseDir, dsn))
	return abs, nil
}

func applySchemaSearchPath(db *gorm.DB, driver, schema string) error {
	schema = strings.TrimSpace(schema)
	if schema == "" || driver != "postgres" {
		return nil
	}
	if !schemaNameRe.MatchString(schema) {
		return fmt.Errorf("invalid schema name: %s", schema)
	}
	stmt := fmt.Sprintf(`SET search_path TO "%s"`, schema)
	return db.Exec(stmt).Error
}

func maskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	parts := strings.SplitN(dsn, "://", 2)
	if len(parts) != 2 {
		return dsn
	}
	scheme := parts[0]
	rest := parts[1]
	credIdx := strings.Index(rest, "@")
	if credIdx == -1 {
		return dsn
	}
	creds := rest[:credIdx]
	sanitized := creds
	if colon := strings.Index(creds, ":"); colon != -1 {
		sanitized = creds[:colon] + ":***"
	} else if creds != "" {
		sanitized = "***"
	}
	return fmt.Sprintf("%s://%s%s", scheme, sanitized, rest[credIdx:])
}

func detectDelegatedMode(cfg *iamConfig) string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("POWERX_PROVIDER_MODE")), "delegated") {
		return "POWERX_PROVIDER_MODE"
	}
	if cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.Context.ProviderMode), "delegated") {
		return "context.provider_mode"
	}
	return ""
}
