package customerfw

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CustomerAuthSource string

const (
	CustomerAuthSourcePlatform   CustomerAuthSource = "platform"
	CustomerAuthSourceDelegate   CustomerAuthSource = "delegate"
	CustomerAuthSourceThirdParty CustomerAuthSource = "third_party"
	CustomerAuthSourceLocalDev   CustomerAuthSource = "local_dev"
	CustomerAuthSourceMock       CustomerAuthSource = "mock"
	CustomerAuthSourceLocal      CustomerAuthSource = "local"
	CustomerAuthSourceCore       CustomerAuthSource = "core"
	CustomerAuthSourceWeChat     CustomerAuthSource = "wechat"
)

type CustomerContext struct {
	TenantUUID     string             `json:"tenant_uuid,omitempty"`
	CustomerUUID   string             `json:"customer_uuid"`
	MembershipUUID string             `json:"membership_uuid,omitempty"`
	Profile        CustomerAttributes `json:"profile,omitempty"`
	Roles          []string           `json:"roles,omitempty"`
	Scopes         []string           `json:"scopes,omitempty"`
	Source         CustomerAuthSource `json:"source"`
	Authenticated  bool               `json:"authenticated"`
	TokenExpiresAt *time.Time         `json:"token_expires_at,omitempty"`
	Attributes     map[string]any     `json:"attributes,omitempty"`
	RawClaims      map[string]any     `json:"raw_claims,omitempty"`
}

type CustomerAttributes struct {
	DisplayName string `json:"display_name,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	GivenName   string `json:"given_name,omitempty"`
	FamilyName  string `json:"family_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

type customerContextKey struct{}

const GinCustomerContextKey = "customer_ctx"

var ErrCustomerContextMissing = errors.New("customer context missing")

func WithContext(ctx context.Context, cc *CustomerContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, customerContextKey{}, NormalizeContext(cc))
}

func ContextFrom(ctx context.Context) (*CustomerContext, bool) {
	if ctx == nil {
		return nil, false
	}
	cc, ok := ctx.Value(customerContextKey{}).(*CustomerContext)
	return cc, ok && cc != nil
}

func MustContextFrom(ctx context.Context) *CustomerContext {
	cc, ok := ContextFrom(ctx)
	if !ok {
		panic(ErrCustomerContextMissing)
	}
	return cc
}

func SetGinContext(c *gin.Context, cc *CustomerContext) {
	if c == nil || cc == nil {
		return
	}
	cc = NormalizeContext(cc)
	c.Set(GinCustomerContextKey, cc)
	if c.Request != nil {
		c.Request = c.Request.WithContext(WithContext(c.Request.Context(), cc))
	}
}

func ContextFromGin(c *gin.Context) (*CustomerContext, bool) {
	if c == nil {
		return nil, false
	}
	if v, ok := c.Get(GinCustomerContextKey); ok && v != nil {
		if cc, ok := v.(*CustomerContext); ok && cc != nil {
			return cc, true
		}
	}
	if c.Request != nil {
		return ContextFrom(c.Request.Context())
	}
	return nil, false
}

func MustContextFromGin(c *gin.Context) *CustomerContext {
	cc, ok := ContextFromGin(c)
	if !ok {
		panic(ErrCustomerContextMissing)
	}
	return cc
}

func NormalizeContext(cc *CustomerContext) *CustomerContext {
	if cc == nil {
		return nil
	}
	copy := *cc
	copy.TenantUUID = normalizeID(copy.TenantUUID)
	copy.CustomerUUID = normalizeID(copy.CustomerUUID)
	copy.MembershipUUID = normalizeID(copy.MembershipUUID)
	copy.Profile = NormalizeAttributes(copy.Profile)
	copy.Source = NormalizeSource(copy.Source)
	copy.Roles = compactStrings(copy.Roles)
	copy.Scopes = compactStrings(copy.Scopes)
	return &copy
}

func NormalizeAttributes(profile CustomerAttributes) CustomerAttributes {
	return CustomerAttributes{
		DisplayName: strings.TrimSpace(profile.DisplayName),
		Nickname:    strings.TrimSpace(profile.Nickname),
		GivenName:   strings.TrimSpace(profile.GivenName),
		FamilyName:  strings.TrimSpace(profile.FamilyName),
		AvatarURL:   strings.TrimSpace(profile.AvatarURL),
		Locale:      strings.TrimSpace(profile.Locale),
		Timezone:    strings.TrimSpace(profile.Timezone),
	}
}

func NormalizeSource(source CustomerAuthSource) CustomerAuthSource {
	value := CustomerAuthSource(strings.ToLower(strings.TrimSpace(string(source))))
	switch value {
	case CustomerAuthSourcePlatform, CustomerAuthSourceDelegate, CustomerAuthSourceThirdParty,
		CustomerAuthSourceLocalDev, CustomerAuthSourceMock, CustomerAuthSourceLocal,
		CustomerAuthSourceCore, CustomerAuthSourceWeChat:
		return value
	default:
		if value == "" {
			return CustomerAuthSourceDelegate
		}
		return value
	}
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
