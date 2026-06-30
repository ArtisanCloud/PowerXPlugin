package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	defaultGRPCPort        = 9101
	defaultGRPCPortRetries = 10
)

// Config 插件配置结构
type Config struct {
	// 服务配置
	Server *ServerConfig `yaml:"server" json:"server"`

	// 数据库配置
	Database *DatabaseConfig `yaml:"database" json:"database"`

	// 运行时配置
	Runtime *RuntimeConfig `yaml:"runtime" json:"runtime"`

	// RuntimeOpsDefaults 运行时治理默认值（可被 host-values.yaml 覆盖）
	RuntimeOps *RuntimeOpsDefaults `yaml:"runtime_ops" json:"runtime_ops"`

	// PowerX 上下文配置
	Context *ContextConfig `yaml:"context" json:"context"`

	// 宿主标准配置，由 PowerX host-values.yaml 注入。
	Host *HostConfig `yaml:"host" json:"host"`

	// 安全配置
	Security *SecurityConfig `yaml:"security" json:"security"`

	// SecurityBaseline 保存从 security_baseline.yaml 解析出的安全基线默认值。
	SecurityBaseline *SecurityBaselineConfig `yaml:"-" json:"security_baseline"`

	// 监控配置
	Monitoring MonitoringConfig `yaml:"monitoring" json:"monitoring"`

	// 日志配置
	Logging *LoggingConfig `yaml:"logging" json:"logging"`

	// gRPC 配置
	GRPCUpstream *GRPCUpstream `yaml:"grpc_upstream" json:"grpc_upstream"`
	GRPCServer   *GRPCServer   `yaml:"grpc_server" json:"grpc_server"`

	// Integration 集成协议相关配置。
	Integration *IntegrationConfig `yaml:"integration" json:"integration"`

	// Marketplace 配置。
	Marketplace *MarketplaceConfig `yaml:"marketplace" json:"marketplace"`

	// Operations 配置。
	Operations *OperationsConfig `yaml:"operations" json:"operations"`

	// AdminConsole 配置。
	AdminConsole *AdminConsoleConfig `yaml:"admin_console" json:"admin_console"`

	// Gateway 能力调用配置。
	Gateway *GatewayConfig `yaml:"gateway" json:"gateway"`

	// CustomerAuth (mini-app / 2C) 鉴权配置。
	CustomerAuth *CustomerAuthConfig `yaml:"customer_auth" json:"customer_auth"`

	// EventBridge (TaskBus / 本地事件桥) 配置。
	EventBridge *EventBridgeConfig `yaml:"event_bridge" json:"event_bridge"`

	// WS Bus 配置（standalone 模式：内存/Redis）。
	WSBus *WSBusConfig `yaml:"ws_bus" json:"ws_bus"`

	// 向后兼容的字段（从环境变量或旧配置中填充）
	BindAddr   string `yaml:"-" json:"bind_addr,omitempty"`
	LogLevel   string `yaml:"-" json:"log_level,omitempty"`
	DevMode    bool   `yaml:"-" json:"dev_mode,omitempty"`
	DBDSN      string `yaml:"-" json:"db_dsn,omitempty"`
	DBSchema   string `yaml:"-" json:"db_schema,omitempty"`
	RunMigrate bool   `yaml:"-" json:"run_migrate,omitempty"`
}

// EventBridgeConfig 控制事件桥接（本地 emitter / TaskBus emitter / 双写）的行为。
type EventBridgeConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	Mode            string `yaml:"mode" json:"mode"` // local|taskbus|dual（为空时按 enabled 推断）
	FallbackToLocal bool   `yaml:"fallback_to_local" json:"fallback_to_local"`
	LocalQueueSize  int    `yaml:"local_queue_size" json:"local_queue_size"`
	TaskBusProvider string `yaml:"taskbus_provider" json:"taskbus_provider"` // host|redis
	RedisURL        string `yaml:"redis_url" json:"redis_url"`
	RedisStream     string `yaml:"redis_stream" json:"redis_stream"`
	RedisGroup      string `yaml:"redis_group" json:"redis_group"`
	RedisConsumer   string `yaml:"redis_consumer" json:"redis_consumer"`
	RedisMaxLen     int64  `yaml:"redis_max_len" json:"redis_max_len"`
	SourcePlugin    string `yaml:"source_plugin" json:"source_plugin"`
	PayloadVersion  string `yaml:"payload_version" json:"payload_version"`
}

// WSBusConfig 控制 WS Bus 发布订阅的后端实现。
type WSBusConfig struct {
	Provider string `yaml:"provider" json:"provider"`   // memory|redis
	RedisURL string `yaml:"redis_url" json:"redis_url"` // redis://...
	Channel  string `yaml:"channel" json:"channel"`     // default: powerx.wsbus
}

// CustomerAuthConfig 控制 mini-app（2C）鉴权模式。
type CustomerAuthConfig struct {
	Mode             string `yaml:"mode" json:"mode"`
	DelegateEndpoint string `yaml:"delegate_endpoint" json:"delegate_endpoint"`
	DelegateTimeout  string `yaml:"delegate_timeout" json:"delegate_timeout"`
	JWTIssuer        string `yaml:"jwt_issuer" json:"jwt_issuer"`
	JWTAudience      string `yaml:"jwt_audience" json:"jwt_audience"`
	JWTSecret        string `yaml:"jwt_secret" json:"jwt_secret"`
	CacheTTLSeconds  int    `yaml:"cache_ttl_seconds" json:"cache_ttl_seconds"`
	BreakGlassLocal  bool   `yaml:"break_glass_local" json:"break_glass_local"`
	BreakGlassReason string `yaml:"break_glass_reason" json:"break_glass_reason"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	BindAddr            string `yaml:"bind_addr" json:"bind_addr"`
	Port                int    `yaml:"port"`                  // HTTP 端口
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`  // 读取超时
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"` // 写入超时
	Mode                string `yaml:"mode"`                  // gin 模式: debug/release
	APIPrefix           string `yaml:"api_prefix"`            // API 前缀
	WSPrefix            string `yaml:"ws_prefix"`             // API 前缀
	SecretKey           string `yaml:"secret_key"`
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	RunMigrate  bool                `yaml:"run_migrate" json:"run_migrate"`
	GinMode     string              `yaml:"gin_mode" json:"gin_mode"`
	HTTPLog     *bool               `yaml:"http_log" json:"http_log"`
	RouteLog    *bool               `yaml:"route_log" json:"route_log"`
	Logging     *LoggingConfig      `yaml:"logging" json:"logging"`
	Monitoring  *MonitoringConfig   `yaml:"monitoring" json:"monitoring"`
	RuntimeOps  *RuntimeOpsDefaults `yaml:"runtime_ops" json:"runtime_ops"`
	EventBridge *EventBridgeConfig  `yaml:"event_bridge" json:"event_bridge"`
	WSBus       *WSBusConfig        `yaml:"ws_bus" json:"ws_bus"`
	Integration *IntegrationConfig  `yaml:"integration" json:"integration"`
	Cache       *RuntimeCacheConfig `yaml:"cache" json:"cache"`
}

type RuntimeCacheConfig struct {
	Provider string `yaml:"provider" json:"provider"`
	RedisURL string `yaml:"redis_url" json:"redis_url"`
}

func applyRuntimeNamespacesToLegacy(cfg *Config) {
	if cfg == nil || cfg.Runtime == nil {
		return
	}
	if cfg.Runtime.Logging != nil {
		cfg.Logging = cfg.Runtime.Logging
	}
	if cfg.Runtime.Monitoring != nil {
		cfg.Monitoring = *cfg.Runtime.Monitoring
	}
	if cfg.Runtime.RuntimeOps != nil {
		cfg.RuntimeOps = cfg.Runtime.RuntimeOps
	}
	if cfg.Runtime.EventBridge != nil {
		cfg.EventBridge = cfg.Runtime.EventBridge
	}
	if cfg.Runtime.WSBus != nil {
		cfg.WSBus = cfg.Runtime.WSBus
	}
	if cfg.Runtime.Integration != nil {
		cfg.Integration = cfg.Runtime.Integration
	}
	if cfg.Logging != nil {
		if ginMode := strings.TrimSpace(cfg.Runtime.GinMode); ginMode != "" {
			cfg.Logging.GinMode = strings.ToLower(ginMode)
		}
		if cfg.Runtime.HTTPLog != nil {
			cfg.Logging.HTTPAccess = *cfg.Runtime.HTTPLog
		}
		if cfg.Runtime.RouteLog != nil {
			cfg.Logging.RouteLog = *cfg.Runtime.RouteLog
		}
	}
}

func syncLegacyNamespacesToRuntime(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Runtime == nil {
		cfg.Runtime = &RuntimeConfig{}
	}
	cfg.Runtime.Logging = cfg.Logging
	monitoring := cfg.Monitoring
	cfg.Runtime.Monitoring = &monitoring
	cfg.Runtime.RuntimeOps = cfg.RuntimeOps
	cfg.Runtime.EventBridge = cfg.EventBridge
	cfg.Runtime.WSBus = cfg.WSBus
	cfg.Runtime.Integration = cfg.Integration
	if cfg.Logging != nil {
		cfg.Runtime.GinMode = cfg.Logging.GinMode
		httpLog := cfg.Logging.HTTPAccess
		routeLog := cfg.Logging.RouteLog
		cfg.Runtime.HTTPLog = &httpLog
		cfg.Runtime.RouteLog = &routeLog
	}
}

// RuntimeOpsDefaults 定义 runtime ops 所需的默认限值与窗口
type RuntimeOpsDefaults struct {
	HeartbeatSeconds           int                 `yaml:"heartbeat_seconds" json:"heartbeat_seconds"`
	HeartbeatMisses            int                 `yaml:"heartbeat_misses" json:"heartbeat_misses"`
	QuotaWindowMinutes         int                 `yaml:"quota_window_minutes" json:"quota_window_minutes"`
	RestartBackoffStartSeconds int                 `yaml:"restart_backoff_start_seconds" json:"restart_backoff_start_seconds"`
	RestartBackoffMaxSeconds   int                 `yaml:"restart_backoff_max_seconds" json:"restart_backoff_max_seconds"`
	LogRetentionDays           int                 `yaml:"log_retention_days" json:"log_retention_days"`
	CPUDefault                 string              `yaml:"cpu_default" json:"cpu_default"`
	MemoryDefault              string              `yaml:"memory_default" json:"memory_default"`
	NetworkProfile             string              `yaml:"network_profile" json:"network_profile"`
	Observability              ObservabilityConfig `yaml:"observability" json:"observability"`
	Alerts                     AlertThresholds     `yaml:"alerts" json:"alerts"`
}

// ObservabilityConfig captures metrics/logging exporters.
type ObservabilityConfig struct {
	LokiEndpoint  string `yaml:"loki_endpoint" json:"loki_endpoint"`
	TempoEndpoint string `yaml:"tempo_endpoint" json:"tempo_endpoint"`
}

// AlertThresholds defines default alert thresholds for runtime ops.
type AlertThresholds struct {
	HealthFailureRate float64 `yaml:"health_failure_rate" json:"health_failure_rate"`
	P95LatencyMs      int     `yaml:"p95_latency_ms" json:"p95_latency_ms"`
	ErrorRate         float64 `yaml:"error_rate" json:"error_rate"`
	QuotaUsage        float64 `yaml:"quota_usage" json:"quota_usage"`
	BillingAnomaly    float64 `yaml:"billing_anomaly" json:"billing_anomaly"`
}

// NotificationsConfig 通知配置
type NotificationsConfig struct {
	Enabled bool        `yaml:"enabled" json:"enabled"`
	Email   EmailConfig `yaml:"email" json:"email"`
	Slack   SlackConfig `yaml:"slack" json:"slack"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	SMTPHost string `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port" json:"smtp_port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	From     string `yaml:"from" json:"from"`
}

// SlackConfig Slack 配置
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	WebhookURL string `yaml:"webhook_url" json:"webhook_url"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	RedisURL string        `yaml:"redis_url" json:"redis_url"`
	TTL      time.Duration `yaml:"ttl" json:"ttl"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableCORS       bool            `yaml:"enable_cors" json:"enable_cors"`
	CORSOrigins      []string        `yaml:"cors_origins" json:"cors_origins"`
	RateLimit        RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	GatewayAllowlist []string        `yaml:"gateway_allowlist" json:"gateway_allowlist"`
	RequireTLS13     bool            `yaml:"require_tls13" json:"require_tls13"`
	ToolGrantSecret  string          `yaml:"toolgrant_secret" json:"toolgrant_secret"`
}

type HostConfig struct {
	WebAdminOrigins []string `yaml:"web_admin_origins" json:"web_admin_origins"`
}

// GatewayConfig 描述 Integration Gateway 所需配置。
type GatewayConfig struct {
	BaseURL      string        `yaml:"base_url" json:"base_url"`
	APIPrefix    string        `yaml:"api_prefix" json:"api_prefix"`
	AuthScheme   string        `yaml:"auth_scheme" json:"auth_scheme"`
	APIKey       string        `yaml:"api_key" json:"api_key"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	UserAgent    string        `yaml:"user_agent" json:"user_agent"`
	UseMock      []string      `yaml:"use_mock" json:"use_mock"`
	RefreshToken string        `yaml:"refresh_token" json:"refresh_token"`
	AuthBaseURL  string        `yaml:"auth_base_url" json:"auth_base_url"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled" json:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute" json:"requests_per_minute"`
}

// SecurityBaselineConfig 定义安全基线文件中的核心字段。
type SecurityBaselineConfig struct {
	BaselineVersion string                  `yaml:"baseline_version" json:"baseline_version"`
	MaskingRules    MaskingRulesConfig      `yaml:"masking_rules" json:"masking_rules"`
	AuditLog        AuditLogConfig          `yaml:"audit_log" json:"audit_log"`
	ToolGrant       ToolGrantBaselineConfig `yaml:"tool_grant" json:"tool_grant"`
	ConsentDefaults ConsentDefaultsConfig   `yaml:"consent_defaults" json:"consent_defaults"`
}

// MaskingRulesConfig 控制日志/数据脱敏策略。
type MaskingRulesConfig struct {
	PIIFields    []string           `yaml:"pii_fields" json:"pii_fields"`
	LogRedaction LogRedactionConfig `yaml:"log_redaction" json:"log_redaction"`
}

// LogRedactionConfig 描述日志脱敏行为。
type LogRedactionConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Placeholder string `yaml:"placeholder" json:"placeholder"`
}

// ToolGrantBaselineConfig 控制 ToolGrant 生命周期策略。
type ToolGrantBaselineConfig struct {
	TTLHours                int  `yaml:"ttl_hours" json:"ttl_hours"`
	RenewalThresholdMinutes int  `yaml:"renewal_threshold_minutes" json:"renewal_threshold_minutes"`
	RevokeOnLogout          bool `yaml:"revoke_on_logout" json:"revoke_on_logout"`
}

// ConsentDefaultsConfig 定义宿主未提供策略时的默认隐私行为。
type ConsentDefaultsConfig struct {
	RetentionDays int    `yaml:"retention_days" json:"retention_days"`
	AuditChannel  string `yaml:"audit_channel" json:"audit_channel"`
	ExportBucket  string `yaml:"export_bucket" json:"export_bucket"`
}

// AuditLogConfig 描述审计日志的保留策略与导出脚本。
type AuditLogConfig struct {
	RetentionDays int    `yaml:"retention_days" json:"retention_days"`
	ExportScript  string `yaml:"export_script" json:"export_script"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Metrics     MetricsConfig     `yaml:"metrics" json:"metrics"`
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level                     string `yaml:"level" json:"level"`
	Format                    string `yaml:"format" json:"format"`
	Output                    string `yaml:"output" json:"output"`
	FilePath                  string `yaml:"file_path" json:"file_path"`
	MaxSize                   int    `yaml:"max_size" json:"max_size"`
	MaxBackups                int    `yaml:"max_backups" json:"max_backups"`
	MaxAge                    int    `yaml:"max_age" json:"max_age"`
	HTTPAccess                bool   `yaml:"http_access" json:"http_access"`
	GinMode                   string `yaml:"gin_mode" json:"gin_mode"`
	RouteLog                  bool   `yaml:"route_log" json:"route_log"`
	DebugMode                 bool   `yaml:"debug_mode" json:"debug_mode"`
	GovernanceMode            string `yaml:"governance_mode" json:"governance_mode"`
	GovernanceDeadlineVersion string `yaml:"governance_deadline_version" json:"governance_deadline_version"`
	PluginVersion             string `yaml:"plugin_version" json:"plugin_version"`
}

// GRPCUpstream PowerX gRPC 上游配置
type GRPCUpstream struct {
	Address    string `yaml:"address" json:"address"`         // PowerX 网关/服务地址，如 "localhost:9001"
	Token      string `yaml:"token" json:"token"`             // Capability Token（插件安装后下发）
	TenantUUID string `yaml:"tenant_uuid" json:"tenant_uuid"` // 当前租户（UUID）
	UseTLS     bool   `yaml:"use_tls" json:"use_tls"`         // 上线后建议 true
	CACert     string `yaml:"ca_cert" json:"ca_cert"`         // 可选：根证书（UseTLS=true 时）
	// STS 交换短期令牌（可选）：若配置，则优先通过 STS 获取内存 Token
	STSClientID     string        `yaml:"sts_client_id" json:"sts_client_id"`
	STSClientSecret string        `yaml:"sts_client_secret" json:"sts_client_secret"`
	STSAudience     string        `yaml:"sts_audience" json:"sts_audience"`
	STSScope        string        `yaml:"sts_scope" json:"sts_scope"`
	STSTTL          time.Duration `yaml:"sts_ttl" json:"sts_ttl"`

	// 连接策略
	// eager: 启动时立刻连接（默认）
	// lazy: 首次调用时再连接（开发模式友好）
	ConnectMode string `yaml:"connect_mode" json:"connect_mode"`
	// 可选连接：为 true 时，连接失败不致命（仅建议在开发模式）
	Optional bool `yaml:"optional" json:"optional"`
}

// GRPCServer 插件 gRPC 服务器配置
type GRPCServer struct {
	Enable         bool   `yaml:"enable" json:"enable"`                     // 是否启用插件自己的 gRPC Server
	Addr           string `yaml:"addr" json:"addr"`                         // 插件 gRPC 监听，如 ":9101"
	Port           int    `yaml:"port" json:"port"`                         // 仅提供端口时自动拼接监听地址
	PortMaxRetries int    `yaml:"port_max_retries" json:"port_max_retries"` // 端口冲突时最多尝试次数
	UseTLS         bool   `yaml:"use_tls" json:"use_tls"`
	Cert           string `yaml:"cert" json:"cert"`
	Key            string `yaml:"key" json:"key"`
}

// ContextConfig PowerX 上下文相关配置
type ContextConfig struct {
	// HMAC 模式配置
	HMACSecret string `yaml:"hmac_secret" json:"hmac_secret"`
	KeyID      string `yaml:"key_id" json:"key_id"`

	// JWT 模式配置
	JWKSURL  string        `yaml:"jwks_url" json:"jwks_url"`
	Issuer   string        `yaml:"issuer" json:"issuer"`
	Audience string        `yaml:"audience" json:"audience"`
	TTL      time.Duration `yaml:"ttl" json:"ttl"`

	// IAM 模式（可选）：delegated / local，留空按环境变量规则推断
	IAMMode string `yaml:"iam_mode" json:"iam_mode"`
}

// Load 加载配置，优先级：YAML 文件 > 默认值（不再从环境变量覆盖）
func Load() (*Config, error) {
	loadEnvFiles()

	// 设置默认配置
	cfg := getDefaultConfig()

	// 尝试加载 YAML 配置文件
	configDir, err := loadYAMLConfig(cfg)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
			"module":     "config",
			"biz_scene":  "config_load",
			"biz_domain": "runtime_ops",
			"component":  "config.loader",
			"error":      err.Error(),
		}), "Failed to load YAML config, using defaults only")
	}
	if strings.TrimSpace(configDir) != "" {
		loadEnvFiles(configDir, filepath.Dir(configDir))
	}
	applyRuntimeNamespacesToLegacy(cfg)

	loadSecurityBaselineConfig(cfg)

	// 宿主注入的环境变量优先级最高，用于覆盖敏感配置（例如数据库凭据）
	loadEnvConfig(cfg)

	// 统一归一化配置值，避免大小写/空白差异导致校验失败
	normalizeConfig(cfg)
	syncLegacyNamespacesToRuntime(cfg)

	if cfg.Database != nil {
		cfg.Database.ApplyDefaults()
		if configDir != "" {
			cfg.Database.ResolvePaths(configDir)
		}
	}

	// 同步向后兼容字段
	syncBackwardCompatibility(cfg)
	overrideBindAddrFromEnv(cfg)

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getDefaultConfig 获取默认配置
func defaultSecurityBaselineConfig() *SecurityBaselineConfig {
	return &SecurityBaselineConfig{
		BaselineVersion: "2025.10",
		MaskingRules: MaskingRulesConfig{
			PIIFields: []string{"email", "phone_number", "national_id"},
			LogRedaction: LogRedactionConfig{
				Enabled:     true,
				Placeholder: "[REDACTED]",
			},
		},
		AuditLog: AuditLogConfig{
			RetentionDays: 365,
			ExportScript:  "scripts/security/audit_export.sh",
		},
		ToolGrant: ToolGrantBaselineConfig{
			TTLHours:                24,
			RenewalThresholdMinutes: 60,
			RevokeOnLogout:          true,
		},
		ConsentDefaults: ConsentDefaultsConfig{
			RetentionDays: 90,
			AuditChannel:  "stdout",
			ExportBucket:  "",
		},
	}
}

func getDefaultConfig() *Config {
	return &Config{
		DevMode: true,
		Server: &ServerConfig{
			BindAddr: ":8078",
		},
		Integration: &IntegrationConfig{
			Idempotency: IntegrationIdempotencyConfig{
				Provider: "redis",
				RedisURL: "redis://localhost:6379",
				TTLHours: 24,
			},
			Envelope: IntegrationEnvelopeConfig{
				PayloadThresholdBytes: 1 << 20,
			},
			Webhook: IntegrationWebhookConfig{
				RetryPolicy: []int{60, 300, 900},
				DLQTopic:    "plugin.webhook.dlq",
			},
			Secrets: IntegrationSecretsConfig{
				RotationDaysDefault: 30,
			},
			Billing: IntegrationBillingConfig{
				TaxProvider: "stripe_tax",
				StripeTax: IntegrationStripeTaxConfig{
					Location:       "US",
					APIBaseURL:     "https://api.stripe.com",
					TimeoutSeconds: 15,
				},
				Avalara: IntegrationAvalaraConfig{
					Environment:    "sandbox",
					BaseURL:        "https://sandbox-rest.avatax.com",
					TimeoutSeconds: 15,
				},
				Reconciliation: IntegrationRevenueSplitConfig{
					VendorShare:   0.80,
					PlatformShare: 0.15,
					FeeShare:      0.05,
					Currency:      "USD",
				},
				AsyncQueue:          "marketplace.billing.async",
				HTTPTimeoutSeconds:  15,
				RetryBackoffSeconds: []int{5, 30, 120},
			},
		},
		Database: &DatabaseConfig{
			Driver: "memory",
			Schema: "px_plugin_base",
		},
		Runtime: &RuntimeConfig{
			RunMigrate: false,
		},
		RuntimeOps: &RuntimeOpsDefaults{
			HeartbeatSeconds:           15,
			HeartbeatMisses:            3,
			QuotaWindowMinutes:         5,
			RestartBackoffStartSeconds: 5,
			RestartBackoffMaxSeconds:   120,
			LogRetentionDays:           7,
			CPUDefault:                 "500m",
			MemoryDefault:              "512Mi",
			NetworkProfile:             "standard",
			Observability: ObservabilityConfig{
				LokiEndpoint:  "",
				TempoEndpoint: "",
			},
			Alerts: AlertThresholds{
				HealthFailureRate: 0.5,
				P95LatencyMs:      500,
				ErrorRate:         0.05,
				QuotaUsage:        0.9,
				BillingAnomaly:    0.2,
			},
		},
		Operations: &OperationsConfig{
			Scheduler: OperationsSchedulerConfig{
				RetryMaxAttempts:   3,
				PauseStrategy:      "pause_on_retry_exhausted",
				ResumeRoleRequired: "ops_admin_only",
			},
		},
		Context: &ContextConfig{
			TTL: 300 * time.Second, // 5分钟
		},
		Security: &SecurityConfig{
			EnableCORS: true,
			CORSOrigins: []string{
				"http://localhost:3036",
				"http://localhost:3000",
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
			},
			GatewayAllowlist: []string{"localhost", "127.0.0.1"},
			RequireTLS13:     false,
			ToolGrantSecret:  "dev-toolgrant-secret",
		},
		Monitoring: MonitoringConfig{
			Metrics: MetricsConfig{
				Enabled: true,
				Path:    "/api/v1/admin/runtime/metrics",
			},
			HealthCheck: HealthCheckConfig{
				Enabled: true,
				Path:    "/health",
			},
		},
		Logging: &LoggingConfig{
			Level:                     "info",
			Format:                    "json",
			Output:                    "stdout",
			MaxSize:                   100,
			MaxBackups:                3,
			MaxAge:                    28,
			HTTPAccess:                true,
			GinMode:                   "",
			RouteLog:                  false,
			DebugMode:                 true,
			GovernanceMode:            "warn",
			GovernanceDeadlineVersion: "",
			PluginVersion:             "",
		},
		GRPCUpstream: &GRPCUpstream{
			Address:     "localhost:9001",
			Token:       "",
			TenantUUID:  "",
			UseTLS:      false,
			CACert:      "",
			STSAudience: "powerx:api",
			STSScope:    "access",
			STSTTL:      300 * time.Second,
			ConnectMode: "eager",
			Optional:    false,
		},
		GRPCServer: &GRPCServer{
			Enable:         true,
			Addr:           ":9101",
			Port:           9101,
			PortMaxRetries: 10,
			UseTLS:         false,
			Cert:           "",
			Key:            "",
		},
		SecurityBaseline: defaultSecurityBaselineConfig(),
		Gateway: &GatewayConfig{
			BaseURL:   "http://127.0.0.1:8077",
			APIPrefix: "/api/v1",
			UseMock:   []string{},
		},
		CustomerAuth: &CustomerAuthConfig{
			Mode:             "local",
			DelegateEndpoint: "",
			DelegateTimeout:  "3s",
			JWTIssuer:        "",
			JWTAudience:      "",
			JWTSecret:        "",
			CacheTTLSeconds:  0,
		},
	}
}

// loadYAMLConfig 加载 YAML 配置文件
func loadYAMLConfig(cfg *Config) (string, error) {
	candidates := resolveConfigCandidates()

	var configFile string
	for _, path := range candidates {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		configFile = path
		break
	}

	if configFile == "" {
		return "", fmt.Errorf("config file not found (searched: %s)", strings.Join(candidates, ", "))
	}

	// 读取文件
	file, err := os.Open(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to open config file %s: %w", configFile, err)
	}
	defer file.Close()

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return "", fmt.Errorf("failed to parse YAML config: %w", err)
	}

	dir := filepath.Dir(configFile)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir, nil
	}
	return absDir, nil
}

func loadSecurityBaselineConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	baselinePath := locateSecurityBaseline()
	if baselinePath == "" {
		if cfg.SecurityBaseline == nil {
			cfg.SecurityBaseline = defaultSecurityBaselineConfig()
		}
		return
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
			"module":     "config",
			"biz_scene":  "security_baseline_load",
			"biz_domain": "security",
			"component":  "config.loader",
			"path":       baselinePath,
			"error":      err.Error(),
		}), "Failed to read security baseline config")
		return
	}

	baseline := defaultSecurityBaselineConfig()
	if err := yaml.Unmarshal(data, baseline); err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
			"module":     "config",
			"biz_scene":  "security_baseline_load",
			"biz_domain": "security",
			"component":  "config.loader",
			"path":       baselinePath,
			"error":      err.Error(),
		}), "Failed to parse security baseline config")
		return
	}

	cfg.SecurityBaseline = baseline
}

func locateSecurityBaseline() string {
	candidates := resolveSecurityBaselineCandidates()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate
	}
	return ""
}

func resolveSecurityBaselineCandidates() []string {
	var candidates []string

	if raw := os.Getenv("CONFIG_PATH"); raw != "" {
		resolved := resolveConfigValue(raw)
		if resolved != "" {
			info, err := os.Stat(resolved)
			if err == nil && info.IsDir() {
				candidates = append(candidates, filepath.Join(resolved, "security_baseline.yaml"))
			} else {
				dir := filepath.Dir(resolved)
				candidates = append(candidates, filepath.Join(dir, "security_baseline.yaml"))
			}
		}
	}

	candidates = append(candidates,
		filepath.Join("config", "security_baseline.yaml"),
		filepath.Join("backend", "etc", "security_baseline.yaml"),
		filepath.Join("skeleton", "backend", "etc", "security_baseline.yaml"),
		"security_baseline.yaml",
	)

	return candidates
}

// SecurityBaselineConfig returns the loaded security baseline configuration,
// falling back to defaults when unavailable.
func (c *Config) SecurityBaselineConfig() *SecurityBaselineConfig {
	if c == nil {
		return defaultSecurityBaselineConfig()
	}
	if c.SecurityBaseline == nil {
		return defaultSecurityBaselineConfig()
	}
	return c.SecurityBaseline
}

// ToolGrantTTL returns the ToolGrant TTL derived from the baseline (default 24h).
func (c *Config) ToolGrantTTL() time.Duration {
	baseline := c.SecurityBaselineConfig()
	if baseline.ToolGrant.TTLHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(baseline.ToolGrant.TTLHours) * time.Hour
}

// ConsentRetentionDays returns the retention window for consent data (default 90 days).
func (c *Config) ConsentRetentionDays() int {
	baseline := c.SecurityBaselineConfig()
	if baseline.ConsentDefaults.RetentionDays <= 0 {
		return 90
	}
	return baseline.ConsentDefaults.RetentionDays
}

// AuditLogRetentionDays returns the number of days audit logs must be retained (default 365).
func (c *Config) AuditLogRetentionDays() int {
	baseline := c.SecurityBaselineConfig()
	if baseline.AuditLog.RetentionDays <= 0 {
		return 365
	}
	return baseline.AuditLog.RetentionDays
}

// AuditLogExportScript returns the recommended export helper script path.
func (c *Config) AuditLogExportScript() string {
	baseline := c.SecurityBaselineConfig()
	if baseline.AuditLog.ExportScript == "" {
		return "scripts/security/audit_export.sh"
	}
	return baseline.AuditLog.ExportScript
}

func resolveConfigCandidates() []string {
	var candidates []string

	if rawConfigPath := os.Getenv("CONFIG_PATH"); rawConfigPath != "" {
		configPath := resolveConfigValue(rawConfigPath)
		if configPath != "" {
			ext := strings.ToLower(filepath.Ext(configPath))
			if ext == ".yaml" || ext == ".yml" {
				candidates = append(candidates, configPath)
			} else {
				candidates = append(candidates,
					filepath.Join(configPath, "host-values.yaml"),
					filepath.Join(configPath, "config.yaml"),
				)
			}
		}
	}

	candidates = append(candidates,
		"./config/host-values.yaml",
		"./config/config.yaml",
		"./config.yaml",
		"./backend/etc/config.yaml",
		"./skeleton/backend/etc/config.yaml",
		"./etc/config.yaml",
		"../config/host-values.yaml",
		"../config/config.yaml",
		"../etc/config.yaml",
		"../backend/etc/config.yaml",
	)

	return uniqueNonEmptyStrings(candidates)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}

func firstEnvValue(keys ...string) string {
	for _, key := range keys {
		if v := resolveConfigValue(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func parseBoolEnvValue(raw string) (bool, bool) {
	v := strings.TrimSpace(strings.ToLower(raw))
	switch v {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

// loadEnvConfig 从环境变量加载配置，作为 YAML 的覆盖层
func loadEnvConfig(cfg *Config) {
	// 服务配置
	if addr := resolveConfigValue(os.Getenv("POWERX_BIND_ADDR")); addr != "" {
		cfg.Server.BindAddr = addr
	}
	if level := resolveConfigValue(os.Getenv("POWERX_LOG_LEVEL")); level != "" {
		normalized := strings.ToLower(level)
		cfg.Logging.Level = normalized
	}
	if format := resolveConfigValue(os.Getenv("POWERX_LOG_FORMAT")); format != "" {
		cfg.Logging.Format = strings.ToLower(format)
	}
	if output := resolveConfigValue(os.Getenv("POWERX_LOG_OUTPUT")); output != "" {
		cfg.Logging.Output = strings.ToLower(output)
	}
	if filePath := resolveConfigValue(os.Getenv("POWERX_LOG_FILE")); filePath != "" {
		cfg.Logging.FilePath = filePath
	}
	if httpLog := firstEnvValue("POWERX_PLUGIN_HTTP_LOG", "POWERX_HTTP_LOG"); httpLog != "" {
		if parsed, ok := parseBoolEnvValue(httpLog); ok {
			cfg.Logging.HTTPAccess = parsed
		}
	}
	if routeLog := firstEnvValue("POWERX_PLUGIN_ROUTE_LOG", "POWERX_ROUTE_LOG"); routeLog != "" {
		if parsed, ok := parseBoolEnvValue(routeLog); ok {
			cfg.Logging.RouteLog = parsed
		}
	}
	if ginMode := firstEnvValue("POWERX_PLUGIN_GIN_MODE", "POWERX_GIN_MODE", "GIN_MODE"); ginMode != "" {
		cfg.Logging.GinMode = strings.ToLower(ginMode)
	}
	if debugMode := resolveConfigValue(os.Getenv("POWERX_DEBUG_MODE")); debugMode != "" {
		cfg.Logging.DebugMode = (debugMode == "1" || strings.EqualFold(debugMode, "true"))
	}
	if governanceMode := resolveConfigValue(os.Getenv("POWERX_LOG_GOVERNANCE_MODE")); governanceMode != "" {
		cfg.Logging.GovernanceMode = strings.ToLower(strings.TrimSpace(governanceMode))
	}
	if deadlineVersion := resolveConfigValue(os.Getenv("POWERX_LOG_GOVERNANCE_DEADLINE_VERSION")); deadlineVersion != "" {
		cfg.Logging.GovernanceDeadlineVersion = strings.TrimSpace(deadlineVersion)
	}
	if pluginVersion := resolveConfigValue(os.Getenv("POWERX_PLUGIN_VERSION")); pluginVersion != "" {
		cfg.Logging.PluginVersion = strings.TrimSpace(pluginVersion)
	}
	if devMode := resolveConfigValue(os.Getenv("POWERX_DEV_MODE")); devMode != "" {
		cfg.Logging.DebugMode = (devMode == "1" || strings.EqualFold(devMode, "true"))
	}
	if sec := resolveConfigValue(os.Getenv("POWERX_SERVER_SECRET_KEY")); sec != "" {
		cfg.Server.SecretKey = sec
	}

	// 数据库配置
	if dsn := resolveConfigValue(os.Getenv("POWERX_DB_DSN")); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if schema := resolveConfigValue(os.Getenv("POWERX_DB_SCHEMA")); schema != "" {
		cfg.Database.Schema = schema
	}
	if secret := resolveConfigValue(os.Getenv("POWERX_TOOLGRANT_SECRET")); secret != "" {
		cfg.Security.ToolGrantSecret = secret
	}

	// 运行时配置
	if runMigrate := resolveConfigValue(os.Getenv("POWERX_RUN_MIGRATE")); strings.EqualFold(runMigrate, "true") {
		cfg.Runtime.RunMigrate = true
	}

	// 上下文配置
	if hmacSecret := resolveConfigValue(os.Getenv("PLUGIN_CTX_HMAC_SECRET")); hmacSecret != "" {
		cfg.Context.HMACSecret = hmacSecret
	}
	if keyID := resolveConfigValue(os.Getenv("PLUGIN_CTX_KID")); keyID != "" {
		cfg.Context.KeyID = keyID
	}
	if jwksURL := resolveConfigValue(os.Getenv("POWERX_CTX_JWKS_URL")); jwksURL != "" {
		cfg.Context.JWKSURL = jwksURL
	}
	if issuer := resolveConfigValue(os.Getenv("POWERX_CTX_ISSUER")); issuer != "" {
		cfg.Context.Issuer = issuer
	}
	if audience := resolveConfigValue(os.Getenv("POWERX_CTX_AUDIENCE")); audience != "" {
		cfg.Context.Audience = audience
	}
	if ttlStr := resolveConfigValue(os.Getenv("POWERX_CTX_TTL")); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			cfg.Context.TTL = ttl
		}
	}
	if iamMode := resolveConfigValue(os.Getenv("IAM_MODE")); iamMode != "" {
		cfg.Context.IAMMode = iamMode
	} else if iamModeCamel := resolveConfigValue(os.Getenv("IAMMode")); iamModeCamel != "" {
		cfg.Context.IAMMode = iamModeCamel
	}

	// gRPC 上游配置
	if grpcAddr := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_ADDRESS")); grpcAddr != "" {
		cfg.GRPCUpstream.Address = grpcAddr
	}
	if grpcToken := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_TOKEN")); grpcToken != "" {
		cfg.GRPCUpstream.Token = grpcToken
	}
	if grpcTenantUUID := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_TENANT_UUID")); grpcTenantUUID != "" {
		cfg.GRPCUpstream.TenantUUID = grpcTenantUUID
	} else if grpcTenantUuid := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_TENANT_ID")); grpcTenantUuid != "" {
		// 兼容旧变量 POWERX_GRPC_UPSTREAM_TENANT_ID，后续统一迁移为 *_TENANT_UUID。
		cfg.GRPCUpstream.TenantUUID = grpcTenantUuid
	}
	if grpcUseTLS := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_USE_TLS")); strings.EqualFold(grpcUseTLS, "true") {
		cfg.GRPCUpstream.UseTLS = true
	}
	if grpcCACert := resolveConfigValue(os.Getenv("POWERX_GRPC_UPSTREAM_CA_CERT")); grpcCACert != "" {
		cfg.GRPCUpstream.CACert = grpcCACert
	}

	// STS 相关环境变量（可选）
	if v := resolveConfigValue(os.Getenv("POWERX_STS_CLIENT_ID")); v != "" {
		cfg.GRPCUpstream.STSClientID = v
	}
	if v := resolveConfigValue(os.Getenv("POWERX_STS_CLIENT_SECRET")); v != "" {
		cfg.GRPCUpstream.STSClientSecret = v
	}
	if v := resolveConfigValue(os.Getenv("POWERX_STS_AUDIENCE")); v != "" {
		cfg.GRPCUpstream.STSAudience = v
	}
	if v := resolveConfigValue(os.Getenv("POWERX_STS_SCOPE")); v != "" {
		cfg.GRPCUpstream.STSScope = v
	}
	if v := resolveConfigValue(os.Getenv("POWERX_STS_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.GRPCUpstream.STSTTL = d
		}
	}

	// gRPC 服务器配置
	if grpcServerEnable := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_ENABLE")); strings.EqualFold(grpcServerEnable, "false") {
		cfg.GRPCServer.Enable = false
	}
	if grpcServerAddr := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_ADDR")); grpcServerAddr != "" {
		cfg.GRPCServer.Addr = grpcServerAddr
	}
	if grpcServerPort := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_PORT")); grpcServerPort != "" {
		if portVal, err := strconv.Atoi(grpcServerPort); err == nil {
			cfg.GRPCServer.Port = portVal
		}
	}
	if grpcServerRetry := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_PORT_MAX_RETRIES")); grpcServerRetry != "" {
		if retryVal, err := strconv.Atoi(grpcServerRetry); err == nil {
			cfg.GRPCServer.PortMaxRetries = retryVal
		}
	}
	if grpcServerUseTLS := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_USE_TLS")); strings.EqualFold(grpcServerUseTLS, "true") {
		cfg.GRPCServer.UseTLS = true
	}
	if grpcServerCert := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_CERT")); grpcServerCert != "" {
		cfg.GRPCServer.Cert = grpcServerCert
	}
	if grpcServerKey := resolveConfigValue(os.Getenv("POWERX_GRPC_SERVER_KEY")); grpcServerKey != "" {
		cfg.GRPCServer.Key = grpcServerKey
	}

	// Gateway 能力调用配置
	if cfg.Gateway == nil {
		cfg.Gateway = &GatewayConfig{}
	}
	if baseURL := resolveConfigValue(os.Getenv("PX_GATEWAY_BASE_URL")); baseURL != "" {
		cfg.Gateway.BaseURL = baseURL
	}
	if apiPrefix := resolveConfigValue(os.Getenv("PX_GATEWAY_API_PREFIX")); apiPrefix != "" {
		cfg.Gateway.APIPrefix = apiPrefix
	}
	if authScheme := resolveConfigValue(os.Getenv("PX_GATEWAY_AUTH_SCHEME")); authScheme != "" {
		cfg.Gateway.AuthScheme = authScheme
	}
	if apiKey := resolveConfigValue(os.Getenv("PX_GATEWAY_API_KEY")); apiKey != "" {
		cfg.Gateway.APIKey = apiKey
	}
	if timeout := resolveConfigValue(os.Getenv("PX_GATEWAY_TIMEOUT")); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Gateway.Timeout = d
		}
	}
	if ua := resolveConfigValue(os.Getenv("PX_GATEWAY_USER_AGENT")); ua != "" {
		cfg.Gateway.UserAgent = ua
	}
	if mockModules := resolveConfigValue(os.Getenv("PX_USE_MOCK")); mockModules != "" {
		cfg.Gateway.UseMock = splitCSV(mockModules)
	}
	if authBase := resolveConfigValue(os.Getenv("PX_AUTH_BASE_URL")); authBase != "" {
		cfg.Gateway.AuthBaseURL = authBase
	}
}

// syncBackwardCompatibility 同步向后兼容字段
func syncBackwardCompatibility(cfg *Config) {
	cfg.BindAddr = cfg.Server.BindAddr
	cfg.LogLevel = cfg.Logging.Level
	cfg.DevMode = cfg.Logging.DebugMode
	cfg.DBDSN = cfg.Database.DSN
	cfg.DBSchema = cfg.Database.Schema
	cfg.RunMigrate = cfg.Runtime.RunMigrate
}

func overrideBindAddrFromEnv(cfg *Config) {
	if cfg == nil {
		return
	}
	// 优先使用宿主明确下发的完整地址
	addr := strings.TrimSpace(os.Getenv("POWERX_HTTP_ADDR"))

	// 其次使用宿主注入的动态端口（拼成 :<port>）
	if addr == "" {
		if v := strings.TrimSpace(os.Getenv("POWERX_DYNAMIC_PORT")); v != "" {
			addr = ":" + v
		}
	}

	// 兜底：常见 PaaS 的 PORT
	if addr == "" {
		if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
			addr = ":" + v
		}
	}

	if addr != "" {
		if cfg.Server == nil {
			cfg.Server = &ServerConfig{}
		}
		cfg.Server.BindAddr = addr
		cfg.BindAddr = addr
	} else {
		// 至少保持一致
		cfg.BindAddr = cfg.Server.BindAddr
	}
}

func normalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	applyHostWebAdminOrigins(cfg)
	if cfg.Server != nil {
		cfg.Server.BindAddr = resolveConfigValue(cfg.Server.BindAddr)
	}
	if cfg.Runtime == nil {
		cfg.Runtime = &RuntimeConfig{}
	}
	if cfg.Gateway != nil {
		cfg.Gateway.BaseURL = resolveConfigValue(cfg.Gateway.BaseURL)
		cfg.Gateway.APIPrefix = normalizeGatewayAPIPrefix(resolveConfigValue(cfg.Gateway.APIPrefix))
		cfg.Gateway.AuthScheme = normalizeGatewayAuthScheme(resolveConfigValue(cfg.Gateway.AuthScheme))
		cfg.Gateway.RefreshToken = ""
		cfg.Gateway.APIKey = resolveConfigValue(cfg.Gateway.APIKey)
		cfg.Gateway.AuthBaseURL = resolveConfigValue(cfg.Gateway.AuthBaseURL)

		if cfg.Gateway.AuthScheme == "" {
			cfg.Gateway.AuthScheme = inferGatewayAuthScheme(cfg)
		}

		// Dev 模式下：Gateway 配置不完整时不阻塞启动，改为打印提示并自动关闭 Gateway。
		// 生产/非 Dev 场景仍保持严格校验（见 Validate）。
		if cfg.Logging != nil && cfg.Logging.DebugMode && !isHostDelegatedMode(cfg) {
			baseURL := strings.TrimSpace(cfg.Gateway.BaseURL)
			apiKey := strings.TrimSpace(cfg.Gateway.APIKey)

			hasAny := baseURL != "" || apiKey != ""
			incomplete := baseURL == "" || !hasGatewayCredential(cfg.Gateway)

			if hasAny && incomplete {
				pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
					"module":              "config",
					"biz_scene":           "gateway_normalize",
					"biz_domain":          "integration",
					"component":           "config.loader",
					"gateway.base_url":    baseURL,
					"gateway.auth_scheme": cfg.Gateway.AuthScheme,
					"gateway.api_key":     apiKey != "",
				}), "Gateway config is incomplete; gateway disabled in dev mode (set gateway.base_url + selected credential)")

				cfg.Gateway.BaseURL = ""
				cfg.Gateway.AuthScheme = ""
				cfg.Gateway.APIKey = ""
			}
		}
	}
	if cfg.Logging != nil {
		cfg.Logging.Level = strings.ToLower(resolveConfigValue(cfg.Logging.Level))
		cfg.Logging.Format = strings.ToLower(resolveConfigValue(cfg.Logging.Format))
		cfg.Logging.Output = strings.ToLower(resolveConfigValue(cfg.Logging.Output))
		cfg.Logging.GovernanceMode = strings.ToLower(strings.TrimSpace(resolveConfigValue(cfg.Logging.GovernanceMode)))
		cfg.Logging.GovernanceDeadlineVersion = strings.TrimSpace(resolveConfigValue(cfg.Logging.GovernanceDeadlineVersion))
		cfg.Logging.PluginVersion = strings.TrimSpace(resolveConfigValue(cfg.Logging.PluginVersion))
	}
	if cfg.GRPCUpstream != nil {
		cfg.GRPCUpstream.Address = resolveConfigValue(cfg.GRPCUpstream.Address)
		cfg.GRPCUpstream.Token = resolveConfigValue(cfg.GRPCUpstream.Token)
		cfg.GRPCUpstream.TenantUUID = strings.ToLower(resolveConfigValue(cfg.GRPCUpstream.TenantUUID))
		cfg.GRPCUpstream.STSClientID = resolveConfigValue(cfg.GRPCUpstream.STSClientID)
		cfg.GRPCUpstream.STSClientSecret = resolveConfigValue(cfg.GRPCUpstream.STSClientSecret)
		cfg.GRPCUpstream.STSAudience = resolveConfigValue(cfg.GRPCUpstream.STSAudience)
		cfg.GRPCUpstream.STSScope = resolveConfigValue(cfg.GRPCUpstream.STSScope)
	}
	if cfg.GRPCServer != nil {
		cfg.GRPCServer.Addr = resolveConfigValue(cfg.GRPCServer.Addr)
		normalizeGRPCServerConfig(cfg.GRPCServer)
	}
	if cfg.CustomerAuth != nil {
		cfg.CustomerAuth.Mode = strings.ToLower(resolveConfigValue(cfg.CustomerAuth.Mode))
		cfg.CustomerAuth.DelegateEndpoint = resolveConfigValue(cfg.CustomerAuth.DelegateEndpoint)
		cfg.CustomerAuth.DelegateTimeout = resolveConfigValue(cfg.CustomerAuth.DelegateTimeout)
		cfg.CustomerAuth.JWTIssuer = resolveConfigValue(cfg.CustomerAuth.JWTIssuer)
		cfg.CustomerAuth.JWTAudience = resolveConfigValue(cfg.CustomerAuth.JWTAudience)
		cfg.CustomerAuth.JWTSecret = resolveConfigValue(cfg.CustomerAuth.JWTSecret)
	}
}

func isHostDelegatedMode(cfg *Config) bool {
	if strings.TrimSpace(os.Getenv("POWERX_PROXY")) != "1" {
		return false
	}
	return cfg != nil && cfg.Context != nil && strings.ToLower(strings.TrimSpace(cfg.Context.IAMMode)) == "delegated"
}

func normalizeGRPCServerConfig(server *GRPCServer) {
	if server == nil {
		return
	}
	if server.Port <= 0 {
		if port := extractPort(server.Addr); port > 0 {
			server.Port = port
		} else {
			server.Port = defaultGRPCPort
		}
	}
	if server.PortMaxRetries <= 0 {
		server.PortMaxRetries = defaultGRPCPortRetries
	}
}

func extractPort(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0
	}
	if !strings.Contains(addr, ":") {
		return 0
	}
	if _, portStr, err := net.SplitHostPort(addr); err == nil {
		if portVal, err := strconv.Atoi(portStr); err == nil {
			return portVal
		}
	}
	return 0
}

func applyHostWebAdminOrigins(cfg *Config) {
	if cfg == nil || cfg.Host == nil || len(cfg.Host.WebAdminOrigins) == 0 {
		return
	}
	if cfg.Security == nil {
		cfg.Security = &SecurityConfig{}
	}
	cfg.Security.EnableCORS = true
	cfg.Security.CORSOrigins = appendUniqueStrings(cfg.Security.CORSOrigins, cfg.Host.WebAdminOrigins...)
}

func appendUniqueStrings(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))
	for _, value := range append(base, values...) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func resolveConfigValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	const maxDepth = 4
	resolved, _ := resolvePlaceholder(trimmed, 0, maxDepth)
	return resolved
}

func resolvePlaceholder(value string, depth, maxDepth int) (string, bool) {
	if depth > maxDepth {
		return value, false
	}
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return value, false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}"))
	if inner == "" {
		return "", true
	}
	varName := inner
	defaultVal := ""
	if idx := strings.Index(inner, ":-"); idx >= 0 {
		varName = inner[:idx]
		defaultVal = inner[idx+2:]
	}
	varName = strings.TrimSpace(varName)
	if varName == "" {
		return strings.TrimSpace(defaultVal), true
	}
	if envVal, ok := os.LookupEnv(varName); ok {
		envVal = strings.TrimSpace(envVal)
		if envVal != "" && envVal != value {
			return resolveConfigValueWithDepth(envVal, depth+1, maxDepth), true
		}
	}
	return strings.TrimSpace(defaultVal), true
}

func resolveConfigValueWithDepth(value string, depth, maxDepth int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "${") || !strings.HasSuffix(trimmed, "}") {
		return trimmed
	}
	resolved, _ := resolvePlaceholder(trimmed, depth, maxDepth)
	return resolved
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func splitCSV(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeGatewayAuthScheme(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bearer":
		return "bearer"
	case "apikey", "api_key", "api-key":
		return "apikey"
	default:
		return ""
	}
}

func normalizeGatewayAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}

func inferGatewayAuthScheme(cfg *Config) string {
	if cfg != nil && cfg.Gateway != nil && strings.TrimSpace(cfg.Gateway.APIKey) != "" {
		return "apikey"
	}
	return "bearer"
}

func hasGatewayCredential(cfg *GatewayConfig) bool {
	if cfg == nil {
		return false
	}
	switch normalizeGatewayAuthScheme(cfg.AuthScheme) {
	case "bearer":
		return true
	case "apikey":
		return strings.TrimSpace(cfg.APIKey) != ""
	default:
		return false
	}
}

// GetString 获取字符串配置，支持默认值
func GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt 获取整数配置，支持默认值
func GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetBool 获取布尔配置，支持默认值
func GetBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "true" || value == "1" {
		return true
	}
	if value == "false" || value == "0" {
		return false
	}
	return defaultValue
}

// GetDuration 获取时间间隔配置，支持默认值
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	if c == nil || c.Logging == nil {
		return !c.DevMode
	}
	return !c.Logging.DebugMode
}

// IsHMACMode 判断是否使用 HMAC 模式
func (c *Config) IsHMACMode() bool {
	return c.Context.HMACSecret != ""
}

// IsJWTMode 判断是否使用 JWT 模式
func (c *Config) IsJWTMode() bool {
	return c.Context.JWKSURL != ""
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 数据库配置验证
	if c.Database == nil {
		return NewConfigError("database config is required")
	}
	c.Database.ApplyDefaults()
	if err := c.Database.Validate(); err != nil {
		return NewConfigError(err.Error())
	}

	// 认证模式验证
	if c.IsProduction() && !c.IsHMACMode() && !c.IsJWTMode() {
		return NewConfigError("either HMAC or JWT mode must be configured in production")
	}

	// Customer 鉴权配置验证
	if c.CustomerAuth == nil {
		c.CustomerAuth = &CustomerAuthConfig{Mode: "local", DelegateTimeout: "3s"}
	}
	if mode := strings.ToLower(strings.TrimSpace(c.CustomerAuth.Mode)); mode == "" {
		c.CustomerAuth.Mode = "local"
	} else {
		c.CustomerAuth.Mode = mode
	}
	switch c.CustomerAuth.Mode {
	case "local", "local_dev", "mock", "delegate", "platform", "third_party":
	default:
		return NewConfigError("customer_auth.mode must be one of: local, local_dev, mock, delegate, platform, third_party")
	}

	if strings.TrimSpace(c.CustomerAuth.DelegateTimeout) == "" {
		c.CustomerAuth.DelegateTimeout = "3s"
	}
	if _, err := time.ParseDuration(c.CustomerAuth.DelegateTimeout); err != nil {
		return NewConfigError("customer_auth.delegate_timeout must be a valid duration (e.g. 3s, 500ms)")
	}
	if c.CustomerAuth.CacheTTLSeconds < 0 {
		return NewConfigError("customer_auth.cache_ttl_seconds must be non-negative")
	}

	if c.CustomerAuth.Mode == "delegate" || c.CustomerAuth.Mode == "platform" || c.CustomerAuth.Mode == "third_party" {
		endpoint := strings.TrimSpace(c.CustomerAuth.DelegateEndpoint)
		if endpoint == "" {
			return NewConfigError("customer_auth.delegate_endpoint is required when customer_auth.mode delegates to platform")
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return NewConfigError("customer_auth.delegate_endpoint must be an absolute URL")
		}
	}

	sourceMode := customerfw.CustomerAuthSource(c.CustomerAuth.Mode)
	if c.CustomerAuth.Mode == "local" {
		sourceMode = customerfw.CustomerAuthSourceLocal
	}
	if err := customerfw.ValidateSourcePolicy(customerfw.SourcePolicy{
		Mode:       sourceMode,
		Production: c.IsProduction(),
		BreakGlass: c.CustomerAuth.BreakGlassLocal,
	}); err != nil {
		return NewConfigError(err.Error())
	}
	if c.IsProduction() && c.CustomerAuth.BreakGlassLocal && strings.TrimSpace(c.CustomerAuth.BreakGlassReason) == "" {
		return NewConfigError("customer_auth.break_glass_reason is required when break_glass_local is enabled")
	}

	// EventBridge 配置默认值与验证
	if c.EventBridge == nil {
		c.EventBridge = &EventBridgeConfig{
			Enabled:         false,
			Mode:            "",
			FallbackToLocal: true,
			LocalQueueSize:  1024,
			TaskBusProvider: "redis",
			RedisURL:        "redis://localhost:6379",
			RedisStream:     "powerx.taskbus.events",
			RedisGroup:      "powerx.plugin",
			RedisConsumer:   "",
			RedisMaxLen:     10000,
			SourcePlugin:    "",
			PayloadVersion:  "v1",
		}
	}
	if c.EventBridge.LocalQueueSize <= 0 {
		c.EventBridge.LocalQueueSize = 1024
	}
	if strings.TrimSpace(c.EventBridge.PayloadVersion) == "" {
		c.EventBridge.PayloadVersion = "v1"
	}
	providerMode := strings.ToLower(strings.TrimSpace(c.EventBridge.TaskBusProvider))
	if providerMode == "" {
		c.EventBridge.TaskBusProvider = "redis"
	} else {
		switch providerMode {
		case "host", "redis":
			c.EventBridge.TaskBusProvider = providerMode
		default:
			return NewConfigError("event_bridge.taskbus_provider must be one of: host, redis")
		}
	}
	if strings.TrimSpace(c.EventBridge.RedisURL) == "" {
		c.EventBridge.RedisURL = "redis://localhost:6379"
	}
	if strings.TrimSpace(c.EventBridge.RedisStream) == "" {
		c.EventBridge.RedisStream = "powerx.taskbus.events"
	}
	if strings.TrimSpace(c.EventBridge.RedisGroup) == "" {
		c.EventBridge.RedisGroup = "powerx.plugin"
	}
	if c.EventBridge.RedisMaxLen <= 0 {
		c.EventBridge.RedisMaxLen = 10000
	}
	mode := strings.ToLower(strings.TrimSpace(c.EventBridge.Mode))
	if mode == "" {
		if c.EventBridge.Enabled {
			mode = "taskbus"
		} else {
			mode = "local"
		}
	}
	switch mode {
	case "local", "taskbus", "dual":
		c.EventBridge.Mode = mode
	default:
		return NewConfigError("event_bridge.mode must be one of: local, taskbus, dual")
	}
	if !c.EventBridge.Enabled && c.EventBridge.Mode != "local" {
		c.EventBridge.Mode = "local"
	}

	// WS Bus 配置默认值与验证
	if c.WSBus == nil {
		c.WSBus = &WSBusConfig{
			Provider: "memory",
			Channel:  "powerx.wsbus",
		}
	}
	provider := strings.ToLower(strings.TrimSpace(c.WSBus.Provider))
	if provider == "" {
		provider = "memory"
	}
	switch provider {
	case "memory":
		c.WSBus.Provider = "memory"
	case "redis":
		c.WSBus.Provider = "redis"
		if strings.TrimSpace(c.WSBus.RedisURL) == "" {
			return NewConfigError("ws_bus.redis_url is required when provider=redis")
		}
	default:
		return NewConfigError("ws_bus.provider must be one of: memory, redis")
	}
	if strings.TrimSpace(c.WSBus.Channel) == "" {
		c.WSBus.Channel = "powerx.wsbus"
	}

	// Scheduler 配置默认值与验证
	if c.Operations == nil {
		c.Operations = &OperationsConfig{}
	}
	if c.Operations.Scheduler.RetryMaxAttempts == 0 {
		c.Operations.Scheduler.RetryMaxAttempts = 3
	}
	if c.Operations.Scheduler.RetryMaxAttempts < 1 || c.Operations.Scheduler.RetryMaxAttempts > 10 {
		return NewConfigError("operations.scheduler.retry_max_attempts must be between 1 and 10")
	}
	if strings.TrimSpace(c.Operations.Scheduler.PauseStrategy) == "" {
		c.Operations.Scheduler.PauseStrategy = "pause_on_retry_exhausted"
	}
	switch strings.ToLower(strings.TrimSpace(c.Operations.Scheduler.PauseStrategy)) {
	case "pause_on_retry_exhausted":
		c.Operations.Scheduler.PauseStrategy = "pause_on_retry_exhausted"
	default:
		return NewConfigError("operations.scheduler.pause_strategy must be: pause_on_retry_exhausted")
	}
	if strings.TrimSpace(c.Operations.Scheduler.ResumeRoleRequired) == "" {
		c.Operations.Scheduler.ResumeRoleRequired = "ops_admin_only"
	}
	switch strings.ToLower(strings.TrimSpace(c.Operations.Scheduler.ResumeRoleRequired)) {
	case "ops_admin_only":
		c.Operations.Scheduler.ResumeRoleRequired = "ops_admin_only"
	default:
		return NewConfigError("operations.scheduler.resume_role_required must be: ops_admin_only")
	}

	// 安全配置验证
	if c.Security.RateLimit.Enabled && c.Security.RateLimit.RequestsPerMinute <= 0 {
		return NewConfigError("rate limit requests per minute must be positive when enabled")
	}

	baseline := c.SecurityBaselineConfig()
	if baseline.ToolGrant.TTLHours <= 0 {
		return NewConfigError("security baseline: tool_grant.ttl_hours must be positive")
	}
	if baseline.ConsentDefaults.RetentionDays <= 0 {
		return NewConfigError("security baseline: consent_defaults.retention_days must be positive")
	}
	if baseline.AuditLog.RetentionDays <= 0 {
		return NewConfigError("security baseline: audit_log.retention_days must be positive")
	}

	// 日志配置验证
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return NewConfigError("logging level must be one of: debug, info, warn, error")
	}

	validLogFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validLogFormats[c.Logging.Format] {
		return NewConfigError("logging format must be one of: json, text")
	}

	validLogOutputs := map[string]bool{
		"stdout": true, "stderr": true, "file": true,
	}
	if !validLogOutputs[c.Logging.Output] {
		return NewConfigError("logging output must be one of: stdout, stderr, file")
	}

	if c.Logging.Output == "file" && c.Logging.FilePath == "" {
		return NewConfigError("logging file path must be specified when output is 'file'")
	}
	if c.Logging.GovernanceMode == "" {
		c.Logging.GovernanceMode = "warn"
	}
	switch c.Logging.GovernanceMode {
	case "detect", "warn", "block":
	default:
		return NewConfigError("logging governance_mode must be one of: detect, warn, block")
	}

	if c.GRPCUpstream != nil {
		if tenantUUID := strings.TrimSpace(c.GRPCUpstream.TenantUUID); tenantUUID != "" {
			if _, err := uuid.Parse(tenantUUID); err != nil {
				return NewConfigError("grpc_upstream.tenant_uuid must be a valid UUID string")
			}
		}
	}

	if c.GRPCServer != nil {
		if c.GRPCServer.Port <= 0 {
			return NewConfigError("grpc_server.port must be positive")
		}
		if c.GRPCServer.PortMaxRetries < 1 {
			return NewConfigError("grpc_server.port_max_retries must be positive")
		}
	}

	if c.Gateway != nil {
		hasGatewayFields := strings.TrimSpace(c.Gateway.BaseURL) != "" ||
			strings.TrimSpace(c.Gateway.APIKey) != ""
		if hasGatewayFields {
			c.Gateway.AuthScheme = normalizeGatewayAuthScheme(c.Gateway.AuthScheme)
			if c.Gateway.AuthScheme == "" {
				c.Gateway.AuthScheme = inferGatewayAuthScheme(c)
			}
			if isHostDelegatedMode(c) {
				if strings.TrimSpace(c.Gateway.BaseURL) == "" || c.Gateway.AuthScheme != "bearer" {
					return NewConfigError("host delegated gateway config requires base_url + bearer auth_scheme")
				}
				if c.GRPCUpstream == nil ||
					strings.TrimSpace(c.GRPCUpstream.STSClientID) == "" ||
					strings.TrimSpace(c.GRPCUpstream.STSClientSecret) == "" ||
					strings.TrimSpace(c.GRPCUpstream.Address) == "" ||
					strings.TrimSpace(c.GRPCUpstream.TenantUUID) == "" {
					return NewConfigError("host delegated gateway config requires base_url + bearer auth_scheme + STS client")
				}
				return nil
			}
			if strings.TrimSpace(c.Gateway.BaseURL) == "" || !hasGatewayCredential(c.Gateway) {
				return NewConfigError("gateway config requires base_url + credential matching auth_scheme")
			}
		}
	}

	return nil
}

// ShouldBlockLegacyLogging reports whether direct legacy logging must be blocked
// under current governance policy.
func (c *Config) ShouldBlockLegacyLogging() (bool, string) {
	if c == nil || c.Logging == nil {
		return false, ""
	}
	mode := strings.ToLower(strings.TrimSpace(c.Logging.GovernanceMode))
	switch mode {
	case "block":
		return true, "governance_mode=block"
	case "detect", "warn":
	default:
		mode = "warn"
	}

	deadline := strings.TrimSpace(c.Logging.GovernanceDeadlineVersion)
	current := strings.TrimSpace(c.Logging.PluginVersion)
	if deadline == "" || current == "" {
		return false, ""
	}
	if compareVersion(current, deadline) >= 0 {
		return true, fmt.Sprintf("plugin_version(%s) reached governance_deadline_version(%s)", current, deadline)
	}
	return false, ""
}

func compareVersion(current, target string) int {
	curParts := normalizeVersionParts(current)
	tgtParts := normalizeVersionParts(target)
	maxLen := len(curParts)
	if len(tgtParts) > maxLen {
		maxLen = len(tgtParts)
	}
	for i := 0; i < maxLen; i++ {
		curVal := 0
		if i < len(curParts) {
			curVal = curParts[i]
		}
		tgtVal := 0
		if i < len(tgtParts) {
			tgtVal = tgtParts[i]
		}
		if curVal > tgtVal {
			return 1
		}
		if curVal < tgtVal {
			return -1
		}
	}
	return 0
}

func normalizeVersionParts(raw string) []int {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(raw), "v"))
	if trimmed == "" {
		return []int{0}
	}
	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	parts := make([]int, 0, len(segments))
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		numericPrefix := seg
		for i, ch := range seg {
			if ch < '0' || ch > '9' {
				numericPrefix = seg[:i]
				break
			}
		}
		if numericPrefix == "" {
			parts = append(parts, 0)
			continue
		}
		value, err := strconv.Atoi(numericPrefix)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return []int{0}
	}
	return parts
}

// ConfigError 配置错误类型
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Message
}

func NewConfigError(message string) *ConfigError {
	return &ConfigError{Message: message}
}

func tenantUUIDFromJWT(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	tid, _ := claims["tid"].(string)
	return strings.ToLower(strings.TrimSpace(tid))
}
