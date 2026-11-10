//go:build darwin || linux

package publisher

import (
	"archive/zip"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This is a controlled pure-JS build fixture, not a Linux/Nuxt runtime test.
// Production has no flag or environment override for executor admission.
func buildFixture(t *testing.T) (string, *Engine, Plan) {
	t.Helper()
	root := workspace(t)
	files := map[string]string{
		"backend/go.mod":                         "module example.com/publisher-fixture/backend\n\ngo 1.24.0\n",
		"backend/internal/shared/app/version.go": "package app\nvar PluginVersion = \"0.1.0\"\n",
		"backend/cmd/plugin/main.go":             "package main\nimport \"example.com/publisher-fixture/backend/internal/shared/app\"\nfunc main(){println(app.PluginVersion)}\n",
		"web-admin/package.json":                 "{\"scripts\":{\"build\":\"node ../scripts/dist.mjs\"}}",
		"plugin.d/capabilities.yaml":             "capabilities:\n  imports: [contracts/capability.yaml]\n",
		"plugin.d/rbac.yaml":                     "permissions: []\n",
		"contracts/capability.yaml":              "provides:\n  - path: schema/input.json\n",
		"contracts/schema/input.json":            "{\"type\":\"object\"}\n",
		"Makefile":                               "dist:\n\tmkdir -p $(BUILD_DIR) $(DIST_DIR)/backend/bin\n\tcd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags \"-X example.com/publisher-fixture/backend/internal/shared/app.PluginVersion=$(VERSION)\" -o $(BUILD_DIR)/plugin ./cmd/plugin\n\tcp $(BUILD_DIR)/plugin $(DIST_DIR)/backend/bin/plugin\n\tnode scripts/dist.mjs $(VERSION) $(DIST_DIR)\n",
		"scripts/dist.mjs": `import fs from 'node:fs'; import path from 'node:path';
const [version,dist]=process.argv.slice(2);
for(const name of ['web-admin/.output/server','web-admin/.output/public'])fs.mkdirSync(path.join(dist,name),{recursive:true});
fs.writeFileSync(path.join(dist,'plugin.yaml'),'id: com.example.publisher\nversion: '+version+'\ntarget: {os: linux, arch: amd64}\nruntime: {kind: process, entry: backend/bin/plugin}\ncatalogs: {capabilities: plugin.d/capabilities.yaml}\nfrontend: {admin: {kind: process, process: {entry: node, args: ["./web-admin/.output/server/index.mjs"]}}}\n');
fs.writeFileSync(path.join(dist,'web-admin/.output/server/index.mjs'),'export default {}');
fs.writeFileSync(path.join(dist,'web-admin/.output/public/index.html'),'<div></div>');
console.log('TOKEN=fixture-sensitive-value');
`,
	}
	for rel, data := range files {
		file := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	e := openTest(t, root)
	e.host = Target{OS: "linux", Arch: "amd64"}
	e.toolProbe = func() (BuildTools, error) {
		return BuildTools{Go: "go1.26.7", Node: "v22.13.0", NodeOS: "linux", NodeArch: "x64"}, nil
	}
	p, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	return root, e, p
}
func runFixture(t *testing.T, e *Engine, p Plan) Build {
	t.Helper()
	b, err := e.StartBuild(context.Background(), p.ReleasePlanID)
	if err != nil {
		t.Fatal(err)
	}
	b, err = e.WaitBuild(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func requireReady(t *testing.T, e *Engine, b Build) {
	t.Helper()
	if b.Stage != "ready" {
		log, _ := e.BuildLog(b.BuildID)
		t.Fatalf("stage=%s error=%v log=%s", b.Stage, b.Error, log)
	}
}
func TestBuildIsolationFreezingIntegrityAndRedaction(t *testing.T) {
	root, e, p := buildFixture(t)
	before, err := sourceDigest(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(root, "backend/bin"), 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "backend/bin/plugin")
	if err = os.WriteFile(sentinel, []byte("user-owned-output"), 0600); err != nil {
		t.Fatal(err)
	}
	b := runFixture(t, e, p)
	requireReady(t, e, b)
	if !canonicalUUID(b.BuildID) || b.Source == nil || b.Source.SHA256 != before || b.Source.VCS != "filesystem" || b.Source.Dirty != nil || b.Tools == nil || b.Artifact == nil {
		t.Fatalf("build=%+v", b)
	}
	a, err := e.VerifyArtifact(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	beforeFile, err := os.Stat(filepath.Join(e.state.path, a.Path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.VerifyArtifact(b.BuildID); err != nil {
		t.Fatal(err)
	}
	afterFile, err := os.Stat(filepath.Join(e.state.path, a.Path))
	if err != nil || !os.SameFile(beforeFile, afterFile) || !beforeFile.ModTime().Equal(afterFile.ModTime()) {
		t.Fatal("verification_rebuilt_artifact")
	}
	if b.Artifact.SHA256 != a.SHA256 || !validSHA(a.SHA256) {
		t.Fatal("artifact_digest_mismatch")
	}
	data, _ := os.ReadFile(sentinel)
	if string(data) != "user-owned-output" {
		t.Fatal("user_output_changed")
	}
	after, _ := sourceDigest(context.Background(), root, "")
	if before != after {
		t.Fatal("source_changed")
	}
	r, err := zip.OpenReader(filepath.Join(e.state.path, a.Path))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	seen := map[string]bool{}
	for _, f := range r.File {
		if seen[f.Name] || strings.Contains(f.Name, ".env") || strings.Contains(f.Name, ".px-plugin") || strings.Contains(f.Name, "scripts/") {
			t.Fatal("unsafe_package_entry")
		}
		seen[f.Name] = true
		if f.Name == p.Version+"/backend/bin/plugin" && f.Mode().Perm()&0111 == 0 {
			t.Fatal("execution_permission_lost")
		}
	}
	if !seen[p.Version+"/contracts/schema/input.json"] || !seen[p.Version+"/web-admin/.output/powerx-publisher.json"] {
		t.Fatal("missing_runtime_resource")
	}
	log, err := e.BuildLog(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(log, "fixture-sensitive-value") || strings.Contains(log, root) {
		t.Fatal("private_diagnostic_leaked")
	}
	file := filepath.Join(e.state.path, a.Path)
	original, _ := os.ReadFile(file)
	if err = os.Chmod(file, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, append(original, 'x'), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = e.VerifyArtifact(b.BuildID)
	assertCode(t, err, ArtifactChanged)
	if err = os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(root, "plugin.yaml"), file); err != nil {
		t.Fatal(err)
	}
	_, err = e.VerifyArtifact(b.BuildID)
	assertCode(t, err, ArtifactChanged)
}
func TestBuildAdmissionAndNoLegacyDependencyFallback(t *testing.T) {
	root, e, p := buildFixture(t)
	e.host = Target{OS: "darwin", Arch: "arm64"}
	_, err := e.StartBuild(context.Background(), p.ReleasePlanID)
	assertCode(t, err, BuildPlatform)
	page, err := e.ListBuilds(1, 20)
	if err != nil || page.Total != 0 {
		t.Fatal("unsupported_task_created")
	}
	e.host = Target{OS: "linux", Arch: "amd64"}
	if err = os.WriteFile(filepath.Join(root, "backend/go.mod"), []byte("module example.com/publisher-fixture/backend\ngo 1.24.0\nreplace example.com/other => ../../outside\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b := runFixture(t, e, p)
	if b.Stage != "failed" || b.Error.Code != SourceUnsafe || b.Artifact != nil {
		t.Fatalf("local_replace_accepted=%+v", b)
	}
}
func TestBuildRejectsNativeRuntimeAndMissingReferences(t *testing.T) {
	for _, kind := range []string{"native", "missing_ref", "version", "linked_version", "stripped", "secret"} {
		t.Run(kind, func(t *testing.T) {
			root, e, p := buildFixture(t)
			file := filepath.Join(root, "scripts/dist.mjs")
			data, _ := os.ReadFile(file)
			switch kind {
			case "linked_version", "stripped":
				makefile := filepath.Join(root, "Makefile")
				content, _ := os.ReadFile(makefile)
				if kind == "linked_version" {
					content = []byte(strings.ReplaceAll(string(content), "PluginVersion=$(VERSION)", "PluginVersion=0.1.0"))
				} else {
					content = []byte(strings.ReplaceAll(string(content), "-ldflags \"-X", "-ldflags \"-s -w -X"))
				}
				if err := os.WriteFile(makefile, content, 0600); err != nil {
					t.Fatal(err)
				}
			case "secret":
				file := filepath.Join(root, "config/event_fabric.yaml")
				if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte("secret: fixture-sensitive-value\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "native":
				data = append(data, []byte("\nfs.writeFileSync(path.join(dist,'web-admin/.output/server/addon.node'),'native-fixture');\n")...)
			case "missing_ref":
				if err := os.Remove(filepath.Join(root, "contracts/schema/input.json")); err != nil {
					t.Fatal(err)
				}
			case "version":
				data = []byte(strings.Replace(string(data), "'id: com.example.publisher\\nversion: '+version+", "'id: com.example.publisher\\nversion: '+'9.0.0'+", 1))
			}
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			b := runFixture(t, e, p)
			if b.Stage != "failed" || b.Artifact != nil {
				t.Fatalf("invalid_ready=%+v", b)
			}
			want := ArtifactInvalid
			if kind == "native" {
				want = NativeUnsupported
			}
			if b.Error.Code != want {
				log, _ := e.BuildLog(b.BuildID)
				t.Fatalf("code=%s want=%s log=%s", b.Error.Code, want, log)
			}
		})
	}
}
func TestBuildCancellationOwnProcessGroupAndWorkspaceLock(t *testing.T) {
	root, e, p := buildFixture(t)
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("dist:\n\tsleep 30 & wait\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := e.StartBuild(context.Background(), p.ReleasePlanID)
	if err != nil {
		t.Fatal(err)
	}
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
	_, err = e.StartBuild(context.Background(), p.ReleasePlanID)
	assertCode(t, err, BuildBusy)
	state, err := e.CancelBuild(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stage != "cancelled" || state.PID != 0 || state.Artifact != nil {
		t.Fatalf("cancel=%+v", state)
	}
	if processGroupAlive(pid) {
		t.Fatal("owned_process_group_remains")
	}
	release, err := e.state.lockFile("build.lock")
	if err != nil {
		t.Fatal(err)
	}
	release()
}
func TestSourceMutationAndInterruptedRecovery(t *testing.T) {
	root, e, p := buildFixture(t)
	dest := filepath.Join(t.TempDir(), "snapshot")
	source, err := snapshotSource(context.Background(), root, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "backend/cmd/plugin/main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	assertCode(t, unchangedSource(context.Background(), root, dest, source), SourceChanged)
	b := Build{Schema: BuildSchema, BuildID: uuid.NewString(), ReleasePlanID: p.ReleasePlanID, PluginID: p.PluginID, Version: p.Version, Target: p.Target, Stage: "building", PID: 999999, CreatedAt: time.Now().UTC()}
	if err = e.saveBuild(&b, false); err != nil {
		t.Fatal(err)
	}
	other := openTest(t, root)
	got, err := other.GetBuild(b.BuildID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != "interrupted" || got.PID != 0 || got.Artifact != nil {
		t.Fatalf("recovery=%+v", got)
	}
	var raw map[string]any
	data, _ := json.Marshal(got)
	_ = json.Unmarshal(data, &raw)
	if raw["artifact"] != nil {
		t.Fatal("interrupted_artifact_invented")
	}
}

func processGroupAlive(pid int) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(-pid, 0) != nil {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	return true
}
