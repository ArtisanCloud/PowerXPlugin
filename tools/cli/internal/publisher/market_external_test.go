//go:build darwin || linux

package publisher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Opt-in, secret seed arrives on an inherited pipe, never an env value, argv,
// workspace file or test output. The store is an explicit in-memory fixture.
func TestMarketExternalFixture(t *testing.T) {
	if os.Getenv("PX_PUBLISHER_MARKET_FIXTURE_FD") != "3" {
		t.Skip("PX_PUBLISHER_MARKET_FIXTURE_FD=3")
	}
	pipe := os.NewFile(3, "market-fixture")
	defer pipe.Close()
	var seed struct {
		APIRoot string `json:"api_root"`
		Token   string `json:"token"`
		Enabled bool   `json:"enabled"`
	}
	if json.NewDecoder(pipe).Decode(&seed) != nil {
		t.Fatal("fixture_seed")
	}
	_, e, p := buildFixture(t)
	e.credentials = &memoryCredentials{items: map[string]string{}}
	m, err := e.CreateMarket(MarketInput{Name: "joint-fixture", APIRoot: seed.APIRoot, ConsoleBase: "https://market.example"})
	if err != nil {
		t.Fatal(err)
	}
	m, err = e.BindMarket(context.Background(), m.ConnectionID, seed.Token)
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.StartBuild(context.Background(), p.ReleasePlanID)
	if err != nil {
		t.Fatal(err)
	}
	b, err = e.WaitBuild(b.BuildID)
	if err != nil || b.Stage != "ready" {
		t.Fatal("fixture_build", err)
	}
	u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
	if err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if !seed.Enabled {
		if u.Stage != "pending" || u.Error == nil || u.Error.Code != MarketCapability || u.Attempts != 0 {
			t.Fatal("discovery_gate", u.Stage, u.Error, u.Attempts)
		}
		t.Log("real_market_discovery_gate_passed")
		return
	}
	if u.Stage != "succeeded" || u.Receipt == nil || u.CurrentVersion == nil {
		t.Fatal("joint_upload", u.Stage, u.Error)
	}
	if _, err = e.MarketVersions(context.Background(), m.ConnectionID, 1, 20); err != nil {
		t.Fatal("joint_versions", err)
	}
	if err = os.Remove(filepath.Join(e.work.path, b.BuildID, "artifact.zip")); err != nil {
		t.Fatal(err)
	}
	if _, err = e.ResumeUpload(context.Background(), u.UploadID); err != nil {
		t.Fatal(err)
	}
	recovered, err := e.WaitUpload(u.UploadID)
	if err != nil || recovered.Stage != "succeeded" || recovered.Attempts != 1 || recovered.OperationID != u.OperationID || recovered.Receipt.VersionID != u.Receipt.VersionID {
		t.Fatal("joint_recovery", err, recovered.Stage, recovered.Error)
	}
	t.Logf("joint_upload_and_recovery_passed artifact_sha256=%s size=%d version_status=%s attempts=%d", u.Artifact.SHA256, u.Artifact.Size, u.CurrentVersion.Status, recovered.Attempts)
}
