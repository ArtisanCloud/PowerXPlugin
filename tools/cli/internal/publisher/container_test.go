//go:build darwin || linux

package publisher

import (
	"context"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContainerSnapshotExcludesHostDependenciesAndIDEState(t *testing.T) {
	root := workspace(t)
	for _, name := range []string{"web-admin/node_modules/native/binding.node", "web-admin/.idea/workspace.xml", "web-admin/.env", "web-admin/index.ts"} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(t.TempDir(), "snapshot")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	hash, err := cleanSourceDigest(context.Background(), root, destination)
	if err != nil {
		t.Fatal(err)
	}
	copyHash, err := cleanSourceDigest(context.Background(), destination, "")
	if err != nil || hash != copyHash {
		t.Fatal("source_digest_mismatch", err)
	}
	for _, name := range []string{"web-admin/node_modules", "web-admin/.idea", "web-admin/.env"} {
		if _, err := os.Stat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatal("private_host_state_copied", name)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "web-admin/index.ts")); err != nil {
		t.Fatal(err)
	}
}

func TestContainerSnapshotRejectsUnapprovedExternalContracts(t *testing.T) {
	root := workspace(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "contracts")); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanSourceDigest(context.Background(), root, ""); err == nil || AsError(err).Code != SourceUnsafe {
		t.Fatal("external_contracts_accepted", err)
	}
}

func TestContainerSDKCopyDetectsSubsequentSourceChanges(t *testing.T) {
	source := t.TempDir()
	file := filepath.Join(source, "index.ts")
	if err := os.WriteFile(file, []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}
	sdk, err := copySDK(context.Background(), source, filepath.Join(t.TempDir(), "sdk"))
	if err != nil {
		t.Fatal(err)
	}
	if err = verifySDK(context.Background(), sdk); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, []byte("export const changed=true"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = verifySDK(context.Background(), sdk); err == nil || AsError(err).Code != SourceChanged {
		t.Fatal("sdk_source_mutation_accepted", err)
	}
}

func TestContainerExecutorRejectsUnknownRunnerWithoutStartingBuild(t *testing.T) {
	e := openTest(t, workspace(t))
	plan, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.StartBuildWithExecutor(context.Background(), plan.ReleasePlanID, "shell"); err == nil || AsError(err).Code != InvalidInput {
		t.Fatal("unknown_executor_accepted", err)
	}
	builds, err := e.ListBuilds(1, 20)
	if err != nil || builds.Total != 0 {
		t.Fatal("unknown_executor_created_build", err)
	}
}

func TestFailedContainerBuildCanBeReadAndRecoveredAfterRestart(t *testing.T) {
	root := workspace(t)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	now := time.Now().UTC()
	b := Build{Schema: BuildSchema, BuildID: id, ReleasePlanID: p.ReleasePlanID, PluginID: p.PluginID, Version: p.Version, Target: p.Target, Executor: "container", Stage: "failed", Error: failure(ContainerBuildFailed, nil), CreatedAt: now, UpdatedAt: now, Process: &BuildProcess{ParentPID: os.Getpid(), LeaderPID: 42, GroupID: 42, WorkingDirectory: "work/" + id + "/run", StartedAt: now, ExitedAt: &now}}
	if err = e.saveBuild(&b, false); err != nil {
		t.Fatal(err)
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(root)
	if err != nil {
		t.Fatal("container_history_blocks_executor_startup", err)
	}
	defer restored.Close()
	got, err := restored.GetBuild(id)
	if err != nil || got.Process.WorkingDirectory != b.Process.WorkingDirectory {
		t.Fatal("container_history_not_retained", err)
	}
}
