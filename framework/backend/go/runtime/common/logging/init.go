package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Policy       Policy
	HostMode     bool
	File         FileOptions
	Stdout       io.Writer
	Stderr       io.Writer
	SetDefault   bool
	CleanupFiles bool
}

type Runtime struct {
	Logger        *slog.Logger
	Policy        Policy
	PrimaryWriter io.Writer
}

func Init(cfg Config) (*Runtime, error) {
	policy := ResolveWithHostMode(cfg.Policy, cfg.HostMode)
	if err := ValidatePolicy(policy); err != nil {
		return nil, err
	}
	file := cfg.File
	if cfg.CleanupFiles {
		file.Cleanup = true
	}
	writer, err := ResolvePrimaryWriter(policy, file, cfg.Stdout, cfg.Stderr)
	if err != nil {
		return nil, err
	}
	logger := NewSlogLogger(writer, WriterOptions{
		Format: policy.Format,
		Level:  policy.Level,
	})
	if cfg.SetDefault {
		slog.SetDefault(logger)
	}
	return &Runtime{
		Logger:        logger,
		Policy:        policy,
		PrimaryWriter: writer,
	}, nil
}

func InitDefault(level, format, output string, file FileOptions, hostMode bool) (*Runtime, error) {
	return Init(Config{
		Policy: Policy{
			Mode:   ModeStandalone,
			Sinks:  []SinkType{SinkType(strings.ToLower(strings.TrimSpace(output)))},
			Format: format,
			Level:  level,
			Retry:  DefaultPolicy().Retry,
		},
		HostMode:     hostMode,
		File:         file,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		SetDefault:   true,
		CleanupFiles: file.Cleanup,
	})
}
