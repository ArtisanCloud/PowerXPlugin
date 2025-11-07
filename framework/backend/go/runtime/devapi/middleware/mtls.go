package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

// MTLSConfig holds mTLS configuration
type MTLSConfig struct {
	// Client CA file path for verifying client certificates
	ClientCAFile string
	// Whether to require client certificates
	RequireClientCert bool
}

// NewMTLSMiddleware creates a middleware that verifies client certificates
func NewMTLSMiddleware(config MTLSConfig) bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			// Get the underlying HTTP request
			req := ctx.HTTPRequest()
			if req == nil {
				ctx.JSON(http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
				return
			}

			// Get TLS connection state from request
			tlsConn := req.TLS
			if tlsConn == nil {
				ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error": "TLS connection required",
				})
				return
			}

			// Check if client certificate is present
			if tlsConn.PeerCertificates == nil || len(tlsConn.PeerCertificates) == 0 {
				if config.RequireClientCert {
					ctx.JSON(http.StatusUnauthorized, map[string]string{
						"error": "client certificate required",
					})
					return
				}
				// Allow requests without client cert if not required
				next(ctx)
				return
			}

			// Load and verify client CA
			if config.ClientCAFile != "" {
				clientCA, err := os.ReadFile(config.ClientCAFile)
				if err != nil {
					slog.Error("failed to read client CA file", slog.String("path", config.ClientCAFile), slog.Any("error", err))
					ctx.JSON(http.StatusInternalServerError, map[string]string{
						"error": "internal server error",
					})
					return
				}

				clientCAPool := x509.NewCertPool()
				if !clientCAPool.AppendCertsFromPEM(clientCA) {
					ctx.JSON(http.StatusInternalServerError, map[string]string{
						"error": "failed to parse client CA",
					})
					return
				}

				// Verify the client certificate against the CA
				clientCert := tlsConn.PeerCertificates[0]
				opts := x509.VerifyOptions{
					Roots: clientCAPool,
				}

				if _, err := clientCert.Verify(opts); err != nil {
					slog.Warn("client certificate verification failed",
						slog.String("subject", clientCert.Subject.String()),
						slog.Any("error", err),
					)
					ctx.JSON(http.StatusUnauthorized, map[string]string{
						"error": "client certificate verification failed",
					})
					return
				}

				slog.Debug("mTLS verification successful",
					slog.String("subject", clientCert.Subject.String()),
					slog.String("issuer", clientCert.Issuer.String()),
				)
			}

			// Certificate verified, proceed to next handler
			next(ctx)
		}
	}
}
