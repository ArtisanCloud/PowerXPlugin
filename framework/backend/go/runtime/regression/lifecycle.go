package regression

import (
	"context"
	"encoding/json"
	"time"
)

func (r *Runner) Cancel(ctx context.Context, a Actor, id string) (RunDetail, error) {
	if _, err := r.Get(ctx, a, id); err != nil {
		return RunDetail{}, err
	}
	r.mu.Lock()
	e := r.active[id]
	r.mu.Unlock()
	if e == nil {
		return r.Store.Get(ctx, a.TenantUUID, id)
	}
	e.cancel()
	if err := e.runUpdate(func(v *Run) {
		if !Terminal(v.Phase) {
			v.Phase = "cancelling"
			v.ReasonCode = "REGRESSION_CANCELLED"
			v.MessageKey = MessageKey(v.ReasonCode)
		}
	}, "run.updated"); err != nil {
		return RunDetail{}, err
	}
	return r.Store.Get(ctx, a.TenantUUID, id)
}
func (r *Runner) Close(ctx context.Context) error {
	r.mu.Lock()
	r.closed = true
	active := []*execution{}
	for _, e := range r.active {
		active = append(active, e)
	}
	r.mu.Unlock()
	for _, e := range active {
		e.cancel()
	}
	for _, e := range active {
		select {
		case <-e.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func (r *Runner) Recover(ctx context.Context, a Actor) error {
	if err := ValidateActor(a); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rows, err := r.Store.Active(ctx, a.TenantUUID)
	if err != nil {
		return err
	}
	for _, d := range rows {
		if _, running := r.active[d.Run.UUID]; running {
			continue
		}
		if Terminal(d.Run.Phase) {
			d.Run.CleanupState = "failed"
		} else {
			d.Run.Phase = "interrupted"
			d.Run.Verdict = "blocked"
			d.Run.ReasonCode = Interrupted
			d.Run.MessageKey = MessageKey(Interrupted)
		}
		now := time.Now().UTC()
		d.Run.FinishedAt = &now
		d.Run.UpdatedAt = now
		d.Run.Revision++
		for i := range d.CaseExecutions {
			c := &d.CaseExecutions[i]
			if c.Phase != "finished" {
				c.Phase = "finished"
				c.Verdict = "blocked"
				c.ReasonCode = Interrupted
				c.MessageKey = MessageKey(Interrupted)
				c.FinishedAt = &now
				_ = caseVerdict(c, false)
				completeSteps(c, Interrupted)
			}
			c.Revision = d.Run.Revision
			c.UpdatedAt = now
			if len(c.FixtureResources) > 0 && c.CleanupState != "succeeded" {
				c.CleanupState = "failed"
				d.Run.CleanupState = "failed"
			}
		}
		if d.Run.CleanupState == "pending" {
			d.Run.CleanupState = "not_required"
		}
		e := newEvent(a, d, "run.finished", nil, map[string]any{"run": d.Run})
		if err = r.Store.Commit(ctx, a.TenantUUID, d, e); err != nil {
			return err
		}
		if r.observe != nil {
			r.observe(ctx, e, nil)
		}
	}
	return nil
}
func (r *Runner) Cleanup(ctx context.Context, a Actor, id string) (RunDetail, error) {
	d, err := r.Get(ctx, a, id)
	if err != nil {
		return d, err
	}
	if !Terminal(d.Run.Phase) {
		return d, NewError(Busy)
	}
	profile, pf, err := r.Registry.Profile(d.Run.ProfileUUID)
	if err != nil {
		return d, NewError(CleanupUnavailable)
	}
	if profile.BindingRevision != d.Run.ProfileSnapshot.BindingRevision {
		return d, NewError(CleanupUnavailable)
	}
	if d.Run.CleanupState == "succeeded" || d.Run.CleanupState == "not_required" {
		return d, nil
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return d, NewError(StoreUnavailable)
	}
	if r.busy {
		r.mu.Unlock()
		return d, NewError(Busy)
	}
	r.busy = true
	base, cancel := context.WithCancel(r.parent)
	e := &execution{runner: r, detail: d, actor: a, inputs: map[string]json.RawMessage{}, ctx: base, cancel: cancel, done: make(chan struct{}), secrets: []string{a.Credential}}
	r.active[id] = e
	r.mu.Unlock()
	if err = e.runUpdate(func(v *Run) { v.CleanupState = "running" }, "run.updated"); err != nil {
		cancel()
		r.mu.Lock()
		delete(r.active, id)
		r.busy = false
		r.mu.Unlock()
		close(e.done)
		return RunDetail{}, err
	}
	initial := Clone(e.detail)
	go func() {
		defer func() {
			cancel()
			e.mu.Lock()
			e.actor.Credential = ""
			e.secrets = nil
			e.mu.Unlock()
			r.mu.Lock()
			delete(r.active, id)
			r.busy = false
			r.mu.Unlock()
			close(e.done)
		}()
		for i := range e.detail.CaseExecutions {
			c := &e.detail.CaseExecutions[i]
			if c.CleanupState != "failed" && c.CleanupState != "pending" {
				continue
			}
			current, factory, factoryErr := r.Registry.Case(c.CaseUUID)
			if factoryErr == nil && current.Version != c.DefinitionSnapshot.Version {
				factoryErr = NewError(CleanupUnavailable)
			}
			scope := &Scope{RunUUID: id, CaseExecutionUUID: c.UUID, Actor: a, Definition: c.DefinitionSnapshot, Profile: profile, Input: c.InputSnapshot, Recorder: &recorder{execution: e, index: i, step: -1}}
			rec := scope.Recorder.(*recorder)
			_ = rec.mutate(context.WithoutCancel(base), "cleanup.updated", func(v *CaseExecution) (*AssertionResult, error) { v.CleanupState = "running"; return nil, nil })
			cleanCtx, stop := context.WithTimeout(base, time.Duration(c.DefinitionSnapshot.CleanupTimeoutMS)*time.Millisecond)
			err := factoryErr
			if err == nil {
				scope.Dependencies, _, err = invokeProfile(cleanCtx, pf, a)
			}
			if err == nil {
				var scene Scenario
				scene, err = construct(factory)
				if err == nil {
					err = invoke(cleanCtx, scene.Cleanup, scope)
				}
			}
			if cleanCtx.Err() != nil {
				err = cleanCtx.Err()
			}
			stop()
			_ = rec.mutate(context.WithoutCancel(base), "cleanup.updated", func(v *CaseExecution) (*AssertionResult, error) {
				v.CleanupState = "succeeded"
				if err != nil {
					v.CleanupState = "failed"
				}
				for _, f := range v.FixtureResources {
					if f.CleanupState != "succeeded" {
						v.CleanupState = "failed"
					}
				}
				return nil, nil
			})
		}
		_ = e.runUpdate(func(v *Run) {
			v.CleanupState = "succeeded"
			for _, c := range e.detail.CaseExecutions {
				if c.CleanupState == "failed" {
					v.CleanupState = "failed"
				}
			}
		}, "run.updated")
	}()
	return initial, nil
}
