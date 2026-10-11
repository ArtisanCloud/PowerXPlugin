package realtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

const leaseHandle = "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func leaseFixture() *contracts.DelegatedIdentity {
	now := time.Now().UTC()
	expiry := now.Add(time.Minute)
	return &contracts.DelegatedIdentity{Schema: contracts.IdentityResultSchema, Mode: "delegated", PluginID: "com.example.identity", TenantUUID: "11111111-1111-4111-8111-111111111111", UserUUID: "22222222-2222-4222-8222-222222222222", IsRoot: true, Roles: []string{}, Permissions: []string{}, PolicyVersion: "p1", PermsHash: "hash", SourceMethod: "GET", SourcePath: "/ws", Transport: "ws_upgrade", SourceRequestID: "source", SourceTraceID: "source", TraceID: "resolve", ProofExpiresAt: now.Add(time.Second), UserSessionExpiresAt: expiry, WSRevalidationHandle: leaseHandle, WSRevalidationExpiresAt: &expiry}
}

type leaseRevalidator struct {
	value *contracts.DelegatedIdentity
	calls atomic.Int64
	fn    func(context.Context, *contracts.WSIdentity) error
}

func (r *leaseRevalidator) RevalidateWSIdentity(ctx context.Context, q contracts.WSIdentityQuery) (*contracts.WSIdentity, error) {
	r.calls.Add(1)
	if q.Handle != leaseHandle {
		return nil, iamerrors.New("IAM_CONTEXT_WS_HANDLE_INVALID", "IAM_CONTEXT_WS_HANDLE_INVALID")
	}
	v := *r.value
	v.Schema = contracts.WSIdentityResultSchema
	v.ProofExpiresAt = *v.WSRevalidationExpiresAt
	v.TraceID = "fresh-resolve"
	out := &contracts.WSIdentity{DelegatedIdentity: v}
	if r.fn != nil {
		if err := r.fn(ctx, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func TestIdentityLeaseBoundsExpiryAndClosesWithoutTraffic(t *testing.T) {
	registry := NewIdentityLifecycle()
	defer registry.Close()
	v := leaseFixture()
	v.ProofExpiresAt = time.Now().UTC().Add(40 * time.Millisecond)
	v.UserSessionExpiresAt = v.ProofExpiresAt
	expiry := v.UserSessionExpiresAt
	v.WSRevalidationExpiresAt = &expiry
	r := &leaseRevalidator{value: v}
	closed := make(chan string, 1)
	lease, err := registry.Open(IdentityLeaseOptions{Revalidator: r, Identity: v, PluginID: v.PluginID, TTL: time.Hour, ObservedRevision: registry.Revision(), AuthorizeTopic: func(*contracts.DelegatedIdentity, string) bool { return true }, Close: func(reason string) { closed <- reason }})
	if err != nil {
		t.Fatal(err)
	}
	if lease.ExpiresAt.After(v.UserSessionExpiresAt) {
		t.Fatal("lifetime extended session")
	}
	select {
	case reason := <-closed:
		if reason != WSIdentityExpired {
			t.Fatal(reason)
		}
	case <-time.After(time.Second):
		t.Fatal("idle lease did not expire")
	}
	if r.calls.Load() != 0 {
		t.Fatal("idle connection polled identity")
	}
	if lease.AuthorizeTopic(context.Background(), "_topic.allowed") == nil {
		t.Fatal("expired lease authorized")
	}
}
func TestIdentityLeaseTopicRevocationAndHydrationFence(t *testing.T) {
	registry := NewIdentityLifecycle()
	defer registry.Close()
	v := leaseFixture()
	r := &leaseRevalidator{value: v}
	var closed atomic.Int32
	revision := registry.Revision()
	opts := IdentityLeaseOptions{Revalidator: r, Identity: v, PluginID: v.PluginID, TTL: time.Minute, ObservedRevision: revision, AuthorizeTopic: func(_ *contracts.DelegatedIdentity, topic string) bool { return topic == "_topic.allowed" }, Close: func(string) { closed.Add(1) }}
	lease, err := registry.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	if lease.AuthorizeTopic(context.Background(), "_topic.allowed") != nil {
		t.Fatal("authorized topic rejected")
	}
	if registry.Revoke(IdentityRevocation{TenantUUID: "33333333-3333-4333-8333-333333333333"}) != 0 {
		t.Fatal("cross tenant revoke")
	}
	if registry.Revoke(IdentityRevocation{TenantUUID: v.TenantUUID, UserUUID: v.UserUUID}) != 1 || closed.Load() != 1 || lease.AuthorizeTopic(context.Background(), "_topic.allowed") == nil {
		t.Fatal("revocation did not close")
	}
	if _, err = registry.Open(opts); iamerrors.CodeOf(err) != "IAM_CONTEXT_POLICY_CHANGED" {
		t.Fatal("hydration fence missing")
	}
}
func TestWSRevalidationUsesFreshPermissionsAfterOriginalTicketExpires(t *testing.T) {
	registry := NewIdentityLifecycle()
	defer registry.Close()
	v := leaseFixture()
	v.ProofExpiresAt = time.Now().UTC().Add(20 * time.Millisecond)
	v.IsRoot = false
	member := "33333333-3333-4333-8333-333333333333"
	v.MemberUUID = &member
	v.Permissions = []string{"topic:read"}
	r := &leaseRevalidator{value: v}
	var closed atomic.Int32
	lease, err := registry.Open(IdentityLeaseOptions{Revalidator: r, Identity: v, PluginID: v.PluginID, TTL: time.Second, ObservedRevision: registry.Revision(), AuthorizeTopic: func(i *contracts.DelegatedIdentity, _ string) bool { return i.HasPermission("topic:read") }, Close: func(string) { closed.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(25 * time.Millisecond)
	if r.calls.Load() != 0 {
		t.Fatal("periodic identity call")
	}
	if err = lease.AuthorizeTopic(context.Background(), "_topic.allowed"); err != nil {
		t.Fatal(err)
	}
	r.fn = func(_ context.Context, i *contracts.WSIdentity) error {
		i.Permissions = []string{}
		i.PolicyVersion = "p2"
		return nil
	}
	if err = lease.AuthorizeTopic(context.Background(), "_topic.allowed"); iamerrors.CodeOf(err) != WSTopicForbidden || closed.Load() != 1 || r.calls.Load() != 2 {
		t.Fatal("cached permissions authorized revoked message")
	}
}
func TestWSRevalidationDenialMismatchAndLateResultClose(t *testing.T) {
	for _, code := range []string{"IAM_CONTEXT_USER_SESSION_INVALID", "IAM_CONTEXT_USER_FORBIDDEN", "IAM_CONTEXT_SERVICE_FORBIDDEN", "IAM_CONTEXT_POLICY_CHANGED", "IAM_CONTEXT_UNAVAILABLE", "IAM_WS_IDENTITY_MISMATCH", "LATE_RESULT"} {
		t.Run(code, func(t *testing.T) {
			registry := NewIdentityLifecycle()
			defer registry.Close()
			v := leaseFixture()
			r := &leaseRevalidator{value: v}
			var closed atomic.Int32
			authorized := false
			lease, err := registry.Open(IdentityLeaseOptions{Revalidator: r, Identity: v, PluginID: v.PluginID, TTL: time.Minute, ObservedRevision: registry.Revision(), AuthorizeTopic: func(*contracts.DelegatedIdentity, string) bool { authorized = true; return true }, Close: func(string) { closed.Add(1) }})
			if err != nil {
				t.Fatal(err)
			}
			r.fn = func(_ context.Context, i *contracts.WSIdentity) error {
				if code == "IAM_WS_IDENTITY_MISMATCH" {
					i.UserUUID = "44444444-4444-4444-8444-444444444444"
					return nil
				}
				if code == "LATE_RESULT" {
					lease.Close(WSIdentityRevoked)
					return nil
				}
				return iamerrors.New(code, code)
			}
			if err = lease.AuthorizeTopic(context.Background(), "_topic.allowed"); err == nil || authorized || closed.Load() != 1 {
				t.Fatal("failed revalidation authorized message")
			}
		})
	}
}

func TestTopicAuthorizationCancellationCannotCommitSuccess(t *testing.T) {
	registry := NewIdentityLifecycle()
	defer registry.Close()
	v := leaseFixture()
	r := &leaseRevalidator{value: v}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var closed atomic.Int32
	lease, err := registry.Open(IdentityLeaseOptions{Revalidator: r, Identity: v, PluginID: v.PluginID, TTL: time.Minute, ObservedRevision: registry.Revision(), AuthorizeTopic: func(*contracts.DelegatedIdentity, string) bool { cancel(); return true }, Close: func(string) { closed.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	if err = lease.AuthorizeTopic(ctx, "_topic.allowed"); iamerrors.CodeOf(err) != WSIdentityRevoked || closed.Load() != 1 {
		t.Fatal("CANCELED_AUTHORIZATION_COMMITTED")
	}
}
