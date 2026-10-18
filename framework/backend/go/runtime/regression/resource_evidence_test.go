package regression

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestResourceEvidencePreservesNamesRelationsAndReadback(t *testing.T) {
	parent, child := uuid.NewString(), uuid.NewString()
	r, actor := testRunner(t, func(ctx context.Context, scope *Scope) error {
		for _, resource := range []FixtureResource{
			{ResourceUUID: parent, DisplayName: "source", RoleKey: "regression.resourceRoles.source"},
			{ResourceUUID: child, DisplayName: "copy", RoleKey: "regression.resourceRoles.copy", ParentResourceUUID: parent},
		} {
			resource.ResourceType, resource.CleanupKey, resource.CleanupParameters = "fixture", "fixture.delete", JSON(map[string]any{})
			if err := scope.Recorder.Register(ctx, resource); err != nil {
				return err
			}
		}
		return scope.Recorder.Step(ctx, stepID, func(ctx context.Context) error {
			return scope.Recorder.Assert(ctx, "fact.present", "regression.fixture.fact", "passed", true, true, proof())
		})
	}, func(ctx context.Context, scope *Scope) error {
		for _, resource := range scope.Recorder.ListOwned() {
			if err := scope.Recorder.Cleaned(ctx, resource.ResourceUUID, cleanupProof(resource.ResourceUUID)); err != nil {
				return err
			}
		}
		return nil
	})
	detail := submit(t, r, actor)
	final := await(t, r, actor, detail.Run.UUID)
	settle(t, r)
	resources := final.CaseExecutions[0].FixtureResources
	if final.Run.Verdict != "passed" || len(resources) != 2 || resources[1].ParentResourceUUID != parent || resources[1].DisplayName != "copy" {
		t.Fatal(final)
	}
	for _, resource := range resources {
		if resource.CleanupState != "succeeded" || resource.CleanedAt == nil || len(resource.CleanupEvidence) != 1 || resource.CleanupEvidence[0].ResourceUUID != resource.ResourceUUID {
			t.Fatal(resource)
		}
	}
}

func TestResourceEvidenceRejectsUnownedParentAndUnprovenCleanup(t *testing.T) {
	for _, mode := range []string{"unowned_parent", "no_proof", "wrong_resource", "not_readback", "secret_name"} {
		t.Run(mode, func(t *testing.T) {
			resourceID := uuid.NewString()
			r, actor := testRunner(t, func(ctx context.Context, scope *Scope) error {
				recorder := scope.Recorder.(*recorder)
				recorder.execution.secrets = []string{"fixture-private-credential"}
				resource := FixtureResource{ResourceType: "fixture", ResourceUUID: resourceID, DisplayName: "fixture-private-credential", CleanupKey: "fixture.delete", CleanupParameters: JSON(map[string]any{})}
				if mode == "unowned_parent" {
					resource.ParentResourceUUID = uuid.NewString()
				}
				err := scope.Recorder.Register(ctx, resource)
				if mode == "unowned_parent" {
					if CodeOf(err) != InvalidRequest {
						t.Errorf("parent: %v", err)
					}
					return err
				}
				if err != nil {
					return err
				}
				if scope.Recorder.ListOwned()[0].DisplayName != "" {
					t.Error("secret in resource name")
				}
				evidence := cleanupProof(resourceID)
				switch mode {
				case "no_proof":
					evidence = nil
				case "wrong_resource":
					evidence[0].ResourceUUID = uuid.NewString()
				case "not_readback":
					evidence[0].Kind = "request"
				}
				err = scope.Recorder.Cleaned(ctx, resourceID, evidence)
				if mode != "secret_name" && CodeOf(err) != EvidenceMissing {
					t.Errorf("proof: %v", err)
				}
				return err
			}, nil)
			detail := submit(t, r, actor)
			final := await(t, r, actor, detail.Run.UUID)
			settle(t, r)
			for _, resource := range final.CaseExecutions[0].FixtureResources {
				if mode != "secret_name" && (resource.CleanupState == "succeeded" || resource.CleanedAt != nil) {
					t.Fatal(resource)
				}
			}
		})
	}
}
