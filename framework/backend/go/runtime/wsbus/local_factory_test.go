package wsbus

import (
	"context"
	"testing"
)

func TestOpenLocalHubMemory(t *testing.T) {
	hub, closeHub, err := OpenLocalHub(t.Context(), "memory", RedisHubConfig{})
	if err != nil || hub == nil || closeHub == nil {
		t.Fatalf("hub=%T close=%v err=%v", hub, closeHub == nil, err)
	}
	if _, ok := hub.(*MemoryHub); !ok {
		t.Fatalf("hub=%T", hub)
	}
	if err := closeHub(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenLocalHubFailsWithoutFallback(t *testing.T) {
	for _, provider := range []string{"redis", "invalid"} {
		hub, closeHub, err := OpenLocalHub(t.Context(), provider, RedisHubConfig{})
		if err == nil || hub != nil || closeHub != nil {
			t.Fatalf("provider=%s hub=%T err=%v", provider, hub, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	hub, closeHub, err := OpenLocalHub(ctx, "redis", RedisHubConfig{RedisURL: "redis://127.0.0.1:1"})
	if err == nil || hub != nil || closeHub != nil {
		t.Fatalf("cancelled subscription hub=%T err=%v", hub, err)
	}
}
