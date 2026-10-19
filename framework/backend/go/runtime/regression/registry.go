package regression

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

func ValidUUID(v string) bool {
	id, err := uuid.Parse(v)
	return err == nil && id != uuid.Nil && id.String() == v
}
func Clone[T any](v T) T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out T
	if err = json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
func ValidateMeta(m EventMeta, tenant, plugin string) error {
	if !ValidUUID(m.TenantUUID) || m.TenantUUID != tenant || strings.TrimSpace(m.RequestID) == "" || strings.TrimSpace(m.TraceID) == "" || m.SourcePlugin != plugin || strings.TrimSpace(plugin) == "" || m.OccurredAt.IsZero() || m.OccurredAt.Location() != time.UTC || m.PayloadVersion != "v1" {
		return NewError(EventMetaInvalid)
	}
	return nil
}
func key(v string) bool { return strings.TrimSpace(v) != "" && strings.TrimSpace(v) == v }

type Registry struct {
	mu        sync.RWMutex
	suites    map[string]SuiteDefinition
	defs      map[string]CaseDefinition
	scenarios map[string]ScenarioFactory
	profiles  map[string]ProfileDefinition
	factories map[string]ProfileFactory
	schemas   map[string]*jsonschema.Schema
}

func NewRegistry() *Registry {
	return &Registry{suites: map[string]SuiteDefinition{}, defs: map[string]CaseDefinition{}, scenarios: map[string]ScenarioFactory{}, profiles: map[string]ProfileDefinition{}, factories: map[string]ProfileFactory{}, schemas: map[string]*jsonschema.Schema{}}
}
func (r *Registry) RegisterSuite(d SuiteDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !ValidUUID(d.UUID) || !key(d.Key) || !key(d.TitleKey) || !key(d.DescriptionKey) {
		return NewError(InvalidRequest)
	}
	if _, ok := r.suites[d.UUID]; ok {
		return NewError(InvalidRequest)
	}
	r.suites[d.UUID] = Clone(d)
	return nil
}
func (r *Registry) RegisterScenario(factory ScenarioFactory) (err error) {
	defer func() {
		if recover() != nil {
			err = NewError(InvalidRequest)
		}
	}()
	if factory == nil {
		return NewError(InvalidRequest)
	}
	s := factory()
	if s == nil {
		return NewError(InvalidRequest)
	}
	d := s.Definition()
	if !ValidUUID(d.UUID) || !ValidUUID(d.SuiteUUID) || !key(d.Key) || !key(d.Version) || !key(d.TitleKey) || !key(d.DescriptionKey) || len(d.Steps) == 0 || len(d.RequiredAssertionKeys) == 0 || len(d.SupportedProfileKinds) == 0 || d.ExecutionTimeoutMS == 0 || d.CleanupTimeoutMS == 0 || d.ExecutionTimeoutMS > uint64((1<<63-1)/int64(time.Millisecond)) || d.CleanupTimeoutMS > uint64((1<<63-1)/int64(time.Millisecond)) {
		return NewError(InvalidRequest)
	}
	seen := map[string]bool{}
	for _, k := range d.RequiredAssertionKeys {
		if !key(k) || seen[k] {
			return NewError(InvalidRequest)
		}
		seen[k] = true
	}
	seen = map[string]bool{}
	for _, step := range d.Steps {
		if !ValidUUID(step.UUID) || seen[step.UUID] || !key(step.Key) || !key(step.TitleKey) || !key(step.Boundary) || !oneOf(step.Phase, "preflight", "prepare", "execute", "verify", "cleanup") {
			return NewError(InvalidRequest)
		}
		seen[step.UUID] = true
	}
	for _, mode := range d.SupportedProfileKinds {
		if !oneOf(mode, "controlled", "live") {
			return NewError(InvalidRequest)
		}
	}
	c := jsonschema.NewCompiler()
	c.LoadURL = func(string) (io.ReadCloser, error) { return nil, NewError(InvalidRequest) }
	resource := "https://regression.invalid/" + d.UUID
	if !json.Valid(d.InputSchema) {
		return NewError(InvalidRequest)
	}
	if err := c.AddResource(resource, strings.NewReader(string(d.InputSchema))); err != nil {
		return NewError(InvalidRequest)
	}
	schema, err := c.Compile(resource)
	if err != nil {
		return NewError(InvalidRequest)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.suites[d.SuiteUUID]; !ok {
		return NewError(InvalidRequest)
	}
	if _, ok := r.defs[d.UUID]; ok {
		return NewError(InvalidRequest)
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
	r.defs[d.UUID] = Clone(d)
	r.scenarios[d.UUID] = factory
	r.schemas[d.UUID] = schema
	return nil
}
func (r *Registry) RegisterProfile(d ProfileDefinition, f ProfileFactory) error {
	if !ValidUUID(d.UUID) || !key(d.Key) || !key(d.TitleKey) || !key(d.BindingRevision) || !key(d.FixtureNamespace) || !oneOf(d.Kind, "controlled", "live") || f == nil {
		return NewError(InvalidRequest)
	}
	if d.DependencyDeclarations == nil {
		d.DependencyDeclarations = []DependencyDeclaration{}
	}
	for _, dep := range d.DependencyDeclarations {
		if !key(dep.DependencyKey) || !key(dep.Boundary) || !oneOf(dep.Mode, "real", "test_double") {
			return NewError(InvalidRequest)
		}
		if d.Kind == "live" && dep.Mode != "real" {
			return NewError(InvalidRequest)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.profiles[d.UUID]; ok {
		return NewError(InvalidRequest)
	}
	r.profiles[d.UUID] = Clone(d)
	r.factories[d.UUID] = f
	return nil
}
func (r *Registry) Suites() []SuiteDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []SuiteDefinition{}
	for _, v := range r.suites {
		out = append(out, Clone(v))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
func (r *Registry) Cases() []CaseDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []CaseDefinition{}
	for _, v := range r.defs {
		out = append(out, Clone(v))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
func (r *Registry) Profiles() []ProfileDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []ProfileDefinition{}
	for _, v := range r.profiles {
		out = append(out, Clone(v))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
func (r *Registry) Case(id string) (CaseDefinition, ScenarioFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.defs[id]
	if !ok {
		return d, nil, NewError(NotFound)
	}
	return Clone(d), r.scenarios[id], nil
}
func (r *Registry) Profile(id string) (ProfileDefinition, ProfileFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.profiles[id]
	if !ok {
		return d, nil, NewError(NotFound)
	}
	return Clone(d), r.factories[id], nil
}
func (r *Registry) ValidateInput(id string, raw json.RawMessage) error {
	r.mu.RLock()
	schema, ok := r.schemas[id]
	r.mu.RUnlock()
	if !ok {
		return NewError(NotFound)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return NewError(InvalidRequest)
	}
	if err := schema.Validate(value); err != nil {
		return NewError(InvalidRequest)
	}
	return nil
}
func oneOf(v string, values ...string) bool {
	for _, value := range values {
		if value == v {
			return true
		}
	}
	return false
}
