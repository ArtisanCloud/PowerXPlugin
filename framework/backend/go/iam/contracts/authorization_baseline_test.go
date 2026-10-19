package contracts

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

func baselineQuery() AuthorizationBaselineQuery {
	return AuthorizationBaselineQuery{TenantUUID: "20000000-0000-4000-8000-000000000001", MemberUUIDs: []string{"10000000-0000-4000-8000-000000000001"}, PermissionCodes: []string{"orders:read", "cost:read", "scope:read"}}
}

func baselineFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/authorization_baseline_result.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAuthorizationBaselineBranchesAndPrecision(t *testing.T) {
	raw := baselineFixture(t)
	result, err := DecodeAuthorizationBaselineResult(raw, baselineQuery(), "com.powerx.fixture")
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].GrantedAt != nil || result.Items[0].VisibleSince.Nanosecond() != 1000 || result.Items[1].Allowed == nil || *result.Items[1].Allowed || result.Items[2].Allowed != nil {
		t.Fatal(result)
	}
	out, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(out), "2026-10-15T02:00:00.000001Z") {
		t.Fatal(err, string(out))
	}
	if _, err = DecodeAuthorizationBaselineResult(out, baselineQuery(), "com.powerx.fixture"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"native", "reconstructed"} {
		t.Run(kind, func(t *testing.T) {
			value := strings.ReplaceAll(string(raw), `"baseline_kind": "initialized"`, `"baseline_kind": "`+kind+`"`)
			value = strings.Replace(value, `"granted_at": null`, `"granted_at": "2026-10-15T01:00:00.000001Z"`, 1)
			if _, err := DecodeAuthorizationBaselineResult([]byte(value), baselineQuery(), "com.powerx.fixture"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAuthorizationBaselineRejectsInvalidResults(t *testing.T) {
	tests := map[string]func(map[string]any, []any){
		"schema":           func(r map[string]any, _ []any) { r["schema"] = "v0" },
		"tenant":           func(r map[string]any, _ []any) { r["tenant_uuid"] = uuid.NewString() },
		"plugin":           func(r map[string]any, _ []any) { r["plugin_id"] = "com.powerx.other" },
		"unknown":          func(r map[string]any, _ []any) { r["extra"] = false },
		"missing":          func(r map[string]any, items []any) { r["items"] = items[:2] },
		"duplicate":        func(_ map[string]any, items []any) { items[2] = items[0] },
		"extra_member":     func(_ map[string]any, items []any) { items[0].(map[string]any)["member_uuid"] = uuid.NewString() },
		"extra_permission": func(_ map[string]any, items []any) { items[0].(map[string]any)["permission_code"] = "other:read" },
		"missing_allowed":  func(_ map[string]any, items []any) { delete(items[1].(map[string]any), "allowed") },
		"null_allowed":     func(_ map[string]any, items []any) { items[1].(map[string]any)["allowed"] = nil },
		"error_false":      func(_ map[string]any, items []any) { items[2].(map[string]any)["allowed"] = false },
		"error_null":       func(_ map[string]any, items []any) { items[2].(map[string]any)["visible_since"] = nil },
		"error_code":       func(_ map[string]any, items []any) { items[2].(map[string]any)["error_code"] = "UNKNOWN" },
		"revision":         func(_ map[string]any, items []any) { delete(items[0].(map[string]any), "authorization_revision") },
		"native_null":      func(_ map[string]any, items []any) { items[0].(map[string]any)["baseline_kind"] = "native" },
		"initialized_date": func(_ map[string]any, items []any) {
			items[0].(map[string]any)["granted_at"] = "2026-10-15T01:00:00.000001Z"
		},
		"missing_nullable": func(_ map[string]any, items []any) { delete(items[0].(map[string]any), "valid_until") },
		"millisecond":      func(r map[string]any, _ []any) { r["evaluated_at"] = "2026-10-15T02:15:00.001Z" },
		"offset":           func(r map[string]any, _ []any) { r["evaluated_at"] = "2026-10-15T10:15:00.000001+08:00" },
		"number_time":      func(r map[string]any, _ []any) { r["evaluated_at"] = 1792029600000 },
		"future_lower": func(_ map[string]any, items []any) {
			items[0].(map[string]any)["visible_since"] = "2026-10-15T02:16:00.000001Z"
		},
		"early_lower": func(_ map[string]any, items []any) {
			items[0].(map[string]any)["visible_since"] = "2026-10-15T01:59:00.000001Z"
		},
		"expired": func(_ map[string]any, items []any) {
			items[0].(map[string]any)["valid_until"] = "2026-10-15T02:15:00.000001Z"
		},
		"recheck": func(_ map[string]any, items []any) {
			items[0].(map[string]any)["recheck_at"] = "2026-10-15T02:15:00.000001Z"
		},
		"scope":       func(_ map[string]any, items []any) { items[0].(map[string]any)["data_scope"] = "SELF" },
		"fingerprint": func(_ map[string]any, items []any) { items[0].(map[string]any)["scope_fingerprint"] = "opaque" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			var result map[string]any
			if err := json.Unmarshal(baselineFixture(t), &result); err != nil {
				t.Fatal(err)
			}
			mutate(result, result["items"].([]any))
			raw, _ := json.Marshal(result)
			if out, err := DecodeAuthorizationBaselineResult(raw, baselineQuery(), "com.powerx.fixture"); out != nil || !iamerrors.IsCode(err, iamerrors.CodeBaselineResponseInvalid) {
				t.Fatal(out, err)
			}
		})
	}
	for _, raw := range []string{string(baselineFixture(t)) + `{}`, strings.Replace(string(baselineFixture(t)), `"allowed": true`, `"allowed": false,"allowed": true`, 1)} {
		if _, err := DecodeAuthorizationBaselineResult([]byte(raw), baselineQuery(), "com.powerx.fixture"); !iamerrors.IsCode(err, iamerrors.CodeBaselineResponseInvalid) {
			t.Fatal(err)
		}
	}
}

func TestAuthorizationBaselineQueryBoundsAndCanonicalDuplicates(t *testing.T) {
	queries := []AuthorizationBaselineQuery{baselineQuery(), baselineQuery(), baselineQuery(), baselineQuery(), baselineQuery(), baselineQuery()}
	queries[0].TenantUUID = ""
	queries[1].MemberUUIDs = []string{uuid.Nil.String()}
	queries[2].MemberUUIDs = append(queries[2].MemberUUIDs, "{10000000-0000-4000-8000-000000000001}")
	queries[3].PermissionCodes = []string{"orders:read", "orders:read"}
	queries[4].PermissionCodes = []string{" orders:read"}
	queries[5].MemberUUIDs = nil
	for _, q := range queries {
		if _, err := q.Normalize(); !iamerrors.IsCode(err, iamerrors.CodeBaselineInvalidArgument) {
			t.Fatal(q, err)
		}
	}
	q := baselineQuery()
	q.MemberUUIDs = nil
	for range 100 {
		q.MemberUUIDs = append(q.MemberUUIDs, uuid.NewString())
	}
	if _, err := q.Normalize(); err != nil {
		t.Fatal(err)
	}
	q.MemberUUIDs = append(q.MemberUUIDs, uuid.NewString())
	if _, err := q.Normalize(); err == nil {
		t.Fatal(q)
	}
	q = baselineQuery()
	q.PermissionCodes = nil
	for i := range 11 {
		q.PermissionCodes = append(q.PermissionCodes, "scope"+strings.Repeat("x", i)+":read")
	}
	if _, err := q.Normalize(); err == nil {
		t.Fatal(q)
	}
}
