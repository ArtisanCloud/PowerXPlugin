package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthConfig struct {
	Issuer             string   `yaml:"issuer" json:"issuer"`
	AcceptAudiences    []string `yaml:"accept_audiences" json:"accept_audiences"`
	HMACSecret         string   `yaml:"hmac_secret" json:"hmac_secret"`
	ClockSkewSeconds   int      `yaml:"clock_skew_seconds" json:"clock_skew_seconds"`
	Optional           bool     `yaml:"optional" json:"optional"`
	AllowSignedContext bool     `yaml:"allow_signed_context" json:"allow_signed_context"`
	ContextHMACSecret  string   `yaml:"context_hmac_secret" json:"context_hmac_secret"`
	MaxCtxAgeSeconds   int64    `yaml:"max_ctx_age_seconds" json:"max_ctx_age_seconds"`
}

type PowerXClaims struct {
	TenantUUID      TenantClaim `json:"tid"`
	TenantID        int64       `json:"tid_n,omitempty"`
	UserUUID        string      `json:"uid,omitempty"`
	UserID          int64       `json:"uid_n,omitempty"`
	MemberUUID      string      `json:"mid,omitempty"`
	MemberID        int64       `json:"mid_n,omitempty"`
	MemberIDAlias   int64       `json:"member_id,omitempty"`
	IsRoot          bool        `json:"is_root"`
	Roles           []string    `json:"roles"`
	Permissions     []string    `json:"perms"`
	PermissionCodes []string    `json:"permission_codes"`
	PolicyVersion   string      `json:"policy_version"`
	PermsHash       string      `json:"perms_hash"`
	AuthzSource     string      `json:"source"`
	PluginID        string      `json:"plugin_id,omitempty"`
	Scope           string      `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

type TenantClaim string

func (t *TenantClaim) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = TenantClaim(strings.TrimSpace(s))
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*t = TenantClaim(strings.TrimSpace(num.String()))
	return nil
}

func (t TenantClaim) String() string {
	return string(t)
}

func (c *PowerXClaims) UnmarshalJSON(data []byte) error {
	var raw struct {
		TenantUUID      TenantClaim `json:"tid"`
		TenantID        any         `json:"tid_n"`
		UserID          any         `json:"uid"`
		UserIDNumeric   any         `json:"uid_n"`
		MemberID        any         `json:"mid"`
		MemberIDNum     any         `json:"mid_n"`
		MemberIDAlias   any         `json:"member_id"`
		IsRoot          bool        `json:"is_root"`
		Roles           []string    `json:"roles"`
		Permissions     []string    `json:"perms"`
		PermissionCodes []string    `json:"permission_codes"`
		PolicyVersion   string      `json:"policy_version"`
		PermsHash       string      `json:"perms_hash"`
		AuthzSource     string      `json:"source"`
		AuthzSourceAlt  string      `json:"authz_source"`
		PluginID        string      `json:"plugin_id,omitempty"`
		Scope           string      `json:"scope,omitempty"`
		jwt.RegisteredClaims
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	tenantID, err := parseNumericClaim(raw.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseNumericClaim(raw.UserIDNumeric)
	if err != nil {
		return err
	}
	if userID == 0 {
		userID, err = parseNumericClaim(raw.UserID)
		if err != nil {
			return err
		}
	}
	memberID, err := parseNumericClaim(raw.MemberIDNum)
	if err != nil {
		return err
	}
	if memberID == 0 {
		memberID, err = parseNumericClaim(raw.MemberIDAlias)
		if err != nil {
			return err
		}
	}
	c.TenantUUID = raw.TenantUUID
	c.TenantID = tenantID
	c.UserUUID = stringClaim(raw.UserID)
	c.UserID = userID
	c.MemberUUID = stringClaim(raw.MemberID)
	c.MemberID = memberID
	c.MemberIDAlias = memberID
	c.IsRoot = raw.IsRoot
	c.Roles = raw.Roles
	c.PermissionCodes = raw.PermissionCodes
	c.Permissions = raw.PermissionCodes
	if len(c.Permissions) == 0 {
		c.Permissions = raw.Permissions
	}
	c.PolicyVersion = raw.PolicyVersion
	c.PermsHash = raw.PermsHash
	c.AuthzSource = firstNonEmpty(raw.AuthzSource, raw.AuthzSourceAlt)
	c.PluginID = raw.PluginID
	c.Scope = raw.Scope
	c.RegisteredClaims = raw.RegisteredClaims
	return nil
}

func parseNumericClaim(value any) (int64, error) {
	switch v := value.(type) {
	case nil:
		return 0, nil
	case float64:
		return int64(v), nil
	case json.Number:
		return strconv.ParseInt(strings.TrimSpace(v.String()), 10, 64)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return 0, nil
		}
		return parsed, nil
	default:
		return 0, errors.New("invalid numeric claim type")
	}
}

func stringClaim(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		if v == 0 {
			return ""
		}
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}

func ParseFromHeaders(h func(string) string, cfg JWTAuthConfig) (tc TenantContext, rawBearer string, ok bool) {
	// 1) Authorization: Bearer
	authz := h("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		raw := strings.TrimSpace(authz[7:])
		if raw != "" && cfg.HMACSecret != "" {
			if t, err := parseHS256(raw, cfg); err == nil {
				return t, raw, true
			}
		}
	}
	// 2) 回退 Signed-Context
	if cfg.AllowSignedContext && cfg.ContextHMACSecret != "" {
		if t, ok := tryLoadSignedContext(h, cfg.ContextHMACSecret, cfg.MaxCtxAgeSeconds); ok {
			return t, "", true
		}
	}
	return TenantContext{}, "", false
}

func parseHS256(raw string, cfg JWTAuthConfig) (TenantContext, error) {
	leeway := time.Duration(cfg.ClockSkewSeconds)
	if leeway <= 0 {
		leeway = 60
	}
	claims := &PowerXClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected sign method")
		}
		return []byte(cfg.HMACSecret), nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.AcceptAudiences...), jwt.WithLeeway(leeway*time.Second))
	if err != nil || token == nil || !token.Valid {
		return TenantContext{}, errors.New("invalid token")
	}
	authzSource := strings.TrimSpace(claims.AuthzSource)
	if authzSource == "" && hasDelegatedAuthzSnapshot(claims.PermissionCodes, claims.PolicyVersion, claims.PermsHash) {
		authzSource = "signed_claims"
	}
	return TenantContext{
		TenantUUID:    strings.TrimSpace(claims.TenantUUID.String()),
		TenantID:      claims.TenantID,
		UserID:        claims.UserID,
		MemberID:      claims.MemberID,
		IsRoot:        claims.IsRoot,
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		PolicyVersion: claims.PolicyVersion,
		PermsHash:     claims.PermsHash,
		AuthzSource:   authzSource,
		PluginID:      strings.TrimSpace(claims.PluginID),
	}, nil
}

type signedCtx struct {
	TenantUUID     string   `json:"tid"`
	TenantID       int64    `json:"tid_n,omitempty"`
	UserID         int64    `json:"uid"`
	MemberID       int64    `json:"mid,omitempty"`
	IsRoot         bool     `json:"is_root"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permission_codes"`
	PolicyVersion  string   `json:"policy_version"`
	PermsHash      string   `json:"perms_hash"`
	AuthzSource    string   `json:"source"`
	AuthzSourceAlt string   `json:"authz_source"`
	PluginID       string   `json:"plugin_id,omitempty"`
	TS             int64    `json:"ts"`
}

func tryLoadSignedContext(h func(string) string, secret string, maxAgeSec int64) (TenantContext, bool) {
	ctxB64 := h("X-PowerX-CTX")
	if ctxB4 := ctxB64; ctxB4 == "" {
		return TenantContext{}, false
	}
	sigHex := h("X-PowerX-CTX-SIG")
	if sigHex == "" {
		return TenantContext{}, false
	}
	raw, err := base64.StdEncoding.DecodeString(ctxB64)
	if err != nil {
		return TenantContext{}, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ctxB64))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(sigHex)) {
		return TenantContext{}, false
	}
	var sc signedCtx
	if err := json.Unmarshal(raw, &sc); err != nil {
		return TenantContext{}, false
	}
	if maxAgeSec > 0 && (time.Now().Unix()-sc.TS) > maxAgeSec {
		return TenantContext{}, false
	}
	authzSource := firstNonEmpty(sc.AuthzSource, sc.AuthzSourceAlt)
	if authzSource == "" && hasDelegatedAuthzSnapshot(sc.Permissions, sc.PolicyVersion, sc.PermsHash) {
		authzSource = "signed_context"
	}
	return TenantContext{TenantUUID: strings.TrimSpace(sc.TenantUUID), TenantID: sc.TenantID, UserID: sc.UserID, MemberID: sc.MemberID, IsRoot: sc.IsRoot, Roles: sc.Roles,
		Permissions: sc.Permissions, PolicyVersion: sc.PolicyVersion, PermsHash: sc.PermsHash, AuthzSource: authzSource, PluginID: strings.TrimSpace(sc.PluginID)}, true
}

// 供客户端出站兜底：把 TenantContext 签成 X-PowerX-CTX / SIG
func SignContext(tc TenantContext, secret string) (ctxB64, sigHex string, ts int64, err error) {
	sc := signedCtx{TenantUUID: strings.TrimSpace(tc.TenantUUID), TenantID: tc.TenantID, UserID: tc.UserID, MemberID: tc.MemberID, IsRoot: tc.IsRoot, Roles: tc.Roles,
		Permissions: tc.Permissions, PolicyVersion: tc.PolicyVersion, PermsHash: tc.PermsHash, AuthzSource: tc.AuthzSource, PluginID: tc.PluginID, TS: time.Now().Unix()}
	b, e := json.Marshal(&sc)
	if e != nil {
		return "", "", 0, e
	}
	ctxB64 = base64.StdEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ctxB64))
	sigHex = hex.EncodeToString(mac.Sum(nil))
	return ctxB64, sigHex, sc.TS, nil
}

func hasDelegatedAuthzSnapshot(permissionCodes []string, policyVersion string, permsHash string) bool {
	if strings.TrimSpace(policyVersion) == "" || strings.TrimSpace(permsHash) == "" {
		return false
	}
	for _, code := range permissionCodes {
		if strings.TrimSpace(code) != "" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
