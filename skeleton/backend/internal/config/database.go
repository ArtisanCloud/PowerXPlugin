// package config 或者你原来的包
package config

import (
	"errors"
	"fmt"
	"time"
)

type DatabaseConfig struct {
	// 连接串：postgres://user:pass@host:5432/dbname?sslmode=disable
	DSN string `yaml:"dsn" json:"dsn"`
	// 默认 schema（会在连接后执行 CREATE SCHEMA IF NOT EXISTS + SET search_path）
	Schema string `yaml:"schema" json:"schema"`

	// ---------- 连接池 ----------
	MaxIdleConns    int           `yaml:"maxIdleConns" json:"maxIdleConns"`       // 默认 10
	MaxOpenConns    int           `yaml:"maxOpenConns" json:"maxOpenConns"`       // 默认 100
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"` // 默认 1h
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" json:"connMaxIdleTime"` // 默认 30m

	// ---------- 观测 & 性能 ----------
	SlowThreshold        time.Duration `yaml:"slowThreshold" json:"slowThreshold"`               // 默认 200ms（gorm 慢查询阈值）
	PreferSimpleProtocol bool          `yaml:"preferSimpleProtocol" json:"preferSimpleProtocol"` // 默认 true
	PrepareStmt          bool          `yaml:"prepareStmt" json:"prepareStmt"`                   // 默认 false（按需开启）
	SkipDefaultTx        bool          `yaml:"skipDefaultTx" json:"skipDefaultTx"`               // 默认 false

	// ---------- 日志 ----------
	// debug|info|warn|error|silent（不区分大小写）
	LogLevel string `yaml:"logLevel" json:"logLevel"` // 默认 "silent"
	DevMode  bool   `yaml:"devMode" json:"devMode"`   // 默认 false（为 true 时，LogLevel 自动提升到 info）

	// ---------- 运维 ----------
	HealthTimeout time.Duration `yaml:"healthTimeout" json:"healthTimeout"` // 默认 5s
}

func (c *DatabaseConfig) ApplyDefaults() {
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 10
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 100
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = time.Hour
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}
	if c.SlowThreshold == 0 {
		c.SlowThreshold = 200 * time.Millisecond
	}
	// 默认更轻的协议，降低 CPU
	if !c.PreferSimpleProtocol {
		c.PreferSimpleProtocol = true
	}
	if c.LogLevel == "" {
		c.LogLevel = "silent"
	}
	if c.HealthTimeout == 0 {
		c.HealthTimeout = 5 * time.Second
	}
	// Dev 模式自动抬升日志
	if c.DevMode && (c.LogLevel == "silent" || c.LogLevel == "error") {
		c.LogLevel = "info"
	}
}

func (c *DatabaseConfig) Validate() error {
	if c.DSN == "" {
		return errors.New("database dsn is required")
	}
	if c.Schema == "" {
		return errors.New("database schema is required")
	}
	if c.MaxIdleConns < 0 || c.MaxOpenConns < 0 {
		return errors.New("maxIdleConns/maxOpenConns cannot be negative")
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("maxIdleConns(%d) cannot be greater than maxOpenConns(%d)", c.MaxIdleConns, c.MaxOpenConns)
	}
	return nil
}
