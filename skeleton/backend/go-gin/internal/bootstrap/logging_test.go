package bootstrap

import (
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredLoggingIndependentOfProviderAndProxy(t *testing.T) {
	for _, mode := range []string{"local", "delegated"} {
		for _, proxy := range []string{"0", "1"} {
			t.Run(mode+"/"+proxy, func(t *testing.T) {
				t.Setenv("POWERX_PROXY", proxy)
				t.Setenv("POWERX_PROVIDER_MODE", mode)
				cfg := &config.LoggingConfig{Output: "file", Format: "json", Level: "debug", FilePath: filepath.Join(t.TempDir(), "runtime.log"), MaxSize: 1, MaxAge: 1, MaxBackups: 1}
				p, err := configuredLoggingPolicy(cfg)
				if err != nil || runtimelogging.PrimaryOutput(p) != "file" || len(p.Sinks) != 1 || p.Level != "debug" {
					t.Fatal("CONFIGURED_LOGGING_OUTPUT_OVERRIDDEN", err, p)
				}
				if err = initializeConfiguredLogging(cfg); err != nil {
					t.Fatal(err)
				}
				logger.Info("FIXTURE_BOOTSTRAP_LOG")
				data, err := os.ReadFile(cfg.FilePath)
				if err != nil || !strings.Contains(string(data), "FIXTURE_BOOTSTRAP_LOG") {
					t.Fatal("CONFIGURED_LOG_FILE_MISSING", err)
				}
			})
		}
	}
}

func TestConfiguredLoggingRejectsInvalidPolicyAndFile(t *testing.T) {
	if _, err := configuredLoggingPolicy(nil); err == nil {
		t.Fatal("MISSING_LOG_CONFIG_ACCEPTED")
	}
	if _, err := configuredLoggingPolicy(&config.LoggingConfig{Output: "invalid", Format: "json", Level: "info"}); err == nil {
		t.Fatal("INVALID_LOG_POLICY_ACCEPTED")
	}
	if err := initializeConfiguredLogging(&config.LoggingConfig{Output: "file", Format: "json", Level: "info"}); err == nil {
		t.Fatal("INVALID_LOG_FILE_FELL_BACK_TO_CONSOLE")
	}
}
