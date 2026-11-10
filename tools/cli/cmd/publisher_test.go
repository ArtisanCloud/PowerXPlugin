//go:build darwin || linux

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/powerx-plugin/cli/internal/publisher"
)

func TestPublisherCLIPlanRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte("id: com.example.cli-publisher\nversion: 0.1.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := executePublisher(context.Background(), []string{"plan", "create", "--entry", root, "--version", "1.2.0"}, &out, "zh"); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data publisher.Plan `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := executePublisher(context.Background(), []string{"plan", "get", "--entry", root, "--id", envelope.Data.ReleasePlanID}, &out, "en"); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Data publisher.Plan `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Data != envelope.Data {
		t.Fatal("plan_roundtrip_mismatch")
	}
	out.Reset()
	if err := executePublisher(context.Background(), []string{"plan", "list", "--entry", root}, &out, "zh"); err != nil {
		t.Fatal(err)
	}
	var page struct {
		Data publisher.PlanPage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Data.Total != 1 {
		t.Fatal("plan_list_mismatch")
	}
}
func TestPublisherCLIRejectsUnknownCommandsFlagsAndInvalidPlans(t *testing.T) {
	for _, args := range [][]string{
		{"upload"}, {"plan", "create", "--version", "v1.2.0"}, {"plan", "create", "--version", "1.2.0", "--command", "id"}, {"plan", "create", "--version", "1.2.0", "unexpected"},
	} {
		var out bytes.Buffer
		err := executePublisher(context.Background(), args, &out, "zh")
		var typed *publisher.Error
		if !errors.As(err, &typed) || typed.Code != publisher.InvalidInput {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
}
func TestPublisherCLIHelpUsesLocaleCatalog(t *testing.T) {
	for _, locale := range []string{"en", "zh"} {
		var out bytes.Buffer
		if err := executePublisher(context.Background(), []string{"--help"}, &out, locale); err != nil {
			t.Fatal(err)
		}
		if out.String() != publisher.Text(locale, "usage") {
			t.Fatal("help_locale_mismatch")
		}
	}
}
