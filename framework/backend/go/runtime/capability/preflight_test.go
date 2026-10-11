package capability

import (
	"context"
	"errors"
	"fmt"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"testing"
)

type nilUnsafeGrantChecker struct{ calls int }

func (c *nilUnsafeGrantChecker) GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	c.calls++
	return nil, nil
}

func TestRequireGrantsRejectsTypedNilBeforeCallingAdapter(t *testing.T) {
	var pointer *nilUnsafeGrantChecker
	var function checkFunc
	for _, checker := range []GrantChecker{pointer, function} {
		err := RequireGrants(t.Context(), checker, []string{"cap.a"})
		var moduleErr *module.Error
		if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
			t.Fatalf("checker=%T err=%v", checker, err)
		}
	}
}

type checkFunc func(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error)

func (f checkFunc) GrantStatus(ctx context.Context, in powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	return f(ctx, in)
}

func TestRequireGrantsValidationAndErrors(t *testing.T) {
	upstream := errors.New("test_upstream")
	for _, tc := range []struct {
		name, status, id, reason string
		empty                    bool
		err                      error
		want                     bool
	}{
		{"granted", "granted", "cap.a", "CAPABILITY_GRANTED", false, nil, false},
		{"denied", "not_granted", "cap.a", "CAPABILITY_NOT_GRANTED", false, nil, true},
		{"unknown", "unknown", "cap.a", "CAPABILITY_UNKNOWN", false, nil, true},
		{"wrong_id", "granted", "cap.b", "CAPABILITY_GRANTED", false, nil, true},
		{"no_reason", "granted", "cap.a", "", false, nil, true},
		{"invalid_status", "success", "cap.a", "CAPABILITY_GRANTED", false, nil, true},
		{"missing", "", "", "", true, nil, true},
		{"upstream", "", "", "", true, upstream, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := checkFunc(func(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
				if tc.empty {
					return nil, tc.err
				}
				return []powerxcapability.GrantStatusItem{{CapabilityID: tc.id, Status: tc.status, ReasonCode: tc.reason}}, tc.err
			})
			err := RequireGrants(t.Context(), checker, []string{"cap.a"})
			if (err != nil) != tc.want {
				t.Fatalf("err=%v", err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("lost upstream error: %v", err)
			}
		})
	}
	for _, ids := range [][]string{{""}, {" cap.a"}, {"cap.a", "cap.a"}, {"cap.a"}} {
		if err := RequireGrants(t.Context(), nil, ids); err == nil {
			t.Fatalf("accepted %v", ids)
		}
	}
	if err := RequireGrants(t.Context(), nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRequireGrantsBatchesOnlyExplicitRequirements(t *testing.T) {
	ids := make([]string, 201)
	for i := range ids {
		ids[i] = fmt.Sprintf("cap.%d", i)
	}
	calls := 0
	checker := checkFunc(func(_ context.Context, in powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
		calls++
		if len(in.CapabilityIDs) > 100 {
			t.Fatal("batch exceeds Core contract")
		}
		items := make([]powerxcapability.GrantStatusItem, len(in.CapabilityIDs))
		for i, id := range in.CapabilityIDs {
			items[i] = powerxcapability.GrantStatusItem{CapabilityID: id, Status: "granted", ReasonCode: "CAPABILITY_GRANTED"}
		}
		return items, nil
	})
	if err := RequireGrants(t.Context(), checker, ids); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
	}
}
