package iamcontext

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func TestModeResolver_ConfigPriority(t *testing.T) {
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
	if !iamerrors.IsCode(err, iamerrors.CodeModeConflict) {
		t.Fatalf("expected code %s, got %s", iamerrors.CodeModeConflict, iamerrors.CodeOf(err))
	}
	if got := iamerrors.StatusCode(err); got != 409 {
		t.Fatalf("expected status 409, got %d", got)
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
	if mode != contracts.IAMModeDelegated {
		t.Fatalf("expected delegated mode, got %q", mode)
	}
	if record.Audit.Source != "config" {
		t.Fatalf("expected source config, got %q", record.Audit.Source)
	}
}

func TestModeResolver_ResolveFromPowerXProxy(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{
		PowerXProxy: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != contracts.IAMModeDelegated {
		t.Fatalf("expected delegated mode, got %q", mode)
	}
	if record.Audit.Source != "env:POWERX_PROXY" {
		t.Fatalf("expected source env:POWERX_PROXY, got %q", record.Audit.Source)
	}
}

func TestModeResolver_DefaultLocal(t *testing.T) {
	resolver := ModeResolver{}

	mode, record, err := resolver.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != contracts.IAMModeLocal {
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
	if !iamerrors.IsCode(err, iamerrors.CodeModeInvalid) {
		t.Fatalf("expected code %s, got %s", iamerrors.CodeModeInvalid, iamerrors.CodeOf(err))
	}
	if got := iamerrors.StatusCode(err); got != 400 {
		t.Fatalf("expected status 400, got %d", got)
	}
}
