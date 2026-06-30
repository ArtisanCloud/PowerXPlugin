package customerfw

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDelegatedCustomerAuthClientRegisterLoginValidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/customer/auth/register", "/customer/auth/login", "/customer/auth/wechat/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"access_token": "access",
					"context": map[string]any{
						"tenant_uuid":   "tenant-a",
						"customer_uuid": "customer-a",
						"source":        "delegate",
						"authenticated": true,
					},
				},
			})
		case "/customer/auth/validate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"context": map[string]any{
						"tenant_uuid":   "tenant-a",
						"customer_uuid": "customer-a",
						"source":        "third_party",
						"authenticated": true,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewDelegatedCustomerAuthClient(DelegatedClientConfig{BaseURL: server.URL})
	reg, err := client.Register(t.Context(), RegisterInput{Identifier: "a@example.com"})
	if err != nil || reg.Context.CustomerUUID != "customer-a" {
		t.Fatalf("register result=%#v err=%v", reg, err)
	}
	login, err := client.Login(t.Context(), LoginInput{Identifier: "a@example.com"})
	if err != nil || login.AccessToken != "access" {
		t.Fatalf("login result=%#v err=%v", login, err)
	}
	wxLogin, err := client.Login(t.Context(), LoginInput{Channel: CustomerAuthChannelWeChatMiniApp, Code: "wx-code"})
	if err != nil || wxLogin.AccessToken != "access" {
		t.Fatalf("wechat login result=%#v err=%v", wxLogin, err)
	}
	cc, err := client.Validate(t.Context(), "token")
	if err != nil || cc.Source != CustomerAuthSourceThirdParty {
		t.Fatalf("validate context=%#v err=%v", cc, err)
	}
}

func TestDelegatedCustomerAuthClientUnavailable(t *testing.T) {
	client := NewDelegatedCustomerAuthClient(DelegatedClientConfig{})
	_, err := client.Validate(t.Context(), "token")
	if CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("expected delegate unavailable, got %v", err)
	}
}
