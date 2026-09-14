package runtimeexample

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const tenant = "11111111-1111-1111-1111-111111111111"
const other = "22222222-2222-2222-2222-222222222222"

func TestLocalCache(t *testing.T) {
	raw := NewMemoryCache()
	now := time.Now()
	raw.now = func() time.Time { return now }
	r, _ := cache.NewRuntime(provider.ModeLocal, raw, nil)
	s, _ := r.Cache()
	ctx := context.Background()
	scope := cache.Scope{TenantUUID: tenant, Namespace: "test"}
	if err := s.Set(ctx, scope, cache.SetInput{Key: "empty", TTL: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get(ctx, scope, cache.GetInput{Key: "empty"})
	if err != nil || !v.Found || len(v.Value) != 0 {
		t.Fatalf("%+v %v", v, err)
	}
	v, err = s.Get(ctx, cache.Scope{TenantUUID: other, Namespace: "test"}, cache.GetInput{Key: "empty"})
	if err != nil || v.Found {
		t.Fatal(v, err)
	}
	now = now.Add(time.Millisecond)
	v, err = s.Get(ctx, scope, cache.GetInput{Key: "empty"})
	if err != nil || v.Found {
		t.Fatal(v, err)
	}
	if err = s.Delete(ctx, scope, cache.DeleteInput{Key: "empty"}); err != nil {
		t.Fatal(err)
	}
}
func TestTaskIsolationIdempotencyCAS(t *testing.T) {
	_, r, err := Build(provider.ModeLocal, hostapi.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.Tasks()
	ctx := context.Background()
	scope := taskcenter.Scope{TenantUUID: tenant}
	input := taskcenter.CreateInput{Type: "test", IdempotencyKey: "one", Payload: json.RawMessage(`{"n":9007199254740993}`)}
	first, err := s.Create(ctx, scope, input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Create(ctx, scope, input)
	if err != nil || again.TaskUUID != first.TaskUUID {
		t.Fatal(again, err)
	}
	input.Payload = json.RawMessage(`{"n":9007199254740992}`)
	if _, err = s.Create(ctx, scope, input); !errors.Is(err, taskcenter.ErrConflict) {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, taskcenter.Scope{TenantUUID: other}, taskcenter.GetInput{TaskUUID: first.TaskUUID}); !errors.Is(err, taskcenter.ErrNotFound) {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Update(ctx, scope, taskcenter.UpdateInput{TaskUUID: first.TaskUUID, ExpectedRevision: 1, State: taskcenter.Running, Progress: 10})
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, taskcenter.ErrConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal(successes.Load())
	}
	done, err := s.Update(ctx, scope, taskcenter.UpdateInput{TaskUUID: first.TaskUUID, ExpectedRevision: 2, State: taskcenter.Succeeded, Progress: 100})
	if err != nil || done.CompletedAt == nil {
		t.Fatal(done, err)
	}
	if _, err = s.Update(ctx, scope, taskcenter.UpdateInput{TaskUUID: first.TaskUUID, ExpectedRevision: 3, State: taskcenter.Running, Progress: 100}); !errors.Is(err, taskcenter.ErrConflict) {
		t.Fatal(err)
	}
}
func TestDelegatedRevocationNoFallback(t *testing.T) {
	revoked := false
	calls := 0
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer service" || r.Header.Get("X-Tenant-UUID") != "" {
			t.Error("credential boundary")
		}
		if revoked {
			w.WriteHeader(403)
			w.Write([]byte(`{"reason_code":"CACHE_FORBIDDEN"}`))
			return
		}
		w.Write([]byte(`{"code":200,"data":{"found":false,"value_base64":"","expires_at":null}}`))
	}))
	defer host.Close()
	tokens := hostapi.TokenProviderFunc(func(context.Context) (hostapi.Credential, error) {
		return hostapi.Credential{Token: "service", TenantUUID: tenant}, nil
	})
	r, _, err := Build(provider.ModeDelegated, hostapi.Config{BaseURL: host.URL}, tokens, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.Cache()
	scope := cache.Scope{TenantUUID: tenant, Namespace: "test"}
	if _, err = s.Get(context.Background(), scope, cache.GetInput{Key: "k"}); err != nil {
		t.Fatal(err)
	}
	revoked = true
	if _, err = s.Get(context.Background(), scope, cache.GetInput{Key: "k"}); err == nil {
		t.Fatal("revocation swallowed")
	}
	scope.TenantUUID = other
	if _, err = s.Get(context.Background(), scope, cache.GetInput{Key: "k"}); err == nil || calls != 2 {
		t.Fatal(err, calls)
	}
	if _, _, err = Build(provider.ModeDelegated, hostapi.Config{BaseURL: host.URL}, nil, nil); err == nil {
		t.Fatal("missing credential accepted")
	}
	if _, _, err = Build(provider.Mode("bad"), hostapi.Config{}, nil, nil); err == nil {
		t.Fatal("invalid mode")
	}
}
