package eventbridge

import (
	"testing"
)

func TestFactoryDelegatedProviderWithoutProxyDoesNotForceTaskBus(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "0")

	factory, err := NewFactory(Config{
		Enabled:         false,
		Mode:            "local",
		FallbackToLocal: true,
		LocalQueueSize:  8,
	})
	if err != nil {
		t.Fatalf("NewFactory() error = %v", err)
	}
	emitter, err := factory.NewEmitter()
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}
	if emitter == nil {
		t.Fatal("expected local emitter")
	}
}

func TestFactoryProxyModeRequiresTaskBus(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "local")
	t.Setenv("POWERX_PROXY", "1")

	factory, err := NewFactory(Config{
		Enabled:         false,
		Mode:            "local",
		FallbackToLocal: true,
		LocalQueueSize:  8,
	})
	if err != nil {
		t.Fatalf("NewFactory() error = %v", err)
	}
	if _, err := factory.NewEmitter(); err != ErrTaskBusRequired {
		t.Fatalf("error=%v want=%v", err, ErrTaskBusRequired)
	}
}
