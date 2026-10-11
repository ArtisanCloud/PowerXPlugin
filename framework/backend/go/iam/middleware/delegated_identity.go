package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	iamcontext "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/gin-gonic/gin"
)

type DelegatedIdentityOptions struct {
	Resolver  contracts.DelegatedIdentityService
	PluginID  string
	APIPrefix string
	// Assertions must come from an authenticated inbound business credential.
	TenantAssertion func(*gin.Context) string
	UserAssertion   func(*gin.Context) string
	Commit          func(*gin.Context, *contracts.DelegatedIdentity)
}

// ResolveRequestIdentity is shared by HTTP and pre-upgrade WS handlers. It
// never consumes the inbound Bearer as a Core user identity credential.
func ResolveRequestIdentity(c *gin.Context, opts DelegatedIdentityOptions, transport string) (*contracts.DelegatedIdentity, error) {
	if opts.Resolver == nil || opts.PluginID == "" {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	q := contracts.DelegatedIdentityQuery{ContextTicket: c.GetHeader(contracts.IdentityContextHeader)}
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx := delegated.WithRequestMetadata(c.Request.Context(), delegated.RequestMetadata{RequestID: c.GetHeader("X-Request-ID"), TraceID: c.GetHeader("X-Trace-ID")})
	v, err := opts.Resolver.ResolveDelegatedIdentity(ctx, q)
	if err != nil {
		return nil, err
	}
	if err = v.Validate(opts.PluginID, time.Now()); err != nil {
		return nil, err
	}
	sourcePath := c.Request.URL.Path
	if opts.APIPrefix != "" {
		sourcePath = strings.TrimPrefix(sourcePath, strings.TrimRight(opts.APIPrefix, "/"))
	}
	if v.SourceMethod != c.Request.Method || v.SourcePath != sourcePath || v.Transport != transport {
		return nil, iamerrors.New(iamerrors.CodeContextRequestForbidden, iamerrors.CodeContextRequestForbidden)
	}
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" && requestID != v.SourceRequestID {
		return nil, iamerrors.New(iamerrors.CodeContextRequestForbidden, iamerrors.CodeContextRequestForbidden)
	}
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" && traceID != v.SourceTraceID {
		return nil, iamerrors.New(iamerrors.CodeContextRequestForbidden, iamerrors.CodeContextRequestForbidden)
	}
	if opts.TenantAssertion != nil && opts.TenantAssertion(c) != v.TenantUUID {
		return nil, iamerrors.New("IAM_CONTEXT_TENANT_FORBIDDEN", "IAM_CONTEXT_TENANT_FORBIDDEN")
	}
	if opts.UserAssertion != nil && opts.UserAssertion(c) != v.UserUUID {
		return nil, iamerrors.New("IAM_CONTEXT_USER_FORBIDDEN", "IAM_CONTEXT_USER_FORBIDDEN")
	}
	c.Request = c.Request.WithContext(iamcontext.WithDelegatedIdentity(c.Request.Context(), v))
	if opts.Commit != nil {
		opts.Commit(c, v)
	}
	return v, nil
}

func RefreshDelegatedIdentity(opts DelegatedIdentityOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		transport := "http"
		if strings.HasSuffix(c.Request.URL.Path, "/runtime/ws-bus/grant") {
			transport = "ws_grant"
		}
		if _, err := ResolveRequestIdentity(c, opts, transport); err != nil {
			RespondIdentityError(c, err)
			return
		}
		c.Next()
	}
}

// RequireIdentityPermission is the secondary business check after Core proof.
func RequireIdentityPermission(permission func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity, ok := iamcontext.DelegatedIdentityFromContext(c.Request.Context())
		if !ok {
			RespondIdentityError(c, iamerrors.New(iamerrors.CodeProofRequired, iamerrors.CodeProofRequired))
			return
		}
		if permission == nil || !identity.HasPermission(permission(c)) {
			RespondIdentityError(c, iamerrors.New(iamerrors.CodeContextRequestForbidden, iamerrors.CodeContextRequestForbidden))
			return
		}
		c.Next()
	}
}

func RespondIdentityError(c *gin.Context, err error) {
	code := iamerrors.CodeOf(err)
	if code == "" {
		code = iamerrors.CodeContextUnavailable
	}
	status := iamerrors.StatusCode(err)
	details := gin.H{}
	var upstream *delegated.IdentityRequestError
	if errors.As(err, &upstream) {
		details = gin.H{"code": upstream.Code, "reason_code": upstream.ReasonCode, "trace_id": upstream.TraceID, "request_id": upstream.RequestID, "source_trace_id": upstream.SourceTraceID, "source_request_id": upstream.SourceRequestID}
	}
	if status == 500 && upstream == nil {
		status = http.StatusServiceUnavailable
	}
	// The machine code is translated by the consuming UI; no proof/body is sent.
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": code, "details": details}, "trace_id": c.GetHeader("X-Trace-ID"), "request_id": c.GetHeader("X-Request-ID")})
}
