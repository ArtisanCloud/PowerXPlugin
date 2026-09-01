package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFallsBackToMemoryDefaults(t *testing.T) {
	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("CONFIG_PATH", "")
	t.Setenv("POWERX_DB_DSN", "")
	t.Setenv("POWERX_DB_SCHEMA", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载默认配置失败: %v", err)
	}
	if cfg.Logging == nil || !cfg.Logging.DebugMode {
		t.Fatalf("默认配置应启用 logging.debug_mode 便于本地启动")
	}
	if cfg.Database == nil {
		t.Fatal("默认数据库配置缺失")
	}
	if cfg.Database.Driver != "memory" {
		t.Fatalf("默认数据库驱动应为 memory，实际 %q", cfg.Database.Driver)
	}
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		t.Fatal("默认内存数据库 DSN 不应为空")
	}
}

func TestLoadRespectsProviderModeEnv(t *testing.T) {
	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("POWERX_PROVIDER_MODE", "delegated")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Context == nil || cfg.Context.ProviderMode != "delegated" {
		t.Fatalf("POWERX_PROVIDER_MODE 环境变量未生效，期望 delegated 实际 %q", cfg.Context.ProviderMode)
	}
}

func TestLoadIgnoresUnknownProviderModeEnv(t *testing.T) {
	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("POWERX_UNKNOWN_PROVIDER_MODE", "delegated")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Context != nil && cfg.Context.ProviderMode == "delegated" {
		t.Fatalf("legacy IAM mode env must not affect provider mode")
	}
}

func TestLoadAppliesEnvOverrides(t *testing.T) {
	const (
		dsn    = "postgres://user:pass@127.0.0.1:5432/powerx_test?sslmode=disable"
		schema = "px_override"
	)

	t.Setenv("POWERX_DB_DSN", dsn)
	t.Setenv("POWERX_DB_SCHEMA", schema)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_LOG_LEVEL", "INFO")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := "server:\n  bind_addr: \"127.0.0.1:0\"\nlogging:\n  level: WARN\n  format: TEXT\n  output: STDOUT\n"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Database == nil {
		t.Fatal("Database 配置未初始化")
	}
	if cfg.Database.DSN != dsn {
		t.Fatalf("POWERX_DB_DSN 未生效，期望 %q 实际 %q", dsn, cfg.Database.DSN)
	}
	if cfg.Database.Schema != schema {
		t.Fatalf("POWERX_DB_SCHEMA 未生效，期望 %q 实际 %q", schema, cfg.Database.Schema)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("POWERX_LOG_LEVEL 未归一化为小写 info, logging=%q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" || cfg.Logging.Output != "stdout" {
		t.Fatalf("日志配置未归一化: format=%q output=%q", cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadAppliesPluginSchemaEnvWhenDatabaseSchemaEnvMissing(t *testing.T) {
	const (
		dsn    = "postgres://user:pass@127.0.0.1:5432/powerx_test?sslmode=disable"
		schema = "px_com_powerx_plugin_demo"
	)

	t.Setenv("POWERX_DB_DSN", dsn)
	t.Setenv("POWERX_PLUGIN_DB_SCHEMA", schema)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_PROVIDER_MODE", "local")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := "server:\n  bind_addr: \"127.0.0.1:0\"\ndatabase:\n  schema: \"public\"\nlogging:\n  debug_mode: true\n"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Database == nil || cfg.Database.Schema != schema {
		t.Fatalf("POWERX_PLUGIN_DB_SCHEMA 未生效，database=%#v", cfg.Database)
	}
}

func TestLoadForMigrationSkipsRuntimeGatewaySTSValidation(t *testing.T) {
	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("PX_GATEWAY_BASE_URL", "http://127.0.0.1:8077")
	t.Setenv("PX_GATEWAY_AUTH_SCHEME", "bearer")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/powerx_test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_com_powerx_plugins_base")
	t.Setenv("POWERX_PLUGIN_DB_SCHEMA", "px_com_powerx_plugins_base")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "STS client") {
		t.Fatalf("Load() should still enforce runtime gateway STS validation, err=%v", err)
	}
	if _, err := LoadForMigration(); err != nil {
		t.Fatalf("LoadForMigration() should skip runtime gateway STS validation: %v", err)
	}
}

func TestLoadMapsHostWebAdminOriginsToCORS(t *testing.T) {
	tempDir := t.TempDir()
	configContent := strings.Join([]string{
		"security:",
		"  enable_cors: false",
		"  cors_origins:",
		"    - http://localhost:3031",
		"host:",
		"  web_admin_origins:",
		"    - https://admin.example.com",
		"    - http://localhost:3031",
	}, "\n")
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Security == nil || !cfg.Security.EnableCORS {
		t.Fatal("host.web_admin_origins 未启用 CORS")
	}
	want := []string{"http://localhost:3031", "https://admin.example.com"}
	if strings.Join(cfg.Security.CORSOrigins, ",") != strings.Join(want, ",") {
		t.Fatalf("CORS origins 未合并 host.web_admin_origins, got=%v want=%v", cfg.Security.CORSOrigins, want)
	}
}

func TestLoadNormalizesLoggingFromYAML(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "logging:\n  level: ERROR\n  format: JSON\n  output: STDERR\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Logging.Level != "error" || cfg.Logging.Format != "json" || cfg.Logging.Output != "stderr" {
		t.Fatalf("YAML 归一化失败: level=%q format=%q output=%q", cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadResolvesPlaceholderDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "server:\n  bind_addr: \"${POWERX_BIND_ADDR:-:9000}\"\nlogging:\n  level: \"${POWERX_LOG_LEVEL:-INFO}\"\n  format: \"${POWERX_LOG_FORMAT:-JSON}\"\n  output: \"${POWERX_LOG_OUTPUT:-STDOUT}\"\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Server.BindAddr != ":9000" {
		t.Fatalf("server.bind_addr 占位符未解析，得到 %q", cfg.Server.BindAddr)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("logging.level 占位符未解析，得到 %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" || cfg.Logging.Output != "stdout" {
		t.Fatalf("logging format/output 占位符未解析，format=%q output=%q", cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadKeepsGatewayBaseURLInHostDelegatedSTSMode(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "logging:\n  debug_mode: true\ncontext:\n  provider_mode: delegated\ngateway:\n  base_url: http://127.0.0.1:8077\n  auth_scheme: bearer\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("PX_GATEWAY_AUTH_SCHEME", "bearer")
	t.Setenv("POWERX_STS_CLIENT_ID", "com.powerx.plugins.base.tenant")
	t.Setenv("POWERX_STS_CLIENT_SECRET", "secret")
	t.Setenv("POWERX_GRPC_UPSTREAM_ADDRESS", "127.0.0.1:9001")
	t.Setenv("POWERX_GRPC_UPSTREAM_TENANT_UUID", "6b5d0240-9920-46da-b707-88200e0f51ea")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Gateway == nil || cfg.Gateway.BaseURL != "http://127.0.0.1:8077" {
		t.Fatalf("宿主 STS 模式不应清空 gateway.base_url，实际 %#v", cfg.Gateway)
	}
	if cfg.Gateway.AuthScheme != "bearer" {
		t.Fatalf("宿主 STS 模式应保留 bearer auth_scheme，实际 %q", cfg.Gateway.AuthScheme)
	}
}

func TestLoadUsesConfigPathPlaceholder(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}
	configContent := "server:\n  bind_addr: \":0\"\ndatabase:\n  dsn: \"postgres://user:pass@127.0.0.1:5432/test?sslmode=disable\"\n  schema: \"px_test\"\nlogging:\n  level: \"INFO\"\n  format: \"JSON\"\n  output: \"STDOUT\"\n  debug_mode: true\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入 host-values 配置失败: %v", err)
	}
	t.Setenv("POWERX_PLUGIN_CONFIG_DIR", configDir)
	t.Setenv("CONFIG_PATH", "${POWERX_PLUGIN_CONFIG_DIR:-./backend/etc}")
	t.Setenv("POWERX_DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Database == nil || cfg.Database.DSN == "" {
		t.Fatal("未从 CONFIG_PATH 提供的 YAML 中读取到数据库 DSN")
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("CONFIG_PATH 配置未生效，log level=%q", cfg.Logging.Level)
	}
}

func TestLoadSupportsRuntimeNamespacedLogging(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "runtime:\n  logging:\n    level: WARN\n    format: JSON\n    output: STDERR\n  run_migrate: true\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Logging == nil || cfg.Runtime == nil || cfg.Runtime.Logging == nil {
		t.Fatal("runtime.logging 映射失败")
	}
	if cfg.Logging.Level != "warn" || cfg.Logging.Format != "json" || cfg.Logging.Output != "stderr" {
		t.Fatalf("runtime.logging 未生效: level=%q format=%q output=%q", cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	}
	if !cfg.Runtime.RunMigrate {
		t.Fatal("runtime.run_migrate 未生效")
	}
}

func TestLoadSupportsRuntimeNamespacedEventBridge(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "runtime:\n  event_bridge:\n    enabled: true\n    mode: taskbus\n    taskbus_provider: redis\n    redis_url: redis://127.0.0.1:6379\n    redis_stream: px.test.stream\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.EventBridge == nil || cfg.Runtime == nil || cfg.Runtime.EventBridge == nil {
		t.Fatal("runtime.event_bridge 映射失败")
	}
	if cfg.EventBridge.TaskBusProvider != "redis" {
		t.Fatalf("runtime.event_bridge.taskbus_provider 未生效: %q", cfg.EventBridge.TaskBusProvider)
	}
	if cfg.EventBridge.RedisStream != "px.test.stream" {
		t.Fatalf("runtime.event_bridge.redis_stream 未生效: %q", cfg.EventBridge.RedisStream)
	}
}

func TestValidateSchedulerConfigDefaults(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Operations = &OperationsConfig{}
	cfg.Gateway.AuthScheme = "apikey"
	cfg.Gateway.APIKey = "test-key"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate 失败: %v", err)
	}
	if cfg.Operations.Scheduler.RetryMaxAttempts != 3 {
		t.Fatalf("scheduler.retry_max_attempts 默认值错误: %d", cfg.Operations.Scheduler.RetryMaxAttempts)
	}
	if cfg.Operations.Scheduler.PauseStrategy != "pause_on_retry_exhausted" {
		t.Fatalf("scheduler.pause_strategy 默认值错误: %q", cfg.Operations.Scheduler.PauseStrategy)
	}
	if cfg.Operations.Scheduler.ResumeRoleRequired != "ops_admin_only" {
		t.Fatalf("scheduler.resume_role_required 默认值错误: %q", cfg.Operations.Scheduler.ResumeRoleRequired)
	}
}

func TestValidateSchedulerRetryMaxAttemptsRange(t *testing.T) {
	tests := []struct {
		name        string
		attempts    int
		expectError bool
	}{
		{name: "min boundary", attempts: 1, expectError: false},
		{name: "max boundary", attempts: 10, expectError: false},
		{name: "below min", attempts: -1, expectError: true},
		{name: "above max", attempts: 11, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := getDefaultConfig()
			cfg.Operations = &OperationsConfig{
				Scheduler: OperationsSchedulerConfig{
					RetryMaxAttempts: tt.attempts,
				},
			}
			cfg.Gateway.AuthScheme = "apikey"
			cfg.Gateway.APIKey = "test-key"
			err := cfg.Validate()
			if tt.expectError && err == nil {
				t.Fatal("期望 Validate 失败，实际成功")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("期望 Validate 成功，实际失败: %v", err)
			}
			if tt.expectError && !strings.Contains(err.Error(), "operations.scheduler.retry_max_attempts must be between 1 and 10") {
				t.Fatalf("错误信息不符合预期: %v", err)
			}
		})
	}
}

func TestLoadSupportsStandaloneLoggingPolicySwitch(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "logging:\n  level: INFO\n  format: JSON\n  output: FILE\n  file_path: ./tmp/plugin.log\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")
	t.Setenv("POWERX_LOG_OUTPUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Logging.Output != "file" {
		t.Fatalf("standalone 默认策略应保留 file 输出，实际 %q", cfg.Logging.Output)
	}
	if cfg.Logging.FilePath == "" {
		t.Fatal("file 输出下 file_path 不应为空")
	}

	t.Setenv("POWERX_LOG_OUTPUT", "STDOUT")
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("再次加载配置失败: %v", err)
	}
	if cfg2.Logging.Output != "stdout" {
		t.Fatalf("切换策略后 output 应为 stdout，实际 %q", cfg2.Logging.Output)
	}
}

func TestShouldBlockLegacyLoggingByDeadline(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Logging.GovernanceMode = "warn"
	cfg.Logging.GovernanceDeadlineVersion = "1.2.0"
	cfg.Logging.PluginVersion = "1.1.9"
	blocked, reason := cfg.ShouldBlockLegacyLogging()
	if blocked {
		t.Fatalf("截止版本前不应阻断，reason=%s", reason)
	}

	cfg.Logging.PluginVersion = "1.2.0"
	blocked, reason = cfg.ShouldBlockLegacyLogging()
	if !blocked {
		t.Fatal("达到截止版本后应触发阻断")
	}
	if reason == "" {
		t.Fatal("阻断原因不应为空")
	}

	cfg.Logging.GovernanceMode = "block"
	blocked, _ = cfg.ShouldBlockLegacyLogging()
	if !blocked {
		t.Fatal("治理模式 block 应始终阻断")
	}
}
