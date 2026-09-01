package authproxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDelegatedClientDirectoryMemberReadsCoreEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/iam/members/11111111-1111-1111-1111-111111111111" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer service-token" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"member_uuid":"11111111-1111-1111-1111-111111111111","tenant_uuid":"22222222-2222-2222-2222-222222222222","user_uuid":"33333333-3333-3333-3333-333333333333","display_name":"Example","status":1}}`))
	}))
	defer server.Close()

	client, err := NewDelegatedClient(server.URL, "service-token")
	if err != nil {
		t.Fatalf("NewDelegatedClient() error = %v", err)
	}
	member, err := client.GetDirectoryMember(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("GetDirectoryMember() error = %v", err)
	}
	if member.MemberUUID != "11111111-1111-1111-1111-111111111111" || member.DisplayName != "Example" {
		t.Fatalf("member = %#v", member)
	}
}

func TestDelegatedClientBatchResolveDirectoryMembersReadsMissingUUIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/iam/members:batch-resolve" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var payload map[string][]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if got := payload["member_uuids"]; len(got) != 2 || got[0] != "member-a" || got[1] != "member-b" {
			t.Fatalf("member_uuids = %#v", got)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"member_uuid":"member-a","tenant_uuid":"tenant-a","user_uuid":"user-a","display_name":"Alpha","status":1}],"missing_member_uuids":["member-b"]}}`))
	}))
	defer server.Close()

	client, err := NewDelegatedClient(server.URL, "service-token")
	if err != nil {
		t.Fatalf("NewDelegatedClient() error = %v", err)
	}
	result, err := client.BatchResolveDirectoryMembers(context.Background(), []string{"member-a", "member-b"})
	if err != nil {
		t.Fatalf("BatchResolveDirectoryMembers() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].DisplayName != "Alpha" || len(result.MissingMemberUUIDs) != 1 || result.MissingMemberUUIDs[0] != "member-b" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDelegatedClientPhase2DirectoryAndAuthorizationContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer service-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/tenant/iam/departments":
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"department_uuid":"department-a","tenant_uuid":"tenant-a","name":"Operations","code":"ops","parent_department_uuid":"department-root"}]}}`))
		case "/api/v1/tenant/iam/roles":
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"role_uuid":"role-a","tenant_uuid":"tenant-a","code":"operator","name":"Operator"}]}}`))
		case "/api/v1/tenant/iam/permissions":
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"permission_uuid":"permission-a","resource":"iam.member","action":"read","scope":"tenant"}]}}`))
		case "/api/v1/tenant/iam/authorization:check":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if payload["member_uuid"] != "member-a" || payload["resource"] != "iam.member" || payload["action"] != "read" {
				t.Fatalf("payload = %#v", payload)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"allowed":false,"reason_code":"IAM_PERMISSION_DENIED"}}`))
		default:
			t.Fatalf("unexpected path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewDelegatedClient(server.URL, "service-token")
	if err != nil {
		t.Fatalf("NewDelegatedClient() error = %v", err)
	}
	departments, err := client.ListDirectoryDepartments(context.Background())
	if err != nil || len(departments) != 1 || departments[0].DepartmentUUID != "department-a" {
		t.Fatalf("ListDirectoryDepartments() = %#v, %v", departments, err)
	}
	roles, err := client.ListDirectoryRoles(context.Background())
	if err != nil || len(roles) != 1 || roles[0].RoleUUID != "role-a" {
		t.Fatalf("ListDirectoryRoles() = %#v, %v", roles, err)
	}
	permissions, err := client.ListDirectoryPermissions(context.Background())
	if err != nil || len(permissions) != 1 || permissions[0].PermissionUUID != "permission-a" {
		t.Fatalf("ListDirectoryPermissions() = %#v, %v", permissions, err)
	}
	decision, err := client.CheckDirectoryAuthorization(context.Background(), DirectoryAuthorizationRequest{MemberUUID: "member-a", Resource: "iam.member", Action: "read"})
	if err != nil || decision == nil || decision.Allowed || decision.ReasonCode != "IAM_PERMISSION_DENIED" {
		t.Fatalf("CheckDirectoryAuthorization() = %#v, %v", decision, err)
	}
}

func TestDelegatedClientPreservesCoreIAMErrorCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":403,"message":"forbidden","error_code":"IAM_FORBIDDEN","reason_code":"IAM_FORBIDDEN"}`))
	}))
	defer server.Close()

	client, err := NewDelegatedClient(server.URL, "service-token")
	if err != nil {
		t.Fatalf("NewDelegatedClient() error = %v", err)
	}
	_, err = client.GetDirectoryMember(context.Background(), "11111111-1111-1111-1111-111111111111")
	var proxyErr *ProxyError
	if !errors.As(err, &proxyErr) {
		t.Fatalf("error = %v, want ProxyError", err)
	}
	if proxyErr.Status != http.StatusForbidden || proxyErr.ReasonCode != "IAM_FORBIDDEN" {
		t.Fatalf("proxy error = %#v", proxyErr)
	}
}
