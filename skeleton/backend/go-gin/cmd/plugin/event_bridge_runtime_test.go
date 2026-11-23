package main

import (
	"errors"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	fwtaskbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLocalEventRuntimeDoesNotRequireCoreResources(t *testing.T) {
	for _, proxy := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "proxy"}[proxy], func(t *testing.T) {
			var requests atomic.Int32
			core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusForbidden)
			}))
			defer core.Close()
			cfg := &config.Config{
				Gateway:     &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey"},
				EventBridge: &config.EventBridgeConfig{Enabled: true, RedisURL: "redis://127.0.0.1:1", SourcePlugin: "com.example.local-event"},
			}
			mode := fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeLocal, ProxyEnabled: proxy}
			selected := resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
			if _, ok := selected.Provider.(*fwtaskbus.RedisProvider); !ok {
				t.Fatal("LOCAL_TASKBUS_NOT_SELECTED")
			}
			emitter, reliable, err := newEventBridgeEmitter(cfg, mode, selected)
			if err != nil || emitter == nil || reliable != nil {
				t.Fatal("LOCAL_RUNTIME_REQUIRES_HOST_RESOURCES", err)
			}
			defer fweventbridge.Close(emitter)
			if requests.Load() != 0 {
				t.Fatal("LOCAL_STARTUP_REQUESTED_CORE")
			}
			cfg.EventBridge.RedisURL = "invalid-redis-url"
			selected = resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
			if fallback, _, err := newEventBridgeEmitter(cfg, mode, selected); err == nil || fallback != nil {
				t.Fatal("INVALID_LOCAL_REDIS_DOWNGRADED")
			}
		})
	}
}

func eventRuntimeFixture() *config.Config {
	return &config.Config{
		Gateway:      &config.GatewayConfig{BaseURL: "http://127.0.0.1:1", AuthScheme: "apikey", APIKey: "fixture-key"},
		GRPCUpstream: &config.GRPCUpstream{TenantUUID: "22222222-2222-4222-8222-222222222222"},
		EventBridge:  &config.EventBridgeConfig{Enabled: true, RedisURL: "redis://127.0.0.1:1", SourcePlugin: "com.example.event-runtime", TopicBindings: map[string]string{"fixture.changed": "22222222-2222-4222-8222-222222222222.powerx.fixture.changed"}},
	}
}

func TestEventRuntimeAssemblyMatrix(t *testing.T) {
	for _, business := range []string{"local", "delegated"} {
		for _, proxy := range []string{"0", "1"} {
			t.Run(business+proxy, func(t *testing.T) {
				t.Setenv("POWERX_PROVIDER_MODE", business)
				t.Setenv("POWERX_PROXY", proxy)
				cfg := eventRuntimeFixture()
				cfg.Context = &config.ContextConfig{ProviderMode: business}
				resolved := bootstrap.NewProviderResolver(cfg)
				if err := resolved.Err(); err != nil {
					t.Fatal(err)
				}
				mode := resolved.RuntimeMode()
				if proxy == "1" {
					t.Setenv("POWERX_PROXY", "0")
				} else {
					t.Setenv("POWERX_PROXY", "1")
				}
				selected := resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
				_, host := selected.Provider.(*fwtaskbus.HostProvider)
				if host != (business == "delegated") || mode.ProviderMode != fwprovider.Mode(business) {
					t.Fatal("ASSEMBLY_MODE_CHANGED")
				}
				emitter, reliable, err := newEventBridgeEmitter(cfg, mode, selected)
				if err != nil || emitter == nil {
					t.Fatal("VALID_MATRIX_REJECTED", err)
				}
				defer fweventbridge.Close(emitter)
				if (reliable != nil) != (business == "delegated") {
					t.Fatal("EVENT_TRANSPORT_CHANGED")
				}
			})
		}
	}
}

type failedEventProvider struct{ err error }

func (p failedEventProvider) NewEmitter() (fweventbridge.Emitter, error) { return nil, p.err }

func TestEventRuntimeDoesNotDowngradeProviderFailure(t *testing.T) {
	failed := errors.New("FIXTURE_EVENT_PROVIDER_FAILED")
	for _, proxy := range []bool{false, true} {
		mode := fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeLocal, ProxyEnabled: proxy}
		emitter, reliable, err := newEventBridgeEmitter(eventRuntimeFixture(), mode, taskBusRuntime{Provider: failedEventProvider{failed}})
		if !errors.Is(err, failed) || emitter != nil || reliable != nil {
			t.Fatal("EVENT_FAILURE_DOWNGRADED", err)
		}
	}
}

func TestEventRuntimeRejectsMissingHostCredentials(t *testing.T) {
	cfg := eventRuntimeFixture()
	cfg.Gateway.APIKey = ""
	mode := fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeDelegated, ProxyEnabled: true}
	selected := resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
	emitter, reliable, err := newEventBridgeEmitter(cfg, mode, selected)
	if err == nil || emitter != nil || reliable != nil {
		t.Fatal("MISSING_HOST_CREDENTIAL_DOWNGRADED")
	}
}

func TestEventRuntimeReportsMissingHostTopics(t *testing.T) {
	cfg := eventRuntimeFixture()
	cfg.EventBridge.TopicBindings = nil
	mode := fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeDelegated, ProxyEnabled: true}
	selected := resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
	emitter, reliable, err := newEventBridgeEmitter(cfg, mode, selected)
	if !errors.Is(err, errHostEventTopicsRequired) || emitter != nil || reliable != nil {
		t.Fatal("MISSING_TOPIC_DIAGNOSTIC_LOST", err)
	}
}
