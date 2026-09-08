package customerhttp

import (
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	customerobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// Authenticate enforces customer authentication for /mini-app routes.
func Authenticate(deps *app.Deps) gin.HandlerFunc {
	if deps != nil && deps.CustomerRuntime != nil {
		auth, err := deps.CustomerRuntime.Auth()
		if err == nil {
			audit := customerobs.NewAuditLogger(nil)
			return authmw.CustomerAuthWithValidator(customerfw.NewAuthClientValidator(auth), audit)
		}
		return unavailable(err)
	}
	factory := customersvc.NewAuthenticatorFactory(nil, nil)
	if deps != nil && deps.Config != nil {
		factory = customersvc.NewAuthenticatorFactory(deps.Config, nil)
	}
	authenticator := factory.Build()
	audit := customerobs.NewAuditLogger(nil)
	return authmw.CustomerAuth(authenticator, audit)
}

// RequireMembership obtains the adapter selected by CustomerRuntime. Absence
// is an explicit 503; delegated mode must never read plugin-local tables.
func RequireMembership(deps *app.Deps) gin.HandlerFunc {
	if deps != nil && deps.CustomerRuntime != nil {
		resolver, err := deps.CustomerRuntime.Membership()
		if err == nil {
			return authmw.CustomerMembership(resolver)
		}
		return unavailable(err)
	}
	return unavailable(nil)
}

func unavailable(_ error) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatusJSON(503, gin.H{"error": gin.H{"code": "CUSTOMER_DELEGATE_UNAVAILABLE"}})
	}
}
