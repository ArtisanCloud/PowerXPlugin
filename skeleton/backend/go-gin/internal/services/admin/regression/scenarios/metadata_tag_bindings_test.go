package scenarios

import (
	"context"
	Metadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"github.com/google/uuid"
	"testing"
)

type hostStub struct {
	Metadata.Service
	bindings []Metadata.TagBinding
	denied   bool
	target   string
}

func (h *hostStub) ResolveResourceType(context.Context, string) (*Metadata.ResourceType, error) {
	return &Metadata.ResourceType{BindingEnabled: true, ValidatorStatus: "available"}, nil
}
func (h *hostStub) ListTagBindings(context.Context, Metadata.ListTagBindingsRequest) ([]Metadata.TagBinding, error) {
	return h.bindings, nil
}
func (h *hostStub) ReplaceTagBindings(ctx context.Context, in Metadata.ReplaceTagBindingsRequest) ([]Metadata.TagBinding, error) {
	if h.denied {
		return nil, Regression.NewError(Regression.DependencyBlocked)
	}
	if in.ResourceUUID != h.target {
		return nil, Regression.NewError(Regression.InvalidRequest)
	}
	h.bindings = []Metadata.TagBinding{}
	for _, tag := range in.TagUUIDs {
		h.bindings = append(h.bindings, Metadata.TagBinding{UUID: uuid.NewString(), TagUUID: tag, ResourceUUID: in.ResourceUUID, ResourceType: in.ResourceType})
	}
	return h.bindings, nil
}
func TestMetadataScenarioUsesBoundFixtureAndRestoresOriginal(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(map[bool]string{false: "granted", true: "denied"}[denied], func(t *testing.T) {
			ctx := context.Background()
			tenant, resource, tag, original := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
			host := &hostStub{target: resource, denied: denied, bindings: []Metadata.TagBinding{{TagUUID: original}}}
			registry := Regression.NewRegistry()
			if err := registry.RegisterSuite(Regression.SuiteDefinition{UUID: SuiteUUID, Key: "business", TitleKey: "regression.suites.business", DescriptionKey: "regression.suites.description"}); err != nil {
				t.Fatal(err)
			}
			if err := registry.RegisterScenario(func() Regression.Scenario { return &MetadataTagBindings{} }); err != nil {
				t.Fatal(err)
			}
			// Contract-level fake verifies the business scenario; real Core authorization is a separate acceptance.
			scope := &Regression.Scope{Actor: Regression.Actor{TenantUUID: tenant}, Dependencies: &MetadataDependencies{Service: host, ResourceType: "fixture.resource", ResourceUUID: resource}, Input: Regression.JSON(map[string]any{"resource_type": "fixture.resource", "tag_uuid": tag})}
			scene := &MetadataTagBindings{}
			if err := scene.Preflight(ctx, scope); err != nil {
				t.Fatal(err)
			}
			if scene.resourceUUID != resource || scene.originalTags[0] != original {
				t.Fatal(Regression.InvalidRequest)
			}
			records := &captureRecorder{tenant: tenant, run: uuid.NewString(), caseID: uuid.NewString()}
			scope.Recorder = records
			scope.RunUUID = records.run
			scope.CaseExecutionUUID = records.caseID
			if err := scene.Prepare(ctx, scope); err != nil {
				t.Fatal(err)
			}
			err := scene.Execute(ctx, scope)
			if denied {
				if Regression.CodeOf(err) != Regression.DependencyBlocked || len(records.resources) != 0 {
					t.Fatal(err)
				}
			} else {
				if err != nil || records.assertions != 3 {
					t.Fatal(err, records.assertions)
				}
			}
			if err := scene.Cleanup(ctx, scope); err != nil {
				t.Fatal(err)
			}
			if !denied && (host.bindings[0].TagUUID != original || records.resources[0].CleanupState != "succeeded") {
				t.Fatal(Regression.CleanupFailed)
			}

		})
	}
}

type captureRecorder struct {
	tenant, run, caseID string
	resources           []Regression.FixtureResource
	assertions          int
}

func (c *captureRecorder) Step(ctx context.Context, id string, fn func(context.Context) error) error {
	return fn(ctx)
}
func (c *captureRecorder) Assert(ctx context.Context, key, title, status string, expected, actual any, evidence []Regression.Evidence) error {
	if status != "passed" || !Regression.Equal(expected, actual) || len(evidence) == 0 || evidence[0].Boundary != "core.host" {
		return Regression.NewError(Regression.AssertionFailed)
	}
	c.assertions++
	return nil
}
func (c *captureRecorder) Register(ctx context.Context, f Regression.FixtureResource) error {
	f.TenantUUID = c.tenant
	f.RunUUID = c.run
	f.CaseExecutionUUID = c.caseID
	f.CleanupState = "pending"
	c.resources = append(c.resources, f)
	return nil
}
func (c *captureRecorder) ListOwned() []Regression.FixtureResource { return c.resources }
func (c *captureRecorder) Cleaned(ctx context.Context, id string, evidence []Regression.Evidence) error {
	for i := range c.resources {
		if c.resources[i].ResourceUUID == id {
			c.resources[i].CleanupState = "succeeded"
			c.resources[i].CleanupEvidence = evidence
			return nil
		}
	}
	return Regression.NewError(Regression.NotFound)
}
