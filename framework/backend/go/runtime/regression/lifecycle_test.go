package regression

import (
	"context"
	"github.com/google/uuid"
	"sync/atomic"
	"testing"
	"time"
)

func submit(t *testing.T, r *Runner, a Actor) RunDetail {
	t.Helper()
	d, _, err := r.Create(context.Background(), a, CreateRun{ClientRequestUUID: uuid.NewString(), ProfileUUID: profileID, CaseUuids: []string{caseID}, Inputs: JSON(map[string]any{caseID: map[string]any{}})})
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func settle(t *testing.T, r *Runner) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.mu.Lock()
	var done <-chan struct{}
	for _, e := range r.active {
		done = e.done
	}
	r.mu.Unlock()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}
func TestLifecyclePanicTimeoutCancellationAndIndependentCleanup(t *testing.T) {
	for _, mode := range []string{"panic", "timeout", "cancel", "prepare"} {
		t.Run(mode, func(t *testing.T) {
			started := make(chan struct{})
			var cleaned atomic.Bool
			exec := func(ctx context.Context, s *Scope) error {
				return s.Recorder.Step(ctx, stepID, func(ctx context.Context) error {
					if err := s.Recorder.Register(ctx, FixtureResource{ResourceType: "fixture", ResourceUUID: uuid.NewString(), CleanupKey: "fixture.delete", CleanupParameters: JSON(map[string]any{})}); err != nil {
						return err
					}
					close(started)
					if mode == "panic" {
						panic(InvalidRequest)
					}
					<-ctx.Done()
					return ctx.Err()
				})
			}
			cleanup := func(ctx context.Context, s *Scope) error {
				if ctx.Err() != nil {
					t.Error(ctx.Err())
				}
				for _, f := range s.Recorder.ListOwned() {
					if err := s.Recorder.Cleaned(ctx, f.ResourceUUID, cleanupProof(f.ResourceUUID)); err != nil {
						return err
					}
				}
				cleaned.Store(true)
				return nil
			}
			r, a := testRunner(t, exec, cleanup)
			if mode == "timeout" {
				r.Registry.defs[caseID] = func() CaseDefinition { d := r.Registry.defs[caseID]; d.ExecutionTimeoutMS = 2000; return d }()
			}
			if mode == "prepare" {
				old := r.Registry.scenarios[caseID]
				r.Registry.scenarios[caseID] = func() Scenario {
					s := old().(*executionScenario)
					s.prepare = func(ctx context.Context, s *Scope) error {
						if err := s.Recorder.Register(ctx, FixtureResource{ResourceType: "fixture", ResourceUUID: uuid.NewString(), CleanupKey: "fixture.delete", CleanupParameters: JSON(map[string]any{})}); err != nil {
							return err
						}
						return NewError(InvalidRequest)
					}
					return s
				}
			}
			d := submit(t, r, a)
			if mode == "cancel" {
				<-started
				if _, err := r.Cancel(context.Background(), a, d.Run.UUID); err != nil {
					t.Fatal(err)
				}
			}
			final := await(t, r, a, d.Run.UUID)
			settle(t, r)
			if final.Run.Verdict == "passed" || !cleaned.Load() || final.Run.CleanupState != "succeeded" {
				t.Fatal(final.Run)
			}
			for _, s := range final.CaseExecutions[0].Steps {
				if s.Status == "running" {
					t.Fatal(s.Status)
				}
			}
		})
	}
}
func TestCleanupRetryKeepsVerdictAndChecksBinding(t *testing.T) {
	var attempts atomic.Int32
	r, a := testRunner(t, func(ctx context.Context, s *Scope) error {
		if err := s.Recorder.Register(ctx, FixtureResource{ResourceType: "fixture", ResourceUUID: uuid.NewString(), CleanupKey: "fixture.delete", CleanupParameters: JSON(map[string]any{})}); err != nil {
			return err
		}
		return s.Recorder.Step(ctx, stepID, func(ctx context.Context) error {
			return s.Recorder.Assert(ctx, "fact.present", "regression.fixture.fact", "passed", true, true, proof())
		})
	}, func(ctx context.Context, s *Scope) error {
		if attempts.Add(1) == 1 {
			return NewError(CleanupFailed)
		}
		for _, f := range s.Recorder.ListOwned() {
			if err := s.Recorder.Cleaned(ctx, f.ResourceUUID, cleanupProof(f.ResourceUUID)); err != nil {
				return err
			}
		}
		return nil
	})
	d := submit(t, r, a)
	final := await(t, r, a, d.Run.UUID)
	settle(t, r)
	if final.Run.Verdict != "failed" || final.Run.CleanupState != "failed" {
		t.Fatal(final.Run)
	}
	old := r.Registry.profiles[profileID]
	changed := old
	changed.BindingRevision = "v2"
	r.Registry.profiles[profileID] = changed
	if _, err := r.Cleanup(context.Background(), a, d.Run.UUID); CodeOf(err) != CleanupUnavailable {
		t.Fatal(err)
	}
	r.Registry.profiles[profileID] = old
	if _, err := r.Cleanup(context.Background(), a, d.Run.UUID); err != nil {
		t.Fatal(err)
	}
	settle(t, r)
	final, err := r.Get(context.Background(), a, d.Run.UUID)
	if err != nil || final.Run.Verdict != "failed" || final.Run.CleanupState != "succeeded" {
		t.Fatal(final.Run, err)
	}
}
func TestRecoveryMarksUnfinishedWithoutRepeatingBusiness(t *testing.T) {
	r, a := testRunner(t, func(context.Context, *Scope) error { t.Fatal(Interrupted); return nil }, nil)
	s := r.Store.(*memoryStore)
	now := time.Now().UTC()
	d := RunDetail{Run: Run{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, ActorUUID: a.UUID, Revision: 1, Phase: "running", Verdict: "pending", CleanupState: "pending", CreatedAt: now, UpdatedAt: now}, CaseExecutions: []CaseExecution{{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, DefinitionSnapshot: definition(), Phase: "executing", Verdict: "pending", CleanupState: "pending", Steps: []StepResult{}, FixtureResources: []FixtureResource{}, MissingAssertionKeys: []string{}}}}
	d.CaseExecutions[0].RunUUID = d.Run.UUID
	s.details[d.Run.UUID] = d
	if err := r.Recover(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(context.Background(), a, d.Run.UUID)
	if err != nil || got.Run.Phase != "interrupted" || got.Run.Verdict != "blocked" || len(got.CaseExecutions[0].MissingAssertionKeys) != 1 {
		t.Fatal(got, err)
	}
}

func cleanupProof(id string) []Evidence {
	return []Evidence{{Kind: "readback", Boundary: "fixture", SourceRef: "fixture.read_after_delete", ResourceUUID: id, ObservedAt: time.Now().UTC(), SanitizedData: JSON(map[string]any{"exists": false})}}
}
