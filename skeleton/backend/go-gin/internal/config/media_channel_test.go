package config

import "testing"

func TestMediaChannelIsExplicitAndFailsClosed(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("POWERX_PROVIDER_MODE", "local")
	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("PX_GATEWAY_BASE_URL", "https://core.invalid")
	t.Setenv("PX_GATEWAY_AUTH_SCHEME", "apikey")
	t.Setenv("PX_GATEWAY_API_KEY", "controlled")
	t.Setenv("PX_MEDIA_CHANNEL", "")
	cfg, err := Load()
	if err != nil || cfg.MediaChannel() != "" {
		t.Fatal("API_KEY_SILENTLY_CHANGED_MEDIA_CHANNEL")
	}
	t.Setenv("PX_MEDIA_CHANNEL", "core_host")
	cfg, err = Load()
	if err != nil || cfg.MediaChannel() != "core_host" {
		t.Fatal("CORE_HOST_MEDIA_NOT_SELECTED")
	}
	t.Setenv("PX_MEDIA_CHANNEL", "unknown")
	if _, err := Load(); err == nil {
		t.Fatal("UNKNOWN_MEDIA_CHANNEL_ACCEPTED")
	}
	t.Setenv("PX_MEDIA_CHANNEL", "core_host")
	t.Setenv("PX_GATEWAY_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("CORE_HOST_MEDIA_WITHOUT_CREDENTIAL_ACCEPTED")
	}
}
