package main

import (
	"os"
	"strings"
	"testing"
)

// Guard the executable's assembly order: handlers snapshot IAM dependencies.
func TestIAMBindingPrecedesTransportConstruction(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	bind := strings.Index(source, "pluginbootstrap.ResolveAndBindFrameworkIAM(")
	validate := strings.Index(source, "pluginbootstrap.ValidateIAMBinding(")
	for _, transport := range []string{"pluginrouter.NewRouter(", "grpcserver.NewGRPCServer("} {
		at := strings.Index(source, transport)
		if bind < 0 || validate < bind || at < validate {
			t.Fatalf("IAM assembly must precede %s", transport)
		}
	}
}
