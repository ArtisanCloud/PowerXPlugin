package rbac

import "testing"

func TestValidate(t *testing.T) {
	perms := []Permission{{Key: "com.powerx.sample.admin.view"}}
	if err := Validate(perms); err != nil {
		t.Fatalf("permission should be valid: %v", err)
	}
}
