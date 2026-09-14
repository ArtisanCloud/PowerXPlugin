package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
)

type PowerXSTSTokenProvider struct {
	Client *PowerXServiceClient
}

func NewPowerXSTSTokenProvider(client *PowerXServiceClient) *PowerXSTSTokenProvider {
	return &PowerXSTSTokenProvider{Client: client}
}

func (p *PowerXSTSTokenProvider) Token(ctx context.Context) (string, error) {
	if p == nil || p.Client == nil {
		return "", fmt.Errorf("powerx STS client is not configured")
	}
	token := strings.TrimSpace(p.Client.GetToken())
	if token != "" && token != "sts" {
		return token, nil
	}
	token, _, err := p.Client.ExchangeSTS(ctx)
	return strings.TrimSpace(token), err
}

func (p *PowerXSTSTokenProvider) InvalidateToken() {
	if p == nil || p.Client == nil {
		return
	}
	p.Client.InvalidateSTS()
}

func (p *PowerXSTSTokenProvider) TokenFunc() func(context.Context) (string, error) {
	return p.Token
}

// Credential binds the backend STS provider to its bootstrap tenant, never a
// tenant supplied in an HTTP request. Core validates the signed token itself.
func (p *PowerXSTSTokenProvider) Credential(ctx context.Context) (hostapi.Credential, error) {
	token, err := p.Token(ctx)
	if err != nil {
		return hostapi.Credential{}, err
	}
	tenant := p.Client.GetTenantUUID()
	if tenant == "" {
		return hostapi.Credential{}, fmt.Errorf("STS_TENANT_UNAVAILABLE")
	}
	return hostapi.Credential{Token: token, TenantUUID: tenant}, nil
}
