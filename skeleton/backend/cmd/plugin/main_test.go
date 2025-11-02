package main

import (
	"testing"
)

func TestSetupApp(t *testing.T) {
	app, err := setupApp()
	if err != nil {
		t.Fatalf("setupApp returned error: %v", err)
	}
	if app == nil {
		t.Fatalf("setupApp returned nil app")
	}
	if app.Router == nil {
		t.Fatalf("expected router to be attached")
	}
	manifest := app.Manifest()
	if manifest == nil {
		t.Fatalf("expected manifest to be registered")
	}
	if manifest.ID != "com.powerx.sample" {
		t.Fatalf("unexpected manifest id: %s", manifest.ID)
	}
}
