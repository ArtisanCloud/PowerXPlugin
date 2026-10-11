package delegated

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

//go:embed testdata/core_tenant.json
var coreTenantResponseFixture []byte

// Keep the complete Core response independent of Framework contract models.
func writeCoreTenantTestResponse(t *testing.T, w http.ResponseWriter, tenant string, status any) {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(coreTenantResponseFixture, &response); err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data := response["data"].(map[string]any)
	data["tenant_uuid"] = tenant
	data["status"] = status
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Error(err)
	}
}

func TestCoreTenantCompleteNumericResponse(t *testing.T) {
	var fixture struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(coreTenantResponseFixture, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		for _, status := range []int16{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/%d", scheme, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tenant/iam/tenant" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != scheme+" service-credential" {
						t.Error("IAM_INVALID_ARGUMENT")
					}
					writeCoreTenantTestResponse(t, w, displayNameTenant, status)
				}))
				defer server.Close()
				cfg := CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "service-credential", nil })}
				var client *CoreClient
				var err error
				if scheme == "ApiKey" {
					client, err = NewCoreClientWithAPIKey(cfg, "service-credential")
				} else {
					client, err = NewCoreClient(cfg)
				}
				if err != nil {
					t.Fatal(err)
				}
				result, err := client.GetTenant(context.Background(), displayNameTenant)
				if err != nil {
					t.Fatal(err)
				}
				if result == nil || result.TenantUUID != displayNameTenant || result.TenantKey != fixture.Data["key"] || result.Name != fixture.Data["name"] || result.Status != strconv.Itoa(int(status)) {
					t.Fatalf("tenant=%#v", result)
				}
			})
		}
	}
}

func TestCoreTenantRejectsInvalidStatusAndCrossTenant(t *testing.T) {
	for _, tc := range []struct {
		name, asserted string
		status         any
	}{
		{"string-status", displayNameTenant, "1"},
		{"fractional-status", displayNameTenant, 1.5},
		{"status-overflow", displayNameTenant, 32768},
		{"cross-tenant", displayNameOtherTenant, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeCoreTenantTestResponse(t, w, displayNameTenant, tc.status)
			}))
			defer server.Close()
			client, err := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL}, "service-credential")
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.GetTenant(context.Background(), tc.asserted)
			if result != nil || iamerrors.CodeOf(err) != iamerrors.CodeUpstreamDependency {
				t.Fatalf("tenant=%#v code=%s", result, iamerrors.CodeOf(err))
			}
		})
	}
}
