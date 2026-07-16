package client

import (
	"context"
	"fmt"
	"strings"
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
