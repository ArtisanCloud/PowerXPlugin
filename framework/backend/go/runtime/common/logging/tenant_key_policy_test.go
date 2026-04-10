package logging

import "testing"

func TestTenantUUIDPrimaryAndTenantKeyMirror(t *testing.T) {
	fields := NormalizeRuntimeFields(Fields{
		FieldTenantUUID: " tenant-a ",
		FieldTenantKey:  "legacy-tenant-key",
	})

	if got, want := fields[FieldTenantUUID], "tenant-a"; got != want {
		t.Fatalf("%s = %v, want %q", FieldTenantUUID, got, want)
	}
	if got, want := fields[FieldTenantKey], "tenant-a"; got != want {
		t.Fatalf("%s = %v, want %q", FieldTenantKey, got, want)
	}
}

func TestMissingTenantUUIDFallsBackUnknownAndMirror(t *testing.T) {
	fields := NormalizeRuntimeFields(Fields{
		FieldTenantKey: "legacy-key-only",
	})

	if got, want := fields[FieldTenantUUID], FallbackUnknown; got != want {
		t.Fatalf("%s = %v, want %q", FieldTenantUUID, got, want)
	}
	if got, want := fields[FieldTenantKey], FallbackUnknown; got != want {
		t.Fatalf("%s = %v, want %q", FieldTenantKey, got, want)
	}
}
