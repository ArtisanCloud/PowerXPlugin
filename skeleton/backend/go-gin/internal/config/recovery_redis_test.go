package config

import (
	"net/url"
	"testing"
)

func TestRecoveryRedisPreservesDatabaseAndRejectsOtherInstance(t *testing.T) {
	got, err := recoveryRedisURL("redis://old:credential@localhost:6379/7?pool_size=3", "127.0.0.1:6379", "127.0.0.1:16222", "private:@/value")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := u.User.Password()
	if u.Host != "127.0.0.1:16222" || u.Path != "/7" || u.Query().Get("pool_size") != "3" || password != "private:@/value" {
		t.Fatal("Redis binding lost endpoint, database or escaped credentials")
	}
	for _, raw := range []string{"redis://localhost:6380/1", "redis://external.invalid:6379/1", "rediss://localhost:6379/1"} {
		if _, err := recoveryRedisURL(raw, "localhost:6379", "127.0.0.1:16222", ""); err == nil {
			t.Fatal("uncaptured Redis instance accepted")
		}
	}
}

func TestRecoveryRedisRequiresExplicitHostContract(t *testing.T) {
	t.Setenv("POWERX_RECOVERY_READ_ONLY", "true")
	t.Setenv("POWERX_RECOVERY_REDIS_SOURCE_ADDR", "localhost:6379")
	t.Setenv("POWERX_REDIS_ADDR", "127.0.0.1:16222")
	t.Setenv("POWERX_REDIS_PASSWORD", "")
	c := getDefaultConfig()
	c.WSBus = &WSBusConfig{Provider: "redis", RedisURL: "redis://localhost:6379/4"}
	if err := applyRecoveryRedisBindings(c); err != nil {
		t.Fatal(err)
	}
	if c.WSBus.RedisURL != "redis://127.0.0.1:16222/4" {
		t.Fatal("WS Redis not rebound")
	}
	t.Setenv("POWERX_RECOVERY_REDIS_SOURCE_ADDR", "")
	if err := applyRecoveryRedisBindings(c); err == nil {
		t.Fatal("missing frozen source binding accepted")
	}
}

func TestRecoveryEventRedisFollowsProviderInsteadOfProxy(t *testing.T) {
	for _, mode := range []string{"local", "delegated"} {
		for _, proxy := range []string{"0", "1"} {
			t.Run(mode+proxy, func(t *testing.T) {
				t.Setenv("POWERX_RECOVERY_READ_ONLY", "true")
				t.Setenv("POWERX_RECOVERY_REDIS_SOURCE_ADDR", "localhost:6379")
				t.Setenv("POWERX_REDIS_ADDR", "127.0.0.1:16222")
				t.Setenv("POWERX_REDIS_PASSWORD", "")
				t.Setenv("POWERX_PROVIDER_MODE", mode)
				t.Setenv("POWERX_PROXY", proxy)
				c := &Config{Context: &ContextConfig{ProviderMode: mode}, EventBridge: &EventBridgeConfig{Enabled: true, RedisURL: "redis://localhost:6379/7"}}
				if err := applyRecoveryRedisBindings(c); err != nil {
					t.Fatal(err)
				}
				want := "redis://localhost:6379/7"
				if mode == "local" {
					want = "redis://127.0.0.1:16222/7"
				}
				if c.EventBridge.RedisURL != want {
					t.Fatal("RECOVERY_EVENT_REDIS_MODE_MISMATCH")
				}
			})
		}
	}
}
