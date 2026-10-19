package regression

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"time"
)

func JSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
func Redact(raw json.RawMessage) json.RawMessage {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return JSON(nil)
	}
	var clean func(any) any
	clean = func(x any) any {
		switch o := x.(type) {
		case map[string]any:
			for k, val := range o {
				switch strings.ToLower(k) {
				case "password", "secret", "token", "access_token", "refresh_token", "authorization", "credential", "private_key", "api_key", "access_key":
					delete(o, k)
				default:
					o[k] = clean(val)
				}
			}
		case []any:
			for i := range o {
				o[i] = clean(o[i])
			}
		}
		return x
	}
	return JSON(clean(v))
}
func Equal(a, b any) bool {
	var av, bv any
	if json.Unmarshal(JSON(a), &av) != nil || json.Unmarshal(JSON(b), &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// writeOnly is the public input-schema declaration for fields excluded from snapshots.
func RedactInput(raw, schema json.RawMessage) json.RawMessage {
	var value any
	var definition map[string]any
	if json.Unmarshal(raw, &value) != nil || json.Unmarshal(schema, &definition) != nil {
		return JSON(nil)
	}
	var visit func(any, map[string]any)
	visit = func(v any, s map[string]any) {
		if object, ok := v.(map[string]any); ok {
			properties, _ := s["properties"].(map[string]any)
			for k, item := range object {
				child, _ := properties[k].(map[string]any)
				if child["writeOnly"] == true {
					delete(object, k)
					continue
				}
				visit(item, child)
			}
		}
		if list, ok := v.([]any); ok {
			child, _ := s["items"].(map[string]any)
			for _, item := range list {
				visit(item, child)
			}
		}
	}
	visit(value, definition)
	return Redact(JSON(value))
}

func inputSecrets(raw, schema json.RawMessage) []string {
	var value any
	var definition map[string]any
	_ = json.Unmarshal(raw, &value)
	_ = json.Unmarshal(schema, &definition)
	secrets := []string{}
	var collect func(any)
	collect = func(v any) {
		switch x := v.(type) {
		case string:
			if x != "" {
				secrets = append(secrets, x)
			}
		case map[string]any:
			for _, item := range x {
				collect(item)
			}
		case []any:
			for _, item := range x {
				collect(item)
			}
		}
	}
	var visit func(any, map[string]any)
	visit = func(v any, s map[string]any) {
		if s["writeOnly"] == true {
			collect(v)
			return
		}
		switch x := v.(type) {
		case map[string]any:
			props, _ := s["properties"].(map[string]any)
			for k, item := range x {
				child, _ := props[k].(map[string]any)
				visit(item, child)
			}
		case []any:
			child, _ := s["items"].(map[string]any)
			for _, item := range x {
				visit(item, child)
			}
		}
	}
	visit(value, definition)
	return secrets
}
func (e *execution) redact(raw json.RawMessage) json.RawMessage {
	var value any
	_ = json.Unmarshal(Redact(raw), &value)
	var visit func(any) any
	visit = func(v any) any {
		switch x := v.(type) {
		case string:
			for _, s := range e.secrets {
				if s != "" && (x == s || len(s) >= 16 && strings.Contains(x, s)) {
					return nil
				}
			}
		case map[string]any:
			for k, item := range x {
				x[k] = visit(item)
			}
		case []any:
			for i, item := range x {
				x[i] = visit(item)
			}
		}
		return v
	}
	return JSON(visit(value))
}

func completeSteps(c *CaseExecution, reason string) {
	now := time.Now().UTC()
	for i := range c.Steps {
		if c.Steps[i].Status == "running" {
			c.Steps[i].Status = "failed"
			if oneOf(reason, Timeout, "REGRESSION_CANCELLED", Interrupted, DependencyBlocked) {
				c.Steps[i].Status = "blocked"
			}
			c.Steps[i].FinishedAt = &now
		}
	}
	for _, def := range c.DefinitionSnapshot.Steps {
		seen := false
		for _, step := range c.Steps {
			seen = seen || step.StepUUID == def.UUID
		}
		if !seen {
			c.Steps = append(c.Steps, StepResult{UUID: uuid.NewString(), StepUUID: def.UUID, Phase: def.Phase, TitleKey: def.TitleKey, Status: "skipped", Assertions: []AssertionResult{}})
		}
	}
}

type recorder struct {
	execution *execution
	index     int
	step      int
	err       error
}

func (r *recorder) current() *CaseExecution { return &r.execution.detail.CaseExecutions[r.index] }
func (r *recorder) mutate(ctx context.Context, kind string, fn func(*CaseExecution) (*AssertionResult, error)) error {
	e := r.execution
	e.mu.Lock()
	defer e.mu.Unlock()
	a, err := fn(r.current())
	if err != nil {
		r.err = err
		return err
	}
	c := r.current()
	c.Revision = e.detail.Run.Revision + 1
	c.UpdatedAt = time.Now().UTC()
	payload := map[string]any{"case_execution": c}
	if a != nil {
		payload["step_result_uuid"] = c.Steps[r.step].UUID
		payload["assertion"] = a
	}
	if err = e.commitLocked(ctx, kind, &c.UUID, payload); err != nil {
		r.err = err
	}
	return err
}
func (r *recorder) Step(ctx context.Context, id string, fn func(context.Context) error) error {
	if fn == nil {
		return NewError(InvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := r.mutate(ctx, "case.updated", func(c *CaseExecution) (*AssertionResult, error) {
		if r.err != nil {
			return nil, r.err
		}
		var def *StepDefinition
		for i := range c.DefinitionSnapshot.Steps {
			if c.DefinitionSnapshot.Steps[i].UUID == id {
				def = &c.DefinitionSnapshot.Steps[i]
				break
			}
		}
		if def == nil {
			return nil, NewError(InvalidRequest)
		}
		for _, s := range c.Steps {
			if s.StepUUID == id {
				return nil, NewError(InvalidRequest)
			}
		}
		now := time.Now().UTC()
		c.Steps = append(c.Steps, StepResult{UUID: uuid.NewString(), StepUUID: id, Phase: def.Phase, TitleKey: def.TitleKey, Status: "running", StartedAt: &now, Assertions: []AssertionResult{}})
		r.step = len(c.Steps) - 1
		return nil, nil
	})
	if err != nil {
		return err
	}
	err = fn(ctx)
	var assertionError error
	save := r.mutate(context.WithoutCancel(ctx), "case.updated", func(c *CaseExecution) (*AssertionResult, error) {
		s := &c.Steps[r.step]
		s.Status = "passed"
		if err != nil {
			s.Status = "failed"
		}
		for _, a := range s.Assertions {
			if a.Required && a.Status == "failed" {
				assertionError = NewError(AssertionFailed)
			}
			if a.Required && a.Status != "passed" && assertionError == nil {
				assertionError = NewError(DependencyBlocked)
			}
			if a.Status == "failed" {
				s.Status = "failed"
			} else if a.Status == "blocked" && s.Status != "failed" {
				s.Status = "blocked"
			}
		}
		now := time.Now().UTC()
		s.FinishedAt = &now
		r.step = -1
		return nil, nil
	})
	if err != nil {
		return err
	}
	if save == nil && assertionError != nil {
		return assertionError
	}
	return save
}
func (r *recorder) Assert(ctx context.Context, k, title, status string, expected, actual any, evidence []Evidence) error {
	return r.mutate(ctx, "assertion.recorded", func(c *CaseExecution) (*AssertionResult, error) {
		if r.step < 0 || !key(k) || !key(title) || !oneOf(status, "passed", "failed", "blocked", "skipped") {
			return nil, NewError(InvalidRequest)
		}
		for _, s := range c.Steps {
			for _, a := range s.Assertions {
				if a.Key == k {
					return nil, NewError(AssertionDuplicate)
				}
			}
		}
		required := false
		for _, v := range c.DefinitionSnapshot.RequiredAssertionKeys {
			if v == k {
				required = true
			}
		}
		reason := ""
		if required && status == "passed" && len(evidence) == 0 {
			status = "blocked"
			reason = EvidenceMissing
		}
		if status == "passed" && !Equal(expected, actual) {
			status = "failed"
		}
		for i := range evidence {
			v := &evidence[i]
			if !key(v.Kind) || !key(v.Boundary) || !key(v.SourceRef) || v.ObservedAt.IsZero() || !json.Valid(v.SanitizedData) {
				return nil, NewError(EvidenceMissing)
			}
			v.SanitizedData = r.execution.redact(v.SanitizedData)
		}
		if evidence == nil {
			evidence = []Evidence{}
		}
		a := AssertionResult{UUID: uuid.NewString(), Key: k, TitleKey: title, Status: status, Required: required, Expected: r.execution.redact(JSON(expected)), Actual: r.execution.redact(JSON(actual)), Evidence: evidence, ReasonCode: reason}
		c.Steps[r.step].Assertions = append(c.Steps[r.step].Assertions, a)
		return &a, nil
	})
}
func (r *recorder) Register(ctx context.Context, f FixtureResource) error {
	return r.mutate(ctx, "cleanup.updated", func(c *CaseExecution) (*AssertionResult, error) {
		if !ValidUUID(f.ResourceUUID) || !key(f.ResourceType) || !key(f.CleanupKey) {
			return nil, NewError(InvalidRequest)
		}
		var parameters map[string]json.RawMessage
		if json.Unmarshal(f.CleanupParameters, &parameters) != nil || parameters == nil {
			return nil, NewError(InvalidRequest)
		}
		f.UUID = uuid.NewString()
		f.TenantUUID = r.execution.actor.TenantUUID
		f.RunUUID = r.execution.detail.Run.UUID
		f.CaseExecutionUUID = c.UUID
		f.CleanupState = "pending"
		var displayName *string
		if err := json.Unmarshal(r.execution.redact(JSON(f.DisplayName)), &displayName); err != nil {
			return nil, NewError(InvalidRequest)
		}
		f.DisplayName = ""
		if displayName != nil {
			f.DisplayName = *displayName
		}
		f.CleanupEvidence = []Evidence{}
		f.CleanedAt = nil
		if f.ParentResourceUUID != "" {
			parentOwned := false
			for _, parent := range c.FixtureResources {
				parentOwned = parentOwned || parent.ResourceUUID == f.ParentResourceUUID
			}
			if !parentOwned || f.ParentResourceUUID == f.ResourceUUID {
				return nil, NewError(InvalidRequest)
			}
		}
		for _, old := range c.FixtureResources {
			if old.ResourceUUID == f.ResourceUUID && old.ResourceType == f.ResourceType {
				return nil, NewError(InvalidRequest)
			}
		}
		f.CleanupParameters = r.execution.redact(f.CleanupParameters)
		c.FixtureResources = append(c.FixtureResources, f)
		return nil, nil
	})
}
func (r *recorder) ListOwned() []FixtureResource {
	r.execution.mu.Lock()
	defer r.execution.mu.Unlock()
	return Clone(r.current().FixtureResources)
}
func (r *recorder) Cleaned(ctx context.Context, id string, evidence []Evidence) error {
	return r.mutate(ctx, "cleanup.updated", func(c *CaseExecution) (*AssertionResult, error) {
		if len(evidence) == 0 {
			return nil, NewError(EvidenceMissing)
		}
		for i := range evidence {
			v := &evidence[i]
			if v.Kind != "readback" || !key(v.Boundary) || !key(v.SourceRef) || v.ResourceUUID != id || v.ObservedAt.IsZero() || !json.Valid(v.SanitizedData) {
				return nil, NewError(EvidenceMissing)
			}
			v.SanitizedData = r.execution.redact(v.SanitizedData)
		}
		for i := range c.FixtureResources {
			f := &c.FixtureResources[i]
			if f.ResourceUUID == id && f.TenantUUID == r.execution.actor.TenantUUID && f.RunUUID == r.execution.detail.Run.UUID {
				f.CleanupState = "succeeded"
				f.CleanupEvidence = Clone(evidence)
				now := time.Now().UTC()
				f.CleanedAt = &now
				return nil, nil
			}
		}
		return nil, NewError(NotFound)
	})
}
func caseVerdict(c *CaseExecution, normal bool) string {
	failed, blocked := false, false
	seen := map[string]AssertionResult{}
	for _, step := range c.Steps {
		for _, a := range step.Assertions {
			seen[a.Key] = a
			if a.Required && a.Status == "failed" {
				failed = true
			}
			if a.Required && (a.Status != "passed" || len(a.Evidence) == 0) {
				blocked = true
			}
		}
	}
	c.MissingAssertionKeys = []string{}
	for _, k := range c.DefinitionSnapshot.RequiredAssertionKeys {
		if _, ok := seen[k]; !ok {
			c.MissingAssertionKeys = append(c.MissingAssertionKeys, k)
		}
	}
	if normal && len(c.MissingAssertionKeys) > 0 {
		failed = true
		c.ReasonCode = AssertionMissing
	}
	if c.CleanupState == "failed" {
		failed = true
		c.ReasonCode = CleanupFailed
	}
	if len(c.DefinitionSnapshot.RequiredAssertionKeys) == 0 {
		failed = true
		c.ReasonCode = AssertionMissing
	}
	if failed {
		return "failed"
	}
	if blocked || len(c.MissingAssertionKeys) > 0 || !normal {
		return "blocked"
	}
	return "passed"
}
