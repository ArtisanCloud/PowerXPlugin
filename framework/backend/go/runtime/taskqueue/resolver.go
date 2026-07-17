package taskqueue

import (
	"os"
	"strings"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type LinkMode string

const (
	LinkModeLocal LinkMode = "local"
	LinkModeHost  LinkMode = "host"
)

type ResolveInput struct {
	ConfigProviderMode fwprovider.Mode
	EnvMode            string
	PowerXProxy        string
}

type ResolveResult struct {
	Mode   LinkMode
	Source string
}

func ResolveProviderMode(input ResolveInput) ResolveResult {
	if firstNonEmpty(input.PowerXProxy, os.Getenv("POWERX_PROXY")) == "1" {
		return ResolveResult{Mode: LinkModeHost, Source: "env:POWERX_PROXY"}
	}
	envMode := strings.ToLower(strings.TrimSpace(firstNonEmpty(input.EnvMode, os.Getenv("POWERX_PROVIDER_MODE"))))
	if envMode == string(fwprovider.ModeLocal) || envMode == string(fwprovider.ModeDelegated) {
		return ResolveResult{Mode: LinkModeLocal, Source: "env:provider_mode"}
	}
	if input.ConfigProviderMode == fwprovider.ModeLocal || input.ConfigProviderMode == fwprovider.ModeDelegated {
		return ResolveResult{Mode: LinkModeLocal, Source: "config:provider_mode"}
	}
	return ResolveResult{Mode: LinkModeLocal, Source: "default"}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
