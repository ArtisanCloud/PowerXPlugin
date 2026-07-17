package logging

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

type PolicyMode string

const (
	ModeHost       PolicyMode = "host"
	ModeStandalone PolicyMode = "standalone"
)

type SinkType string

const (
	SinkStdout SinkType = "stdout"
	SinkStderr SinkType = "stderr"
	SinkFile   SinkType = "file"
	SinkLoki   SinkType = "loki"
)

type RetryPolicy struct {
	Enabled     bool
	MaxAttempts int
	BackoffMS   int
}

type Policy struct {
	PolicyVersion        string
	Mode                 PolicyMode
	Sinks                []SinkType
	Format               string
	Level                string
	AuthorizedExtraSinks []SinkType
	Retry                RetryPolicy
}

func DefaultPolicy() Policy {
	return Policy{
		Mode:   ModeStandalone,
		Sinks:  []SinkType{SinkStdout},
		Format: "json",
		Level:  "info",
		Retry: RetryPolicy{
			Enabled:     true,
			MaxAttempts: 3,
			BackoffMS:   200,
		},
	}
}

func IsHostProxyMode() bool {
	return strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1"
}

func ResolvePolicy(input Policy) Policy {
	resolved := input
	defaults := DefaultPolicy()

	if resolved.Mode == "" {
		resolved.Mode = defaults.Mode
	}
	if len(resolved.Sinks) == 0 {
		resolved.Sinks = append([]SinkType{}, defaults.Sinks...)
	}
	if strings.TrimSpace(resolved.Format) == "" {
		resolved.Format = defaults.Format
	}
	if strings.TrimSpace(resolved.Level) == "" {
		resolved.Level = defaults.Level
	}
	if resolved.Retry.MaxAttempts <= 0 {
		resolved.Retry = defaults.Retry
	}

	resolved.Format = strings.ToLower(strings.TrimSpace(resolved.Format))
	resolved.Level = strings.ToLower(strings.TrimSpace(resolved.Level))

	if resolved.Mode == ModeHost {
		// Host mode always keeps stdout for PowerX collection.
		if !slices.Contains(resolved.Sinks, SinkStdout) {
			resolved.Sinks = append([]SinkType{SinkStdout}, resolved.Sinks...)
		}
		resolved.Format = "json"
	}

	seen := make(map[SinkType]struct{}, len(resolved.Sinks))
	dedup := make([]SinkType, 0, len(resolved.Sinks))
	for _, sink := range resolved.Sinks {
		normalized := SinkType(strings.ToLower(strings.TrimSpace(string(sink))))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		dedup = append(dedup, normalized)
	}
	resolved.Sinks = dedup

	return resolved
}

func ValidatePolicy(p Policy) error {
	switch p.Mode {
	case ModeHost, ModeStandalone:
	default:
		return fmt.Errorf("invalid policy mode: %s", p.Mode)
	}
	if len(p.Sinks) == 0 {
		return fmt.Errorf("at least one sink is required")
	}

	validSinks := map[SinkType]struct{}{
		SinkStdout: {}, SinkStderr: {}, SinkFile: {}, SinkLoki: {},
	}
	for _, sink := range p.Sinks {
		if _, ok := validSinks[sink]; !ok {
			return fmt.Errorf("invalid sink: %s", sink)
		}
	}

	switch p.Format {
	case "json", "text":
	default:
		return fmt.Errorf("invalid format: %s", p.Format)
	}

	switch p.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid level: %s", p.Level)
	}

	if p.Mode == ModeHost {
		if !slices.Contains(p.Sinks, SinkStdout) {
			return fmt.Errorf("host mode requires stdout sink")
		}
		if p.Format != "json" {
			return fmt.Errorf("host mode requires json format")
		}
		for _, sink := range p.Sinks {
			if sink == SinkStdout {
				continue
			}
			if !slices.Contains(p.AuthorizedExtraSinks, sink) {
				return fmt.Errorf("sink %s is not authorized in host mode", sink)
			}
		}
	}

	if p.Retry.MaxAttempts <= 0 || p.Retry.MaxAttempts > 10 {
		return fmt.Errorf("invalid retry max_attempts: %d", p.Retry.MaxAttempts)
	}
	if p.Retry.BackoffMS <= 0 || p.Retry.BackoffMS > 60000 {
		return fmt.Errorf("invalid retry backoff_ms: %d", p.Retry.BackoffMS)
	}

	return nil
}

func PrimaryOutput(p Policy) string {
	if len(p.Sinks) == 0 {
		return string(SinkStdout)
	}
	first := p.Sinks[0]
	if first == SinkLoki {
		// Legacy skeleton logger supports stdout/stderr/file only.
		return string(SinkStdout)
	}
	return string(first)
}
