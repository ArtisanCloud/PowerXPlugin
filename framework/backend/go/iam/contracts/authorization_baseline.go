package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

const (
	AuthorizationBaselineCapability     = "com.corex.iam.authorization_baseline.service_read"
	AuthorizationBaselineEndpoint       = "core://iam/authorization-baseline"
	AuthorizationBaselineOperation      = "read_member_baselines"
	AuthorizationBaselineQuerySchema    = "powerx.iam.authorization-baseline-query/v1"
	AuthorizationBaselineResultSchema   = "powerx.iam.authorization-baseline-result/v1"
	AuthorizationBaselineTimeLayout     = "2006-01-02T15:04:05.000000Z"
	AuthorizationBaselineMaxMembers     = 100
	AuthorizationBaselineMaxPermissions = 10
)

// AuthorizationBaselineQuery carries a server-owned tenant assertion. Core
// derives its scope from the service credential; TenantUUID is never sent.
type AuthorizationBaselineQuery struct {
	TenantUUID      string   `json:"-"`
	MemberUUIDs     []string `json:"member_uuids"`
	PermissionCodes []string `json:"permission_codes"`
}

type AuthorizationBaselineService interface {
	ReadMemberBaselines(context.Context, AuthorizationBaselineQuery) (*AuthorizationBaselineResult, error)
}

// BaselineTime preserves the exact UTC microsecond wire contract. Other time
// formats, numeric timestamps and precision loss are contract errors.
type BaselineTime struct{ time.Time }

func (v BaselineTime) MarshalJSON() ([]byte, error) {
	if v.IsZero() || v.Nanosecond()%1000 != 0 || v.Year() < 1 || v.Year() > 9999 {
		return nil, baselineInvalid()
	}
	return json.Marshal(v.UTC().Format(AuthorizationBaselineTimeLayout))
}

func (v *BaselineTime) UnmarshalJSON(raw []byte) error {
	var value string
	if json.Unmarshal(raw, &value) != nil || len(value) != 27 {
		return baselineInvalid()
	}
	parsed, err := time.Parse(AuthorizationBaselineTimeLayout, value)
	if err != nil || parsed.IsZero() || parsed.Format(AuthorizationBaselineTimeLayout) != value {
		return baselineInvalid()
	}
	v.Time = parsed.UTC()
	return nil
}

type AuthorizationBaselineResult struct {
	Schema      string                          `json:"schema"`
	TenantUUID  string                          `json:"tenant_uuid"`
	PluginID    string                          `json:"plugin_id"`
	EvaluatedAt BaselineTime                    `json:"evaluated_at"`
	Items       []AuthorizationBaselineDecision `json:"items"`
}

// Allowed is deliberately nullable: an item error is never a business denial.
// Consumers must inspect Status before using Allowed or any baseline fields.
type AuthorizationBaselineDecision struct {
	MemberUUID            string        `json:"member_uuid"`
	PermissionCode        string        `json:"permission_code"`
	Status                string        `json:"status"`
	ErrorCode             string        `json:"error_code,omitempty"`
	Allowed               *bool         `json:"allowed,omitempty"`
	BaselineKind          string        `json:"baseline_kind,omitempty"`
	GrantedAt             *BaselineTime `json:"granted_at"`
	KnownSince            *BaselineTime `json:"known_since,omitempty"`
	VisibleSince          *BaselineTime `json:"visible_since"`
	AuthorizationRevision string        `json:"authorization_revision,omitempty"`
	DataScope             string        `json:"data_scope,omitempty"`
	ScopeFingerprint      string        `json:"scope_fingerprint,omitempty"`
	ValidUntil            *BaselineTime `json:"valid_until"`
	RecheckAt             *BaselineTime `json:"recheck_at"`
}

func (v AuthorizationBaselineDecision) MarshalJSON() ([]byte, error) {
	if v.Status == "error" {
		return json.Marshal(struct {
			MemberUUID     string `json:"member_uuid"`
			PermissionCode string `json:"permission_code"`
			Status         string `json:"status"`
			ErrorCode      string `json:"error_code"`
		}{v.MemberUUID, v.PermissionCode, v.Status, v.ErrorCode})
	}
	type decision AuthorizationBaselineDecision
	return json.Marshal(decision(v))
}

var baselinePermissionPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+:[a-zA-Z0-9_.-]+$`)
var baselineScopePattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func (q AuthorizationBaselineQuery) Normalize() (AuthorizationBaselineQuery, error) {
	fail := func() (AuthorizationBaselineQuery, error) {
		return AuthorizationBaselineQuery{}, iamerrors.New(iamerrors.CodeBaselineInvalidArgument, iamerrors.CodeBaselineInvalidArgument)
	}
	tenant, err := uuid.Parse(strings.TrimSpace(q.TenantUUID))
	if err != nil || tenant == uuid.Nil || len(q.MemberUUIDs) < 1 || len(q.MemberUUIDs) > AuthorizationBaselineMaxMembers || len(q.PermissionCodes) < 1 || len(q.PermissionCodes) > AuthorizationBaselineMaxPermissions {
		return fail()
	}
	out := AuthorizationBaselineQuery{TenantUUID: tenant.String(), MemberUUIDs: make([]string, 0, len(q.MemberUUIDs)), PermissionCodes: append([]string(nil), q.PermissionCodes...)}
	seen := map[string]bool{}
	for _, raw := range q.MemberUUIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil || id == uuid.Nil || seen[id.String()] {
			return fail()
		}
		seen[id.String()] = true
		out.MemberUUIDs = append(out.MemberUUIDs, id.String())
	}
	seen = map[string]bool{}
	for _, code := range out.PermissionCodes {
		if !baselinePermissionPattern.MatchString(code) || seen[code] {
			return fail()
		}
		seen[code] = true
	}
	return out, nil
}

func baselineInvalid() error {
	return iamerrors.New(iamerrors.CodeBaselineResponseInvalid, iamerrors.CodeBaselineResponseInvalid)
}

// baselineObject rejects unknown, missing and duplicate properties, including
// null/missing branch discriminators which Go bool zero values would hide.
func baselineObject(raw []byte, required []string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, baselineInvalid()
	}
	allowed := map[string]bool{}
	for _, name := range required {
		allowed[name] = true
	}
	out := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		name, ok := token.(string)
		if err != nil || !ok || !allowed[name] || out[name] != nil {
			return nil, baselineInvalid()
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil, baselineInvalid()
		}
		out[name] = value
	}
	if _, err = d.Token(); err != nil || d.Decode(new(any)) != io.EOF || len(out) != len(required) {
		return nil, baselineInvalid()
	}
	return out, nil
}

func DecodeAuthorizationBaselineResult(raw []byte, query AuthorizationBaselineQuery, pluginID string) (*AuthorizationBaselineResult, error) {
	if _, err := baselineObject(raw, []string{"schema", "tenant_uuid", "plugin_id", "evaluated_at", "items"}); err != nil {
		return nil, err
	}
	var wire struct {
		Schema      string            `json:"schema"`
		TenantUUID  string            `json:"tenant_uuid"`
		PluginID    string            `json:"plugin_id"`
		EvaluatedAt BaselineTime      `json:"evaluated_at"`
		Items       []json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &wire) != nil {
		return nil, baselineInvalid()
	}
	out := &AuthorizationBaselineResult{Schema: wire.Schema, TenantUUID: wire.TenantUUID, PluginID: wire.PluginID, EvaluatedAt: wire.EvaluatedAt}
	for _, rawItem := range wire.Items {
		var item AuthorizationBaselineDecision
		if json.Unmarshal(rawItem, &item) != nil {
			return nil, baselineInvalid()
		}
		fields := []string{"member_uuid", "permission_code", "status", "error_code"}
		if item.Status == "decided" {
			fields = []string{"member_uuid", "permission_code", "status", "allowed", "granted_at", "visible_since", "authorization_revision", "data_scope", "scope_fingerprint", "valid_until", "recheck_at"}
			if item.Allowed != nil && *item.Allowed {
				fields = append(fields, "baseline_kind", "known_since")
			}
		}
		if _, err := baselineObject(rawItem, fields); err != nil {
			return nil, err
		}
		out.Items = append(out.Items, item)
	}
	if err := out.Validate(query, pluginID); err != nil {
		return nil, err
	}
	return out, nil
}

// Validate is also the contract boundary for injected local providers.
func (r *AuthorizationBaselineResult) Validate(query AuthorizationBaselineQuery, pluginID string) error {
	q, err := query.Normalize()
	if err != nil {
		return err
	}
	if r == nil || pluginID == "" || r.PluginID != pluginID || r.Schema != AuthorizationBaselineResultSchema || r.TenantUUID != q.TenantUUID || !validBaselineTime(&r.EvaluatedAt) || len(r.Items) != len(q.MemberUUIDs)*len(q.PermissionCodes) {
		return baselineInvalid()
	}
	expected := map[string]bool{}
	for _, member := range q.MemberUUIDs {
		for _, code := range q.PermissionCodes {
			expected[member+"/"+code] = true
		}
	}
	for _, item := range r.Items {
		key := item.MemberUUID + "/" + item.PermissionCode
		if !expected[key] {
			return baselineInvalid()
		}
		delete(expected, key)
		if item.Status == "error" {
			switch item.ErrorCode {
			case "IAM_BASELINE_MEMBER_UNAVAILABLE", "IAM_BASELINE_UNAVAILABLE", "IAM_BASELINE_UNSUPPORTED_SCOPE", "IAM_BASELINE_UNSUPPORTED_POLICY", "IAM_BASELINE_ROOT_BYPASS_UNSUPPORTED":
			default:
				return baselineInvalid()
			}
			if item.Allowed != nil || item.BaselineKind != "" || item.GrantedAt != nil || item.KnownSince != nil || item.VisibleSince != nil || item.AuthorizationRevision != "" || item.DataScope != "" || item.ScopeFingerprint != "" || item.ValidUntil != nil || item.RecheckAt != nil {
				return baselineInvalid()
			}
			continue
		}
		if item.Status != "decided" || item.Allowed == nil || item.ErrorCode != "" || strings.TrimSpace(item.AuthorizationRevision) == "" || item.DataScope != "TENANT" || !baselineScopePattern.MatchString(item.ScopeFingerprint) {
			return baselineInvalid()
		}
		for _, value := range []*BaselineTime{item.GrantedAt, item.KnownSince, item.VisibleSince, item.ValidUntil, item.RecheckAt} {
			if value != nil && !validBaselineTime(value) {
				return baselineInvalid()
			}
		}
		if item.RecheckAt != nil && !item.RecheckAt.After(r.EvaluatedAt.Time) {
			return baselineInvalid()
		}
		if !*item.Allowed {
			if item.BaselineKind != "" || item.GrantedAt != nil || item.KnownSince != nil || item.VisibleSince != nil || item.ValidUntil != nil {
				return baselineInvalid()
			}
			continue
		}
		if item.KnownSince == nil || item.VisibleSince == nil || item.VisibleSince.Before(item.KnownSince.Time) || item.VisibleSince.After(r.EvaluatedAt.Time) {
			return baselineInvalid()
		}
		switch item.BaselineKind {
		case "initialized":
			if item.GrantedAt != nil {
				return baselineInvalid()
			}
		case "native", "reconstructed":
			if item.GrantedAt == nil || item.GrantedAt.After(item.VisibleSince.Time) {
				return baselineInvalid()
			}
		default:
			return baselineInvalid()
		}
		if item.ValidUntil != nil && (!item.ValidUntil.After(r.EvaluatedAt.Time) || (item.RecheckAt != nil && item.RecheckAt.After(item.ValidUntil.Time))) {
			return baselineInvalid()
		}
	}
	return nil
}

func validBaselineTime(value *BaselineTime) bool {
	_, offset := value.Zone()
	return !value.IsZero() && value.Nanosecond()%1000 == 0 && offset == 0 && value.Year() >= 1 && value.Year() <= 9999
}
