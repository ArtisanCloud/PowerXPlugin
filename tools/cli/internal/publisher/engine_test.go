//go:build darwin || linux

package publisher

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func workspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte("id: com.example.publisher\nname: publisher-fixture\nversion: 0.1.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}
func openTest(t *testing.T, root string) *Engine {
	t.Helper()
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}
func input() CreatePlan {
	return CreatePlan{Version: "1.2.0", Channel: "stable", Target: Target{OS: "linux", Arch: "amd64"}, ReleaseNotes: "fixture-notes"}
}
func assertCode(t *testing.T, err error, want string) {
	t.Helper()
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != want {
		t.Fatalf("code=%v, want=%s", err, want)
	}
}

func TestPlansPersistWithoutChangingManifest(t *testing.T) {
	root := workspace(t)
	before, err := os.ReadFile(filepath.Join(root, "plugin.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	if !canonicalUUID(p.ReleasePlanID) || p.CreatedAt.IsZero() || p.Schema != PlanSchema {
		t.Fatalf("plan=%+v", p)
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openTest(t, root)
	got, err := reopened.Get(p.ReleasePlanID)
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Fatalf("got=%+v want=%+v", got, p)
	}
	after, _ := os.ReadFile(filepath.Join(root, "plugin.yaml"))
	if string(before) != string(after) {
		t.Fatal("source_manifest_changed")
	}
	for rel, want := range map[string]os.FileMode{".px-plugin/publisher": 0700, ".px-plugin/publisher/plans": 0700, ".px-plugin/publisher/plans/" + p.ReleasePlanID + ".json": 0600} {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("mode %s: %v %v", rel, info, err)
		}
	}
}
func TestConcurrentPlansAndBoundedPagination(t *testing.T) {
	e := openTest(t, workspace(t))
	var wg sync.WaitGroup
	errs := make(chan error, 205)
	for i := 0; i < 205; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := e.Create(input()); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	first, err := e.List(1, 100)
	if err != nil {
		t.Fatal(err)
	}
	last, err := e.List(3, 100)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 205 || len(first.Items) != 100 || len(last.Items) != 5 {
		t.Fatalf("pagination=%d/%d/%d", first.Total, len(first.Items), len(last.Items))
	}
	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		p, err := e.List(page, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range p.Items {
			if seen[item.ReleasePlanID] {
				t.Fatal("duplicate_plan")
			}
			seen[item.ReleasePlanID] = true
		}
	}
	empty, err := e.List(math.MaxInt, 100)
	if err != nil || len(empty.Items) != 0 || empty.Items == nil {
		t.Fatalf("overflow=%+v %v", empty, err)
	}
	for _, pair := range [][2]int{{0, 20}, {1, 0}, {1, 101}, {-1, 20}} {
		_, err = e.List(pair[0], pair[1])
		assertCode(t, err, InvalidInput)
	}
}
func TestInvalidInputsAreNotPersisted(t *testing.T) {
	for name, mutate := range map[string]func(*CreatePlan){
		"prefix":          func(p *CreatePlan) { p.Version = "v1.2.0" },
		"prerelease":      func(p *CreatePlan) { p.Version = "1.2.0-beta.1" },
		"make_expression": func(p *CreatePlan) { p.Version = "$(shell id)" },
		"leading_zero":    func(p *CreatePlan) { p.Version = "01.2.0" },
		"channel":         func(p *CreatePlan) { p.Channel = "beta" },
		"target":          func(p *CreatePlan) { p.Target.Arch = "arm64" },
		"notes":           func(p *CreatePlan) { p.ReleaseNotes = strings.Repeat("x", MaxNotesBytes+1) },
		"utf8":            func(p *CreatePlan) { p.ReleaseNotes = string([]byte{0xff}) },
	} {
		t.Run(name, func(t *testing.T) {
			e := openTest(t, workspace(t))
			p := input()
			mutate(&p)
			_, err := e.Create(p)
			assertCode(t, err, InvalidInput)
			page, err := e.List(1, 20)
			if err != nil || page.Total != 0 {
				t.Fatal("invalid_plan_written")
			}
		})
	}
}
func TestCorruptRecordsFailClosed(t *testing.T) {
	for _, kind := range []string{"truncated", "schema", "foreign_plugin", "foreign_uuid", "unknown_field", "world_readable", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := workspace(t)
			e := openTest(t, root)
			p, err := e.Create(input())
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(root, ".px-plugin/publisher/plans", p.ReleasePlanID+".json")
			data, _ := os.ReadFile(file)
			switch kind {
			case "truncated":
				data = []byte("{")
			case "schema":
				data = []byte(strings.Replace(string(data), PlanSchema, "legacy", 1))
			case "foreign_plugin":
				data = []byte(strings.Replace(string(data), p.PluginID, "com.other.publisher", 1))
			case "foreign_uuid":
				data = []byte(strings.Replace(string(data), p.ReleasePlanID, uuid.NewString(), 1))
			case "unknown_field":
				data = []byte(strings.Replace(string(data), "{", "{\"command\":\"id\",", 1))
			case "world_readable":
				if err = os.Chmod(file, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err = os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(filepath.Join(root, "plugin.yaml"), file); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "world_readable" && kind != "symlink" {
				if err = os.WriteFile(file, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err = e.List(1, 20)
			if err == nil {
				t.Fatal("corruption_accepted")
			}
		})
	}
}
func TestStateCannotFollowOrEscapeSymlinkDirectories(t *testing.T) {
	for _, rel := range []string{".px-plugin", ".px-plugin/publisher", ".px-plugin/publisher/plans"} {
		t.Run(rel, func(t *testing.T) {
			root := workspace(t)
			outside := t.TempDir()
			link := filepath.Join(root, rel)
			if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			_, err := Open(root)
			assertCode(t, err, StorageFailure)
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatal("escaped_state_write")
			}
		})
	}
	t.Run("replaced_parent", func(t *testing.T) {
		root := workspace(t)
		e := openTest(t, root)
		outside := t.TempDir()
		state := filepath.Join(root, ".px-plugin/publisher")
		if err := os.Rename(state, state+"-pinned"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, state); err != nil {
			t.Fatal(err)
		}
		p, err := e.Create(input())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(filepath.Join(state+"-pinned", "plans", p.ReleasePlanID+".json")); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(outside)
		if err != nil || len(entries) != 0 {
			t.Fatal("escaped_pinned_root")
		}
	})
}
func TestIdentityAndSourceChangesAreExplicit(t *testing.T) {
	root := workspace(t)
	e := openTest(t, root)
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte("id: com.example.publisher\nversion: 0.2.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := e.Create(input())
	assertCode(t, err, WorkspaceChanged)
	_, err = e.Get("../../plugin.yaml")
	assertCode(t, err, InvalidInput)
	_, err = e.Get(uuid.NewString())
	assertCode(t, err, NotFound)
	for _, manifest := range []string{"plugin_id: com.example.publisher\nversion: 0.1.0\n", "id: com.powerx.plugin.legacy\nversion: 0.1.0\n", "id: com.example.publisher\nid: com.other.publisher\nversion: 0.1.0\n", "id: com.example.publisher\nversion: 0.1.0\n---\nid: com.other.publisher\nversion: 0.1.0\n"} {
		other := workspace(t)
		if err = os.WriteFile(filepath.Join(other, "plugin.yaml"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
		_, err = Open(other)
		assertCode(t, err, InvalidWorkspace)
	}
}
func TestStrictJSONRejectsAmbiguousOrUnknownFields(t *testing.T) {
	valid, _ := json.Marshal(input())
	var invalidUTF8 CreatePlan
	if decodeStrict([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}, &invalidUTF8) == nil {
		t.Fatal("invalid_utf8_accepted")
	}
	for name, data := range map[string]string{
		"unknown":          strings.Replace(string(valid), "{", "{\"command\":\"id\",", 1),
		"duplicate":        strings.Replace(string(valid), "{", "{\"version\":\"2.0.0\",", 1),
		"nested_duplicate": strings.Replace(string(valid), `"os":"linux"`, `"os":"linux","os":"darwin"`, 1),
		"trailing":         string(valid) + "{}",
	} {
		t.Run(name, func(t *testing.T) {
			var p CreatePlan
			if decodeStrict([]byte(data), &p) == nil {
				t.Fatalf("accepted=%s", name)
			}
		})
	}
}
func TestLocaleCatalogParity(t *testing.T) {
	for key := range catalog["en"] {
		if catalog["zh"][key] == "" {
			t.Fatalf("missing_locale=%s", key)
		}
	}
	for key := range catalog["zh"] {
		if catalog["en"][key] == "" {
			t.Fatalf("missing_locale=%s", key)
		}
	}
	err := &DisplayError{Err: invalid("version"), Locale: "zh_CN.UTF-8"}
	if !strings.Contains(err.Error(), Text("zh", InvalidInput)) || err.ExitCode() != 2 {
		t.Fatal("locale_mapping_failed")
	}
	if got := fmt.Sprint(failure(Forbidden, errors.New("private-diagnostic"))); got != Forbidden {
		t.Fatal("diagnostic_leaked")
	}
}
