package regression

import (
	"context"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.Mutex
	details map[string]RunDetail
	events  map[string][]RegressionEvent
}

func newMemory() *memoryStore {
	return &memoryStore{details: map[string]RunDetail{}, events: map[string][]RegressionEvent{}}
}
func (s *memoryStore) FindRequest(ctx context.Context, a Actor, id string) (RunDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.details {
		if d.Run.TenantUUID == a.TenantUUID && d.Run.ActorUUID == a.UUID && d.Run.ClientRequestUUID == id {
			v := Clone(d)
			v.RequestHash = d.RequestHash
			return v, nil
		}
	}
	return RunDetail{}, NewError(NotFound)
}
func (s *memoryStore) Create(ctx context.Context, a Actor, d RunDetail, e RegressionEvent) (RunDetail, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d.LastEventUUID = &e.UUID
	s.details[d.Run.UUID] = Clone(d)
	v := s.details[d.Run.UUID]
	v.RequestHash = d.RequestHash
	s.details[d.Run.UUID] = v
	s.events[d.Run.UUID] = []RegressionEvent{e}
	return d, false, nil
}
func (s *memoryStore) Get(ctx context.Context, tenant, id string) (RunDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.details[id]
	if !ok || d.Run.TenantUUID != tenant {
		return d, NewError(NotFound)
	}
	out := Clone(d)
	out.RequestHash = d.RequestHash
	return out, nil
}
func (s *memoryStore) List(context.Context, string, uint64, uint64, string) ([]Run, uint64, error) {
	return nil, 0, nil
}
func (s *memoryStore) Commit(ctx context.Context, tenant string, d RunDetail, e RegressionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.details[d.Run.UUID]
	if old.Run.TenantUUID != tenant || old.Run.Revision+1 != d.Run.Revision {
		return NewError(RequestConflict)
	}
	if err := ValidateMeta(e.Meta, tenant, e.Meta.SourcePlugin); err != nil {
		return err
	}
	d.LastEventUUID = &e.UUID
	s.details[d.Run.UUID] = Clone(d)
	s.events[d.Run.UUID] = append(s.events[d.Run.UUID], Clone(e))
	v := s.details[d.Run.UUID]
	v.RequestHash = old.RequestHash
	s.details[d.Run.UUID] = v
	return nil
}
func (s *memoryStore) Events(ctx context.Context, tenant, id, cursor string) ([]RegressionEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.details[id].Run.TenantUUID != tenant {
		return nil, NewError(NotFound)
	}
	return Clone(s.events[id]), nil
}
func (s *memoryStore) Active(ctx context.Context, tenant string) ([]RunDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []RunDetail{}
	for _, d := range s.details {
		if d.Run.TenantUUID == tenant && !Terminal(d.Run.Phase) {
			out = append(out, Clone(d))
		}
	}
	return out, nil
}

type executionScenario struct {
	testScenario
	execute func(context.Context, *Scope) error
	cleanup func(context.Context, *Scope) error
	prepare func(context.Context, *Scope) error
}

func (s *executionScenario) Prepare(ctx context.Context, scope *Scope) error {
	if s.prepare != nil {
		return s.prepare(ctx, scope)
	}
	return nil
}

func (s *executionScenario) Execute(ctx context.Context, scope *Scope) error {
	return s.execute(ctx, scope)
}
func (s *executionScenario) Cleanup(ctx context.Context, scope *Scope) error {
	if s.cleanup != nil {
		return s.cleanup(ctx, scope)
	}
	return nil
}
func testRunner(t *testing.T, exec func(context.Context, *Scope) error, cleanup func(context.Context, *Scope) error) (*Runner, Actor) {
	t.Helper()
	registry := NewRegistry()
	if err := registry.RegisterSuite(SuiteDefinition{UUID: suiteID, Key: "fixture", TitleKey: "regression.fixture.title", DescriptionKey: "regression.fixture.description"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScenario(func() Scenario {
		def := definition()
		// Fixture bookkeeping can be slower under race instrumentation and load.
		// Only tests explicitly exercising timeouts should race a short deadline.
		def.ExecutionTimeoutMS, def.CleanupTimeoutMS = 10000, 5000
		return &executionScenario{testScenario: testScenario{def: def}, execute: exec, cleanup: cleanup}
	}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterProfile(ProfileDefinition{UUID: profileID, Key: "controlled", TitleKey: "regression.profiles.controlled", Kind: "controlled", FixtureNamespace: "regression", BindingRevision: "v1", DependencyDeclarations: []DependencyDeclaration{}}, func(context.Context, Actor) (any, []DependencyEvidence, error) {
		return nil, []DependencyEvidence{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	runner, err := NewRunner(context.Background(), registry, newMemory(), JSON(map[string]any{"source_revision": "fixture"}), nil)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{TenantUUID: uuid.NewString(), UUID: uuid.NewString(), RequestID: uuid.NewString(), TraceID: uuid.NewString(), PluginID: "com.powerx.fixture"}
	return runner, actor
}
func await(t *testing.T, r *Runner, a Actor, id string) RunDetail {
	t.Helper()
	notify, close := r.Subscribe(a.TenantUUID, id)
	defer close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for {
		d, err := r.Get(ctx, a, id)
		if err != nil {
			t.Fatal(err)
		}
		if Terminal(d.Run.Phase) {
			return d
		}
		select {
		case <-notify:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}
func proof() []Evidence {
	return []Evidence{{Kind: "readback", Boundary: "service", SourceRef: "fixture.read", ObservedAt: time.Now().UTC(), SanitizedData: JSON(true)}}
}
func TestRunnerRequiresDeclaredAssertionsAndEvidence(t *testing.T) {
	tests := []struct{ mode, verdict string }{{"passed", "passed"}, {"missing", "failed"}, {"optional", "failed"}, {"duplicate", "failed"}, {"blocked", "blocked"}, {"no_evidence", "blocked"}, {"mismatch", "failed"}}
	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			r, a := testRunner(t, func(ctx context.Context, s *Scope) error {
				return s.Recorder.Step(ctx, stepID, func(ctx context.Context) error {
					if tc.mode == "missing" {
						return nil
					}
					k := "fact.present"
					if tc.mode == "optional" {
						k = "optional"
					}
					status := "passed"
					if tc.mode == "blocked" {
						status = "blocked"
					}
					evidence := proof()
					if tc.mode == "no_evidence" {
						evidence = nil
					}
					actual := true
					if tc.mode == "mismatch" {
						actual = false
					}
					err := s.Recorder.Assert(ctx, k, "regression.fixture.assert", status, true, actual, evidence)
					if tc.mode == "duplicate" {
						err = s.Recorder.Assert(ctx, k, "regression.fixture.assert", status, true, actual, evidence)
					}
					return err
				})
			}, nil)
			d, _, err := r.Create(context.Background(), a, CreateRun{ClientRequestUUID: uuid.NewString(), ProfileUUID: profileID, CaseUuids: []string{caseID}})
			if err != nil {
				t.Fatal(err)
			}
			end := await(t, r, a, d.Run.UUID)
			if end.Run.Verdict != tc.verdict {
				t.Fatalf("%s %s %s", tc.mode, end.Run.Verdict, end.CaseExecutions[0].ReasonCode)
			}
			if _, err = r.Get(context.Background(), Actor{TenantUUID: uuid.NewString(), UUID: a.UUID, RequestID: a.RequestID, TraceID: a.TraceID, PluginID: a.PluginID}, d.Run.UUID); CodeOf(err) != NotFound {
				t.Fatal(err)
			}
		})
	}
}
