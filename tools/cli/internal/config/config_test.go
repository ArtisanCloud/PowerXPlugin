package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check global defaults
	if config.Global.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got %q", config.Global.LogLevel)
	}

	if config.Global.CacheDir == "" {
		t.Error("Expected non-empty CacheDir")
	}

	// Check DevAPI defaults
	if config.DevAPI.BaseURL != "http://localhost:8077" {
		t.Errorf("Expected BaseURL 'http://localhost:8077', got %q", config.DevAPI.BaseURL)
	}

	if config.DevAPI.Timeout != 30 {
		t.Errorf("Expected Timeout 30, got %d", config.DevAPI.Timeout)
	}

	// Check Dev defaults
	if config.Dev.WatchInterval != 500 {
		t.Errorf("Expected WatchInterval 500, got %d", config.Dev.WatchInterval)
	}

	if config.Dev.DebounceDelay != 250 {
		t.Errorf("Expected DebounceDelay 250, got %d", config.Dev.DebounceDelay)
	}

	// Check Security defaults
	if !config.Security.AutoRotate {
		t.Error("Expected AutoRotate to be true")
	}

	if config.Security.RotationCheck != 5 {
		t.Errorf("Expected RotationCheck 5, got %d", config.Security.RotationCheck)
	}

	// Check Performance defaults
	if config.Performance.MaxConcurrency != 10 {
		t.Errorf("Expected MaxConcurrency 10, got %d", config.Performance.MaxConcurrency)
	}

	if config.Performance.BatchSize != 50 {
		t.Errorf("Expected BatchSize 50, got %d", config.Performance.BatchSize)
	}

	// Check Audit defaults
	if !config.Audit.Enabled {
		t.Error("Expected Audit.Enabled to be true")
	}

	if config.Audit.MaxSize != 10*1024*1024 {
		t.Errorf("Expected MaxSize 10MB, got %d", config.Audit.MaxSize)
	}

	// Check Watch defaults
	if !config.Watch.Recursive {
		t.Error("Expected Watch.Recursive to be true")
	}

	if !config.Watch.IgnoreDotFiles {
		t.Error("Expected Watch.IgnoreDotFiles to be true")
	}

	// Check Build defaults
	if !config.Build.Enabled {
		t.Error("Expected Build.Enabled to be true")
	}

	if !config.Build.Incremental {
		t.Error("Expected Build.Incremental to be true")
	}
}

func TestConfigManager_AddSource(t *testing.T) {
	cm := NewConfigManager()

	source := &FileSource{
		Path:   "/tmp/test-config.json",
		Format: "json",
	}

	cm.AddSource(source)

	if len(cm.sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(cm.sources))
	}
}

func TestConfigManager_Get(t *testing.T) {
	cm := NewConfigManager()

	config := &Config{
		Global: GlobalConfig{
			Debug: true,
		},
	}

	cm.Set(config)

	retrieved := cm.Get()
	if retrieved.Global.Debug != true {
		t.Error("Expected Debug to be true")
	}
}

func TestFileSource_Load(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	config := DefaultConfig()
	config.Global.Debug = true
	config.Global.Verbose = true

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the config
	source := NewFileSource(configFile, "json", false)
	loadedConfig, err := source.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedConfig.Global.Debug != true {
		t.Error("Expected Debug to be true")
	}

	if loadedConfig.Global.Verbose != true {
		t.Error("Expected Verbose to be true")
	}
}

func TestFileSource_Load_NonExistent(t *testing.T) {
	source := NewFileSource("/tmp/nonexistent-config.json", "json", false)

	config, err := source.Load()
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got %v", err)
	}

	if config == nil {
		t.Error("Expected non-nil config")
	}
}

func TestEnvironmentSource_Load(t *testing.T) {
	// Set environment variables
	os.Setenv("PX_DEBUG", "true")
	os.Setenv("PX_VERBOSE", "true")
	os.Setenv("PX_GLOBAL_LOGLEVEL", "debug")
	defer os.Unsetenv("PX_DEBUG")
	defer os.Unsetenv("PX_VERBOSE")
	defer os.Unsetenv("PX_GLOBAL_LOGLEVEL")

	source := NewEnvironmentSource("PX", false)

	config, err := source.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// The actual parsing would need to be implemented
	// This is just a basic test
	if config == nil {
		t.Error("Expected non-nil config")
	}
}

func TestCommandLineSource_Load(t *testing.T) {
	args := map[string]string{
		"global_debug":   "true",
		"devapi_timeout": "60",
	}

	source := NewCommandLineSource(args, false)

	config, err := source.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config == nil {
		t.Error("Expected non-nil config")
	}
}

func TestLoadConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.DevAPI.BaseURL = "https://dev.powerx.local"
	cfg.DevAPI.EnableMTLS = true
	cfg.DevAPI.CertPath = "/tmp/client.crt"
	cfg.DevAPI.KeyPath = "/tmp/client.key"
	cfg.DevAPI.CACertPath = "/tmp/ca.crt"

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded, err := LoadConfigFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFile returned error: %v", err)
	}

	if loaded.DevAPI.BaseURL != cfg.DevAPI.BaseURL {
		t.Fatalf("expected baseURL %s, got %s", cfg.DevAPI.BaseURL, loaded.DevAPI.BaseURL)
	}
	if !loaded.DevAPI.EnableMTLS {
		t.Fatalf("expected EnableMTLS true")
	}
}

func TestConfigManager_Load(t *testing.T) {
	cm := NewConfigManager()

	// Add a file source with default config
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	config := DefaultConfig()
	config.Global.Debug = true

	data, _ := json.Marshal(config)
	os.WriteFile(configFile, data, 0644)

	source := NewFileSource(configFile, "json", false)
	cm.AddSource(source)

	err := cm.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	loadedConfig := cm.Get()
	if !loadedConfig.Global.Debug {
		t.Error("Expected Debug to be true after load")
	}
}

func TestConfigManager_AddWatcher(t *testing.T) {
	cm := NewConfigManager()

	watcher := func(config *Config) {
	}

	cm.AddWatcher("test-path", watcher)

	if len(cm.watchers["test-path"]) != 1 {
		t.Errorf("Expected 1 watcher, got %d", len(cm.watchers["test-path"]))
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	config := DefaultConfig()

	// Modify to ensure it's valid
	config.Global.LogLevel = "debug"
	config.DevAPI.BaseURL = "https://api.example.com"
	config.DevAPI.Timeout = 30
	config.Performance.MaxConcurrency = 5

	err := validateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	config := DefaultConfig()
	config.Global.LogLevel = "invalid"

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid log level")
	}
}

func TestValidateConfig_InvalidBaseURL(t *testing.T) {
	config := DefaultConfig()
	config.DevAPI.BaseURL = "invalid-url"

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid base URL")
	}
}

func TestValidateConfig_NegativeTimeout(t *testing.T) {
	config := DefaultConfig()
	config.DevAPI.Timeout = -1

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for negative timeout")
	}
}

func TestValidateConfig_InvalidTLSVersion(t *testing.T) {
	config := DefaultConfig()
	config.Security.MinTLSVersion = "invalid"

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid TLS version")
	}
}

func TestValidateConfig_ZeroConcurrency(t *testing.T) {
	config := DefaultConfig()
	config.Performance.MaxConcurrency = 0

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for zero concurrency")
	}
}

func TestParseEnvironment(t *testing.T) {
	os.Setenv("TEST_DEBUG", "true")
	os.Setenv("TEST_VERBOSE", "true")
	defer os.Unsetenv("TEST_DEBUG")
	defer os.Unsetenv("TEST_VERBOSE")

	envConfig := parseEnvironment("TEST")

	if envConfig["debug"] != "true" {
		t.Error("Expected debug to be true")
	}

	if envConfig["verbose"] != "true" {
		t.Error("Expected verbose to be true")
	}
}

func TestParseCommandLine(t *testing.T) {
	args := map[string]string{
		"global_debug":   "true",
		"devapi_timeout": "60",
	}

	cmdConfig := parseCommandLine(args)

	global, ok := cmdConfig["global"].(map[string]interface{})
	if !ok || global["debug"] != "true" {
		t.Error("Expected global.debug to be true")
	}

	devapi, ok := cmdConfig["devapi"].(map[string]interface{})
	if !ok || devapi["timeout"] != "60" {
		t.Error("Expected devapi.timeout to be 60")
	}
}

func TestSetNestedValue(t *testing.T) {
	data := make(map[string]interface{})
	keys := []string{"global", "log", "level"}
	value := "debug"

	setNestedValue(data, keys, value)

	// Check if nested value was set
	if global, ok := data["global"]; ok {
		if log, ok := global.(map[string]interface{})["log"]; ok {
			if level, ok := log.(map[string]interface{})["level"]; ok {
				if level != "debug" {
					t.Errorf("Expected level to be 'debug', got %v", level)
				}
			} else {
				t.Error("Expected level to be set in nested map")
			}
		} else {
			t.Error("Expected log to be set in nested map")
		}
	} else {
		t.Error("Expected global to be set in map")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"debug", "info", "warn", "error"}

	if !contains(slice, "info") {
		t.Error("Expected contains to return true for existing item")
	}

	if contains(slice, "invalid") {
		t.Error("Expected contains to return false for non-existing item")
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &Config{
		Global: GlobalConfig{
			Debug: false,
		},
	}

	override := &Config{
		Global: GlobalConfig{
			Debug: true,
		},
	}

	merged := mergeConfigs(base, override)

	if !merged.Global.Debug {
		t.Error("Expected Debug to be true after merge")
	}
}

func TestGetDefaultCacheDir(t *testing.T) {
	dir := getDefaultCacheDir()
	if dir == "" {
		t.Error("Expected non-empty cache directory")
	}

	if !filepath.IsAbs(dir) {
		t.Error("Expected absolute path for cache directory")
	}
}

func TestGetDefaultTempDir(t *testing.T) {
	dir := getDefaultTempDir()
	if dir == "" {
		t.Error("Expected non-empty temp directory")
	}

	if !filepath.IsAbs(dir) {
		t.Error("Expected absolute path for temp directory")
	}
}

func TestGetDefaultCertDir(t *testing.T) {
	dir := getDefaultCertDir()
	if dir == "" {
		t.Error("Expected non-empty cert directory")
	}

	if !filepath.IsAbs(dir) {
		t.Error("Expected absolute path for cert directory")
	}
}

func TestGetDefaultAuditDir(t *testing.T) {
	dir := getDefaultAuditDir()
	if dir == "" {
		t.Error("Expected non-empty audit directory")
	}

	if !filepath.IsAbs(dir) {
		t.Error("Expected absolute path for audit directory")
	}
}
