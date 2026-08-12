package manifest

import "testing"

func TestValidate(t *testing.T) {
	plugin := Plugin{
		ID:      "com.powerx.sample",
		Name:    "Sample",
		Version: "0.1.0",
		Menus: []Menu{{
			Path:  "/_p/com.powerx.sample/admin",
			Title: "Sample",
			Children: []Menu{
				{Path: "/_p/com.powerx.sample/admin/intro", Title: "Intro"},
			},
		}},
		Permissions: []string{"sample.admin:view"},
	}
	if err := Validate(plugin); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
}

func TestValidateAllowsRuntimeManifestWithoutPermissions(t *testing.T) {
	plugin := Plugin{
		ID:      "com.powerx.sample",
		Name:    "Sample",
		Version: "0.1.0",
		Menus: []Menu{{
			Path:  "/_p/com.powerx.sample/admin",
			Title: "Sample",
		}},
	}
	if err := Validate(plugin); err != nil {
		t.Fatalf("expected manifest without runtime permissions to be valid: %v", err)
	}
}
