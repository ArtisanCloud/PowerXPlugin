package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config represents the main configuration structure
type Config struct {
	// Global settings
	Global GlobalConfig `json:"global"`

	// Dev API settings
	DevAPI DevAPIConfig `json:"devApi"`

	// Publish API settings
	PublishAPI PublishAPIConfig `json:"publishApi"`

	// Dev command settings
	Dev DevConfig `json:"dev"`

	// Security settings
	Security SecurityConfig `json:"security"`

	// Performance settings
	Performance PerformanceConfig `json:"performance"`

	// Audit settings
	Audit AuditConfig `json:"audit"`

	// Watch settings
	Watch WatchConfig `json:"watch"`

	// Build settings
	Build BuildConfig `json:"build"`

	// Metadata
	Version   string                 `json:"version,omitempty"`
	CreatedAt time.Time              `json:"createdAt,omitempty"`
	UpdatedAt time.Time              `json:"updatedAt,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// GlobalConfig contains global configuration
type GlobalConfig struct {
	Debug      bool   `json:"debug,omitempty"`
	Verbose    bool   `json:"verbose,omitempty"`
	NoColor    bool   `json:"noColor,omitempty"`
	LogLevel   string `json:"logLevel,omitempty"` // debug, info, warn, error
	LogFile    string `json:"logFile,omitempty"`
	WorkingDir string `json:"workingDir,omitempty"`
	CacheDir   string `json:"cacheDir,omitempty"`
	TempDir    string `json:"tempDir,omitempty"`
}

// DevAPIConfig contains Dev API configuration
type DevAPIConfig struct {
	BaseURL    string `json:"baseUrl,omitempty"`
	APIKey     string `json:"apiKey,omitempty"`
	Timeout    int    `json:"timeout,omitempty"` // seconds
	Retries    int    `json:"retries,omitempty"`
	RetryDelay int    `json:"retryDelay,omitempty"` // seconds
	EnableMTLS bool   `json:"enableMtls,omitempty"`
	CertPath   string `json:"certPath,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"`
	CACertPath string `json:"caCertPath,omitempty"`
}

// PublishAPIConfig contains PowerX Registry configuration
type PublishAPIConfig struct {
	BaseURL string `json:"baseUrl,omitempty"`
	APIKey  string `json:"apiKey,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// DevConfig contains dev command configuration
type DevConfig struct {
	Watch         bool     `json:"watch,omitempty"`
	EntryPath     string   `json:"entryPath,omitempty"`
	Tenant        string   `json:"tenant,omitempty"`
	Ignore        []string `json:"ignore,omitempty"`
	DevAPI        string   `json:"devApi,omitempty"`
	WatchInterval int      `json:"watchInterval,omitempty"` // milliseconds
	DebounceDelay int      `json:"debounceDelay,omitempty"` // milliseconds
	MaxWorkers    int      `json:"maxWorkers,omitempty"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	EnableMTLS         bool   `json:"enableMtls,omitempty"`
	CertDir            string `json:"certDir,omitempty"`
	AutoRotate         bool   `json:"autoRotate,omitempty"`
	RotationCheck      int    `json:"rotationCheck,omitempty"` // minutes
	MinTLSVersion      string `json:"minTlsVersion,omitempty"`
	MaxTLSVersion      string `json:"maxTlsVersion,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}

// PerformanceConfig contains performance configuration
type PerformanceConfig struct {
	HashCacheSize  int   `json:"hashCacheSize,omitempty"`
	BatchSize      int   `json:"batchSize,omitempty"`
	MaxConcurrency int   `json:"maxConcurrency,omitempty"`
	HTTPClientPool int   `json:"httpClientPool,omitempty"`
	RateLimit      int   `json:"rateLimit,omitempty"`    // per second
	MemoryLimit    int64 `json:"memoryLimit,omitempty"`  // bytes
	CPUThreshold   int   `json:"cpuThreshold,omitempty"` // percent
}

// AuditConfig contains audit configuration
type AuditConfig struct {
	Enabled   bool   `json:"enabled,omitempty"`
	Directory string `json:"directory,omitempty"`
	MaxSize   int64  `json:"maxSize,omitempty"` // bytes
	MaxFiles  int    `json:"maxFiles,omitempty"`
	Compress  bool   `json:"compress,omitempty"`
}

// WatchConfig contains file watching configuration
type WatchConfig struct {
	Recursive      bool     `json:"recursive,omitempty"`
	IgnoreDotFiles bool     `json:"ignoreDotFiles,omitempty"`
	IgnorePatterns []string `json:"ignorePatterns,omitempty"`
	MaxFileSize    int64    `json:"maxFileSize,omitempty"` // bytes
	MaxFiles       int      `json:"maxFiles,omitempty"`
	Paths          []string `json:"paths,omitempty"`
}

// BuildConfig contains build configuration
type BuildConfig struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Command         string `json:"command,omitempty"`
	OutputPath      string `json:"outputPath,omitempty"`
	Incremental     bool   `json:"incremental,omitempty"`
	CleanOnStart    bool   `json:"cleanOnStart,omitempty"`
	Parallel        bool   `json:"parallel,omitempty"`
	MaxParallelJobs int    `json:"maxParallelJobs,omitempty"`
}

// ConfigManager manages configuration
type ConfigManager struct {
	mu           sync.RWMutex
	config       *Config
	sources      []Source
	watchers     map[string][]Watcher
	reload       bool
	lastModified time.Time
}

// Source represents a configuration source
type Source interface {
	Load() (*Config, error)
	Watch() (<-chan bool, error)
	GetPath() string
}

// Watcher watches for configuration changes
type Watcher func(*Config)

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:   &Config{},
		sources:  make([]Source, 0),
		watchers: make(map[string][]Watcher),
		reload:   true,
	}
}

// AddSource adds a configuration source
func (cm *ConfigManager) AddSource(source Source) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.sources = append(cm.sources, source)
}

// Load loads configuration from all sources
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config := DefaultConfig()

	// Load from all sources
	for _, source := range cm.sources {
		cfg, err := source.Load()
		if err != nil {
			return fmt.Errorf("failed to load config from %s: %w", source.GetPath(), err)
		}

		// Merge configurations
		config = mergeConfigs(config, cfg)
	}

	// Set timestamps
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	cm.config = config
	return nil
}

// Get returns the current configuration
func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// Set updates the configuration
func (cm *ConfigManager) Set(config *Config) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	config.UpdatedAt = time.Now()
	cm.config = config
}

// AddWatcher adds a configuration change watcher
func (cm *ConfigManager) AddWatcher(path string, watcher Watcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.watchers[path] = append(cm.watchers[path], watcher)
}

// Reload reloads configuration from sources
func (cm *ConfigManager) Reload() error {
	return cm.Load()
}

// EnableAutoReload enables automatic reloading
func (cm *ConfigManager) EnableAutoReload() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.reload = true

	// Start watchers
	for _, source := range cm.sources {
		go func(s Source) {
			ch, err := s.Watch()
			if err != nil {
				return
			}

			for range ch {
				cm.reloadConfig(s)
			}
		}(source)
	}
}

func (cm *ConfigManager) reloadConfig(source Source) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.reload {
		return
	}

	// Load new config
	newConfig, err := source.Load()
	if err != nil {
		return
	}

	// Notify watchers
	path := source.GetPath()
	for _, watcher := range cm.watchers[path] {
		watcher(newConfig)
	}
}

// FileSource represents a file-based configuration source
type FileSource struct {
	Path     string
	Format   string // "yaml", "json", "toml"
	Override bool
}

// NewFileSource creates a new file source
func NewFileSource(path, format string, override bool) *FileSource {
	return &FileSource{
		Path:     path,
		Format:   format,
		Override: override,
	}
}

// Load loads configuration from file
func (fs *FileSource) Load() (*Config, error) {
	data, err := os.ReadFile(fs.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var config *Config
	switch strings.ToLower(fs.Format) {
	case "json":
		config, err = parseJSON(data)
	case "yaml", "yml":
		config, err = parseYAML(data)
	case "toml":
		config, err = parseTOML(data)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", fs.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", fs.Path, err)
	}

	return config, nil
}

// Watch watches for file changes
func (fs *FileSource) Watch() (<-chan bool, error) {
	return watchFile(fs.Path)
}

// GetPath returns the file path
func (fs *FileSource) GetPath() string {
	return fs.Path
}

// EnvironmentSource represents an environment variable source
type EnvironmentSource struct {
	Prefix    string
	Overwrite bool
}

// NewEnvironmentSource creates a new environment source
func NewEnvironmentSource(prefix string, overwrite bool) *EnvironmentSource {
	return &EnvironmentSource{
		Prefix:    prefix,
		Overwrite: overwrite,
	}
}

// Load loads configuration from environment variables
func (es *EnvironmentSource) Load() (*Config, error) {
	config := &Config{}
	envConfig := parseEnvironment(es.Prefix)

	// Merge environment variables into config
	config = mergeEnvConfig(config, envConfig)

	return config, nil
}

// Watch watches for environment changes
func (es *EnvironmentSource) Watch() (<-chan bool, error) {
	// Environment variables don't change at runtime
	// Return a nil channel
	return nil, nil
}

// GetPath returns the environment prefix
func (es *EnvironmentSource) GetPath() string {
	return "environment:" + es.Prefix
}

// CommandLineSource represents command-line arguments
type CommandLineSource struct {
	Args      map[string]string
	Overwrite bool
}

// NewCommandLineSource creates a new command line source
func NewCommandLineSource(args map[string]string, overwrite bool) *CommandLineSource {
	return &CommandLineSource{
		Args:      args,
		Overwrite: overwrite,
	}
}

// Load loads configuration from command line
func (cls *CommandLineSource) Load() (*Config, error) {
	config := &Config{}
	cmdConfig := parseCommandLine(cls.Args)

	// Merge command line into config
	config = mergeCommandLineConfig(config, cmdConfig)

	return config, nil
}

// Watch does nothing for command line
func (cls *CommandLineSource) Watch() (<-chan bool, error) {
	return nil, nil
}

// GetPath returns the command line identifier
func (cls *CommandLineSource) GetPath() string {
	return "command-line"
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Debug:    false,
			Verbose:  false,
			NoColor:  false,
			LogLevel: "info",
			CacheDir: getDefaultCacheDir(),
			TempDir:  getDefaultTempDir(),
		},
		DevAPI: DevAPIConfig{
			BaseURL:    "http://localhost:8077",
			Timeout:    30,
			Retries:    3,
			RetryDelay: 1,
			EnableMTLS: false,
		},
		PublishAPI: PublishAPIConfig{
			Timeout: 30,
		},
		Dev: DevConfig{
			Watch:         false,
			WatchInterval: 500,
			DebounceDelay: 250,
			MaxWorkers:    4,
		},
		Security: SecurityConfig{
			EnableMTLS:    false,
			CertDir:       getDefaultCertDir(),
			AutoRotate:    true,
			RotationCheck: 5,
			MinTLSVersion: "1.2",
			MaxTLSVersion: "1.3",
		},
		Performance: PerformanceConfig{
			HashCacheSize:  1000,
			BatchSize:      50,
			MaxConcurrency: 10,
			HTTPClientPool: 10,
			RateLimit:      1000,
			MemoryLimit:    500 * 1024 * 1024, // 500MB
			CPUThreshold:   80,
		},
		Audit: AuditConfig{
			Enabled:   true,
			Directory: getDefaultAuditDir(),
			MaxSize:   10 * 1024 * 1024, // 10MB
			MaxFiles:  5,
			Compress:  true,
		},
		Watch: WatchConfig{
			Recursive:      true,
			IgnoreDotFiles: true,
			IgnorePatterns: []string{".git", "node_modules", "dist", "build"},
			MaxFileSize:    10 * 1024 * 1024, // 10MB
			MaxFiles:       10000,
		},
		Build: BuildConfig{
			Enabled:         true,
			Incremental:     true,
			CleanOnStart:    false,
			Parallel:        true,
			MaxParallelJobs: 4,
		},
		Custom: make(map[string]interface{}),
	}
}

// Helper functions

// parseJSON parses JSON configuration
func parseJSON(data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}

// parseYAML parses YAML configuration (placeholder)
// In a real implementation, you would use a YAML library
func parseYAML(data []byte) (*Config, error) {
	// Placeholder: would use gopkg.in/yaml.v3 or similar
	return nil, fmt.Errorf("YAML parsing not implemented")
}

// parseTOML parses TOML configuration (placeholder)
// In a real implementation, you would use a TOML library
func parseTOML(data []byte) (*Config, error) {
	// Placeholder: would use github.com/BurntSushi/toml or similar
	return nil, fmt.Errorf("TOML parsing not implemented")
}

// parseEnvironment parses environment variables
func parseEnvironment(prefix string) map[string]interface{} {
	envConfig := make(map[string]interface{})

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if key starts with prefix
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}

		// Convert key to nested map
		if prefix != "" {
			key = strings.TrimPrefix(key, prefix+"_")
		}

		// Simple conversion: split by underscore and create nested map
		keys := strings.Split(strings.ToLower(key), "_")
		setNestedValue(envConfig, keys, value)
	}

	return envConfig
}

// parseCommandLine parses command line arguments
func parseCommandLine(args map[string]string) map[string]interface{} {
	cmdConfig := make(map[string]interface{})

	for key, value := range args {
		keys := strings.Split(strings.ToLower(key), "_")
		setNestedValue(cmdConfig, keys, value)
	}

	return cmdConfig
}

// setNestedValue sets a value in a nested map
func setNestedValue(data map[string]interface{}, keys []string, value interface{}) {
	current := data
	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
		} else {
			if _, ok := current[key]; !ok {
				current[key] = make(map[string]interface{})
			}
			current = current[key].(map[string]interface{})
		}
	}
}

// mergeConfigs merges two configurations
func mergeConfigs(base, override *Config) *Config {
	if override == nil {
		return base
	}

	if base == nil {
		return override
	}

	// Simple merge: override takes precedence
	// In a real implementation, you would use a more sophisticated merge strategy
	merged := *base

	// Merge global config
	if override.Global.Debug != base.Global.Debug {
		merged.Global.Debug = override.Global.Debug
	}
	if override.Global.Verbose != base.Global.Verbose {
		merged.Global.Verbose = override.Global.Verbose
	}
	if override.Global.NoColor != base.Global.NoColor {
		merged.Global.NoColor = override.Global.NoColor
	}
	if override.Global.LogLevel != "" {
		merged.Global.LogLevel = override.Global.LogLevel
	}
	if override.Global.LogFile != "" {
		merged.Global.LogFile = override.Global.LogFile
	}
	if override.Global.WorkingDir != "" {
		merged.Global.WorkingDir = override.Global.WorkingDir
	}
	if override.Global.CacheDir != "" {
		merged.Global.CacheDir = override.Global.CacheDir
	}
	if override.Global.TempDir != "" {
		merged.Global.TempDir = override.Global.TempDir
	}

	if override.PublishAPI.BaseURL != "" {
		merged.PublishAPI.BaseURL = override.PublishAPI.BaseURL
	}
	if override.PublishAPI.APIKey != "" {
		merged.PublishAPI.APIKey = override.PublishAPI.APIKey
	}
	if override.PublishAPI.Timeout > 0 {
		merged.PublishAPI.Timeout = override.PublishAPI.Timeout
	}

	// Merge other configs...
	// (Similar pattern for other fields)

	return &merged
}

// mergeEnvConfig merges environment config into main config
func mergeEnvConfig(config *Config, envConfig map[string]interface{}) *Config {
	// Implementation would traverse envConfig and update config fields
	return config
}

// mergeCommandLineConfig merges command line config into main config
func mergeCommandLineConfig(config *Config, cmdConfig map[string]interface{}) *Config {
	// Implementation would traverse cmdConfig and update config fields
	return config
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate global config
	if config.Global.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		if !contains(validLevels, config.Global.LogLevel) {
			return fmt.Errorf("invalid log level: %s", config.Global.LogLevel)
		}
	}

	// Validate Dev API config
	if config.DevAPI.BaseURL != "" {
		if !strings.HasPrefix(config.DevAPI.BaseURL, "http://") &&
			!strings.HasPrefix(config.DevAPI.BaseURL, "https://") {
			return fmt.Errorf("DevAPI baseURL must start with http:// or https://")
		}
	}

	if config.DevAPI.Timeout < 0 {
		return fmt.Errorf("DevAPI timeout must be non-negative")
	}

	if config.DevAPI.Retries < 0 {
		return fmt.Errorf("DevAPI retries must be non-negative")
	}

	if config.PublishAPI.BaseURL != "" {
		if !strings.HasPrefix(config.PublishAPI.BaseURL, "http://") &&
			!strings.HasPrefix(config.PublishAPI.BaseURL, "https://") {
			return fmt.Errorf("PublishAPI baseUrl must start with http:// or https://")
		}
	}
	if config.PublishAPI.Timeout < 0 {
		return fmt.Errorf("PublishAPI timeout must be non-negative")
	}

	// Validate security config
	if config.Security.MinTLSVersion != "" {
		validVersions := []string{"1.2", "1.3"}
		if !contains(validVersions, config.Security.MinTLSVersion) {
			return fmt.Errorf("invalid MinTLSVersion: %s", config.Security.MinTLSVersion)
		}
	}

	// Validate performance config
	if config.Performance.MaxConcurrency < 1 {
		return fmt.Errorf("MaxConcurrency must be at least 1")
	}

	if config.Performance.RateLimit < 0 {
		return fmt.Errorf("RateLimit must be non-negative")
	}

	if config.Performance.MemoryLimit < 0 {
		return fmt.Errorf("MemoryLimit must be non-negative")
	}

	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getDefaultCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".px-plugin", "cache")
}

func getDefaultTempDir() string {
	return filepath.Join(os.TempDir(), "px-plugin")
}

func getDefaultCertDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".px-plugin", "certs")
}

func getDefaultAuditDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".px-plugin", "audit")
}

func getDefaultConfigFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".px-plugin", "config.json")
}

// DefaultConfigPath returns the default config file path (~/.px-plugin/config.json).
func DefaultConfigPath() string {
	return getDefaultConfigFile()
}

// LoadConfigFile loads configuration from the provided JSON file.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// LoadDefaultConfigFile loads configuration from ~/.px-plugin/config.json.
func LoadDefaultConfigFile() (*Config, error) {
	path := getDefaultConfigFile()
	return LoadConfigFile(path)
}

// watchFile watches for file changes (placeholder)
func watchFile(path string) (<-chan bool, error) {
	// In a real implementation, you would use fsnotify or similar
	return nil, fmt.Errorf("file watching not implemented")
}
