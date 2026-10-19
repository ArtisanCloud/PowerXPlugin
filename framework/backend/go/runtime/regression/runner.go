package regression

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"sync"
	"time"
)

type Observer func(context.Context, RegressionEvent, error)
type Runner struct {
	Registry      *Registry
	Store         Store
	parent        context.Context
	source        json.RawMessage
	observe       Observer
	mu            sync.Mutex
	busy          bool
	closed        bool
	active        map[string]*execution
	faults        map[string]error
	subscriptions map[string]map[chan struct{}]bool
}
type execution struct {
	mu      sync.Mutex
	runner  *Runner
	detail  RunDetail
	actor   Actor
	inputs  map[string]json.RawMessage
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	secrets []string
}

func NewRunner(ctx context.Context, registry *Registry, store Store, source json.RawMessage, observer Observer) (*Runner, error) {
	if ctx == nil || registry == nil || store == nil || !json.Valid(source) {
		return nil, NewError(StoreUnavailable)
	}
	return &Runner{Registry: registry, Store: store, parent: ctx, source: Redact(source), observe: observer, active: map[string]*execution{}, faults: map[string]error{}, subscriptions: map[string]map[chan struct{}]bool{}}, nil
}
func ValidateActor(a Actor) error {
	if !ValidUUID(a.TenantUUID) || !ValidUUID(a.UUID) || !key(a.PluginID) || !key(a.RequestID) || !key(a.TraceID) {
		return NewError(InvalidRequest)
	}
	return nil
}
func requestHash(in CreateRun) string {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(JSON(in)))
	decoder.UseNumber()
	_ = decoder.Decode(&value)
	h := sha256.Sum256(JSON(value))
	return hex.EncodeToString(h[:])
}
func newEvent(a Actor, d RunDetail, kind string, caseUUID *string, payload any) RegressionEvent {
	now := time.Now().UTC()
	return RegressionEvent{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, RunUUID: d.Run.UUID, CaseExecutionUUID: caseUUID, Sequence: d.Run.Revision, Revision: d.Run.Revision, Kind: kind, Payload: Redact(JSON(payload)), Meta: EventMeta{TenantUUID: a.TenantUUID, RequestID: a.RequestID, TraceID: a.TraceID, SourcePlugin: a.PluginID, OccurredAt: now, PayloadVersion: "v1"}, CreatedAt: now, UpdatedAt: now}
}
func (r *Runner) Create(ctx context.Context, a Actor, in CreateRun) (RunDetail, bool, error) {
	if err := ValidateActor(a); err != nil {
		return RunDetail{}, false, err
	}
	if !ValidUUID(in.ClientRequestUUID) || !ValidUUID(in.ProfileUUID) || len(in.CaseUuids) == 0 {
		return RunDetail{}, false, NewError(InvalidRequest)
	}
	hash := requestHash(in)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.parent.Err() != nil {
		return RunDetail{}, false, NewError(StoreUnavailable)
	}
	if old, err := r.Store.FindRequest(ctx, a, in.ClientRequestUUID); err == nil {
		if old.RequestHash != hash {
			return RunDetail{}, false, NewError(RequestConflict)
		}
		return old, true, nil
	} else if CodeOf(err) != NotFound {
		return RunDetail{}, false, err
	}
	if r.busy {
		return RunDetail{}, false, NewError(Busy)
	}
	profile, _, err := r.Registry.Profile(in.ProfileUUID)
	if err != nil {
		return RunDetail{}, false, err
	}
	var rerun *string
	if in.RerunOfRunUUID != "" {
		rerun = &in.RerunOfRunUUID
		if _, err = r.Store.Get(ctx, a.TenantUUID, in.RerunOfRunUUID); err != nil {
			return RunDetail{}, false, err
		}
	}
	inputs := map[string]json.RawMessage{}
	if len(in.Inputs) > 0 {
		if err = json.Unmarshal(in.Inputs, &inputs); err != nil || inputs == nil {
			return RunDetail{}, false, NewError(InvalidRequest)
		}
	}
	seen := map[string]bool{}
	now := time.Now().UTC()
	run := Run{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, ActorUUID: a.UUID, ProfileUUID: profile.UUID, ClientRequestUUID: in.ClientRequestUUID, RerunOfRunUUID: rerun, ProfileSnapshot: profile, SelectionSnapshot: []CaseDefinition{}, InputSnapshot: JSON(map[string]any{}), SourceSnapshot: r.source, Phase: "queued", Verdict: "pending", CleanupState: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now}
	if len(run.InputSnapshot) == 0 || string(run.InputSnapshot) == "null" {
		run.InputSnapshot = JSON(map[string]any{})
	}
	d := RunDetail{Run: run, CaseExecutions: []CaseExecution{}, RequestHash: hash}
	secrets := []string{a.Credential}
	for i, id := range in.CaseUuids {
		if seen[id] {
			return RunDetail{}, false, NewError(InvalidRequest)
		}
		seen[id] = true
		def, _, err := r.Registry.Case(id)
		if err != nil {
			return RunDetail{}, false, err
		}
		supported := false
		for _, kind := range def.SupportedProfileKinds {
			supported = supported || kind == profile.Kind
		}
		if !supported {
			return RunDetail{}, false, NewError(InvalidRequest)
		}
		input, ok := inputs[id]
		if !ok {
			input = JSON(map[string]any{})
		}
		if err = r.Registry.ValidateInput(id, input); err != nil {
			return RunDetail{}, false, err
		}
		inputs[id] = input
		d.Run.SelectionSnapshot = append(d.Run.SelectionSnapshot, def)
		sanitized := RedactInput(input, def.InputSchema)
		secrets = append(secrets, inputSecrets(input, def.InputSchema)...)
		var snapshots map[string]json.RawMessage
		_ = json.Unmarshal(d.Run.InputSnapshot, &snapshots)
		snapshots[id] = sanitized
		d.Run.InputSnapshot = JSON(snapshots)
		d.CaseExecutions = append(d.CaseExecutions, CaseExecution{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, RunUUID: run.UUID, CaseUUID: id, Ordinal: uint64(i + 1), DefinitionSnapshot: def, InputSnapshot: sanitized, DependencyEvidence: []DependencyEvidence{}, Phase: "pending", Verdict: "pending", Steps: []StepResult{}, FixtureResources: []FixtureResource{}, MissingAssertionKeys: []string{}, CleanupState: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now})
	}
	for id := range inputs {
		if !seen[id] {
			return RunDetail{}, false, NewError(InvalidRequest)
		}
	}
	sanitizer := &execution{secrets: secrets}
	d.Run.InputSnapshot = sanitizer.redact(d.Run.InputSnapshot)
	for i := range d.CaseExecutions {
		d.CaseExecutions[i].InputSnapshot = sanitizer.redact(d.CaseExecutions[i].InputSnapshot)
	}
	event := newEvent(a, d, "run.created", nil, map[string]any{"run": d.Run})
	created, duplicate, err := r.Store.Create(ctx, a, d, event)
	if err != nil || duplicate {
		return created, duplicate, err
	}
	base, cancel := context.WithCancel(r.parent)
	e := &execution{runner: r, detail: created, actor: a, inputs: inputs, ctx: base, cancel: cancel, done: make(chan struct{}), secrets: secrets}
	r.busy = true
	r.active[run.UUID] = e
	go e.work()
	return created, false, nil
}
func (r *Runner) Get(ctx context.Context, a Actor, id string) (RunDetail, error) {
	if err := ValidateActor(a); err != nil {
		return RunDetail{}, err
	}
	r.mu.Lock()
	fault := r.faults[a.TenantUUID+":"+id]
	r.mu.Unlock()
	if fault != nil {
		return RunDetail{}, fault
	}
	return r.Store.Get(ctx, a.TenantUUID, id)
}
func (r *Runner) Notify(tenant, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subscriptions[tenant+":"+id] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (r *Runner) Subscribe(tenant, id string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	k := tenant + ":" + id
	r.mu.Lock()
	if r.subscriptions[k] == nil {
		r.subscriptions[k] = map[chan struct{}]bool{}
	}
	r.subscriptions[k][ch] = true
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.subscriptions[k], ch)
		if len(r.subscriptions[k]) == 0 {
			delete(r.subscriptions, k)
		}
	}
}
func Terminal(phase string) bool { return oneOf(phase, "completed", "cancelled", "interrupted") }
func (e *execution) commitLocked(ctx context.Context, kind string, id *string, payload map[string]any) error {
	previous := e.detail.Run.Revision
	e.detail.Run.Revision++
	e.detail.Run.UpdatedAt = time.Now().UTC()
	payload["run"] = e.detail.Run
	event := newEvent(e.actor, e.detail, kind, id, payload)
	event.Payload = e.redact(event.Payload)
	err := e.runner.Store.Commit(ctx, e.actor.TenantUUID, e.detail, event)
	if err != nil {
		e.detail.Run.Revision = previous
		e.runner.mu.Lock()
		e.runner.faults[e.actor.TenantUUID+":"+e.detail.Run.UUID] = NewError(StoreUnavailable)
		e.runner.mu.Unlock()
		e.cancel()
	}
	if e.runner.observe != nil {
		e.runner.observe(ctx, event, err)
	}
	if err == nil {
		e.detail.LastEventUUID = &event.UUID
	}
	e.runner.Notify(e.actor.TenantUUID, e.detail.Run.UUID)
	return err
}
func (e *execution) runUpdate(fn func(*Run), kind string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	fn(&e.detail.Run)
	return e.commitLocked(context.WithoutCancel(e.ctx), kind, nil, map[string]any{})
}
func invoke(ctx context.Context, fn func(context.Context, *Scope) error, s *Scope) (err error) {
	defer func() {
		if recover() != nil {
			err = NewError(InvalidRequest)
		}
	}()
	return fn(ctx, s)
}
func construct(factory ScenarioFactory) (scenario Scenario, err error) {
	defer func() {
		if recover() != nil {
			err = NewError(InvalidRequest)
		}
	}()
	scenario = factory()
	if scenario == nil {
		err = NewError(InvalidRequest)
	}
	return
}
func invokeProfile(ctx context.Context, factory ProfileFactory, actor Actor) (dependencies any, evidence []DependencyEvidence, err error) {
	defer func() {
		if recover() != nil {
			err = NewError(DependencyBlocked)
		}
	}()
	return factory(ctx, actor)
}
func (e *execution) work() {
	defer func() {
		e.cancel()
		e.mu.Lock()
		e.actor.Credential = ""
		e.secrets = nil
		e.mu.Unlock()
		e.runner.mu.Lock()
		delete(e.runner.active, e.detail.Run.UUID)
		e.runner.busy = false
		e.runner.mu.Unlock()
		close(e.done)
	}()
	if err := e.runUpdate(func(r *Run) { now := time.Now().UTC(); r.Phase = "running"; r.StartedAt = &now }, "run.updated"); err != nil {
		return
	}
	for i := range e.detail.CaseExecutions {
		if e.ctx.Err() != nil {
			rec := &recorder{execution: e, index: i, step: -1}
			_ = rec.mutate(context.WithoutCancel(e.ctx), "case.updated", func(c *CaseExecution) (*AssertionResult, error) {
				now := time.Now().UTC()
				c.Phase = "finished"
				c.Verdict = "skipped"
				c.ReasonCode = "REGRESSION_CANCELLED"
				c.MessageKey = MessageKey(c.ReasonCode)
				c.CleanupState = "not_required"
				c.FinishedAt = &now
				c.MissingAssertionKeys = append([]string{}, c.DefinitionSnapshot.RequiredAssertionKeys...)
				completeSteps(c, c.ReasonCode)
				return nil, nil
			})
			continue
		}
		e.runCase(i)
	}
	_ = e.runUpdate(func(r *Run) {
		r.Verdict = "passed"
		r.CleanupState = "not_required"
		for _, c := range e.detail.CaseExecutions {
			if c.CleanupState == "failed" {
				r.CleanupState = "failed"
			} else if len(c.FixtureResources) > 0 && r.CleanupState != "failed" {
				r.CleanupState = "succeeded"
			}
			if c.Verdict == "failed" {
				r.Verdict = "failed"
			} else if c.Verdict != "passed" && r.Verdict != "failed" {
				r.Verdict = "blocked"
			}
		}
		r.Phase = "completed"
		if e.ctx.Err() != nil {
			r.Phase = "cancelled"
			if r.Verdict != "failed" {
				r.Verdict = "blocked"
			}
		}
		now := time.Now().UTC()
		r.FinishedAt = &now
	}, "run.finished")
}
func (e *execution) runCase(index int) {
	rec := &recorder{execution: e, index: index, step: -1}
	def := e.detail.CaseExecutions[index].DefinitionSnapshot
	_, factory, err := e.runner.Registry.Case(def.UUID)
	if err != nil {
		return
	}
	scenario, constructionErr := construct(factory)
	ctx, cancel := context.WithTimeout(e.ctx, time.Duration(def.ExecutionTimeoutMS)*time.Millisecond)
	defer cancel()
	scope := &Scope{RunUUID: e.detail.Run.UUID, CaseExecutionUUID: e.detail.CaseExecutions[index].UUID, Actor: e.actor, Definition: Clone(def), Profile: Clone(e.detail.Run.ProfileSnapshot), Input: e.inputs[def.UUID], Recorder: rec}
	_, profileFactory, profileErr := e.runner.Registry.Profile(scope.Profile.UUID)
	stage := func(name string) error {
		return rec.mutate(context.WithoutCancel(ctx), "case.updated", func(c *CaseExecution) (*AssertionResult, error) {
			c.Phase = name
			if c.StartedAt == nil {
				now := time.Now().UTC()
				c.StartedAt = &now
			}
			return nil, nil
		})
	}
	if err = stage("preflight"); err != nil {
		return
	}
	normal := false
	if constructionErr != nil {
		err = constructionErr
	} else if profileErr != nil {
		err = profileErr
	} else {
		var evidence []DependencyEvidence
		scope.Dependencies, evidence, err = invokeProfile(ctx, profileFactory, e.actor)
		if evidence == nil {
			evidence = []DependencyEvidence{}
		}
		if err == nil {
			seen := map[string]bool{}
			for _, declaration := range scope.Profile.DependencyDeclarations {
				matched := false
				for _, item := range evidence {
					if item.DependencyKey == declaration.DependencyKey {
						if seen[item.DependencyKey] || item.DeclaredMode != declaration.Mode || item.ActualMode != declaration.Mode || item.Boundary != declaration.Boundary || item.ReasonCode != "" {
							err = NewError(DependencyBlocked)
						}
						seen[item.DependencyKey] = true
						matched = true
					}
				}
				if !matched {
					err = NewError(DependencyBlocked)
				}
			}
			if len(evidence) != len(scope.Profile.DependencyDeclarations) {
				err = NewError(DependencyBlocked)
			}
		}
		if saveErr := rec.mutate(context.WithoutCancel(ctx), "case.updated", func(c *CaseExecution) (*AssertionResult, error) { c.DependencyEvidence = evidence; return nil, nil }); saveErr != nil {
			err = saveErr
		}
	}
	if err == nil {
		err = invoke(ctx, scenario.Preflight, scope)
	}
	preflightBlocked := err != nil
	if err == nil {
		if err = stage("preparing"); err == nil {
			err = invoke(ctx, scenario.Prepare, scope)
		}
	}
	if err == nil {
		if err = stage("executing"); err == nil {
			err = invoke(ctx, scenario.Execute, scope)
			normal = err == nil
		}
	}
	e.mu.Lock()
	recordErr := rec.err
	e.mu.Unlock()
	if recordErr != nil {
		err = recordErr
		normal = false
	}
	reason := CodeOf(err)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		reason = Timeout
		normal = false
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		reason = "REGRESSION_CANCELLED"
		normal = false
	}
	_ = stage("cleaning")
	cleanCtx, cleanCancel := context.WithTimeout(context.WithoutCancel(e.ctx), time.Duration(def.CleanupTimeoutMS)*time.Millisecond)
	var cleanErr error
	if scenario != nil {
		cleanErr = invoke(cleanCtx, scenario.Cleanup, scope)
	}
	if cleanCtx.Err() != nil {
		cleanErr = cleanCtx.Err()
	}
	cleanCancel()
	_ = rec.mutate(context.WithoutCancel(e.ctx), "case.updated", func(c *CaseExecution) (*AssertionResult, error) {
		c.CleanupState = "not_required"
		if len(c.FixtureResources) > 0 {
			c.CleanupState = "succeeded"
		}
		for _, f := range c.FixtureResources {
			if f.CleanupState != "succeeded" {
				c.CleanupState = "failed"
			}
		}
		if cleanErr != nil {
			c.CleanupState = "failed"
		}
		c.ReasonCode = reason
		c.Verdict = caseVerdict(c, normal)
		if err != nil && !preflightBlocked && !oneOf(reason, Timeout, "REGRESSION_CANCELLED", DependencyBlocked) {
			c.Verdict = "failed"
			c.ReasonCode = reason
		}
		c.Phase = "finished"
		completeSteps(c, c.ReasonCode)
		c.MessageKey = ""
		if c.ReasonCode != "" {
			c.MessageKey = MessageKey(c.ReasonCode)
		}
		now := time.Now().UTC()
		c.FinishedAt = &now
		return nil, nil
	})
}
