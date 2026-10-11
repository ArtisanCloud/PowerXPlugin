package contracts

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

const (
	IdentityContextCapability = "com.corex.iam.identity_context.service_read"
	IdentityContextEndpoint   = "core://iam/identity-context"
	IdentityContextHeader     = "X-PowerX-Identity-Context"
	IdentityQuerySchema       = "powerx.iam.plugin-request-identity-query/v1"
	IdentityResultSchema      = "powerx.iam.plugin-request-identity/v1"
	WSIdentityQuerySchema     = "powerx.iam.plugin-ws-identity-query/v1"
	WSIdentityResultSchema    = "powerx.iam.plugin-ws-identity/v1"
)

// DelegatedIdentityQuery accepts only the opaque proof issued by Core.
type DelegatedIdentityQuery struct {
	ContextTicket string `json:"context_ticket"`
}

func (DelegatedIdentityQuery) String() string   { return "DelegatedIdentityQuery{redacted}" }
func (DelegatedIdentityQuery) GoString() string { return "DelegatedIdentityQuery{redacted}" }

var ticketPattern = regexp.MustCompile(`^pxctx_v1_[A-Za-z0-9_-]{43}$`)

func (q DelegatedIdentityQuery) Validate() error {
	if q.ContextTicket == "" {
		return iamerrors.New(iamerrors.CodeProofRequired, iamerrors.CodeProofRequired)
	}
	if !ticketPattern.MatchString(q.ContextTicket) {
		return iamerrors.New(iamerrors.CodeProofInvalid, iamerrors.CodeProofInvalid)
	}
	return nil
}

type DelegatedIdentityService interface {
	ResolveDelegatedIdentity(context.Context, DelegatedIdentityQuery) (*DelegatedIdentity, error)
}
type WSIdentityRevalidationService interface {
	RevalidateWSIdentity(context.Context, WSIdentityQuery) (*WSIdentity, error)
}
type WSIdentityQuery struct {
	Handle string `json:"-"`
}

func (WSIdentityQuery) String() string   { return "WSIdentityQuery{redacted}" }
func (WSIdentityQuery) GoString() string { return "WSIdentityQuery{redacted}" }

var wsHandlePattern = regexp.MustCompile(`^pxwsi_v1_[A-Za-z0-9_-]{43}$`)

func (q WSIdentityQuery) Validate() error {
	if q.Handle == "" {
		return iamerrors.New("IAM_CONTEXT_WS_HANDLE_REQUIRED", "IAM_CONTEXT_WS_HANDLE_REQUIRED")
	}
	if !wsHandlePattern.MatchString(q.Handle) {
		return iamerrors.New("IAM_CONTEXT_WS_HANDLE_INVALID", "IAM_CONTEXT_WS_HANDLE_INVALID")
	}
	return nil
}

type WSIdentity struct{ DelegatedIdentity }

func (v *WSIdentity) Validate(pluginID string, now time.Time) error {
	if v == nil {
		return iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	return v.DelegatedIdentity.validateSchema(pluginID, now, WSIdentityResultSchema)
}

// DelegatedIdentity is the complete Core v1 result. Null membership is valid
// only for a verified Root; a valid selected tenant is always required.
type DelegatedIdentity struct {
	// Server-only fields are deliberately excluded from JSON and fmt output.
	WSRevalidationHandle     string     `json:"-"`
	WSRevalidationExpiresAt  *time.Time `json:"-"`
	Schema                   string     `json:"schema"`
	Mode                     string     `json:"mode"`
	PluginID                 string     `json:"plugin_id"`
	TenantUUID               string     `json:"tenant_uuid"`
	UserUUID                 string     `json:"user_uuid"`
	MemberUUID               *string    `json:"member_uuid"`
	IsRoot                   bool       `json:"is_root"`
	Roles                    []string   `json:"roles"`
	Permissions              []string   `json:"permissions"`
	PolicyVersion            string     `json:"policy_version"`
	PermsHash                string     `json:"perms_hash"`
	SourceMethod             string     `json:"source_method"`
	SourcePath               string     `json:"source_path"`
	SourceRequiredPermission string     `json:"source_required_permission"`
	Transport                string     `json:"transport"`
	SourceRequestID          string     `json:"source_request_id"`
	SourceTraceID            string     `json:"source_trace_id"`
	TraceID                  string     `json:"trace_id"`
	ProofExpiresAt           time.Time  `json:"proof_expires_at"`
	UserSessionExpiresAt     time.Time  `json:"user_session_expires_at"`
}

func (v *DelegatedIdentity) Validate(pluginID string, now time.Time) error {
	return v.validateSchema(pluginID, now, IdentityResultSchema)
}
func (v DelegatedIdentity) String() string   { return "DelegatedIdentity{redacted}" }
func (v DelegatedIdentity) GoString() string { return "DelegatedIdentity{redacted}" }
func (v *DelegatedIdentity) validateSchema(pluginID string, now time.Time, schema string) error {
	bad := func() error {
		return iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	if v == nil || pluginID == "" || v.PluginID != pluginID || v.Schema != schema || v.Mode != "delegated" {
		return bad()
	}
	for _, s := range []string{v.TenantUUID, v.UserUUID} {
		if !canonicalUUID(s) {
			return bad()
		}
	}
	if v.MemberUUID == nil {
		if !v.IsRoot {
			return bad()
		}
	} else if !canonicalUUID(*v.MemberUUID) {
		return bad()
	}
	if v.Roles == nil || v.Permissions == nil || v.PolicyVersion == "" || v.PermsHash == "" || v.SourceMethod == "" || !strings.HasPrefix(v.SourcePath, "/") || v.SourceRequestID == "" || v.SourceTraceID == "" || v.TraceID == "" {
		return bad()
	}
	for _, values := range [][]string{v.Roles, v.Permissions} {
		for _, s := range values {
			if strings.TrimSpace(s) == "" {
				return bad()
			}
		}
	}
	switch v.Transport {
	case "http", "ws_grant", "ws_upgrade":
	default:
		return bad()
	}
	_, proofOffset := v.ProofExpiresAt.Zone()
	_, userOffset := v.UserSessionExpiresAt.Zone()
	if proofOffset != 0 || userOffset != 0 || v.ProofExpiresAt.IsZero() || v.UserSessionExpiresAt.IsZero() || v.ProofExpiresAt.After(v.UserSessionExpiresAt) {
		return bad()
	}
	if !now.Before(v.UserSessionExpiresAt) {
		return iamerrors.New(iamerrors.CodeUserSessionExpired, iamerrors.CodeUserSessionExpired)
	}
	if v.Transport == "http" {
		if schema == WSIdentityResultSchema || v.WSRevalidationHandle != "" || v.WSRevalidationExpiresAt != nil {
			return bad()
		}
	} else {
		if !wsHandlePattern.MatchString(v.WSRevalidationHandle) || v.WSRevalidationExpiresAt == nil || !v.WSRevalidationExpiresAt.Equal(v.UserSessionExpiresAt) {
			return bad()
		}
		if !now.Before(*v.WSRevalidationExpiresAt) {
			return iamerrors.New("IAM_CONTEXT_WS_HANDLE_EXPIRED", "IAM_CONTEXT_WS_HANDLE_EXPIRED")
		}
		if schema == WSIdentityResultSchema && !v.ProofExpiresAt.Equal(*v.WSRevalidationExpiresAt) {
			return bad()
		}
	}
	if !now.Before(v.ProofExpiresAt) {
		return iamerrors.New(iamerrors.CodeProofExpired, iamerrors.CodeProofExpired)
	}
	return nil
}

// DecodeDelegatedIdentity rejects omitted required fields, including false Root
// and explicit nullable member. It never accepts the superseded expires_at DTO.
func DecodeDelegatedIdentity(raw []byte) (*DelegatedIdentity, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	required := []string{"schema", "mode", "plugin_id", "tenant_uuid", "user_uuid", "member_uuid", "is_root", "roles", "permissions", "policy_version", "perms_hash", "source_method", "source_path", "source_required_permission", "transport", "source_request_id", "source_trace_id", "trace_id", "proof_expires_at", "user_session_expires_at"}
	for _, key := range required {
		value, ok := fields[key]
		if !ok || (string(value) == "null" && key != "member_uuid") {
			return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
		}
	}
	var out DelegatedIdentity
	if json.Unmarshal(raw, &out) != nil {
		return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	if rawHandle, ok := fields["ws_revalidation_handle"]; ok {
		if json.Unmarshal(rawHandle, &out.WSRevalidationHandle) != nil || string(rawHandle) == "null" {
			return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
		}
	}
	if rawExpiry, ok := fields["ws_revalidation_expires_at"]; ok {
		var expiry time.Time
		if json.Unmarshal(rawExpiry, &expiry) != nil || string(rawExpiry) == "null" {
			return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
		}
		out.WSRevalidationExpiresAt = &expiry
	}
	return &out, nil
}
func DecodeWSIdentity(raw []byte) (*WSIdentity, error) {
	v, err := DecodeDelegatedIdentity(raw)
	if err != nil {
		return nil, err
	}
	return &WSIdentity{DelegatedIdentity: *v}, nil
}

func canonicalUUID(s string) bool {
	v, err := uuid.Parse(s)
	return err == nil && v != uuid.Nil && v.String() == s
}

// HasPermission uses the Core business permission format. Root bypass is only
// based on an already validated authoritative identity.
func (v *DelegatedIdentity) HasPermission(required string) bool {
	if v == nil || required == "" {
		return false
	}
	if v.IsRoot {
		return true
	}
	at := strings.LastIndex(required, ":")
	if at <= 0 {
		return false
	}
	for _, p := range v.Permissions {
		if p == required || p == "*" || p == "*:*" || p == required[:at]+":*" || p == "*"+required[at:] {
			return true
		}
	}
	return false
}
