package eventfabric

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	Tenant        string         `json:"tenant_uuid"`
	Publisher     string         `json:"publisher_plugin_id"`
	Subscriber    string         `json:"subscriber_id"`
	Business      string         `json:"business_topic"`
	Physical      string         `json:"physical_topic"`
	Envelope      event.Event    `json:"envelope"`
	Delivery      Delivery       `json:"delivery"`
	PublishResult PublishReceipt `json:"publish_result"`
	AckResult     NackResult     `json:"ack_result"`
	NackResult    NackResult     `json:"nack_result"`
	Resolve       struct {
		PublishRequest map[string]any `json:"publish_request"`
		PublishResult  ResolvedTopics `json:"publish_result"`
		ConsumeRequest map[string]any `json:"consume_request"`
		ConsumeResult  ResolvedTopics `json:"consume_result"`
	} `json:"resolve_topics"`
	Errors []struct {
		Status int    `json:"status"`
		Code   string `json:"code"`
	} `json:"errors"`
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/event-fabric-service.v1.json")
	require.NoError(t, err)
	var f fixture
	require.NoError(t, json.Unmarshal(raw, &f))
	return f
}
func (f fixture) publication() PublishRequest {
	raw, _ := json.Marshal(f.Envelope)
	return PublishRequest{f.Tenant, f.Publisher, Event{EventID: f.Delivery.EventID, IdempotencyKey: f.Delivery.IdempotencyKey, Topic: f.Physical, Version: "1", Payload: raw}}
}
func (f fixture) subscription() SubscribeRequest {
	return SubscribeRequest{TenantUUID: f.Tenant, PluginID: f.Subscriber, SubscriberID: f.Subscriber, Topics: []string{f.Physical}, SupportedVersions: []string{"1"}, CompatibilityMode: "strict", BatchSize: 1, ExpectedPublishers: map[string][]string{f.Physical: {f.Publisher}}, ReconnectBackoff: time.Millisecond}
}
func wire(w http.ResponseWriter, result any, protocol string, fallback bool) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "success", "data": map[string]any{"status": "completed", "protocol_used": protocol, "fallback_used": fallback, "result": result}})
}
func frame(w http.ResponseWriter, kind string, value any) {
	raw, _ := json.MarshalIndent(value, "", " ")
	_, _ = fmt.Fprintf(w, "event: %s\n", kind)
	for _, line := range splitLines(raw) {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = fmt.Fprint(w, "\n")
	w.(http.Flusher).Flush()
}
func splitLines(raw []byte) []string {
	var out []string
	start := 0
	for i, b := range raw {
		if b == '\n' {
			out = append(out, string(raw[start:i]))
			start = i + 1
		}
	}
	return append(out, string(raw[start:]))
}
func ready(w http.ResponseWriter, f fixture) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Trace-ID", "core-stream-trace")
	w.Header().Set("X-Request-ID", "core-stream-request")
	frame(w, "ready", map[string]string{"contract_version": Contract, "tenant_uuid": f.Tenant, "subscriber_id": f.Subscriber})
}
func apiClient(t *testing.T, s *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(Config{BaseURL: s.URL + "/fixture/v2", APIPrefix: "/fixture/v2", AuthScheme: "apikey", APIKey: "fixture-service-key"}, s.Client())
	require.NoError(t, err)
	return c
}

func TestPublishFixedBindingCredentialsAndOriginalEnvelope(t *testing.T) {
	f := loadFixture(t)
	for _, mode := range []string{"apikey", "sts"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			tokenCalls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/fixture/v2/tenant/invocations", r.URL.Path)
				want := "ApiKey fixture-service-key"
				if mode == "sts" {
					want = "Bearer fixture-sts"
				}
				require.Equal(t, want, r.Header.Get("Authorization"))
				var body struct {
					Capability string `json:"capability_id"`
					Protocol   string `json:"preferred_protocol"`
					Payload    struct {
						Body struct {
							Contract string `json:"contract_version"`
							Event    Event  `json:"event"`
						} `json:"body"`
					} `json:"payload"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, PublishCapability, body.Capability)
				require.Equal(t, Protocol, body.Protocol)
				require.Equal(t, Contract, body.Payload.Body.Contract)
				require.Equal(t, f.Delivery.EventID, body.Payload.Body.Event.EventID)
				require.Equal(t, "1", body.Payload.Body.Event.Version)
				raw, _ := json.Marshal(f.Envelope)
				require.JSONEq(t, string(raw), string(body.Payload.Body.Event.Payload))
				wire(w, f.PublishResult, Protocol, false)
			}))
			defer s.Close()
			cfg := Config{BaseURL: s.URL, APIPrefix: "/fixture/v2", AuthScheme: mode, APIKey: "fixture-service-key"}
			c, err := NewClientWithTokenProvider(cfg, TokenProviderFunc(func(context.Context) (string, error) { tokenCalls++; return "fixture-sts", nil }), s.Client())
			require.NoError(t, err)
			out, err := c.Publish(middleware.WithBearerToken(context.Background(), "fixture-admin-token"), f.publication())
			require.NoError(t, err)
			require.Equal(t, f.PublishResult, *out)
			require.Equal(t, 1, calls)
			if mode == "apikey" {
				require.Zero(t, tokenCalls)
			} else {
				require.Equal(t, 1, tokenCalls)
			}
		})
	}
}
func TestPublishRejectsIncompleteAndFallbackResults(t *testing.T) {
	f := loadFixture(t)
	for _, result := range []struct {
		Protocol string
		Fallback bool
		Result   any
	}{{"rest", false, f.PublishResult}, {Protocol, true, f.PublishResult}, {Protocol, false, map[string]any{"accepted": true}}, {Protocol, false, PublishReceipt{Contract, true, "foreign", "trace"}}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { wire(w, result.Result, result.Protocol, result.Fallback) }))
		c := apiClient(t, s)
		_, err := c.Publish(context.Background(), f.publication())
		require.Error(t, err)
		s.Close()
	}
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("EVENT_FABRIC_INVALID_INPUT_SENT") }))
	defer s.Close()
	c := apiClient(t, s)
	in := f.publication()
	in.Event.IdempotencyKey = ""
	_, err := c.Publish(context.Background(), in)
	require.Error(t, err)
}
func TestUpstreamErrorsPreservedWithoutAutomaticBusinessRetry(t *testing.T) {
	f := loadFixture(t)
	for _, item := range f.Errors {
		t.Run(item.Code, func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Trace-ID", "upstream-trace")
				w.Header().Set("X-Request-ID", "upstream-request")
				w.WriteHeader(item.Status)
				_ = json.NewEncoder(w).Encode(map[string]any{"error_code": item.Code, "reason_code": item.Code, "details": map[string]any{"reason_code": item.Code}})
			}))
			defer s.Close()
			_, err := apiClient(t, s).Publish(context.Background(), f.publication())
			var up *HTTPError
			require.ErrorAs(t, err, &up)
			require.Equal(t, item.Status, up.StatusCode)
			require.Equal(t, item.Code, up.ReasonCode)
			require.Equal(t, "upstream-trace", up.TraceID)
			require.Equal(t, "upstream-request", up.RequestID)
			require.Contains(t, up.Body, item.Code)
			require.Equal(t, 1, calls)
		})
	}
}
func TestSSERecoveryKeepsIdentityAndNewReceipt(t *testing.T) {
	f := loadFixture(t)
	var calls atomic.Int32
	firstBody := make(chan []byte, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/fixture/v2/tenant/invocations/stream", r.URL.Path)
		require.Equal(t, "ApiKey fixture-service-key", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		raw, _ := json.Marshal(body)
		if calls.Add(1) == 1 {
			firstBody <- raw
			ready(w, f)
			frame(w, "delivery", f.Delivery)
			return
		}
		require.JSONEq(t, string(<-firstBody), string(raw))
		ready(w, f)
		d := f.Delivery
		d.DeliveryID = "66666666-6666-4666-8666-666666666666"
		d.Attempt = 2
		frame(w, "delivery", d)
		<-r.Context().Done()
	}))
	defer s.Close()
	sub, err := apiClient(t, s).Subscribe(context.Background(), f.subscription())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	one, err := sub.Recv(ctx)
	require.NoError(t, err)
	two, err := sub.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, one.EventID, two.EventID)
	require.Equal(t, one.IdempotencyKey, two.IdempotencyKey)
	require.NotEqual(t, one.DeliveryID, two.DeliveryID)
	require.Equal(t, 2, two.Attempt)
	require.NoError(t, sub.Close())
	require.EqualValues(t, 2, calls.Load())
}
func TestSSERejectsScopeAndIncompleteDelivery(t *testing.T) {
	f := loadFixture(t)
	for _, mutate := range []func(*Delivery){func(d *Delivery) { d.TenantUUID = "77777777-7777-4777-8777-777777777777" }, func(d *Delivery) { d.PublisherPluginID = "foreign" }, func(d *Delivery) { d.SubscriberID = "foreign" }, func(d *Delivery) { d.DeliveryID = "" }, func(d *Delivery) { d.AckDeadline = time.Time{} }, func(d *Delivery) { d.Version = "v1" }, func(d *Delivery) { d.Attempt = 0 }, func(d *Delivery) { d.Topic = "foreign" }} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ready(w, f)
			d := f.Delivery
			mutate(&d)
			frame(w, "delivery", d)
		}))
		sub, err := apiClient(t, s).Subscribe(context.Background(), f.subscription())
		require.NoError(t, err)
		_, err = sub.Recv(context.Background())
		var up *HTTPError
		require.ErrorAs(t, err, &up)
		require.Equal(t, "EVENT_FABRIC_RESPONSE_INVALID", up.ReasonCode)
		_ = sub.Close()
		s.Close()
	}
}
func TestSSEAuthorizationStopsAndCloseCancels(t *testing.T) {
	f := loadFixture(t)
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		ready(w, f)
		frame(w, "error", map[string]string{"code": "EVENT_FABRIC_GRANT_REQUIRED"})
	}))
	sub, err := apiClient(t, s).Subscribe(context.Background(), f.subscription())
	require.NoError(t, err)
	_, err = sub.Recv(context.Background())
	var up *HTTPError
	require.ErrorAs(t, err, &up)
	require.Equal(t, 403, up.StatusCode)
	require.Equal(t, "core-stream-trace", up.TraceID)
	require.Equal(t, "core-stream-request", up.RequestID)
	_ = sub.Close()
	require.EqualValues(t, 1, calls.Load())
	s.Close()
	closed := make(chan struct{})
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ready(w, f); <-r.Context().Done(); close(closed) }))
	defer s.Close()
	sub, err = apiClient(t, s).Subscribe(context.Background(), f.subscription())
	require.NoError(t, err)
	require.NoError(t, sub.Close())
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("EVENT_FABRIC_CANCEL_LEAK")
	}
}
func TestAckAndNackResultsAndUncertainty(t *testing.T) {
	f := loadFixture(t)
	for _, operation := range []string{"ack", "nack"} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			payload := body["payload"].(map[string]any)["body"].(map[string]any)
			require.Equal(t, ConsumeCapability, body["capability_id"])
			require.Equal(t, operation, payload["operation"])
			require.Equal(t, f.Delivery.DeliveryID, payload["delivery_id"])
			out := f.AckResult
			if operation == "nack" {
				out = f.NackResult
				require.Equal(t, "consumer_failed", payload["reason"])
			}
			wire(w, out, Protocol, false)
		}))
		c := apiClient(t, s)
		if operation == "ack" {
			require.NoError(t, c.Ack(context.Background(), f.Delivery.Receipt()))
		} else {
			out, err := c.Nack(context.Background(), f.Delivery.Receipt(), "consumer_failed")
			require.NoError(t, err)
			require.Equal(t, f.NackResult, *out)
		}
		s.Close()
	}
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(503)
		_, _ = fmt.Fprint(w, `{"error_code":"EVENT_FABRIC_UNAVAILABLE"}`)
	}))
	defer s.Close()
	c := apiClient(t, s)
	err := c.Ack(context.Background(), f.Delivery.Receipt())
	var settlement *SettlementError
	require.ErrorAs(t, err, &settlement)
	require.True(t, settlement.Uncertain)
	require.EqualValues(t, 1, calls.Load())
	receipt := f.Delivery.Receipt()
	receipt.AckDeadline = time.Now().Add(-time.Second)
	err = c.Ack(context.Background(), receipt)
	require.ErrorAs(t, err, &settlement)
	require.False(t, settlement.Uncertain)
	require.EqualValues(t, 1, calls.Load())
}
func TestNoRedirectOrCredentialFallback(t *testing.T) {
	f := loadFixture(t)
	var leaked atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { leaked.Add(1) }))
	defer target.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer s.Close()
	_, err := apiClient(t, s).Publish(context.Background(), f.publication())
	require.Error(t, err)
	require.Zero(t, leaked.Load())
	_, err = NewClientWithTokenProvider(Config{BaseURL: s.URL, AuthScheme: "apikey"}, TokenProviderFunc(func(context.Context) (string, error) { t.Error("EVENT_FABRIC_CREDENTIAL_FALLBACK"); return "", nil }), s.Client())
	require.Error(t, err)
}

func TestNormalReconnectAndBoundedEmptyEOF(t *testing.T) {
	f := loadFixture(t)
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready(w, f)
		if calls.Add(1) == 1 {
			frame(w, "reconnect", map[string]string{"contract_version": Contract})
			return
		}
		frame(w, "delivery", f.Delivery)
		<-r.Context().Done()
	}))
	sub, err := apiClient(t, s).Subscribe(context.Background(), f.subscription())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = sub.Recv(ctx)
	require.NoError(t, err)
	_ = sub.Close()
	require.EqualValues(t, 2, calls.Load())
	s.Close()
	calls.Store(0)
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); ready(w, f) }))
	defer s.Close()
	in := f.subscription()
	in.ReconnectLimit = 2
	sub, err = apiClient(t, s).Subscribe(context.Background(), in)
	require.NoError(t, err)
	_, err = sub.Recv(ctx)
	require.Error(t, err)
	_ = sub.Close()
	require.EqualValues(t, 3, calls.Load())
}

func TestMissingRoutingEvidenceAndIncompleteAcknowledgementFail(t *testing.T) {
	f := loadFixture(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"status": "completed", "protocol_used": Protocol, "result": f.PublishResult}})
	}))
	_, err := apiClient(t, s).Publish(context.Background(), f.publication())
	require.Error(t, err)
	s.Close()
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wire(w, map[string]any{"contract_version": Contract, "confirmed": true, "delivery_id": f.Delivery.DeliveryID, "outcome": "acked"}, Protocol, false)
	}))
	defer s.Close()
	err = apiClient(t, s).Ack(context.Background(), f.Delivery.Receipt())
	var settle *SettlementError
	require.ErrorAs(t, err, &settle)
	require.True(t, settle.Uncertain)
}

func TestInvalidStartupConfigurationFails(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "not-a-service-url", AuthScheme: "apikey", APIKey: "fixture"}, nil)
	require.Error(t, err)
	_, err = NewClientWithTokenProvider(Config{BaseURL: "http://core.invalid"}, TokenProviderFunc(nil), nil)
	require.Error(t, err)
}
