package provider

import "testing"

func TestModeResolver_ConfigConflict(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{
		ConfigMode:  "local",
		EnvMode:     "delegated",
		Environment: "test",
	})
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	if mode != "" {
		t.Fatalf("expected empty mode on conflict, got %q", mode)
	}
	if !record.ConflictDetected {
		t.Fatalf("expected conflict flag to be true")
	}
	if !IsConflict(err) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestModeResolver_ResolveFromConfigWhenNoConflict(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{
		ConfigMode:  "delegated",
		EnvMode:     "delegated",
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ModeDelegated {
		t.Fatalf("expected delegated mode, got %q", mode)
	}
	if record.Audit.Source != "config" {
		t.Fatalf("expected source config, got %q", record.Audit.Source)
	}
}

func TestModeResolver_DoesNotResolveFromPowerXProxy(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{
		PowerXProxy: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ModeLocal {
		t.Fatalf("expected local mode, got %q", mode)
	}
	if record.Audit.Source != "default" {
		t.Fatalf("expected source default, got %q", record.Audit.Source)
	}
}

func TestModeResolver_DefaultLocal(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ModeLocal {
		t.Fatalf("expected local mode, got %q", mode)
	}
	if record.Audit.Source != "default" {
		t.Fatalf("expected source default, got %q", record.Audit.Source)
	}
}

func TestModeResolver_InvalidMode(t *testing.T) {
	resolver := ModeResolver{}

	mode, _, err := resolver.Resolve(ResolveInput{
		ConfigMode: "invalid",
	})
	if err == nil {
		t.Fatalf("expected invalid mode error")
	}
	if mode != "" {
		t.Fatalf("expected empty mode, got %q", mode)
	}
	if !IsInvalid(err) {
		t.Fatalf("expected invalid error, got %v", err)
	}
}
