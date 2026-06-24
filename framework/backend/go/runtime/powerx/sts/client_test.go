package sts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClientRequiresConfig(t *testing.T) {
	_, err := NewClient(Config{})
	require.ErrorIs(t, err, ErrMissingConfig)
}

func TestClientCachesToken(t *testing.T) {
	client, err := NewClient(Config{TokenEndpoint: "https://powerx.example/sts", ClientID: "client", ClientSecret: "secret"})
	require.NoError(t, err)
	first, err := client.Token(context.Background())
	require.NoError(t, err)
	second, err := client.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, first.AccessToken, second.AccessToken)
}
