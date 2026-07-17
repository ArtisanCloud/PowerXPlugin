package customerfw

import (
	"context"
	"testing"
)

func TestBootstrapResolverNormalizesTenant(t *testing.T) {
	resolver := BootstrapResolver(func(_ context.Context, input BootstrapInput) (*BootstrapContext, error) {
		if input.Scene != "scene-a" {
			t.Fatalf("unexpected scene %q", input.Scene)
		}
		return &BootstrapContext{TenantUUID: " TENANT-A ", EntryType: "scene"}, nil
	})
	out, err := resolver.ResolveEntry(context.Background(), BootstrapInput{Scene: "scene-a"})
	if err != nil {
		t.Fatalf("resolve entry: %v", err)
	}
	out = NormalizeBootstrapContext(out)
	if out.TenantUUID != "tenant-a" {
		t.Fatalf("expected normalized tenant, got %#v", out)
	}
}
