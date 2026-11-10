package publisher

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Plugin struct {
	ID            string `json:"plugin_id"`
	Name          string `json:"name"`
	SourceVersion string `json:"source_version"`
}
type Engine struct {
	root         string
	plugin       Plugin
	state, plans *directory
	builds, work *directory
	mu           sync.Mutex
	jobs         map[string]*buildJob
	closing      bool
	wg           sync.WaitGroup
	host         Target
	toolProbe    func() (BuildTools, error)
	toolsOnce    sync.Once
	tools        BuildTools
	toolsErr     error
}

func Open(entry string) (*Engine, error) {
	root, err := filepath.Abs(entry)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return nil, failure(InvalidWorkspace, err)
	}
	plugin, err := readPlugin(root)
	if err != nil {
		return nil, err
	}
	dir, err := openDirectory(root)
	if err != nil {
		return nil, err
	}
	defer dir.close()
	px, err := dir.child(".px-plugin", false)
	if err != nil {
		return nil, err
	}
	defer px.close()
	state, err := px.child("publisher", true)
	if err != nil {
		return nil, err
	}
	plans, err := state.child("plans", true)
	if err != nil {
		state.close()
		return nil, err
	}
	builds, err := state.child("builds", true)
	if err != nil {
		plans.close()
		state.close()
		return nil, err
	}
	work, err := state.child("work", true)
	if err != nil {
		builds.close()
		plans.close()
		state.close()
		return nil, err
	}
	e := &Engine{root: root, plugin: plugin, state: state, plans: plans, builds: builds, work: work, jobs: map[string]*buildJob{}, host: nativeTarget(), toolProbe: probeTools}
	if err = e.recoverBuilds(); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}
func (e *Engine) Close() error {
	e.mu.Lock()
	e.closing = true
	for _, job := range e.jobs {
		job.cancel()
	}
	e.mu.Unlock()
	e.wg.Wait()
	e.builds.close()
	e.work.close()
	a := e.plans.close()
	b := e.state.close()
	if a != nil {
		return a
	}
	return b
}
func (e *Engine) cancelBuilds() {
	e.mu.Lock()
	for _, job := range e.jobs {
		job.cancel()
	}
	e.mu.Unlock()
	e.wg.Wait()
}
func (e *Engine) Plugin() Plugin { return e.plugin }
func readPlugin(root string) (Plugin, error) {
	file := filepath.Join(root, "plugin.yaml")
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	f, err := os.Open(file)
	if err != nil {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	var document map[string]any
	d := yaml.NewDecoder(bytes.NewReader(data))
	if err = d.Decode(&document); err != nil {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	if err = fwmanifest.ValidatePluginDocument(document); err != nil {
		return Plugin{}, failure(InvalidWorkspace, err)
	}
	version, ok := document["version"].(string)
	if !ok || version == "" {
		return Plugin{}, failure(InvalidWorkspace, nil)
	}
	name, _ := document["name"].(string)
	return Plugin{ID: document["id"].(string), Name: name, SourceVersion: version}, nil
}
func (e *Engine) Create(input CreatePlan) (Plan, error) {
	if err := input.Validate(); err != nil {
		return Plan{}, err
	}
	current, err := readPlugin(e.root)
	if err != nil {
		return Plan{}, err
	}
	if current.ID != e.plugin.ID || current.SourceVersion != e.plugin.SourceVersion {
		return Plan{}, failure(WorkspaceChanged, nil)
	}
	p := Plan{Schema: PlanSchema, ReleasePlanID: uuid.NewString(), PluginID: e.plugin.ID, CreatePlan: input, CreatedAt: time.Now().UTC()}
	data, err := json.Marshal(p)
	if err != nil {
		return Plan{}, failure(StorageFailure, err)
	}
	if err = e.plans.write(p.ReleasePlanID+".json", append(data, '\n'), false); err != nil {
		return Plan{}, err
	}
	return p, nil
}
func (e *Engine) Get(id string) (Plan, error) {
	if !canonicalUUID(id) {
		return Plan{}, invalid("release_plan_id")
	}
	data, err := e.plans.read(id+".json", MaxRecordBytes)
	if err != nil {
		return Plan{}, err
	}
	var p Plan
	if err = decodeStrict(data, &p); err != nil {
		return Plan{}, failure(InvalidRecord, err)
	}
	if p.Schema != PlanSchema || p.ReleasePlanID != id || p.PluginID != e.plugin.ID || p.CreatedAt.IsZero() || p.Validate() != nil {
		return Plan{}, failure(InvalidRecord, nil)
	}
	return p, nil
}
func (e *Engine) List(page, size int) (PlanPage, error) {
	if page < 1 || size < 1 || size > MaxPageSize {
		return PlanPage{}, invalid("pagination")
	}
	names, err := e.plans.names()
	if err != nil {
		return PlanPage{}, err
	}
	items := make([]Plan, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, ".pending-") {
			continue
		}
		if !strings.HasSuffix(name, ".json") || !canonicalUUID(strings.TrimSuffix(name, ".json")) {
			return PlanPage{}, failure(InvalidRecord, nil)
		}
		p, err := e.Get(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return PlanPage{}, err
		}
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ReleasePlanID < items[j].ReleasePlanID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	out := PlanPage{Items: []Plan{}, Total: len(items), Page: page, Size: size}
	// Division bounds the offset before multiplication, including hostile ints.
	if page-1 > len(items)/size {
		return out, nil
	}
	start := (page - 1) * size
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	out.Items = items[start:end]
	return out, nil
}
