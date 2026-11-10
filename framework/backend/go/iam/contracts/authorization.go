package contracts

import (
	"encoding/json"
	"strings"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

// NormalizeForCore validates the published Core authorization query. TenantUUID
// is a caller assertion used to check the response; it never enters the wire DTO.
// The only resource/action-only Core query is the explicitly retained IAM read.
func (in AuthorizationRequest) NormalizeForCore() (AuthorizationRequest, error) {
	invalid := func() (AuthorizationRequest, error) {
		return AuthorizationRequest{}, iamerrors.New(iamerrors.CodeInvalidArgument, iamerrors.CodeInvalidArgument)
	}
	for _, field := range []*string{&in.TenantUUID, &in.MemberUUID, &in.UserUUID} {
		id, err := uuid.Parse(strings.TrimSpace(*field))
		if err != nil || id == uuid.Nil {
			return invalid()
		}
		*field = id.String()
	}
	in.PluginID, in.PermissionCode = strings.TrimSpace(in.PluginID), strings.TrimSpace(in.PermissionCode)
	in.Resource, in.Action = strings.TrimSpace(in.Resource), strings.TrimSpace(in.Action)
	in.TraceID = strings.TrimSpace(in.TraceID)
	if in.PolicyToken != "" || strings.ContainsAny(in.TraceID, "\r\n\x00") {
		return invalid()
	}
	if in.PluginID != "" || in.PermissionCode != "" {
		if in.PluginID == "" || in.PermissionCode == "" || strings.ContainsAny(in.PluginID, "/*\\:\r\n\x00") || strings.ContainsAny(in.PermissionCode, "\r\n\x00") {
			return invalid()
		}
	} else if in.Resource != "iam.member" || in.Action != "read" {
		return invalid()
	}
	return in, nil
}

// DecodeCoreAuthorizationDecision requires explicit allowed/denied data and
// matches every subject and permission to the query, including denied results.
func DecodeCoreAuthorizationDecision(raw []byte, query AuthorizationRequest) (*AuthorizationDecision, error) {
	invalid := func() (*AuthorizationDecision, error) {
		return nil, iamerrors.New(iamerrors.CodeAuthorizationResponseInvalid, iamerrors.CodeAuthorizationResponseInvalid)
	}
	var err error
	query, err = query.NormalizeForCore()
	if err != nil {
		return invalid()
	}
	var wire struct {
		Allowed        *bool   `json:"allowed"`
		ReasonCode     *string `json:"reason_code"`
		PluginID       string  `json:"plugin_id"`
		PermissionCode string  `json:"permission_code"`
		TenantUUID     string  `json:"tenant_uuid"`
		MemberUUID     string  `json:"member_uuid"`
		UserUUID       string  `json:"user_uuid"`
		Resource       string  `json:"resource"`
		Action         string  `json:"action"`
		TraceID        string  `json:"trace_id"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Allowed == nil || wire.ReasonCode == nil {
		return invalid()
	}
	if wire.PluginID != query.PluginID || wire.PermissionCode != query.PermissionCode || wire.Resource == "" || wire.Action == "" {
		return invalid()
	}
	for _, pair := range [][2]string{{wire.TenantUUID, query.TenantUUID}, {wire.MemberUUID, query.MemberUUID}, {wire.UserUUID, query.UserUUID}} {
		actual, err := uuid.Parse(pair[0])
		expected, expectedErr := uuid.Parse(pair[1])
		if err != nil || expectedErr != nil || actual == uuid.Nil || actual != expected {
			return invalid()
		}
	}
	if (query.Resource != "" && wire.Resource != query.Resource) || (query.Action != "" && wire.Action != query.Action) {
		return invalid()
	}
	if (*wire.Allowed && *wire.ReasonCode != "") || (!*wire.Allowed && *wire.ReasonCode != iamerrors.CodePermissionDenied) {
		return invalid()
	}
	return &AuthorizationDecision{Allowed: *wire.Allowed, ReasonCode: *wire.ReasonCode, PluginID: wire.PluginID, PermissionCode: wire.PermissionCode, TenantUUID: query.TenantUUID, MemberUUID: query.MemberUUID, UserUUID: query.UserUUID, Resource: wire.Resource, Action: wire.Action, TraceID: wire.TraceID, Mode: string(IAMAdapterModeDelegated)}, nil
}
