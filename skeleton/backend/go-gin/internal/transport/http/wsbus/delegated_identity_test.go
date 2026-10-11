package wsbus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fwadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwdelegated "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	fwrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

func TestCanceledConnectionDoesNotDeliverQueuedMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sent := false
	ws := &wsConn{ctx: ctx, sendHook: func(wsResponse) { sent = true }}
	ws.sendMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ws.send(wsResponse{Type: "event", Topic: "_topic.identity.test"})
	}()
	cancel()
	ws.sendMu.Unlock()
	<-done
	if sent {
		t.Fatal("CANCELED_CONNECTION_DELIVERED")
	}
}

func TestDelegatedWebSocketUsesRealtimeRevalidationAndCloses(t *testing.T) {
	for _, direction := range []string{"inbound", "outbound"} {
		for _, rejected := range []string{"logout", "member-disabled", "service-grant", "topic-permission", "binding", "dependency"} {
			t.Run(direction+"/"+rejected, func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				const tenant = "11111111-1111-4111-8111-111111111111"
				const user = "22222222-2222-4222-8222-222222222222"
				const ticket = "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				const topic = "_topic.identity.test"
				var calls atomic.Int64
				var reject atomic.Bool
				now := time.Now().UTC()
				identity := fwcontracts.DelegatedIdentity{Schema: fwcontracts.IdentityResultSchema, Mode: "delegated", PluginID: app.PluginID, TenantUUID: tenant, UserUUID: user, IsRoot: true, Roles: []string{}, Permissions: []string{}, PolicyVersion: "p1", PermsHash: "hash", SourceMethod: "GET", SourcePath: "/ws", SourceRequiredPermission: "runtime.realtime:read", Transport: "ws_upgrade", SourceRequestID: "source", SourceTraceID: "source", TraceID: "resolve", ProofExpiresAt: now.Add(time.Minute), UserSessionExpiresAt: now.Add(5 * time.Minute)}
				member := "33333333-3333-4333-8333-333333333333"
				identity.IsRoot = false
				identity.MemberUUID = &member
				identity.Permissions = []string{"runtime.realtime:read", "topic:read"}
				core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "ApiKey backend-key" {
						t.Error("service credential mismatch")
					}
					var request struct {
						Payload struct {
							Body struct {
								Operation string            `json:"operation"`
								Request   map[string]string `json:"request"`
							} `json:"body"`
						} `json:"payload"`
					}
					_ = json.NewDecoder(r.Body).Decode(&request)
					output := identity
					if request.Payload.Body.Operation == "revalidate_ws_identity" {
						calls.Add(1)
						if request.Payload.Body.Request["ws_revalidation_handle"] != "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
							t.Error("handle missing")
						}
						output.Schema = fwcontracts.WSIdentityResultSchema
						output.ProofExpiresAt = identity.UserSessionExpiresAt
						if reject.Load() {
							code := ""
							status := 403
							switch rejected {
							case "logout":
								code = "IAM_CONTEXT_USER_SESSION_INVALID"
								status = 401
							case "member-disabled":
								code = "IAM_CONTEXT_USER_FORBIDDEN"
							case "service-grant":
								code = "IAM_CONTEXT_SERVICE_FORBIDDEN"
							case "dependency":
								code = "IAM_CONTEXT_UNAVAILABLE"
								status = 503
							case "binding":
								output.UserUUID = "44444444-4444-4444-8444-444444444444"
							case "topic-permission":
								output.IsRoot = false
								member := "33333333-3333-4333-8333-333333333333"
								output.MemberUUID = &member
								output.Permissions = []string{"runtime.realtime:read"}
							}
							if code != "" {
								w.Header().Set("X-Trace-ID", "revalidate-trace")
								w.WriteHeader(status)
								_ = json.NewEncoder(w).Encode(map[string]any{"code": status, "error_code": code})
								return
							}
						}
					}
					raw, _ := json.Marshal(output)
					var fields map[string]any
					_ = json.Unmarshal(raw, &fields)
					fields["ws_revalidation_handle"] = "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
					fields["ws_revalidation_expires_at"] = identity.UserSessionExpiresAt
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"result": map[string]any{"result": fields}}})
				}))
				defer core.Close()
				client, err := fwdelegated.NewCoreClientWithAPIKey(fwdelegated.CoreClientConfig{BaseURL: core.URL, PluginID: app.PluginID}, "backend-key")
				if err != nil {
					t.Fatal(err)
				}
				bundle, err := fwdelegated.NewBundle(client)
				if err != nil {
					t.Fatal(err)
				}
				registry := fwadapters.NewRegistry()
				if err = registry.Bind(fwcontracts.IAMAdapterModeDelegated, bundle); err != nil {
					t.Fatal(err)
				}
				lifecycle := fwrealtime.NewIdentityLifecycle()
				defer lifecycle.Close()
				hub := fwwsbus.NewMemoryHub()
				deps := &app.Deps{IAMAdapterMode: iamservice.IAMAdapterModeDelegated, IAMRegistry: registry, WSBusHub: hub, WSIdentityLifecycle: lifecycle, RealtimeDescriptors: []fwrealtime.Descriptor{{Key: topic, Protocols: []fwrealtime.Protocol{fwrealtime.ProtocolWS}, Actions: []fwrealtime.Action{fwrealtime.ActionSubscribe}, Scope: fwrealtime.ScopeTenant, EventTypes: []string{"message"}, Permission: "topic:read"}}}
				cfg := middleware.JWTAuthConfig{Issuer: "core-test", AcceptAudiences: []string{"plugin:" + app.PluginID}, HMACSecret: "test-business-signing-key"}
				token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.PowerXClaims{TenantUUID: middleware.TenantClaim(tenant), UserUUID: user, RegisteredClaims: jwt.RegisteredClaims{Issuer: cfg.Issuer, Audience: jwt.ClaimStrings(cfg.AcceptAudiences), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}}).SignedString([]byte(cfg.HMACSecret))
				if err != nil {
					t.Fatal(err)
				}
				engine := gin.New()
				RegisterRoutes(engine, deps, cfg, "/api/v1")
				server := httptest.NewServer(engine)
				defer server.Close()
				headers := http.Header{"Authorization": []string{"Bearer " + token}, fwcontracts.IdentityContextHeader: []string{ticket}}
				socket, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws?authorization=must-not-override-core-business-bearer", headers)
				if err != nil {
					t.Fatalf("upgrade failed: %v HTTP=%v", err, response)
				}
				defer socket.Close()
				if response.StatusCode != 101 {
					t.Fatal("WS handshake was not accepted")
				}
				_ = socket.SetReadDeadline(time.Now().Add(2 * time.Second))
				_ = socket.WriteJSON(wsRequest{Type: "subscribe", Topics: []string{topic}})
				var ack wsResponse
				if err = socket.ReadJSON(&ack); err != nil {
					t.Fatal(err)
				}
				if ack.Type != "ack" || calls.Load() != 1 {
					t.Fatal("subscription was not revalidated")
				}
				_ = hub.Publish(context.Background(), topic, map[string]any{"ok": true}, fwwsbus.PublishOptions{TenantUUID: tenant})
				var event wsResponse
				if err = socket.ReadJSON(&event); err != nil || event.Type != "event" || calls.Load() != 2 {
					t.Fatalf("event was not revalidated: %v", err)
				}
				reject.Store(true)
				if direction == "outbound" {
					_ = hub.Publish(context.Background(), topic, map[string]any{"must_not_deliver": true}, fwwsbus.PublishOptions{TenantUUID: tenant})
				} else {
					_ = socket.WriteJSON(wsRequest{Type: "subscribe", Topics: []string{topic}})
				}
				if _, _, err = socket.ReadMessage(); err == nil {
					t.Fatal("revalidation denial delivered a message instead of closing")
				}
				if calls.Load() != 3 {
					t.Fatal("message did not trigger exactly one revalidation")
				}
			})
		}
	}

}
