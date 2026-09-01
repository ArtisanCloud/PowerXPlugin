package agent

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
)

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type StaticBearerTokenProvider struct {
	TokenValue string
}

func (p StaticBearerTokenProvider) Token(context.Context) (string, error) {
	if p.TokenValue == "" {
		return "", newError(ErrCodeAuthInvalid, "bearer token is required")
	}
	return p.TokenValue, nil
}

type STSTokenProvider struct {
	Client *sts.Client
}

func (p STSTokenProvider) Token(ctx context.Context) (string, error) {
	if p.Client == nil {
		return "", newError(ErrCodeAuthInvalid, "sts client is required")
	}
	tok, err := p.Client.Token(ctx)
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}
