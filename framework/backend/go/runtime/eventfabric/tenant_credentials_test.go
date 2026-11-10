package eventfabric

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolvedRuntimeDoesNotTreatMappingAsGrantOrReuseReceiptAcrossCredentials(t *testing.T) {
	f := loadFixture(t)
	var revoked atomic.Bool
	var rejected atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, in, operation := readTopicsRequest(t, r)
		if revoked.Load() || (operation == "ack" && r.Header.Get("Authorization") != "ApiKey first-credential") {
			rejected.Add(1)
			w.WriteHeader(403)
			_ = json.NewEncoder(w).Encode(map[string]string{"error_code": "EVENT_FABRIC_GRANT_REQUIRED"})
			return
		}
		switch operation {
		case "resolve_topics":
			wire(w, ResolvedTopics{Contract, in.TenantUUID, in.PluginID, map[string]string{f.Business: f.Physical}}, Protocol, false)
		case "publish":
			wire(w, f.PublishResult, Protocol, false)
		case "ack":
			wire(w, f.AckResult, Protocol, false)
		default:
			t.Error("UNEXPECTED_OPERATION")
		}
	}))
	defer s.Close()
	first, err := NewTenantHostRuntime(context.Background(), tenantOptions(s, f, f.Tenant, "first-credential"))
	require.NoError(t, err)
	defer first.Close()
	second, err := NewTenantHostRuntime(context.Background(), tenantOptions(s, f, f.Tenant, "second-credential"))
	require.NoError(t, err)
	defer second.Close()
	receipt := f.Delivery.Receipt()
	receipt.PluginID = f.Publisher
	receipt.SubscriberID = f.Publisher
	require.NoError(t, first.Client.Ack(context.Background(), receipt))
	err = second.Client.Ack(context.Background(), receipt)
	var settlement *SettlementError
	require.ErrorAs(t, err, &settlement)
	require.False(t, settlement.Uncertain)
	require.EqualValues(t, 1, rejected.Load())
	revoked.Store(true)
	_, err = first.PublishEvent(context.Background(), publishFixture(f, f.Envelope))
	var denied *HTTPError
	require.ErrorAs(t, err, &denied)
	require.Equal(t, 403, denied.StatusCode)
	require.EqualValues(t, 2, rejected.Load())
	_, err = NewTenantHostRuntime(context.Background(), tenantOptions(s, f, f.Tenant, "first-credential"))
	require.Error(t, err)
	require.EqualValues(t, 3, rejected.Load())
}

func TestRuntimeCloseCancelsPendingPublication(t *testing.T) {
	f := loadFixture(t)
	started := make(chan struct{})
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, in, operation := readTopicsRequest(t, r)
		if operation == "resolve_topics" {
			wire(w, ResolvedTopics{Contract, in.TenantUUID, in.PluginID, map[string]string{f.Business: f.Physical}}, Protocol, false)
			return
		}
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer s.Close()
	defer close(release)
	r, err := NewTenantHostRuntime(context.Background(), tenantOptions(s, f, f.Tenant, "fixture"))
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, err := r.PublishEvent(context.Background(), publishFixture(f, f.Envelope)); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("PUBLISH_NOT_STARTED")
	}
	require.NoError(t, r.Close())
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("PUBLISH_NOT_CANCELLED")
	}
}

func TestResolvedRuntimeReconnectUsesSameCredentialAndSubscriber(t *testing.T) {
	f := loadFixture(t)
	f.Subscriber = f.Publisher
	f.Delivery.SubscriberID = f.Publisher
	var streams atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "ApiKey tenant-key", r.Header.Get("Authorization"))
		if r.URL.Path == "/api/v1/tenant/invocations/stream" {
			var in topicInvocation
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			require.JSONEq(t, `"`+f.Publisher+`"`, string(in.Payload.Body["subscriber_id"]))
			require.NotContains(t, in.Payload.Body, "topic_keys")
			ready(w, f)
			if streams.Add(1) == 1 {
				frame(w, "reconnect", map[string]any{"contract_version": Contract, "reason": "fixture"})
				return
			}
			frame(w, "delivery", f.Delivery)
			_, _ = io.Copy(io.Discard, r.Body)
			<-r.Context().Done()
			return
		}
		_, in, _ := readTopicsRequest(t, r)
		wire(w, ResolvedTopics{Contract, in.TenantUUID, in.PluginID, map[string]string{f.Business: f.Physical}}, Protocol, false)
	}))
	defer s.Close()
	opts := tenantOptions(s, f, f.Tenant, "tenant-key")
	opts.PublishTopicKeys = nil
	opts.ConsumeTopicKeys = []string{f.Business}
	r, err := NewTenantHostRuntime(context.Background(), opts)
	require.NoError(t, err)
	defer r.Close()
	in := f.subscription()
	in.ReconnectBackoff = time.Millisecond
	sub, err := r.Subscribe(context.Background(), in)
	require.NoError(t, err)
	defer sub.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	delivery, err := sub.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, f.Delivery.EventID, delivery.EventID)
	require.EqualValues(t, 2, streams.Load())
}
