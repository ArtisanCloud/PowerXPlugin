package eventbridge

import (
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

type reliableFixture struct{ *TaskBusStub }

func (reliableFixture) ReliableHost() bool { return true }

func TestFactoryUsesRuntimeSnapshot(t *testing.T) {
	for _, business := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		for _, proxy := range []bool{false, true} {
			mode := provider.RuntimeMode{ProviderMode: business, ProxyEnabled: proxy}
			factory, err := NewFactory(Config{Enabled: true, RuntimeMode: mode})
			if err != nil {
				t.Fatal(err)
			}
			if proxy {
				t.Setenv("POWERX_PROXY", "0")
			} else {
				t.Setenv("POWERX_PROXY", "1")
			}
			if business == provider.ModeLocal {
				t.Setenv("POWERX_PROVIDER_MODE", "delegated")
			} else {
				t.Setenv("POWERX_PROVIDER_MODE", "local")
			}
			var selected Emitter = NewTaskBusStub()
			if business == provider.ModeDelegated {
				selected = reliableFixture{NewTaskBusStub()}
			}
			factory.WithTaskBusProvider(func() (Emitter, error) { return selected, nil })
			got, err := factory.NewEmitter()
			if err != nil || got != selected {
				t.Fatal("RUNTIME_SNAPSHOT_CHANGED", err)
			}
		}
	}
}

func TestFactoryNeverFallsBackAfterProviderFailure(t *testing.T) {
	failed := errors.New("FIXTURE_PROVIDER_UNAVAILABLE")
	for _, proxy := range []bool{false, true} {
		factory, err := NewFactory(Config{Enabled: true, RuntimeMode: provider.RuntimeMode{ProviderMode: provider.ModeLocal, ProxyEnabled: proxy}})
		if err != nil {
			t.Fatal(err)
		}
		factory.WithTaskBusProvider(func() (Emitter, error) { return nil, failed })
		if emitter, err := factory.NewEmitter(); emitter != nil || !errors.Is(err, failed) {
			t.Fatal("PROVIDER_FAILURE_DOWNGRADED", err)
		}
	}
}

func TestFactoryRejectsWrongTransport(t *testing.T) {
	for _, business := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		for _, proxy := range []bool{false, true} {
			factory, err := NewFactory(Config{Enabled: true, RuntimeMode: provider.RuntimeMode{ProviderMode: business, ProxyEnabled: proxy}})
			if err != nil {
				t.Fatal(err)
			}
			var wrong Emitter = reliableFixture{NewTaskBusStub()}
			if business == provider.ModeDelegated {
				wrong = NewTaskBusStub()
			}
			factory.WithTaskBusProvider(func() (Emitter, error) { return wrong, nil })
			if _, err := factory.NewEmitter(); !errors.Is(err, ErrReliableHostRequired) {
				t.Fatal("WRONG_TRANSPORT_ACCEPTED", err)
			}
		}
	}
}

func TestFactoryDisabledBusDoesNotInitializeProvider(t *testing.T) {
	factory, err := NewFactory(Config{Enabled: false, RuntimeMode: provider.RuntimeMode{ProviderMode: provider.ModeLocal, ProxyEnabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	factory.WithTaskBusProvider(func() (Emitter, error) { t.Fatal("DISABLED_PROVIDER_INITIALIZED"); return nil, nil })
	if emitter, err := factory.NewEmitter(); err != nil || emitter == nil {
		t.Fatal("DISABLED_LOCAL_QUEUE_MISSING", err)
	}
}

type closeTrackedFixture struct {
	*TaskBusStub
	closed bool
}

func (e *closeTrackedFixture) Close() error { e.closed = true; return nil }
func TestFactoryClosesRejectedTransport(t *testing.T) {
	selected := &closeTrackedFixture{TaskBusStub: NewTaskBusStub()}
	factory, err := NewFactory(Config{Enabled: true, RuntimeMode: provider.RuntimeMode{ProviderMode: provider.ModeDelegated}})
	if err != nil {
		t.Fatal(err)
	}
	factory.WithTaskBusProvider(func() (Emitter, error) { return selected, nil })
	if _, err := factory.NewEmitter(); !errors.Is(err, ErrReliableHostRequired) || !selected.closed {
		t.Fatal("REJECTED_TRANSPORT_NOT_CLOSED", err)
	}
}
