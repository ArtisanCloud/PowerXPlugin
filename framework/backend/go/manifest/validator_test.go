package manifest

import "testing"

func TestValidate(t *testing.T) {
    plugin := Plugin{
        ID:   "com.powerx.sample",
        Name: "Sample",
        Version: "0.1.0",
        Menus: []Menu{{
            Path:  "/_p/com.powerx.sample/admin",
            Title: "Sample",
        }},
        Permissions: []string{"com.powerx.sample.admin.view"},
    }
    if err := Validate(plugin); err != nil {
        t.Fatalf("expected valid manifest: %v", err)
    }
}
