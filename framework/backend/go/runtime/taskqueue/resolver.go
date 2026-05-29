package taskqueue

import (
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
)

type ProviderMode string

const (
	ProviderModeLocal ProviderMode = "local"
	ProviderModeHost  ProviderMode = "host"
)

type ResolveInput struct {
	ConfigIAMMode contracts.IAMMode
	EnvMode       string
	PowerXProxy   string
}

type ResolveResult struct {
	Mode   ProviderMode
	Source string
}

func ResolveProviderMode(input ResolveInput) ResolveResult {
	if firstNonEmpty(input.PowerXProxy, os.Getenv("POWERX_PROXY")) == "1" {
		return ResolveResult{Mode: ProviderModeHost, Source: "env:POWERX_PROXY"}
	}
	envMode := strings.ToLower(strings.TrimSpace(firstNonEmpty(input.EnvMode, os.Getenv("IAMMode"), os.Getenv("IAM_MODE"))))
	if envMode == string(contracts.IAMModeLocal) {
		return ResolveResult{Mode: ProviderModeLocal, Source: "env:iam_mode"}
	}
	if envMode == string(contracts.IAMModeDelegated) {
		return ResolveResult{Mode: ProviderModeHost, Source: "env:iam_mode"}
	}
	if input.ConfigIAMMode == contracts.IAMModeLocal {
		return ResolveResult{Mode: ProviderModeLocal, Source: "config:iam_mode"}
	}
	if input.ConfigIAMMode == contracts.IAMModeDelegated {
		return ResolveResult{Mode: ProviderModeHost, Source: "config:iam_mode"}
	}
	return ResolveResult{Mode: ProviderModeLocal, Source: "iam_mode"}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
