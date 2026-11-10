package host_contract

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamdelegated "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

type pluginAuthorizationProbe struct {
	input fwiamcontracts.AuthorizationRequest
	calls int
	err   error
}

func (s *pluginAuthorizationProbe) Authorize(_ context.Context, q fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	s.input = q
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return &fwiamcontracts.AuthorizationDecision{PluginID: q.PluginID, PermissionCode: q.PermissionCode, Allowed: false, ReasonCode: fwiamerrors.CodePermissionDenied, TenantUUID: q.TenantUUID, MemberUUID: q.MemberUUID, UserUUID: q.UserUUID, Resource: "conversation", Action: "reply", Mode: "delegated", TraceID: "core-trace"}, nil
}

func TestProbeForwardsExplicitPluginPermissionAndScopedDecision(t *testing.T) {
	s := &pluginAuthorizationProbe{}
	h := NewHandler(&app.Deps{IAMDirectoryService: &directoryStub{}, IAMAuthzService: s})
	input := map[string]any{"plugin_id": "com.example.messaging", "permission_code": "messages.conversation:reply", "member_uuid": probeMemberUUID, "user_uuid": "33333333-3333-4333-8333-333333333333"}
	probe := map[string]any{"module": "iam", "operation": "authorization.check", "input": input}
	r := executeProbe(t, h, probe)
	if r.Code != http.StatusOK || s.input.PluginID != input["plugin_id"] || s.input.PermissionCode != input["permission_code"] || s.input.TenantUUID != probeTenantUUID || s.input.Resource != "" {
		t.Fatal("PLUGIN_PERMISSION_PROBE_INPUT_LOST", r.Code, r.Body.String())
	}
	var envelope struct {
		Data hostcontract.ProbeResult `json:"data"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	decision := envelope.Data.Result["decision"].(map[string]any)
	if decision["plugin_id"] != input["plugin_id"] || decision["permission_code"] != input["permission_code"] || decision["allowed"] != false {
		t.Fatal("PLUGIN_PERMISSION_PROBE_OUTPUT_LOST", decision)
	}
	s.err = &fwiamdelegated.AuthorizationRequestError{IdentityRequestError: fwiamdelegated.IdentityRequestError{Status: 404, Code: fwiamerrors.CodePluginPermissionNotRegistered, ReasonCode: fwiamerrors.CodePluginPermissionNotRegistered, TraceID: "upstream-trace", RequestID: "upstream-request"}}
	r = executeProbe(t, h, probe)
	var failure struct {
		Error hostcontract.ErrorEnvelope `json:"error"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if r.Code != 404 || failure.Error.HTTPStatus != 404 || failure.Error.ErrorCode != fwiamerrors.CodePluginPermissionNotRegistered || failure.Error.RequestID != "upstream-request" || failure.Error.TraceID != "upstream-trace" {
		t.Fatal("PLUGIN_PERMISSION_PROBE_ERROR_LOST", r.Code, failure.Error)
	}
	delete(input, "permission_code")
	before := s.calls
	r = executeProbe(t, h, probe)
	if r.Code != 400 || s.calls != before {
		t.Fatal("INCOMPLETE_PLUGIN_PERMISSION_ACCEPTED", r.Code, s.calls)
	}
}
