//go:build darwin || linux

package publisher

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixtureService(t *testing.T) (string, *Engine, Plan, *Server, Session) {
	t.Helper()
	root, e, p := buildFixture(t)
	server, err := Start(context.Background(), e, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	data, err := e.state.read("tool-session.json", MaxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	var boot Bootstrap
	if json.Unmarshal(data, &boot) != nil {
		t.Fatal("bootstrap_invalid")
	}
	return root, e, p, server, exchange(t, server, boot)
}
func TestBuildHTTPSharedEngineAndStrictRequest(t *testing.T) {
	_, e, p, server, session := fixtureService(t)
	bad := request(t, server.Info().Endpoint, http.MethodPost, "/v1/builds", session.Token, `{"release_plan_id":"`+p.ReleasePlanID+`","command":"id"}`)
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatal("arbitrary_command_accepted")
	}
	b := decodeData[Build](t, request(t, server.Info().Endpoint, http.MethodPost, "/v1/builds", session.Token, `{"release_plan_id":"`+p.ReleasePlanID+`"}`), http.StatusAccepted)
	if b.Stage != "queued" {
		t.Fatal("request_not_asynchronous")
	}
	ready, err := e.WaitBuild(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	requireReady(t, e, ready)
	got := decodeData[Build](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/builds/"+b.BuildID, session.Token, ""), http.StatusOK)
	if got.Artifact.SHA256 != ready.Artifact.SHA256 {
		t.Fatal("http_engine_mismatch")
	}
	a := decodeData[Artifact](t, request(t, server.Info().Endpoint, http.MethodPost, "/v1/builds/"+b.BuildID+"/verify", session.Token, ""), http.StatusOK)
	if a.SHA256 != ready.Artifact.SHA256 {
		t.Fatal("http_digest_mismatch")
	}
	page := decodeData[BuildPage](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/builds?page=1&size=20", session.Token, ""), http.StatusOK)
	if page.Total != 1 {
		t.Fatal("http_build_pagination")
	}
	if ready.Process == nil || ready.Process.ExitedAt == nil || ready.Process.ParentPID != os.Getpid() {
		t.Fatal("process_provenance_missing")
	}
	t.Logf("build_id=%s artifact_sha256=%s parent_pid=%d group_id=%d", b.BuildID, a.SHA256, ready.Process.ParentPID, ready.Process.GroupID)
}
func TestBuildServerShutdownCancelsOwnedChildProcesses(t *testing.T) {
	root, e, p, server, session := fixtureService(t)
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("dist:\n\tsleep 30 & wait\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b := decodeData[Build](t, request(t, server.Info().Endpoint, http.MethodPost, "/v1/builds", session.Token, `{"release_plan_id":"`+p.ReleasePlanID+`"}`), http.StatusAccepted)
	deadline := time.Now().Add(10 * time.Second)
	pid := 0
	for time.Now().Before(deadline) {
		state, err := e.GetBuild(b.BuildID)
		if err != nil {
			t.Fatal(err)
		}
		if state.PID != 0 {
			pid = state.PID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("build_process_not_started")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := e.GetBuild(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != "cancelled" || got.PID != 0 || processGroupAlive(pid) {
		t.Fatalf("shutdown_left_build=%+v", got)
	}
}
