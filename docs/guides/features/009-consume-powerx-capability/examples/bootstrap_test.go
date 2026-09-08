package guideexamples

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	corecap "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	corenotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Test fixture only: it does not implement tenant isolation or persistence.
type localProbe struct{ calls int }

var _ notifications.Publisher = (*localProbe)(nil)

func (p *localProbe) Create(context.Context, corenotifications.CreateInput) (*corenotifications.Notification, error) {
	p.calls++
	return &corenotifications.Notification{UUID: "11111111-1111-4111-8111-111111111111"}, nil
}

func TestLocalSelectionAndMissingAdapter(t *testing.T) {
	ctx := context.Background()
	local := &localProbe{}
	pub, err := BuildPublisher(ctx, provider.ModeLocal, local, "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pub.Create(ctx, corenotifications.CreateInput{}); err != nil {
		t.Fatal(err)
	}
	if local.calls != 1 {
		t.Fatalf("local_calls=%d", local.calls)
	}
	var missing *localProbe
	_, err = BuildPublisher(ctx, provider.ModeLocal, missing, "", nil, nil, nil)
	var coded *module.Error
	if !errors.As(err, &coded) || coded.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("error=%v", err)
	}
	_, err = BuildPublisher(ctx, provider.ModeDelegated, local, "", nil, nil, nil)
	if err == nil {
		t.Fatal("error=nil")
	}
}

func TestDelegatedPreflightAndNoFallback(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(map[bool]string{false: "granted", true: "not_granted"}[denied], func(t *testing.T) {
			ctx := context.Background()
			local := &localProbe{}
			createCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer fixture-sts" {
					t.Error("request_contract_mismatch")
				}
				if r.Header.Get("X-Tenant-UUID") != "" || r.URL.RawQuery != "" {
					t.Error("identity_override")
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/v1/tenant/capabilities:grant-status":
					var input corecap.GrantStatusInput
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
						return
					}
					items := make([]corecap.GrantStatusItem, 0, len(input.CapabilityIDs))
					for _, id := range input.CapabilityIDs {
						status, reason := "granted", "CAPABILITY_GRANTED"
						if denied && id == "com.corex.notifications.create" {
							status, reason = "not_granted", "CAPABILITY_NOT_GRANTED"
						}
						items = append(items, corecap.GrantStatusItem{CapabilityID: id, Status: status, ReasonCode: reason})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"items": items}})
				case "/api/v1/notifications":
					createCalls++
					if createCalls > 1 {
						w.WriteHeader(http.StatusForbidden)
						_ = json.NewEncoder(w).Encode(map[string]any{"reason_code": "NOTIFICATION_FORBIDDEN"})
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "22222222-2222-4222-8222-222222222222"}})
				default:
					t.Errorf("path=%s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			pub, err := BuildPublisher(ctx, provider.ModeDelegated, local, server.URL,
				func(context.Context) (string, error) { return "fixture-sts", nil },
				[]string{"com.corex.capabilities.grant_status.read", "com.corex.notifications.create"}, server.Client())
			if denied {
				var coded *module.Error
				if !errors.As(err, &coded) || coded.Code != "FRAMEWORK_REQUIRED_CAPABILITY_UNAVAILABLE" {
					t.Fatalf("error=%v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				// Machine keys stand in for the plugin's locale-rendered content.
				input := corenotifications.CreateInput{Title: "notification.title", Content: "notification.content"}
				if _, err := pub.Create(ctx, input); err != nil {
					t.Fatal(err)
				}
				_, err = pub.Create(ctx, input)
				var httpErr *corenotifications.HTTPError
				if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
					t.Fatalf("error=%v", err)
				}
			}
			if local.calls != 0 || (denied && createCalls != 0) {
				t.Fatalf("local_calls=%d create_calls=%d", local.calls, createCalls)
			}
		})
	}
}
