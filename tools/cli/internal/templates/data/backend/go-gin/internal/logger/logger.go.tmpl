package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/sirupsen/logrus"
)

// Logger 全局日志实例
var Logger *logrus.Logger
var httpAccessEnabled = true
var (
	backendMu sync.RWMutex
	backend   *slog.Logger
)

// Fields 日志字段类型别名
type Fields = logrus.Fields
type Entry = logrus.Entry
type LoggerType = logrus.Logger
type Level = logrus.Level

const (
	PanicLevel Level = logrus.PanicLevel
	FatalLevel Level = logrus.FatalLevel
	ErrorLevel Level = logrus.ErrorLevel
	WarnLevel  Level = logrus.WarnLevel
	InfoLevel  Level = logrus.InfoLevel
	DebugLevel Level = logrus.DebugLevel
	TraceLevel Level = logrus.TraceLevel
)

// Init initializes unified logging backend and compatibility bridge.
func Init(level, format, output, filePath string, maxSize, maxBackups, maxAge int, httpAccess bool) {
	Logger = logrus.New()
	httpAccessEnabled = httpAccess

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Logger.SetLevel(logLevel)

	logFormat := strings.ToLower(strings.TrimSpace(format))
	if logFormat == "" {
		logFormat = "json"
	}
	runtime, err := runtimelogging.Init(runtimelogging.Config{
		Policy: runtimelogging.Policy{
			Mode:   runtimelogging.ModeStandalone,
			Sinks:  []runtimelogging.SinkType{runtimelogging.SinkType(strings.ToLower(strings.TrimSpace(output)))},
			Format: logFormat,
			Level:  logLevel.String(),
			Retry: runtimelogging.RetryPolicy{
				Enabled:     true,
				MaxAttempts: 3,
				BackoffMS:   200,
			},
		},
		File: runtimelogging.FileOptions{
			Path:       strings.TrimSpace(filePath),
			MaxSizeMB:  maxSize,
			MaxBackups: maxBackups,
			MaxAgeDays: maxAge,
			Cleanup:    true,
		},
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		SetDefault:   true,
		CleanupFiles: true,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "logger output=%s fallback to stdout: %v\n", output, err)
		runtime, _ = runtimelogging.Init(runtimelogging.Config{
			Policy:       runtimelogging.DefaultPolicy(),
			Stdout:       os.Stdout,
			Stderr:       os.Stderr,
			SetDefault:   true,
			CleanupFiles: false,
		})
	}
	setBackendLogger(runtime.Logger)
	slog.SetDefault(Slog())

	// logrus 仅作为兼容层，统一转发到 slog backend
	Logger.SetOutput(io.Discard)
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.ReplaceHooks(make(logrus.LevelHooks))
	Logger.AddHook(&slogForwardHook{backend: Slog()})

	// 添加调用位置信息（仅在 debug 模式）
	if logLevel == logrus.DebugLevel || logLevel == logrus.TraceLevel {
		Logger.SetReportCaller(true)
	}

	// 标准 logrus 也走同一转发路径，覆盖遗留 logrus.WithField 直调调用点
	std := logrus.StandardLogger()
	std.SetLevel(logLevel)
	std.SetOutput(io.Discard)
	std.SetReportCaller(Logger.ReportCaller)
	std.ReplaceHooks(make(logrus.LevelHooks))
	std.AddHook(&slogForwardHook{backend: Slog()})
}

func mapLogrusLevel(level logrus.Level) slog.Leveler {
	switch level {
	case logrus.TraceLevel:
		return slog.LevelDebug - 4
	case logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.WarnLevel:
		return slog.LevelWarn
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func mapEntryLevel(level logrus.Level) slog.Level {
	switch level {
	case logrus.TraceLevel:
		return slog.LevelDebug - 4
	case logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

type slogForwardHook struct {
	backend *slog.Logger
}

func (h *slogForwardHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *slogForwardHook) Fire(entry *logrus.Entry) error {
	logger := h.backend
	if logger == nil {
		logger = Slog()
	}
	attrs := make([]slog.Attr, 0, len(entry.Data)+1)
	for k, v := range entry.Data {
		attrs = append(attrs, slog.Any(k, v))
	}
	if entry.Caller != nil {
		attrs = append(attrs, slog.String("caller", fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line)))
		attrs = append(attrs, slog.String("func", path.Base(entry.Caller.Function)))
	}

	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}
	logger.LogAttrs(ctx, mapEntryLevel(entry.Level), entry.Message, attrs...)
	return nil
}

func setBackendLogger(l *slog.Logger) {
	backendMu.Lock()
	defer backendMu.Unlock()
	backend = l
}

// Slog returns the unified backend logger used by both slog and logrus compatibility path.
func Slog() *slog.Logger {
	backendMu.RLock()
	defer backendMu.RUnlock()
	if backend == nil {
		return slog.Default()
	}
	return backend
}

// SetOutput 设置日志输出
func SetOutput(output io.Writer) {
	if Logger != nil {
		Logger.SetOutput(output)
	}
}

// HTTPAccessEnabled controls request logging in middleware.
func HTTPAccessEnabled() bool {
	return httpAccessEnabled
}

// WithFields 创建带字段的日志条目
func WithFields(fields logrus.Fields) *logrus.Entry {
	if Logger == nil {
		return logrus.WithFields(fields)
	}
	return Logger.WithFields(fields)
}

// WithField 创建带单个字段的日志条目
func WithField(key string, value interface{}) *logrus.Entry {
	if Logger == nil {
		return logrus.WithField(key, value)
	}
	return Logger.WithField(key, value)
}

// WithError 创建带错误的日志条目
func WithError(err error) *logrus.Entry {
	if Logger == nil {
		return logrus.WithError(err)
	}
	return Logger.WithError(err)
}

// New returns a logrus-compatible logger from skeleton logger package.
func New() *logrus.Logger {
	if Logger != nil {
		return Logger
	}
	return logrus.New()
}

// StandardLogger returns the global standard logger.
func StandardLogger() *logrus.Logger {
	if Logger != nil {
		return Logger
	}
	return logrus.StandardLogger()
}

// NewEntry creates a new log entry with a target logger.
func NewEntry(l *logrus.Logger) *logrus.Entry {
	if l == nil {
		l = StandardLogger()
	}
	return logrus.NewEntry(l)
}

// Debug 调试日志
func Debug(args ...interface{}) {
	if Logger == nil {
		logrus.Debug(args...)
		return
	}
	Logger.Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Debugf(format, args...)
		return
	}
	Logger.Debugf(format, args...)
}

// Info 信息日志
func Info(args ...interface{}) {
	if Logger == nil {
		logrus.Info(args...)
		return
	}
	Logger.Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Infof(format, args...)
		return
	}
	Logger.Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...interface{}) {
	if Logger == nil {
		logrus.Warn(args...)
		return
	}
	Logger.Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Warnf(format, args...)
		return
	}
	Logger.Warnf(format, args...)
}

// Error 错误日志
func Error(args ...interface{}) {
	if Logger == nil {
		logrus.Error(args...)
		return
	}
	Logger.Error(args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Errorf(format, args...)
		return
	}
	Logger.Errorf(format, args...)
}

// Fatal 致命错误日志
func Fatal(args ...interface{}) {
	if Logger == nil {
		logrus.Fatal(args...)
		return
	}
	Logger.Fatal(args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Fatalf(format, args...)
		return
	}
	Logger.Fatalf(format, args...)
}

// Panic panic 日志
func Panic(args ...interface{}) {
	if Logger == nil {
		logrus.Panic(args...)
		return
	}
	Logger.Panic(args...)
}

// Panicf 格式化 panic 日志
func Panicf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Panicf(format, args...)
		return
	}
	Logger.Panicf(format, args...)
}

// HTTPMiddleware 创建 HTTP 中间件日志
func HTTPMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "http",
	})
}

// DBMiddleware 创建数据库中间件日志
func DBMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "database",
	})
}

// AuthMiddleware 创建认证中间件日志
func AuthMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "auth",
	})
}

// ServiceLogger 创建服务层日志
func ServiceLogger(service string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "service",
		"service":   service,
	})
}

// RepoLogger 创建仓储层日志
func RepoLogger(repo string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "repository",
		"repo":      repo,
	})
}

// HandlerLogger 创建处理器日志
func HandlerLogger(handler string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "handler",
		"handler":   handler,
	})
}
