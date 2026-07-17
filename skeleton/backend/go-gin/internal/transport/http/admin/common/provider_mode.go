package common

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/gin-gonic/gin"
)

const (
	ProviderModeLocal     = "local"
	ProviderModeDelegated = "delegated"
)

type ProviderDiagnostics struct {
	Mode               string `json:"mode"`
	DelegatedAvailable bool   `json:"delegated_available"`
	LocalAvailable     bool   `json:"local_available"`
	Provider           string `json:"provider"`
	ReadOnly           bool   `json:"read_only,omitempty"`
	Source             string `json:"source,omitempty"`
}

func NewProviderDiagnostics(mode string, delegatedAvailable, localAvailable bool) ProviderDiagnostics {
	mode = NormalizeProviderMode(mode)
	provider := ProviderModeLocal
	if mode == ProviderModeDelegated {
		provider = ProviderModeDelegated
	}
	return ProviderDiagnostics{
		Mode:               mode,
		DelegatedAvailable: delegatedAvailable,
		LocalAvailable:     localAvailable,
		Provider:           provider,
	}
}

func NormalizeProviderMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ProviderModeDelegated:
		return ProviderModeDelegated
	default:
		return ProviderModeLocal
	}
}

func ProviderUnavailable(c *gin.Context, code, message string, diagnostics ProviderDiagnostics) {
	if strings.TrimSpace(code) == "" {
		code = "PROVIDER_NOT_CONFIGURED"
	}
	if strings.TrimSpace(message) == "" {
		message = "provider is not configured"
	}
	contracts.ResponseErrorWithDetails(c, http.StatusServiceUnavailable, code, message, diagnostics)
}
