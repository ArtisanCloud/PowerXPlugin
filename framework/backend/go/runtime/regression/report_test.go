package regression

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
)

func TestReportsPreserveVerdictAndRedactDeclaredInputs(t *testing.T) {
	input := RedactInput(JSON(map[string]any{"public": "fixture.value", "opaque": "fixture.secret", "nested": map[string]any{"token": "fixture.token"}}), JSON(map[string]any{"properties": map[string]any{"opaque": map[string]any{"writeOnly": true}}}))
	if strings.Contains(string(input), "fixture.secret") || strings.Contains(string(input), "fixture.token") {
		t.Fatal(EvidenceMissing)
	}
	d := RunDetail{Run: Run{Phase: "completed", Verdict: "failed", InputSnapshot: input}, CaseExecutions: []CaseExecution{{DefinitionSnapshot: CaseDefinition{Key: "passed"}, Verdict: "passed"}, {DefinitionSnapshot: CaseDefinition{Key: "failed"}, Verdict: "failed", ReasonCode: AssertionMissing}, {DefinitionSnapshot: CaseDefinition{Key: "blocked"}, Verdict: "blocked", ReasonCode: DependencyBlocked}}}
	report := Export(d, []RegressionEvent{})
	raw, err := json.Marshal(report)
	if err != nil || report.SchemaVersion != "v1" || !strings.Contains(string(raw), "completed") {
		t.Fatal(err)
	}
	b, err := JUnit(d)
	if err != nil {
		t.Fatal(err)
	}
	var suite junitSuite
	if err = xml.Unmarshal(b, &suite); err != nil || suite.Cases[0].Failure != nil || suite.Cases[1].Failure == nil || suite.Cases[2].Skipped == nil {
		t.Fatal(string(b), err)
	}
}
