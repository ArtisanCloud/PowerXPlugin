package wsbus

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type captureSubscriber struct {
	handler func(fwwsbus.Event)
}

func (s *captureSubscriber) Subscribe(_ string, handler func(fwwsbus.Event)) func() {
	s.handler = handler
	return func() {}
}

func TestSubscribeFiltersMemberScopedEvents(t *testing.T) {
	subscriber := &captureSubscriber{}
	conn := &wsConn{
		subs:       map[string]func(){},
		tenantUUID: "tenant-1",
		memberUUID: "11111111-1111-4111-8111-111111111111",
	}
	sent := 0
	conn.sendHook = func(wsResponse) {
		sent++
	}
	conn.subscribe(subscriber, []string{"_topic.notify.member.11111111-1111-4111-8111-111111111111"}, []frameworkrealtime.Descriptor{{Key: "_topic.notify.member.11111111-1111-4111-8111-111111111111", Protocols: []frameworkrealtime.Protocol{frameworkrealtime.ProtocolWS}, Actions: []frameworkrealtime.Action{frameworkrealtime.ActionSubscribe}, Scope: frameworkrealtime.ScopeMember, EventTypes: []string{"message"}}})
	if subscriber.handler == nil {
		t.Fatal("expected subscription handler")
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.member.22222222-2222-4222-8222-222222222222",
		TenantUUID: "tenant-1",
		MemberUUID: "22222222-2222-4222-8222-222222222222",
	})
	if sent != 0 {
		t.Fatalf("cross-member event was delivered, sent=%d", sent)
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.member.11111111-1111-4111-8111-111111111111",
		TenantUUID: "tenant-1",
		MemberUUID: "11111111-1111-4111-8111-111111111111",
	})
	if sent != 1 {
		t.Fatalf("own member event not delivered, sent=%d", sent)
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.tenant.tenant-1",
		TenantUUID: "tenant-1",
	})
	if sent != 2 {
		t.Fatalf("tenant broadcast event not delivered, sent=%d", sent)
	}
}

func TestMemoryHubCarriesMemberUUID(t *testing.T) {
	hub := fwwsbus.NewMemoryHub()
	received := make(chan fwwsbus.Event, 1)
	hub.Subscribe("_topic.notify.member.11111111-1111-4111-8111-111111111111", func(ev fwwsbus.Event) {
		received <- ev
	})
	err := hub.Publish(context.Background(), "_topic.notify.member.11111111-1111-4111-8111-111111111111", map[string]any{"ok": true}, fwwsbus.PublishOptions{
		TenantUUID: "tenant-1",
		MemberUUID: "11111111-1111-4111-8111-111111111111",
		TraceID:    "trace-1",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	ev := <-received
	if ev.MemberUUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("member_uuid=%q", ev.MemberUUID)
	}
}

func TestResolveIdentityUsesVerifiedMemberUUIDClaim(t *testing.T) {
	const tenantUUID = "11111111-1111-4111-8111-111111111111"
	const memberUUID = "22222222-2222-4222-8222-222222222222"
	const secret = "wsbus-test-secret"
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.PowerXClaims{
		TenantUUID: middleware.TenantClaim(tenantUUID),
		MemberUUID: memberUUID,
		MemberID:   42,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wsbus-test",
			Audience:  jwt.ClaimStrings{"wsbus-test"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/api/ws", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	tenant, member, ok := resolveIdentity(ctx, middleware.JWTAuthConfig{
		HMACSecret: secret, Issuer: "wsbus-test", AcceptAudiences: []string{"wsbus-test"},
	})
	if !ok || tenant != tenantUUID || member != memberUUID {
		t.Fatalf("identity = tenant:%q member:%q ok:%t", tenant, member, ok)
	}
}

func TestResolveIdentityUsesVerifiedExplicitMemberUUIDClaim(t *testing.T) {
	const tenantUUID = "11111111-1111-4111-8111-111111111111"
	const memberUUID = "22222222-2222-4222-8222-222222222222"
	const secret = "wsbus-test-secret"
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":         "wsbus-test",
		"aud":         []string{"wsbus-test"},
		"tid":         tenantUUID,
		"member_uuid": memberUUID,
		"exp":         now.Add(time.Minute).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/api/ws?tenant_uuid=must-not-authorize", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	tenant, member, ok := resolveIdentity(ctx, middleware.JWTAuthConfig{
		HMACSecret: secret, Issuer: "wsbus-test", AcceptAudiences: []string{"wsbus-test"},
	})
	if !ok || tenant != tenantUUID || member != memberUUID {
		t.Fatalf("identity = tenant:%q member:%q ok:%t", tenant, member, ok)
	}
}

func TestResolveIdentityDoesNotTrustTenantQuery(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/api/ws?tenant_uuid=must-not-authorize", nil)
	if _, _, ok := resolveIdentity(ctx, middleware.JWTAuthConfig{}); ok {
		t.Fatal("tenant_uuid query must not authenticate a websocket connection")
	}
}
