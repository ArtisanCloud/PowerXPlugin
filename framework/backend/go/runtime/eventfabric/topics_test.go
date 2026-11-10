package eventfabric

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type topicInvocation struct {
	Capability string `json:"capability_id"`
	Protocol   string `json:"preferred_protocol"`
	Payload    struct {
		Body map[string]json.RawMessage `json:"body"`
	} `json:"payload"`
}

func TestResolveTopicsDirectionsAndServiceCredentials(t *testing.T) {
	f := loadFixture(t)
	for _, scheme := range []string{"apikey", "sts"} {
		for _, capability := range []string{PublishCapability, ConsumeCapability} {
			t.Run(scheme+"/"+capability, func(t *testing.T) {
				owner, expectedBody, result := f.Publisher, f.Resolve.PublishRequest, f.Resolve.PublishResult
				if capability == ConsumeCapability {
					owner, expectedBody, result = f.Subscriber, f.Resolve.ConsumeRequest, f.Resolve.ConsumeResult
				}
				calls := 0
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					require.Equal(t, "/fixture/v2/tenant/invocations", r.URL.Path)
					require.Equal(t, http.MethodPost, r.Method)
					expected := "ApiKey fixture-key"
					if scheme == "sts" {
						expected = "Bearer fixture-sts"
					}
					require.Equal(t, expected, r.Header.Get("Authorization"))
					var request topicInvocation
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, capability, request.Capability)
					require.Equal(t, Protocol, request.Protocol)
					body := request.Payload.Body
					actual, _ := json.Marshal(body)
					expectedJSON, _ := json.Marshal(expectedBody)
					require.JSONEq(t, string(expectedJSON), string(actual))
					require.JSONEq(t, `"resolve_topics"`, string(body["operation"]))
					require.JSONEq(t, `["`+f.Business+`"]`, string(body["topic_keys"]))
					require.NotContains(t, body, "topics")
					require.NotContains(t, body, "event")
					require.NotContains(t, body, "headers")
					if capability == ConsumeCapability {
						require.JSONEq(t, `"`+owner+`"`, string(body["subscriber_id"]))
					} else {
						require.NotContains(t, body, "subscriber_id")
					}
					wire(w, result, Protocol, false)
				}))
				defer s.Close()
				cfg := Config{BaseURL: s.URL + "/fixture/v2", APIPrefix: "/fixture/v2", AuthScheme: scheme, APIKey: "fixture-key"}
				providerCalls := 0
				client, err := NewClientWithTokenProvider(cfg, TokenProviderFunc(func(context.Context) (string, error) { providerCalls++; return "fixture-sts", nil }), s.Client())
				require.NoError(t, err)
				in := ResolveTopicsRequest{TenantUUID: f.Tenant, PluginID: owner, CapabilityID: capability, TopicKeys: []string{f.Business}}
				if capability == ConsumeCapability {
					in.SubscriberID = owner
				}
				out, err := client.ResolveTopics(context.Background(), in)
				require.NoError(t, err)
				require.Equal(t, f.Physical, out.TopicBindings[f.Business])
				require.Equal(t, 1, calls)
				if scheme == "apikey" {
					require.Zero(t, providerCalls)
				} else {
					require.Equal(t, 1, providerCalls)
				}
			})
		}
	}
}

func TestResolveTopicsRejectsInvalidRequestsBeforeNetwork(t *testing.T) {
	f := loadFixture(t)
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer s.Close()
	client := apiClient(t, s)
	for _, change := range []func(*ResolveTopicsRequest){
		func(in *ResolveTopicsRequest) { in.TenantUUID = "" }, func(in *ResolveTopicsRequest) { in.PluginID = "" },
		func(in *ResolveTopicsRequest) { in.CapabilityID = "rest" }, func(in *ResolveTopicsRequest) { in.TopicKeys = nil },
		func(in *ResolveTopicsRequest) { in.TopicKeys = []string{f.Business, f.Business} }, func(in *ResolveTopicsRequest) { in.TopicKeys = []string{" "} },
		func(in *ResolveTopicsRequest) { in.TopicKeys = make([]string, 1001) }, func(in *ResolveTopicsRequest) { in.SubscriberID = f.Publisher },
		func(in *ResolveTopicsRequest) { in.CapabilityID = ConsumeCapability },
	} {
		in := ResolveTopicsRequest{TenantUUID: f.Tenant, PluginID: f.Publisher, CapabilityID: PublishCapability, TopicKeys: []string{f.Business}}
		change(&in)
		_, err := client.ResolveTopics(context.Background(), in)
		require.Error(t, err)
	}
	require.Zero(t, calls.Load())
}

func TestResolveTopicsRejectsForeignMissingExtraAndDuplicateBindings(t *testing.T) {
	f := loadFixture(t)
	for name, change := range map[string]func(*ResolvedTopics){
		"contract": func(out *ResolvedTopics) { out.ContractVersion = "foreign" },
		"tenant":   func(out *ResolvedTopics) { out.TenantUUID = otherTenant },
		"owner":    func(out *ResolvedTopics) { out.PluginID = "com.example.foreign" },
		"missing":  func(out *ResolvedTopics) { delete(out.TopicBindings, f.Business) },
		"extra":    func(out *ResolvedTopics) { out.TopicBindings["extra"] = f.Physical + ".extra" },
		"wrong-key": func(out *ResolvedTopics) {
			delete(out.TopicBindings, f.Business)
			out.TopicBindings["extra"] = f.Physical
		},
		"foreign-topic": func(out *ResolvedTopics) {
			out.TopicBindings[f.Business] = otherTenant + ".powerx.fixture.changed"
		},
		"empty-topic": func(out *ResolvedTopics) { out.TopicBindings[f.Business] = f.Tenant + "." },
		"nil":         func(out *ResolvedTopics) { out.TopicBindings = nil },
	} {
		t.Run(name, func(t *testing.T) {
			out := ResolvedTopics{Contract, f.Tenant, f.Publisher, map[string]string{f.Business: f.Physical}}
			change(&out)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { wire(w, out, Protocol, false) }))
			defer s.Close()
			_, err := apiClient(t, s).ResolveTopics(context.Background(), ResolveTopicsRequest{TenantUUID: f.Tenant, PluginID: f.Publisher, CapabilityID: PublishCapability, TopicKeys: []string{f.Business}})
			var failure *HTTPError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "EVENT_FABRIC_RESPONSE_INVALID", failure.ReasonCode)
		})
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wire(w, ResolvedTopics{Contract, f.Tenant, f.Publisher, map[string]string{"a": f.Physical, "b": f.Physical}}, Protocol, false)
	}))
	defer s.Close()
	_, err := apiClient(t, s).ResolveTopics(context.Background(), ResolveTopicsRequest{TenantUUID: f.Tenant, PluginID: f.Publisher, CapabilityID: PublishCapability, TopicKeys: []string{"a", "b"}})
	require.Error(t, err)
}

func TestResolveTopicsPreservesExplicitFailuresWithoutRetry(t *testing.T) {
	f := loadFixture(t)
	for code, status := range map[string]int{"EVENT_FABRIC_GRANT_REQUIRED": 403, "EVENT_FABRIC_OWNER_FORBIDDEN": 403, "EVENT_FABRIC_SERVICE_REQUIRED": 401, "EVENT_FABRIC_TOPIC_BINDING_REQUIRED": 409, "EVENT_FABRIC_UNAVAILABLE": 503} {
		t.Run(code, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"error_code": code, "message": code})
			}))
			defer s.Close()
			_, err := apiClient(t, s).ResolveTopics(context.Background(), ResolveTopicsRequest{TenantUUID: f.Tenant, PluginID: f.Publisher, CapabilityID: PublishCapability, TopicKeys: []string{f.Business}})
			var up *HTTPError
			require.ErrorAs(t, err, &up)
			require.Equal(t, status, up.StatusCode)
			require.Equal(t, code, up.ReasonCode)
			require.EqualValues(t, 1, calls.Load())
		})
	}
}
