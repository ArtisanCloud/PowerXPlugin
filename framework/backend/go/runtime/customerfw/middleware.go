package customerfw

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AuditHook func(fields AuditFields)

type ErrorWriter func(c *gin.Context, err error)

type RequestTenantResolver func(c *gin.Context) string

type TenantInjector func(c *gin.Context, tenantUUID string)

type authOptions struct {
	requireTenant         bool
	audit                 AuditHook
	errorWriter           ErrorWriter
	requestTenantResolver RequestTenantResolver
	tenantInjector        TenantInjector
	bootstrapResolver     MiniAppBootstrapClient
	bootstrapInput        func(*gin.Context) BootstrapInput
}

type AuthOption func(*authOptions)

func RequireTenant() AuthOption {
	return func(opts *authOptions) {
		opts.requireTenant = true
	}
}

func WithAuditHook(hook AuditHook) AuthOption {
	return func(opts *authOptions) {
		opts.audit = hook
	}
}

func WithErrorWriter(writer ErrorWriter) AuthOption {
	return func(opts *authOptions) {
		opts.errorWriter = writer
	}
}

func WithRequestTenantResolver(resolver RequestTenantResolver) AuthOption {
	return func(opts *authOptions) {
		opts.requestTenantResolver = resolver
	}
}

func WithTenantInjector(injector TenantInjector) AuthOption {
	return func(opts *authOptions) {
		opts.tenantInjector = injector
	}
}

func WithBootstrapResolver(resolver MiniAppBootstrapClient, input func(*gin.Context) BootstrapInput) AuthOption {
	return func(opts *authOptions) {
		opts.bootstrapResolver = resolver
		opts.bootstrapInput = input
	}
}

func Authenticate(validator CustomerTokenValidator, options ...AuthOption) gin.HandlerFunc {
	opts := authOptions{
		errorWriter:           defaultErrorWriter,
		requestTenantResolver: DefaultRequestTenantResolver,
		tenantInjector:        DefaultTenantInjector,
	}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}

	return func(c *gin.Context) {
		if c.Request != nil && c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		requestTenant := ""
		if opts.requestTenantResolver != nil {
			requestTenant = opts.requestTenantResolver(c)
		}
		bootstrap, err := resolveBootstrap(c, opts)
		if err != nil {
			emitAudit(opts.audit, requestTenant, "", "", false, 0, err)
			opts.errorWriter(c, err)
			c.Abort()
			return
		}
		credentials := ExtractTokenCredentials(c)
		if len(credentials) == 0 {
			err := NewError(CodeCustomerTokenMissing, "customer token missing")
			emitAudit(opts.audit, requestTenant, "", "", false, 0, err)
			opts.errorWriter(c, err)
			c.Abort()
			return
		}

		start := time.Now()
		cc, err := ValidateTokenCredentials(requestContext(c), validator, requestTenant, credentials)
		latency := time.Since(start)
		if err != nil {
			emitAudit(opts.audit, requestTenant, "", "", false, latency, err)
			opts.errorWriter(c, mapValidatorError(err))
			c.Abort()
			return
		}

		bootstrapTenant := ""
		if bootstrap != nil {
			bootstrapTenant = bootstrap.TenantUUID
		}
		resolvedTenant, err := ResolveTenant(requestTenant, cc.TenantUUID, bootstrapTenant, opts.requireTenant)
		if err != nil {
			emitAudit(opts.audit, requestTenant, cc.CustomerUUID, cc.Source, false, latency, err)
			opts.errorWriter(c, err)
			c.Abort()
			return
		}
		if resolvedTenant != "" {
			cc.TenantUUID = resolvedTenant
			if opts.tenantInjector != nil {
				opts.tenantInjector(c, resolvedTenant)
			}
		}

		SetGinContext(c, cc)
		emitAudit(opts.audit, resolvedTenant, cc.CustomerUUID, cc.Source, true, latency, nil)
		c.Next()
	}
}

func resolveBootstrap(c *gin.Context, opts authOptions) (*BootstrapContext, error) {
	if opts.bootstrapResolver == nil {
		return nil, nil
	}
	input := BootstrapInput{}
	if opts.bootstrapInput != nil {
		input = opts.bootstrapInput(c)
	}
	bootstrap, err := opts.bootstrapResolver.ResolveEntry(requestContext(c), input)
	if err != nil {
		return nil, WrapError(CodeCustomerBootstrapFailed, "customer bootstrap failed", err)
	}
	return NormalizeBootstrapContext(bootstrap), nil
}

func ExtractTokenCredentials(c *gin.Context) []tokenCredential {
	if c == nil {
		return nil
	}
	var credentials []tokenCredential
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw != "" && strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		token := strings.TrimSpace(raw[len("Bearer "):])
		if token != "" {
			credentials = append(credentials, tokenCredential{Name: "authorization", Token: token})
		}
	}
	if token := strings.TrimSpace(c.GetHeader("X-Customer-Token")); token != "" {
		credentials = append(credentials, tokenCredential{Name: "x-customer-token", Token: token})
	}
	return credentials
}

func DefaultRequestTenantResolver(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if c.Request != nil {
		if tenantUUID, ok := TenantUUIDFromContext(c.Request.Context()); ok {
			return tenantUUID
		}
	}
	if tenantUUID := strings.TrimSpace(c.GetHeader("tenant_uuid")); tenantUUID != "" {
		return normalizeID(tenantUUID)
	}
	if tenantUUID := strings.TrimSpace(c.Query("tenant_uuid")); tenantUUID != "" {
		return normalizeID(tenantUUID)
	}
	return ""
}

func DefaultTenantInjector(c *gin.Context, tenantUUID string) {
	if c == nil || strings.TrimSpace(tenantUUID) == "" {
		return
	}
	tenantUUID = normalizeID(tenantUUID)
	c.Set("tenant_uuid", tenantUUID)
	if c.Request != nil {
		c.Request = c.Request.WithContext(WithTenantUUID(c.Request.Context(), tenantUUID))
	}
}

func defaultErrorWriter(c *gin.Context, err error) {
	code := CodeOf(err)
	c.AbortWithStatusJSON(HTTPStatusForCode(code), gin.H{
		"error": gin.H{
			"code":    string(code),
			"message": err.Error(),
		},
	})
}

func mapValidatorError(err error) error {
	code := CodeOf(err)
	if code != CodeCustomerTokenInvalid {
		return err
	}
	return NewError(CodeCustomerTokenInvalid, "customer token invalid")
}

func emitAudit(hook AuditHook, tenantUUID, customerUUID string, source CustomerAuthSource, ok bool, latency time.Duration, err error) {
	if hook == nil {
		return
	}
	hook(ValidationAuditFields(tenantUUID, customerUUID, source, ok, latency, err))
}

func requestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil && c.Request.Context() != nil {
		return c.Request.Context()
	}
	return context.Background()
}
