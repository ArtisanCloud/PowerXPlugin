package regression

import (
	"context"
	"encoding/json"
	"testing"
)

const suiteID = "11111111-1111-4111-8111-111111111111"
const caseID = "22222222-2222-4222-8222-222222222222"
const stepID = "33333333-3333-4333-8333-333333333333"
const profileID = "44444444-4444-4444-8444-444444444444"

func definition() CaseDefinition {
	return CaseDefinition{UUID: caseID, SuiteUUID: suiteID, Key: "fixture", Version: "v1", TitleKey: "regression.fixture.title", DescriptionKey: "regression.fixture.description", Tags: []string{}, SupportedProfileKinds: []string{"controlled"}, InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false}`), ExecutionTimeoutMS: 1000, CleanupTimeoutMS: 1000, Steps: []StepDefinition{{UUID: stepID, Key: "verify", TitleKey: "regression.fixture.verify", Phase: "verify", Boundary: "service"}}, RequiredAssertionKeys: []string{"fact.present"}}
}

type testScenario struct{ def CaseDefinition }

func (s *testScenario) Definition() CaseDefinition            { return s.def }
func (*testScenario) Preflight(context.Context, *Scope) error { return nil }
func (*testScenario) Prepare(context.Context, *Scope) error   { return nil }
func (*testScenario) Execute(context.Context, *Scope) error   { return nil }
func (*testScenario) Cleanup(context.Context, *Scope) error   { return nil }
func TestRegistryRejectsIncompleteAndDuplicateDefinitions(t *testing.T) {
	edits := []func(*CaseDefinition){func(d *CaseDefinition) { d.UUID = "1" }, func(d *CaseDefinition) { d.RequiredAssertionKeys = nil }, func(d *CaseDefinition) { d.RequiredAssertionKeys = []string{"fact.present", "fact.present"} }, func(d *CaseDefinition) { d.RequiredAssertionKeys = []string{" "} }, func(d *CaseDefinition) { d.Steps = nil }, func(d *CaseDefinition) { d.CleanupTimeoutMS = 0 }, func(d *CaseDefinition) { d.InputSchema = json.RawMessage(`{"$ref":"https://invalid.example/schema"}`) }}
	for i, edit := range edits {
		r := NewRegistry()
		if err := r.RegisterSuite(SuiteDefinition{UUID: suiteID, Key: "fixture", TitleKey: "regression.fixture.title", DescriptionKey: "regression.fixture.description"}); err != nil {
			t.Fatal(err)
		}
		d := definition()
		edit(&d)
		if err := r.RegisterScenario(func() Scenario { return &testScenario{def: d} }); err == nil {
			t.Fatalf("%d", i)
		}
	}
	r := NewRegistry()
	_ = r.RegisterSuite(SuiteDefinition{UUID: suiteID, Key: "fixture", TitleKey: "regression.fixture.title", DescriptionKey: "regression.fixture.description"})
	d := definition()
	factory := func() Scenario { return &testScenario{def: d} }
	if err := r.RegisterScenario(factory); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterScenario(factory); err == nil {
		t.Fatal(InvalidRequest)
	}
}
func TestRegistryFreezesDefinitionsAndValidatesInputs(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterSuite(SuiteDefinition{UUID: suiteID, Key: "fixture", TitleKey: "regression.fixture.title", DescriptionKey: "regression.fixture.description"})
	d := definition()
	if err := r.RegisterScenario(func() Scenario { return &testScenario{def: d} }); err != nil {
		t.Fatal(err)
	}
	d.RequiredAssertionKeys[0] = "mutated"
	items := r.Cases()
	if items[0].RequiredAssertionKeys[0] != "fact.present" {
		t.Fatal(InvalidRequest)
	}
	items[0].RequiredAssertionKeys[0] = "mutated"
	if r.Cases()[0].RequiredAssertionKeys[0] != "fact.present" {
		t.Fatal(InvalidRequest)
	}
	if err := r.ValidateInput(caseID, json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := r.ValidateInput(caseID, json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Fatal(InvalidRequest)
	}
}

func TestRegistryRequiresAvailableProfileFactory(t *testing.T) {
	r := NewRegistry()
	p := ProfileDefinition{UUID: profileID, Key: "controlled", TitleKey: "regression.profiles.controlled", Kind: "controlled", FixtureNamespace: "regression", BindingRevision: "v1"}
	if err := r.RegisterProfile(p, nil); err == nil {
		t.Fatal(InvalidRequest)
	}
	p.Kind = "live"
	p.DependencyDeclarations = []DependencyDeclaration{{DependencyKey: "core", Mode: "test_double", Boundary: "core.host"}}
	if err := r.RegisterProfile(p, func(context.Context, Actor) (any, []DependencyEvidence, error) { return nil, nil, nil }); err == nil {
		t.Fatal(InvalidRequest)
	}
}
