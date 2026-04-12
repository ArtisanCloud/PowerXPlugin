package iam

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestSeedLocalAdminSkipsInDelegatedMode(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	if err := SeedLocalAdmin(context.Background(), nil, &config.Config{}, IAMModeLocal); err != nil {
		t.Fatalf("expected skip without error, got %v", err)
	}
}

func TestLoadSeedOptionsDefaults(t *testing.T) {
	t.Setenv("PLUGIN_IAM_TENANT_KEY", "")
	t.Setenv("PLUGIN_IAM_TENANT_NAME", "")
	t.Setenv("PLUGIN_IAM_ADMIN_EMAIL", "")
	t.Setenv("PLUGIN_IAM_ADMIN_PASSWORD", "")
	t.Setenv("PLUGIN_IAM_ADMIN_NAME", "")

	opts, _ := loadSeedOptionsFromEnv()
	if opts.TenantKey != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("unexpected tenant key %s", opts.TenantKey)
	}
	if opts.TenantName != "Local Tenant" {
		t.Fatalf("unexpected tenant name %s", opts.TenantName)
	}
	if opts.AdminEmail != "admin@local.test" {
		t.Fatalf("unexpected admin email %s", opts.AdminEmail)
	}
	if opts.AdminName != "admin" {
		t.Fatalf("unexpected admin name %s", opts.AdminName)
	}
}

func TestSeedOptionsValidate(t *testing.T) {
	opts := SeedOptions{AdminPwd: "12345"}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected validation error for weak password")
	}
	opts.AdminPwd = "123456"
	if err := opts.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
