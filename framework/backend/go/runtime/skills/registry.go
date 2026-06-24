package skills

import (
	"sort"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	byKey     map[string]PluginSkillManifest
	executors map[string]ExecutorHandler
}

func NewRegistry() *Registry {
	return &Registry{
		byKey:     map[string]PluginSkillManifest{},
		executors: map[string]ExecutorHandler{},
	}
}

func (r *Registry) RegisterManifest(m PluginSkillManifest) error {
	if err := ValidateManifest(m); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := m.RegistryKey()
	if _, exists := r.byKey[key]; exists {
		return NewError(ErrCodeDuplicateManifest, "duplicate skill_id and version: "+key)
	}
	r.byKey[key] = m
	return nil
}

func (r *Registry) MustRegisterManifest(m PluginSkillManifest) {
	if err := r.RegisterManifest(m); err != nil {
		panic(err)
	}
}

func (r *Registry) List() []PluginSkillManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]PluginSkillManifest, 0, len(r.byKey))
	for _, m := range r.byKey {
		items = append(items, m)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SkillID == items[j].SkillID {
			return items[i].Version < items[j].Version
		}
		return items[i].SkillID < items[j].SkillID
	})
	return items
}

func (r *Registry) Get(skillID, version string) (PluginSkillManifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if version != "" {
		m, ok := r.byKey[skillID+"@"+version]
		return m, ok
	}
	var latest PluginSkillManifest
	found := false
	for _, m := range r.byKey {
		if m.SkillID != skillID {
			continue
		}
		if !found || m.Version > latest.Version {
			latest = m
			found = true
		}
	}
	return latest, found
}

func (r *Registry) Schema(skillID, version string) (PluginSkillSchema, bool) {
	m, ok := r.Get(skillID, version)
	if !ok {
		return PluginSkillSchema{}, false
	}
	return PluginSkillSchema{
		SkillID:      m.SkillID,
		Version:      m.Version,
		InputSchema:  m.InputSchema,
		OutputSchema: m.OutputSchema,
	}, true
}
