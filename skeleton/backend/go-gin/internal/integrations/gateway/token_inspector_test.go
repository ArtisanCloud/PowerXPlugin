package gateway

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTokenExpiryFromExp(t *testing.T) {
	expiry := time.Now().Add(2 * time.Hour).Unix()
	token := buildJWT(map[string]any{"exp": expiry})

	got, err := ParseTokenExpiry(token)
	require.NoError(t, err)
	require.WithinDuration(t, time.Unix(expiry, 0).UTC(), got, time.Second)
}

func TestParseTokenExpiryFromExpiresAt(t *testing.T) {
	expiry := time.Now().Add(3 * time.Hour).UTC()
	token := buildJWT(map[string]any{"expires_at": expiry.Format(time.RFC3339)})

	got, err := ParseTokenExpiry(token)
	require.NoError(t, err)
	require.WithinDuration(t, expiry, got, time.Second)
}

func TestParseTokenExpiryErrorOnInvalidToken(t *testing.T) {
	_, err := ParseTokenExpiry("invalid-token")
	require.Error(t, err)
}

func buildJWT(payload map[string]any) string {
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(payloadJSON) + ".signature"
}
